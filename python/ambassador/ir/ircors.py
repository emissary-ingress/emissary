import copy
from typing import TYPE_CHECKING, Any, Dict

from ..config import Config
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


class IRCORS(IRResource):
    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.cors",
        kind: str = "IRCORS",
        name: str = "ir.cors",
        **kwargs,
    ) -> None:
        # print("IRCORS __init__ (%s %s %s)" % (kind, name, kwargs))

        # Convert our incoming kwargs into the things that Envoy actually wants.
        # Note that we have to treat 'origins' specially here, so that comes after
        # this renaming loop.

        new_kwargs: Dict[str, Any] = {}

        for from_key, to_key in [
            ("max_age", "max_age"),
            ("credentials", "allow_credentials"),
            ("methods", "allow_methods"),
            ("headers", "allow_headers"),
            ("exposed_headers", "expose_headers"),
        ]:
            value = kwargs.get(from_key, None)

            if value:
                new_kwargs[to_key] = self._cors_normalize(value)

        # 'origins' cannot be treated like other keys, because we have to transform it; Envoy wants
        # it in a different shape than it is in the CRD.
        origins = kwargs.get("origins", None)
        if origins is not None:
            new_kwargs["allow_origin_string_match"] = [{"exact": origin} for origin in origins]

        super().__init__(ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, **new_kwargs)

    def setup(self, ir: "IR", aconf: Config) -> bool:
        # This IRCORS has not been finalized with an ID, so leave with an 'unset' ID so far.
        self.set_id("unset")

        return True

    def set_id(self, mapping_key: str):
        self["filter_enabled"] = {
            "default_value": {"denominator": "HUNDRED", "numerator": 100},
            "runtime_key": f"routing.cors_enabled.{mapping_key}",
        }

    def dup(self) -> "IRCORS":
        return copy.copy(self)

    @staticmethod
    def _cors_normalize(value: Any) -> Any:
        """
        List values get turned into a comma-separated string. Other values
        are returned unaltered.
        """

        if type(value) == list:
            return ", ".join([str(x) for x in value])
        else:
            return value

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
