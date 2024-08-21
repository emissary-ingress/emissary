from typing import TYPE_CHECKING

from ..config import Config
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


class IRRetryPolicy(IRResource):
    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.retrypolicy",
        kind: str = "IRRetryPolicy",
        name: str = "ir.retrypolicy",
        **kwargs,
    ) -> None:
        # print("IRRetryPolicy __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, **kwargs)

    def setup(self, ir: "IR", aconf: Config) -> bool:
        if not self.validate_retry_policy():
            self.post_error("Invalid retry policy specified: {}".format(self))
            return False

        return True

    def validate_retry_policy(self) -> bool:
        retry_on = self.get("retry_on", None)

        is_valid = False
        if retry_on in {
            "5xx",
            "gateway-error",
            "connect-failure",
            "retriable-4xx",
            "refused-stream",
            "retriable-status-codes",
        }:
            is_valid = True

        return is_valid

    def as_dict(self) -> dict:
        raw_dict = super().as_dict()

        for key in list(raw_dict):
            if key in [
                "_active",
                "_errored",
                "_referenced_by",
                "_rkey",
                "kind",
                "location",
                "name",
                "namespace",
                "metadata_labels",
            ]:
                raw_dict.pop(key, None)

        return raw_dict
