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
                 location: str = "--internal--",
                 type: Optional[str] = None,
                 config: Optional[dict] = None,
                 **kwargs) -> None:
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            location=location,
            type=type,
            config=config,
            **kwargs)

    def config_dict(self) -> Optional[dict]:
        return self.config

    def finalize(self) -> None:
        pass
