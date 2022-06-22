from typing import Any, Dict, List, Optional, Tuple, Union, TYPE_CHECKING

import copy
import logging

from ..config import Config
from ..resource import Resource
from ..utils import RichStatus

if TYPE_CHECKING:
    from .ir import IR # pragma: no cover


class IRResource (Resource):
    """
    A resource within the IR.
    """

    @staticmethod
    def helper_sort_keys(res: 'IRResource', k: str) -> Tuple[str, List[str]]:
        return k, list(sorted(res[k].keys()))

    @staticmethod
    def helper_rkey(res: 'IRResource', k: str) -> Tuple[str, str]:
        return '_rkey', res[k]

    @staticmethod
    def helper_list(res: 'IRResource', k: str) -> Tuple[str, list]:
        return k, list([ x.as_dict() for x in res[k] ])

    __as_dict_helpers: Dict[str, Any] = {
        "apiVersion": "drop",
        "logger": "drop",
        "ir": "drop"
    }

    _active: bool
    _errored: bool
    _cache_key: Optional[str]

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str,
                 kind: str,
                 name: str,
                 namespace: Optional[str]=None,
                 metadata_labels: Optional[Dict[str, str]]=None,
                 location: str = "--internal--",
                 apiVersion: str="ambassador/ir",
                 **kwargs) -> None:
        # print("IRResource __init__ (%s %s)" % (kind, name))

        if not namespace:
            namespace = ir.ambassador_namespace
        self.namespace = namespace

        super().__init__(rkey=rkey, location=location,
                         kind=kind, name=name, namespace=namespace, metadata_labels=metadata_labels,
                         apiVersion=apiVersion,
                         **kwargs)
        self.ir = ir
        self.logger = ir.logger

        self._errored = False

        self.__as_dict_helpers = IRResource.__as_dict_helpers
        self.add_dict_helper("_errors", IRResource.helper_list)
        self.add_dict_helper("_referenced_by", IRResource.helper_sort_keys)
        self.add_dict_helper("rkey", IRResource.helper_rkey)

        # Make certain that _active has a default...
        self.set_active(False)

        # ...and start with an empty cache key...
        self._cache_key = None

        # ...before we override it with the setup results.
        self.set_active(self.setup(ir, aconf))

    # XXX WTFO, I hear you cry. Why is this "type: ignore here?" So here's the deal:
    # mypy doesn't like it if you override just the getter of a property that has a
    # setter, too, and I cannot figure out how else to shut it up.
    @property   # type: ignore
    def cache_key(self) -> str:
        # If you ask for the cache key and it's not set, that is an error.
        assert(self._cache_key is not None)
        return self._cache_key

    def lookup_default(self, key: str, default_value: Optional[Any]=None, lookup_class: Optional[str]=None) -> Any:
        """
        Look up a key in the Ambassador module's "defaults" element.

        The "lookup class" is
        - the lookup_class parameter if one was passed, else
        - self.default_class if that's set, else
        - None.

        We can look in two places for key -- the first match wins:

        1. defaults[lookup class][key] if the lookup key is neither None nor "/"
        2. defaults[key]

        (A lookup class of "/" skips step 1.)

        If we don't find the key in either place, return the given default_value.
        If we _do_ find the key, _return a copy of the data!_ If we return the data itself
        and the caller later modifies it... that's a problem.

        :param key: the key to look up
        :param default_value: the value to return if nothing is found in defaults.
        :param lookup_class: the lookup class, see above
        :return: Any
        """

        defaults = self.ir.ambassador_module.get('defaults', {})

        lclass = lookup_class

        if not lclass:
            lclass = self.get('default_class', None)

        if lclass and (lclass != '/'):
            # Case 1.
            classdict = defaults.get(lclass, None)

            if classdict and (key in classdict):
                return copy.deepcopy(classdict[key])

        # We didn't find anything in case 1. Try case 2.
        if defaults and (key in defaults):
            return copy.deepcopy(defaults[key])

        # We didn't find anything in either case. Return the default value.
        return default_value

    def lookup(self, key: str, *args, default_class: Optional[str]=None, default_key: Optional[str]=None) -> Any:
        """
        Look up a key in this IRResource, with a fallback to the Ambassador module's "defaults"
        element.

        Here's the resolution order:

        - if key is present in self, use its value.
        - if not, use lookup_default above to try to find a value in the Ambassador module
        - if we don't find anything, but a default value was passed in as *args[0], return that.
        - if all else fails, return None.

        :param key: the key to look up
        :param default_class: the default class for the fallback lookup (optional, see above)
        :param default_key: the key for the fallback lookup (optional, defaults to key)
        :param args: an all-else-fails default value can go here, see above
        :return: Any
        """

        value = self.get(key, None)

        default_value = None

        if len(args) > 0:
            default_value = args[0]

        if value is None:
            if not default_key:
                default_key = key

            value = self.lookup_default(default_key, default_value=default_value, lookup_class=default_class)

        return value

    def add_dict_helper(self, key: str, helper) -> None:
        self.__as_dict_helpers[key] = helper

    def set_active(self, active: bool) -> None:
        self._active = active

    def is_active(self) -> bool:
        return self._active

    def __bool__(self) -> bool:
        return self._active and not self._errored

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # If you don't override setup, you end up with an IRResource that's always active.
        return True

    def add_mappings(self, ir: 'IR', aconf: Config) -> None:
        # If you don't override add_mappings, uh, no mappings will get added.
        pass

    def post_error(self, error: Union[str, RichStatus], log_level=logging.INFO):
        self._errored = True

        if not self.ir:
            raise Exception("post_error cannot be called before __init__")

        self.ir.post_error(error, resource=self, log_level=log_level)

    def skip_key(self, k: str) -> bool:
        if k.startswith('__') or k.startswith("_IRResource__"):
            return True

        if self.__as_dict_helpers.get(k, None) == 'drop':
            return True

        return False

    def as_dict(self) -> Dict:
        od: Dict[str, Any] = {}

        for k in self.keys():
            if self.skip_key(k):
                continue

            helper = self.__as_dict_helpers.get(k, None)

            if helper:
                new_k, v = helper(self, k)

                if new_k and v:
                    od[new_k] = v
            elif isinstance(self[k], IRResource):
                od[k] = self[k].as_dict()
            elif self[k] is not None:
                od[k] = self[k]

        return od

    @staticmethod
    def normalize_service(service: str) -> str:
        normalized_service = service

        if service.lower().startswith("http://"):
            normalized_service = service[len("http://"):]
        elif service.lower().startswith("https://"):
            normalized_service = service[len("https://"):]

        return normalized_service
