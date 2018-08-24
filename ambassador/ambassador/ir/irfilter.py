from typing import Optional, TYPE_CHECKING

from ..config import Config

from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR


class IRFilter(IRResource):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str = "ir.filter",
                 kind: str = "IRFilter",
                 name: str = "ir.filter",
                 type: Optional[str] = None,
                 **kwargs) -> None:
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            type=type,
            **kwargs)

    def config_dict(self):
        config = {}

        return config
