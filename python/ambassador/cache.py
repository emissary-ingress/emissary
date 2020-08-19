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

        self.logger.info("Cache initialized")

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

    def delete(self, key: str) -> None:
        """
        Recursively delete the entry named by 'key' and everything it owns.
        """

        worklist = [ key ]
        to_delete = []

        while worklist:
            key = worklist.pop(0)

            if key in self.cache:
                rsrc, on_delete = self.cache[key]

                self.logger.debug(f"CACHE: DEL {key}: will delete {rsrc}")
                to_delete.append((key, rsrc, on_delete))

                if key in self.links:
                    for owned in sorted(self.links[key]):
                        self.logger.debug(f"CACHE: DEL {key}: will check owned {owned}")
                        worklist.append(owned)
                
        for key, rsrc, on_delete in to_delete:
            self.logger.debug(f"CACHE: DEL {key}: smiting!")
            del(self.cache[key])

            if key in self.links:
                del(self.links[key])

            if on_delete:
                self.logger.debug(f"CACHE: DEL {key}: calling {self.fn_name(on_delete)}")
                on_delete(rsrc)

    def __getitem__(self, key: str) -> Any:
        """
        Fetches only the _resource_ for a given key from the cache. If the
        key is not present in the cache, returns None.

        If you need the deletion callback, you'll have to work with
        self.cache manually.
        """

        item: Optional[CacheEntry] = self.cache.get(key, None)

        if item is not None:
            self.logger.debug(f"CACHE: fetch {key}")
            return item[0]
        else:
            self.logger.debug(f"CACHE: missing {key}")
            return None


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
        self.logger.info("NullCache: INIT")
        pass

    def add(self, rsrc: Cacheable, 
            on_delete: Optional[DeletionHandler]=None) -> None:
        pass

    def link(self, owner: Cacheable, owned: Cacheable) -> None:
        pass
    
    def dump(self) -> None:
        self.logger.info("NULLCACHE: empty")

    def delete(self, key: str) -> None:
        pass

    def __getitem__(self, key: str) -> Any:
        return None
