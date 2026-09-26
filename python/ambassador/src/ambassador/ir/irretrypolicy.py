import re
from typing import TYPE_CHECKING, Optional

from ..config import Config
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


# Envoy wants durations in the protobuf JSON encoding: a decimal number of seconds with an "s"
# suffix, e.g. "0.025s" for 25 milliseconds.
DurationRE = re.compile(r"^[0-9]+(\.[0-9]{1,9})?s$")


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

        # validate_retry_back_off posts its own (more specific) errors.
        if not self.validate_retry_back_off():
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

    def validate_retry_back_off(self) -> bool:
        """Validate the retry_back_off block, if any. Envoy rejects the whole configuration if
        the back-off is malformed, so anything Envoy would refuse is an error here."""

        back_off = self.get("retry_back_off", None)

        if back_off is None:
            return True

        if not isinstance(back_off, dict):
            self.post_error("retry_back_off must be a dictionary: {}".format(back_off))
            return False

        base_interval = self.parse_interval(back_off, "base_interval")
        max_interval = self.parse_interval(back_off, "max_interval")

        if base_interval is None:
            if "base_interval" not in back_off:
                self.post_error("retry_back_off requires base_interval")

            # If base_interval was present but malformed, parse_interval already posted an error.
            return False

        if ("max_interval" in back_off) and (max_interval is None):
            return False

        if (max_interval is not None) and (max_interval < base_interval):
            self.post_error(
                'retry_back_off max_interval "{}" must not be less than base_interval "{}"'.format(
                    back_off["max_interval"], back_off["base_interval"]
                )
            )
            return False

        return True

    def parse_interval(self, back_off: dict, key: str) -> Optional[float]:
        """Parse one of the retry_back_off intervals into a number of seconds, posting an error
        and returning None if it isn't a positive Envoy duration."""

        value = back_off.get(key, None)

        if value is None:
            return None

        if not isinstance(value, str) or not DurationRE.match(value):
            self.post_error(
                'retry_back_off {} "{}" is not a number of seconds, e.g. "0.025s"'.format(
                    key, value
                )
            )
            return None

        seconds = float(value[:-1])

        if seconds <= 0:
            self.post_error(
                'retry_back_off {} "{}" must be greater than zero'.format(key, value)
            )
            return None

        return seconds

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
