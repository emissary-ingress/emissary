from typing import ClassVar, Dict, Optional, TYPE_CHECKING
from typing import cast as typecast

import json

from ..config import Config
from ..resource import Resource

if TYPE_CHECKING:
    from .ir import IR


class IRResource (Resource):
    """
    A resource within the IR.
    """

    _active: bool

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str,
                 kind: str,
                 name: str,
                 location: str = "-ir-",
                 apiVersion: str="ambassador/ir",
                 **kwargs) -> None:
        # print("IRResource __init__ (%s %s)" % (kind, name))

        super().__init__(rkey=rkey, location=location,
                         kind=kind, name=name, apiVersion=apiVersion,
                         **kwargs)

        self.logger = ir.logger

        self.set_active(self.setup(ir, aconf))

    def set_active(self, active: bool) -> None:
        self._active = active

    def is_active(self) -> bool:
        return self._active

    def __nonzero__(self) -> bool:
        return self._active and not self._errors

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # If you don't override setup, you end up with an IRResource that's always active.
        return True

    def add_mappings(self, ir: 'IR', aconf: Config) -> None:
        # If you don't override add_mappings, uh, no mappings will get added.
        pass

    def as_dict(self) -> Dict:
        od = {}

        for k in self.keys():
            if (k == 'apiVersion') or (k == 'logger'):
                continue
            elif k == '_referenced_by':
                refd_by = sorted([ "%s: %s" % (k, self._referenced_by[k].location)
                                   for k in self._referenced_by.keys() ])

                od['_referenced_by'] = refd_by
            elif k == 'rkey':
                od['_rkey'] = self[k]
            elif isinstance(self[k], IRResource):
                od[k] = self[k].as_dict()
            elif self[k] is not None:
                od[k] = self[k]

        # print("returning %s" % repr(od))
        return od

    def as_json(self, indent=4, sort_keys=True, **kwargs):
        return json.dumps(self.as_dict(), indent=indent, sort_keys=sort_keys, **kwargs)


