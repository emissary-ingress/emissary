from ambassador.utils import RichStatus
from typing import Any, ClassVar, Dict, List, Optional, Type, Union, TYPE_CHECKING

from ..config import Config

from .irbasemapping import IRBaseMapping, normalize_service_name
from .irbasemappinggroup import IRBaseMappingGroup
from .irtcpmappinggroup import IRTCPMappingGroup

import hashlib

if TYPE_CHECKING:
    from .ir import IR # pragma: no cover


class IRTCPMapping (IRBaseMapping):
    binding: str
    service: str
    group_id: str
    route_weight: List[Union[str, int]]

    AllowedKeys: ClassVar[Dict[str, bool]] = {
        "address": True,
        "circuit_breakers": False,
        "enable_ipv4": True,
        "enable_ipv6": True,
        "host": True,
        "idle_timeout_ms": True,
        "metadata_labels": True,
        "port": True,
        "service": True,
        "tls": True,
        "weight": True,
        "resolver": True,
        # Include the serialization, too.
        "serialization": True,
    }

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str,      # REQUIRED
                 name: str,      # REQUIRED
                 location: str,  # REQUIRED
                 service: str,   # REQUIRED
                 namespace: Optional[str] = None,
                 metadata_labels: Optional[Dict[str, str]] = None,

                 kind: str="IRTCPMapping",
                 apiVersion: str="getambassador.io/v3alpha1",   # Not a typo! See below.
                 precedence: int=0,
                 cluster_tag: Optional[str]=None,
                 **kwargs) -> None:
        # OK, this is a bit of a pain. We want to preserve the name and rkey and
        # such here, unlike most kinds of IRResource. So. Shallow copy the keys
        # we're going to allow from the incoming kwargs...

        new_args = { x: kwargs[x] for x in kwargs.keys() if x in IRTCPMapping.AllowedKeys }

        # XXX The resolver lookup code is duplicated from IRBaseMapping.setup --
        # needs to be fixed after 1.6.1.
        resolver_name = kwargs.get('resolver') or ir.ambassador_module.get('resolver', 'kubernetes-service')

        assert(resolver_name)   # for mypy -- resolver_name cannot be None at this point
        resolver = ir.get_resolver(resolver_name)

        if resolver:
            resolver_kind = resolver.kind
        else:
            # In IRBaseMapping.setup, we post an error if the resolver is unknown.
            # Here, we just don't bother; we're only using it for service
            # qualification.
            resolver_kind = 'KubernetesBogusResolver'

        service = normalize_service_name(ir, service, namespace, resolver_kind, rkey=rkey)
        ir.logger.debug(f"TCPMapping {name} service normalized to {repr(service)}")

        # ...and then init the superclass.
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location, service=service,
            kind=kind, name=name, namespace=namespace, metadata_labels=metadata_labels,
            apiVersion=apiVersion, precedence=precedence, cluster_tag=cluster_tag,
            **new_args
        )

        ir.logger.debug("IRTCPMapping %s: self.host = %s", name, self.get("host") or "i'*'")

    @staticmethod
    def group_class() -> Type[IRBaseMappingGroup]:
        return IRTCPMappingGroup

    def bind_to(self) -> str:
        bind_addr = self.get('address') or '0.0.0.0'
        return f"tcp-{bind_addr}-{self.port}"

    def _group_id(self) -> str:
        # Yes, we're using a cryptographic hash here. Cope. [ :) ]

        h = hashlib.new('sha1')

        # This is a TCP mapping.
        h.update('TCP-'.encode('utf-8'))

        address = self.get('address') or '*'
        h.update(address.encode('utf-8'))

        port = str(self.port)
        h.update(port.encode('utf-8'))

        host = self.get('host') or '*'
        h.update(host.encode('utf-8'))

        return h.hexdigest()

    def _route_weight(self) -> List[Union[str, int]]:
        # These aren't order-dependent? or are they?
        return [ 0 ]
