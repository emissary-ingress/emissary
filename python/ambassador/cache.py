from typing import Any, Dict, Callable, Optional, Set, Tuple, TYPE_CHECKING

import logging

class Cacheable(dict):
    """
    A dictionary that is specifically cacheable, by way of its added
    cache_key property.
    """

    _cache_key: Optional[str]

    @property
    def cache_key(self) -> str:
        if not self._cache_key:
            raise RuntimeError("cache key is not set")

        return self._cache_key

    @cache_key.setter
    def cache_key(self, cache_key: str) -> None:
        self._cache_key = cache_key


DeletionHandler = Callable[[Cacheable], None]
CacheEntry = Tuple[Cacheable, Optional[DeletionHandler]]
CacheLink = Set[str]


class Cache():
    """
    A cache of Cacheables, supporting add/delete/fetch and also linking
    an owning Cacheable to an owned Cacheable. Deletion is cascaded: if you
    delete something, everything it owns is recursively deleted too. THIS ONLY
    HAPPENS IN ONE DIRECTION at present, so deleting a Cacheable in the middle
    of the ownership tree can leave dangling pointers.
    """

    def __init__(self, logger: logging.Logger) -> None:
        self.cache: Dict[str, CacheEntry] = {}
        self.links: Dict[str, CacheLink] = {}
        self.logger = logger

        self.reset_stats()

        self.logger.debug("Cache initialized")

    def reset_stats(self) -> None:
        self.hits = 0
        self.misses = 0
        self.invalidate_calls = 0
        self.invalidated_objects = 0

    @staticmethod
    def fn_name(fn: Optional[Callable]) -> str:
        return fn.__name__ if (fn and fn.__name__) else "-none-"

    def add(self, rsrc: Cacheable,
            on_delete: Optional[DeletionHandler]=None) -> None:
        """
        Adds an entry to the cache, if it's not already present. If
        on_delete is not None, it will called when rsrc is removed from
        the cache.
        """

        key = rsrc.cache_key

        if not key:
            self.logger.info(f"CACHE: ignore, no cache_key: {rsrc}")
        elif key in self.cache:
            # self.logger.info(f"CACHE: ignore, already present: {rsrc}")
            pass
        else:
            self.logger.debug(f"CACHE: adding {key}: {rsrc}, on_delete {self.fn_name(on_delete)}")

            self.cache[key] = (rsrc, on_delete)

    def link(self, owner: Cacheable, owned: Cacheable) -> None:
        """
        Adds a link to the cache. Links are directional from the owner to
        the owned. The basic idea is that if the owner changes, all the owned
        things get invalidated. Both the owner and the owned must be in the
        cache.
        """

        owner_key = owner.cache_key
        owned_key = owned.cache_key

        if not owner_key:
            self.logger.info(f"CACHE: cannot link, owner has no key: {owner}")
            return

        if not owned_key:
            self.logger.info(f"CACHE: cannot link, owned has no key: {owned}")
            return

        if owner_key not in self.cache:
            self.logger.info(f"CACHE: cannot link, owner not cached: {owner}")
            return

        if owned_key not in self.cache:
            self.logger.info(f"CACHE: cannot link, owned not cached: {owned}")
            return

        # self.logger.info(f"CACHE: linking {owner_key} -> {owned_key}")

        links = self.links.setdefault(owner_key, set())
        links.update([ owned_key ])

    def invalidate(self, key: str) -> None:
        """
        Recursively invalidate the entry named by 'key' and everything to which it
        is linked.
        """

        # We use worklist to keep track of things to consider: for starters,
        # it just has our key in it, and as we find owned things, we add them
        # to the worklist to consider.
        #
        # Note that word "consider". If you want to invalidate something from
        # the cache that isn't in the cache, that's not an error -- it'll be
        # silently ignored. That helps with dangling links (e.g. if two Mappings
        # both link to the same Group, and you invalidate the first Mapping, the
        # second will have a dangling link to the now-invalidated Group, and that
        # needs to not break anything).

        self.invalidate_calls += 1

        worklist = [ key ]

        # Under the hood, "invalidating" something from this cache is really
        # deleting it, so we'll use "to_delete" for the set of things we're going
        # to, y'knom, delete. We find all the resources we're going to work with
        # before deleting any of them, because I get paranoid about modifying a
        # data structure while I'm trying to traverse it.
        to_delete: Dict[str, CacheEntry] = {}

        # Keep going until we have nothing else to do.
        while worklist:
            # Pop off the first thing...
            key = worklist.pop(0)

            # ...and check if it's in the cache.
            if key in self.cache:
                # It is, good. We can append it to our set of things to delete...
                rsrc, on_delete = self.cache[key]

                self.logger.debug(f"CACHE: DEL {key}: will delete {rsrc}")

                if key not in to_delete:
                    # We haven't seen this key, so remember to delete it...
                    to_delete[key] = (rsrc, on_delete)

                    # ...and then toss all of its linked objects on our list to
                    # consider.
                    if key in self.links:
                        for owned in sorted(self.links[key]):
                            self.logger.debug(f"CACHE: DEL {key}: will check owned {owned}")
                            worklist.append(owned)

                    # (If we have seen the key already, just ignore it and go to the next
                    # key in the worklist. This is important to not get stuck if we somehow
                    # get a circular link list.)

        # OK, we have a set of things to delete. Get to it.
        for key, rdh in to_delete.items():
            self.logger.debug(f"CACHE: DEL {key}: smiting!")

            self.invalidated_objects += 1
            del(self.cache[key])

            if key in self.links:
                del(self.links[key])

            rsrc, on_delete = rdh

            if on_delete:
                self.logger.debug(f"CACHE: DEL {key}: calling {self.fn_name(on_delete)}")
                on_delete(rsrc)

    def __getitem__(self, key: str) -> Optional[Cacheable]:
        """
        Fetches only the _resource_ for a given key from the cache. If the
        key is not present in the cache, returns None.

        If you need the deletion callback, you'll have to work with
        self.cache manually.
        """

        item: Optional[CacheEntry] = self.cache.get(key, None)

        if item is not None:
            self.logger.debug(f"CACHE: fetch {key}")
            self.hits += 1
            return item[0]
        else:
            self.logger.debug(f"CACHE: missing {key}")
            self.misses += 1
            return None

    def dump(self) -> None:
        """
        Dump the cache to the logger.
        """

        for k in sorted(self.cache.keys()):
            rsrc, on_delete = self.cache[k]

            self.logger.info(f"CACHE: {k}, on_delete {self.fn_name(on_delete)}:")

            if k in self.links:
                for owned in sorted(self.links[k]):
                    self.logger.info(f"CACHE:   -> {owned}")

    def dump_stats(self) -> None:
        total = self.hits + self.misses

        if total > 0:
            ratio = "%.1f%%" % ((float(self.hits) / float(total)) * 100.0)
        else:
            ratio = "--"

        self.logger.info("CACHE: Total requests: %d" % total)
        self.logger.info("CACHE: Hit ratio:      %s" % ratio)
        self.logger.info("CACHE: Invalidations:  %d calls" % self.invalidate_calls)
        self.logger.info("CACHE:                 %d objects" % self.invalidated_objects)


class NullCache(Cache):
    """
    A Cache that doesn't actually cache anything -- basically, a no-op
    implementation of the Cache interface.

    Giving consumers of the Cache class a way to make their cache
    instance non-Optional, without actually requiring the use of the
    cache, makes the consumer code simpler.
    """

    def __init__(self, logger: logging.Logger) -> None:
        self.logger = logger
        self.logger.debug("NullCache: INIT")
        self.reset_stats()
        pass

    def add(self, rsrc: Cacheable,
            on_delete: Optional[DeletionHandler]=None) -> None:
        pass

    def link(self, owner: Cacheable, owned: Cacheable) -> None:
        pass

    def invalidate(self, key: str) -> None:
        self.invalidate_calls += 1
        pass

    def __getitem__(self, key: str) -> Any:
        self.misses += 1
        return None

    def dump(self) -> None:
        self.logger.info("NullCache: empty")
