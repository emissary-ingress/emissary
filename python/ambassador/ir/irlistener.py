from typing import Dict, List, Optional, Tuple, TYPE_CHECKING

import copy
import json

from ..config import Config
from ..utils import dump_json

from .irresource import IRResource
from .irtlscontext import IRTLSContext

if TYPE_CHECKING:
    from .ir import IR # pragma: no cover


class IRListener (IRResource):
    """
    IRListener is a pretty direct translation of the Ambassador Listener resource.
    """

    bind_address: str   # Often "0.0.0.0", but can be overridden.
    service_port: int
    use_proxy_proto: bool
    hostname: str
    context: Optional[IRTLSContext]

    AllowedKeys = {
        'bind_address',
        'l7Depth',
        'hostSelector',
        'port',
        'protocol',
        'protocolStack',
        'securityModel',
    }

    ProtocolStacks: Dict[str, List[str]] = {
        # HTTP: accepts cleartext HTTP/1.1 sessions over TCP.
        "HTTP": [ "HTTP", "TCP" ],

        # HTTPS: accepts encrypted HTTP/1.1 or HTTP/2 sessions using TLS over TCP.
        "HTTPS": [ "TLS", "HTTP", "TCP" ],

        # HTTPPROXY: accepts cleartext HTTP/1.1 sessions using the HAProxy PROXY protocol over TCP.
        "HTTPPROXY": [ "PROXY", "HTTP", "TCP" ],

        # HTTPSPROXY: accepts encrypted HTTP/1.1 or HTTP/2 sessions using the HAProxy PROXY protocol over TLS over TCP.
        "HTTPSPROXY": [ "TLS", "PROXY", "HTTP", "TCP" ],

        # RAWTCP: accepts raw TCP sessions.
        "TCP": [ "TCP" ],

        # TLS: accepts TLS over TCP.
        "TLS": [ "TLS", "TCP" ],

        # # UDP: accepts UDP packets.
        # "UDP": [ "UDP" ],
    }

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str,      # REQUIRED
                 name: str,      # REQUIRED
                 location: str,  # REQUIRED
                 namespace: Optional[str]=None,
                 kind: str="IRListener",
                 apiVersion: str="getambassador.io/v2",
                 **kwargs) -> None:
        ir.logger.debug("IRListener __init__ (%s %s %s)" % (kind, name, kwargs))

        new_args = {
            x: kwargs[x] for x in kwargs.keys()
            if x in IRListener.AllowedKeys
        }

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location,
            kind=kind, name=name, namespace=namespace, apiVersion=apiVersion,
            **new_args
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # Was a bind address specified?
        if not self.get('bind_address', None):
            # Nope, use the default.
            self.bind_address = Config.envoy_bind_address            

        ir.logger.debug(f"Listener {self.name} setting up on {self.bind_address}:{self.port}")

        pstack = self.get("protocolStack", None)
        protocol = self.get("protocol", None)
        securityModel = self.get("securityModel", None)

        if pstack:
            ir.logger.debug(f"Listener {self.name} has pstack {pstack}")
            # It's an error to specify both protocol and protocolStack.
            if protocol:
                self.post_error("protocol and protocolStack may not both be specified; using protocolStack and ignoring protocol")
                self.protocol = None
        elif not protocol:
            # It's also an error to specify neither protocol nor protocolStack.
            self.post_error("one of protocol and protocolStack must be specified")
            return False
        else:
            # OK, we have a protocol, does it have a corresponding protocolStack?
            pstack = IRListener.ProtocolStacks.get(protocol, None)

            # This should be impossible, but just in case.
            if not pstack:
                self.post_error(f"protocol %s is not valid", protocol)
                return False
            
            ir.logger.debug(f"Listener {self.name} forcing pstack {';'.join(pstack)}")
            self.protocolStack = pstack
        
        if not securityModel:
            self.post_error("securityModel is required")
            return False

        return True

    def pretty(self) -> str:
        pstack = "????"

        if self.get("protocolStack"):
            pstack = ";".join(self.protocolStack)

        securityModel = self.get("securityModel") or "????"

        return "<Listener %s on %s:%d (%s -- %s)>" % \
               (self.name, self.bind_address, self.port, securityModel, pstack)

    # Deliberately matches IRTCPMappingGroup.bind_to()
    def bind_to(self) -> str:
        return f"{self.bind_address}-{self.port}"


class ListenerFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        amod = ir.ambassador_module

        listeners = aconf.get_config('listeners')

        if listeners:
            for config in listeners.values():
                ir.logger.debug("ListenerFactory: creating Listener for %s" % repr(config.as_dict()))

                listener = IRListener(ir, aconf, **config)

                if listener.is_active():
                    listener.referenced_by(config)
                    listener.sourced_by(config)

                    ir.logger.debug(f"ListenerFactory: saving Listener {listener.pretty()}")
                    ir.save_listener(listener)
                else:
                    ir.logger.debug(f"ListenerFactory: not saving inactive Listener {listener.pretty()}")

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        pass
