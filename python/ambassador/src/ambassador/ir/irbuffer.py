from typing import TYPE_CHECKING

from ..config import Config
from ..utils import RichStatus
from .irfilter import IRFilter

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


class IRBuffer(IRFilter):
    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.buffer",
        name: str = "ir.buffer",
        kind: str = "IRBuffer",
        **kwargs,
    ) -> None:
        super().__init__(ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, **kwargs)

    def setup(self, ir: "IR", aconf: Config) -> bool:
        max_request_bytes = self.pop("max_request_bytes", None)
        if max_request_bytes is not None:
            self["max_request_bytes"] = max_request_bytes
        else:
            self.post_error(RichStatus.fromError("missing required field: max_request_bytes"))
            return False

        if self.pop("max_request_time", None):
            self.ir.aconf.post_notice("'max_request_time' is no longer supported, ignoring", self)

        return True
