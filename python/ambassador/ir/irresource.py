from typing import Any, Dict, List, Optional, Tuple, Union, TYPE_CHECKING

from ..config import Config
from ..resource import Resource
from ..utils import RichStatus

if TYPE_CHECKING:
    from .ir import IR


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

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str,
                 kind: str,
                 name: str,
                 namespace: Optional[str]=None,
                 metadata_labels: Optional[str]=None,
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

        # ...before we override it with the setup results.
        self.set_active(self.setup(ir, aconf))

    def lookup(self, key: str, *args, default_class: Optional[str]=None, default_key: Optional[str]=None) -> Any:
        """
        Look up a key in this IRResource, with a fallback to the Ambassador module's "defaults"
        element.

        Here's the resolution order:

        - if key is present in self, use its value.
        - if not, we'll try to look up a fallback value in the Ambassador module:
            - the key for the lookup will be the value of "default_key" if that's set,
              otherwise the same key we just tried in self.
            - if "default_class" wasn't passed in, and self.default_class isn't set, just
              look up a fallback value from the "defaults" dict in the Ambassador module.
            - otherwise, look up the default class in Ambassador's "defaults", then look up
              the fallback value from that dict (the passed in "default_class" wins if both
              are set).
            - (if the default class is '/', explictly skip descending into a sub-directory)
        - if no key is present in self, and no fallback is found, but a default value was passed
          in as *args[0], return that.
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
            get_from = self.ir.ambassador_module.get('defaults', {})

            dfl_class = default_class

            if not dfl_class:
                dfl_class = self.get('default_class', None)

            if dfl_class and (dfl_class != '/'):
                get_from = get_from.get(dfl_class, None)

            if get_from:
                if not default_key:
                    default_key = key

                value = get_from.get(default_key, default_value)
            else:
                value = default_value

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

    def post_error(self, error: Union[str, RichStatus]):
        self._errored = True

        if not self.ir:
            raise Exception("post_error cannot be called before __init__")

        self.ir.post_error(error, resource=self)
        # super().post_error(error)
        # self.ir.logger.error("%s: %s" % (self, error))

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
