from typing import Any, TYPE_CHECKING

import copy

from ..config import Config
from ..utils import RichStatus

from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR


class IRCORS (IRResource):
    def __init__(self, ir: 'IR', aconf: Config,

                 rkey: str="ir.cors",
                 kind: str="IRCORS",
                 name: str="ir.cors",
                 **kwargs) -> None:
        # print("IRCORS __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            **kwargs
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # 'origins' cannot be treated like other keys, because if it's a
        # list, then it remains as is, but if it's a string, then it's
        # converted to a list.  It has already been validated by the
        # JSON schema to either be a string or a list of strings.
        origins = self.pop('origins', None)

        if origins is not None:
            if type(origins) is not list:
                origins = origins.split(',')

            self.allow_origin_string_match = [{'exact': origin} for origin in origins]

        for from_key, to_key in [ ( 'max_age', 'max_age' ),
                                  ( 'credentials', 'allow_credentials' ),
                                  ( 'methods', 'allow_methods' ),
                                  ( 'headers', 'allow_headers' ),
                                  ( 'exposed_headers', 'expose_headers' ) ]:
            value = self.pop(from_key, None)

            if value:
                self[to_key] = self._cors_normalize(value)

        # This IRCORS has not been finalized with an ID, so leave with an 'unset' ID so far.
        self.set_id('unset')

        return True

    def set_id(self, group_id: str):
        self['filter_enabled'] = {
            "default_value": {
                "denominator": "HUNDRED",
                "numerator": 100
            },
            "runtime_key": f"routing.cors_enabled.{group_id}"
        }

    def dup(self) -> 'IRCORS':
        return copy.copy(self)

    @staticmethod
    def _cors_normalize(value: Any) -> Any:
        """
        List values get turned into a comma-separated string. Other values
        are returned unaltered.
        """

        if type(value) == list:
            return ", ".join([ str(x) for x in value ])
        else:
            return value

    def as_dict(self) -> dict:
        raw_dict = super().as_dict()

        for key in list(raw_dict):
            if key in ["_active", "_errored", "_referenced_by", "_rkey",
                       "kind", "location", "name", "namespace", "metadata_labels"]:
                raw_dict.pop(key, None)

        return raw_dict
