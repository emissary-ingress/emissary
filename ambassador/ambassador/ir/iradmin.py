from typing import TYPE_CHECKING

from ..config import Config

from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR

class IRAdmin (IRResource):
    def __init__(self, ir: 'IR', aconf: Config,

                 admin_port: int,

                 rkey: str="ir.admin",
                 kind: str="IRAdmin",
                 name: str="ir.admin",
                 **kwargs) -> None:
        # print("IRAdmin __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            admin_port=admin_port,
            **kwargs
        )
