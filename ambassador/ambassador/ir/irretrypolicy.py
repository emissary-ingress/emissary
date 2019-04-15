from typing import Any, TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus

from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR

class IRRetryPolicy (IRResource):
    def __init__(self, ir: 'IR', aconf: Config,

                 rkey: str="ir.retrypolicy",
                 kind: str="IRRetryPolicy",
                 name: str="ir.retrypolicy",
                 **kwargs) -> None:
        # print("IRRetryPolicy __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            **kwargs
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        return True

