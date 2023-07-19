from typing import TYPE_CHECKING, ClassVar, Dict, List, Optional, Tuple

from ..config import Config
from .irfilter import IRFilter
from .irresource import IRResource

if TYPE_CHECKING:
    from ..envoy.v3.v3cidrrange import CIDRRange
    from .ir import IR  # pragma: no cover


class IRIPAllowDeny(IRFilter):
    """
    IRIPAllowDeny is an IRFilter that implements an allow/deny list based
    on IP address.
    """

    parent: IRResource
    action: str
    principals: List[Tuple[str, "CIDRRange"]]

    EnvoyTypeMap: ClassVar[Dict[str, str]] = {"remote": "remote_ip", "peer": "direct_remote_ip"}

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.ipallowdeny",
        name: str = "ir.ipallowdeny",
        kind: str = "IRIPAllowDeny",
        parent: IRResource | None = None,
        action: str | None = None,
        **kwargs,
    ) -> None:
        """
        Initialize an IRIPAllowDeny. In addition to the usual IRFilter parameters,
        parent and action are required:

        parent is the IRResource in which the IRIPAllowDeny is defined; at present,
        this will be the Ambassador module. It's required because it's where errors
        should be posted.

        action must be either "ALLOW" or "DENY". This action will be normalized to
        all-uppercase in setup().
        """

        assert parent is not None
        assert action is not None

        super().__init__(
            ir=ir,
            aconf=aconf,
            rkey=rkey,
            kind=kind,
            name=name,
            parent=parent,
            action=action,
            **kwargs,
        )

    def setup(self, ir: "IR", aconf: Config) -> bool:
        """
        Set up an IRIPAllowDeny based on the action and principals passed into
        __init__.
        """

        assert self.parent

        # These pops will crash if the action or principals are missing. That's
        # OK -- they're required elements.
        action: Optional[str] = self.pop("action")
        principals: Optional[List[Dict[str, str]]] = self.pop("principals")

        assert action is not None
        assert principals is not None

        action = action.upper()

        if (action != "ALLOW") and (action != "DENY"):
            raise RuntimeError(f"IRIPAllowDeny action must be ALLOW or DENY, not {action}")

        self.action = action
        self.principals = []

        ir.logger.debug(f"PRINCIPALS: {principals}")

        # principals looks like
        #
        # [
        #    { 'peer': '127.0.0.1' },
        #    { 'remote': '192.68.0.0/24' },
        #    { 'remote': '::1' }
        # ]
        #
        # or the like, where the key in the dict specifies how Envoy will handle the
        # IP match, and the value is a CIDRRange spec.

        from ..envoy.v3.v3cidrrange import CIDRRange

        for pdict in principals:
            # If we have more than one thing in the dict, that's an error.

            first = True

            for kind, spec in pdict.items():
                if not first:
                    self.parent.post_error(
                        f"ip{self.action.lower()} principals must be separate list elements"
                    )
                    break

                first = False

                envoy_kind = IRIPAllowDeny.EnvoyTypeMap.get(kind, None)

                if not envoy_kind:
                    self.parent.post_error(
                        f"ip{self.action.lower()} principal type {kind} unknown: must be peer or remote"
                    )
                    continue

                cidrrange = CIDRRange(spec)

                if cidrrange:
                    self.principals.append((envoy_kind, cidrrange))
                else:
                    self.parent.post_error(
                        f"ip_{self.action.lower()} principal {spec} is not valid: {cidrrange.error}"
                    )

        if len(self.principals) > 0:
            return True
        else:
            return False

    def __str__(self) -> str:
        pstrs = [str(x) for x in self.principals]
        return f"<IPAllowDeny {self.action}: {', '.join(pstrs)}>"

    def as_dict(self) -> dict:
        return {
            "action": self.action,
            "principals": [{kind: block.as_dict()} for kind, block in self.principals],
        }
