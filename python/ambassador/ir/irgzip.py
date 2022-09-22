from typing import TYPE_CHECKING, Optional
from typing import cast as typecast

from ..config import Config
from ..resource import Resource
from ..utils import RichStatus
from .ircluster import IRCluster
from .irfilter import IRFilter

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


class IRGzip(IRFilter):
    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.gzip",
        name: str = "ir.gzip",
        kind: str = "IRGzip",
        **kwargs
    ) -> None:

        super().__init__(ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, **kwargs)

    def setup(self, ir: "IR", aconf: Config) -> bool:
        self["memory_level"] = self.pop("memory_level", None)
        self["content_length"] = self.pop("min_content_length", None)
        self["compression_level"] = self.pop("compression_level", None)
        self["compression_strategy"] = self.pop("compression_strategy", None)
        self["window_bits"] = self.pop("window_bits", None)
        self["content_type"] = self.pop("content_type", [])
        self["disable_on_etag_header"] = self.pop("disable_on_etag_header", None)
        self["remove_accept_encoding_header"] = self.pop("remove_accept_encoding_header", None)

        return True
