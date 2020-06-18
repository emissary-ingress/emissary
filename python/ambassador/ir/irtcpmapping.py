from ambassador.utils import RichStatus
from typing import Any, ClassVar, Dict, List, Optional, Type, Union, TYPE_CHECKING

from ..config import Config

from .irbasemapping import IRBaseMapping, qualify_service_name
from .irbasemappinggroup import IRBaseMappingGroup
from .irtcpmappinggroup import IRTCPMappingGroup

import hashlib

if TYPE_CHECKING:
    from .ir import IR


class IRTCPMapping (IRBaseMapping):
    binding: str
    service: str
    group_id: str
    route_weight: List[Union[str, int]]
    sni: bool

    AllowedKeys: ClassVar[Dict[str, bool]] = {
        "address": True,
        "enable_ipv4": True,
        "enable_ipv6": True,
        "host": True,
        "idle_timeout_ms": True,
        "port": True,
        "service": True,
        "tls": True,
        "weight": True,

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
                 apiVersion: str="getambassador.io/v2",   # Not a typo! See below.
                 precedence: int=0,
                 cluster_tag: Optional[str]=None,
                 **kwargs) -> None:
        # OK, this is a bit of a pain. We want to preserve the name and rkey and
        # such here, unlike most kinds of IRResource. So. Shallow copy the keys
        # we're going to allow from the incoming kwargs...

        new_args = { x: kwargs[x] for x in kwargs.keys() if x in IRTCPMapping.AllowedKeys }
        service = qualify_service_name(ir, service, namespace)

        # ...and then init the superclass.
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location, service=service,
            kind=kind, name=name, namespace=namespace, metadata_labels=metadata_labels,
            apiVersion=apiVersion, precedence=precedence, cluster_tag=cluster_tag,
            **new_args
        )

        if 'host' in kwargs:
            self.tls_context = self.match_tls_context(kwargs['host'], ir)

    @staticmethod
    def group_class() -> Type[IRBaseMappingGroup]:
        return IRTCPMappingGroup

    def bind_to(self) -> str:
        bind_addr = self.get('address') or '0.0.0.0'
        return "%s-%s" % (bind_addr, self.port)

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
