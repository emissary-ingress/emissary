from typing import Dict, List, Optional, Tuple, TYPE_CHECKING

import copy
import json

from ..config import Config
from ..utils import dump_json

from .irhost import IRHost
from .irresource import IRResource
from .irtlscontext import IRTLSContext
from .irtcpmappinggroup import IRTCPMappingGroup
from .irutils import selector_matches

if TYPE_CHECKING:
    from .ir import IR # pragma: no cover


class IRListener (IRResource):
    """
    IRListener is a pretty direct translation of the Ambassador Listener resource.
    """

    bind_address: str       # Often "0.0.0.0", but can be overridden.
    service_port: int
    use_proxy_proto: bool
    hostname: str
    context: Optional[IRTLSContext]
    insecure_only: bool     # Was this synthesized solely due to an insecure_addl_port?
    namespace_literal: str  # Literal namespace to be matched
    namespace_selector: Dict[str, str]  # Namespace selector
    host_selector: Dict[str, str]   # Host selector

    AllowedKeys = {
        'bind_address',
        'l7Depth',
        'hostBinding',  # Note that hostBinding gets processed and deleted in setup.
        'port',
        'protocol',
        'protocolStack',
        'securityModel',
        'statsPrefix',
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

        # TCP: accepts raw TCP sessions.
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
                 apiVersion: str="x.getambassador.io/v3alpha1",
                 insecure_only: bool=False,
                 **kwargs) -> None:
        ir.logger.debug("IRListener __init__ (%s %s %s)" % (kind, name, kwargs))

        # A note: we copy hostBinding from kwargs in this loop, but we end up processing
        # and deleting it in setup(). This is arranged this way because __init__ can't
        # return an error, but setup() can.

        new_args = {
            x: kwargs[x] for x in kwargs.keys()
            if x in IRListener.AllowedKeys
        }

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location,
            kind=kind, name=name, namespace=namespace, apiVersion=apiVersion,
            insecure_only=insecure_only, 
            **new_args
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # Default hostBinding information early, so that we don't have to worry about it
        # ever being unset. We default to only looking for Hosts in our own namespace, and
        # to not using selectors beyond that.
        self.namespace_literal = self.namespace
        self.namespace_selector = {}
        self.host_selector = {}

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

        # Deal with statsPrefix, if it's not set.
        if not self.get("statsPrefix", ""):
            # OK, we need to default the thing per the protocolStack...
            tlsActive = "TLS" in self.protocolStack
            httpActive = "HTTP" in self.protocolStack

            if httpActive:
                if tlsActive:
                    self.statsPrefix = "ingress_https"
                else:
                    self.statsPrefix = "ingress_http"
            elif tlsActive:
                self.statsPrefix = f"ingress_tls_{self.port}"
            else:
                self.statsPrefix = f"ingress_tcp_{self.port}"

        # Deal with hostBinding. First up, namespaces.
        hostbinding = self.get("hostBinding", None)

        if not hostbinding:
            self.post_error("hostBinding is required")
            return False
        
        # We don't want self.hostBinding any more: the relevant stuff will be stored elsewhere
        # for ease of use.
        # 
        # XXX You can't do del(self.hostBinding) here, because underneath everything, an
        # IRListener is a Resource, and Resources are really much more like dicts than we
        # like to admit.
        del(self["hostBinding"])

        # We are going to require at least one of 'namespace' and 'selector' in the
        # hostBinding. (Really, K8s validation should be enforcing this before we get
        # here, anyway.)

        hb_namespace = hostbinding.get("namespace", None)
        hb_selector = hostbinding.get("selector", None)

        if not hb_namespace and not hb_selector:
            # Bzzt.
            self.post_error("hostBinding must have at least one of namespace or selector")
            return False

        if hb_namespace:
            # Again, technically K8s validation should enforce this, but just in case...
            nsfrom = hb_namespace.get("from", None)

            if not nsfrom:
                self.post_error("hostBinding.namespace.from is required")
                return False

            if nsfrom.lower() == 'all':
                self.namespace_literal = "*"    # Special, obviously.
            elif nsfrom.lower() == 'self':
                self.namespace_literal = self.namespace
            elif nsfrom.lower() == 'selector':
                # Augh. We can't actually support this yet, since the Python side of
                # Ambassador has no sense of Namespace objects, so it can't look at the
                # namespace labels!
                #
                # (K8s validation should prevent this from happening.)
                self.post_error("hostBinding.namespace.from=selector is not yet supported")

                # # When nsfrom == SELECTOR, we must have a selector.
                # nsselector: Optional[Dict[str, Any]] = hb_namespace.get("selector", None)

                # if not nsselector:
                #     self.post_error("hostBinding.namespace.selector is required when hostBinding.namespace.from is SELECTOR")
                #     return False
                
                # match: Optional[Dict[str, str]] = nsselector.get("matchLabels", None)

                # if not match:
                #     self.post_error("hostBinding.namespace.selector currently supports only matchLabels")
                #     return False

                # self.namespace_literal = "*"
                # self.namespace_selector = match

        # OK, after all that, look at the host selector itself.
        if hb_selector:
            if not "matchLabels" in hb_selector:
                self.post_error("hostBinding.selector currently supports only matchLabels")
                return False

            # This is not a typo -- save hb_selector here. The selector_matches function
            # takes it this way for whenever we want to add to it.
            self.host_selector = hb_selector

        return True

    def matches_host(self, host: IRHost) -> bool:
        """
        Returns True IFF this Listener wants to take the given IRHost -- meaning,
        the Host's namespace and selectors match what we want.
        """
        nsmatch = (self.namespace_literal == "*") or (self.namespace_literal == host.namespace)

        if not nsmatch:
            self.ir.logger.debug("    namespace mismatch (we're %s), DROP %s", self.namespace_literal, host)
            return False

        if not selector_matches(self.ir.logger, self.host_selector, host.metadata_labels):
            self.ir.logger.debug("    selector mismatch, DROP %s", host)
            return False

        self.ir.logger.debug("    TAKE %s", host)
        return True

    def __str__(self) -> str:
        pstack = "????"

        if self.get("protocolStack"):
            pstack = ";".join(self.protocolStack)

        securityModel = self.get("securityModel") or "????"

        hsstr = ""

        if self.host_selector:
            hsstr = "; ".join([ f"{k}={v}" for k, v in self.host_selector.items() ])

        nsstr = ""

        if self.namespace_selector:
            nsstr = "; ".join([ f"{k}={v}" for k, v in self.namespace_selector.items() ])

        return "<Listener %s on %s:%d (%s -- %s) ns %s sel %s, host sel %s (statsPrefix %s)>" % \
               (self.name, self.bind_address, self.port, securityModel, pstack,
                self.namespace_literal, nsstr, hsstr, self.statsPrefix)

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

                    ir.logger.debug(f"ListenerFactory: saving Listener {listener}")
                    ir.save_listener(listener)
                else:
                    ir.logger.debug(f"ListenerFactory: not saving inactive Listener {listener}")

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        # If we have no listeners at all, add the default listeners.
        # if not ir.listeners:
        #     # Do we have any Hosts using TLS?
        #     tls_active = False

        #     for host in ir.hosts.values():
        #         if host.context:
        #             tls_active = True

        #     if tls_active:
        #         ir.logger.debug("ListenerFactory: synthesizing default listeners (TLS)")

        #         # Add the default HTTP listener.
        #         # 
        #         # We use protocol HTTPS here so that the TLS inspector is active; that
        #         # lets us make better decisions about the security of a given request.
        #         ir.save_listener(IRListener(
        #             ir, aconf, "-internal-", f"ambassador-listener-8080", "-internal-",
        #             port=8080,
        #             protocol="HTTPS",   # Not a typo! See above.
        #             securityModel="XFP",
        #             hostBinding={
        #                 "namespace": {
        #                     "from": "SELF"
        #                 }
        #             }
        #         ))

        #         # Add the default HTTPS listener.
        #         ir.save_listener(IRListener(
        #             ir, aconf, "-internal-", "ambassador-listener-8443", "-internal-",
        #             port=8443,
        #             protocol="HTTPS",
        #             securityModel="XFP",
        #             hostBinding={
        #                 "namespace": {
        #                     "from": "SELF"
        #                 }
        #             }
        #         ))
        #     else:
        #         ir.logger.debug("ListenerFactory: synthesizing default listener (cleartext)")

        #         # Add the default HTTP listener.
        #         # 
        #         # We use protocol HTTP here because no, we don't want TLS active.
        #         ir.save_listener(IRListener(
        #             ir, aconf, "-internal-", "ambassador-listener-8080", "-internal-",
        #             port=8080,
        #             protocol="HTTP",   # Not a typo! See above.
        #             securityModel="XFP",
        #             hostBinding={
        #                 "namespace": {
        #                     "from": "SELF"
        #                 }
        #             }
        #         ))

        # # After that, cycle over our Hosts and see if any refer to 
        # # insecure.additionalPorts that don't already have Listeners.
        # for host in ir.get_hosts():
        #     # Hosts don't choose bind addresses, so if we see an insecure_addl_port,
        #     # look for it on Config.envoy_bind_address.
        #     if (host.insecure_addl_port is not None) and (host.insecure_addl_port > 0):
        #         listener_key = f"{Config.envoy_bind_address}-{host.insecure_addl_port}"
                
        #         if listener_key not in ir.listeners:
        #             ir.logger.debug("ListenerFactory: synthesizing listener for Host %s insecure.additionalPort %d", 
        #                             host.hostname, host.insecure_addl_port)
                    
        #             name = "insecure-for-%d" % host.insecure_addl_port

        #             # Note that we don't specify the bind address here, so that it
        #             # lands on Config.envoy_bind_address.
        #             ir.save_listener(IRListener(
        #                 ir, aconf, "-internal-", name, "-internal-",
        #                 port=host.insecure_addl_port,
        #                 protocol="HTTPS",   # Not a typo! See "Add the default HTTP listener" above.
        #                 securityModel="INSECURE",
        #                 insecure_only=True,
        #                 hostBinding={
        #                     "namespace": {
        #                         "from": "SELF"
        #                     }
        #                 }
        #             ))

        # Finally, cycle over our TCPMappingGroups and make sure we have
        # Listeners for all of them, too.
        for group in ir.ordered_groups():
            if not isinstance(group, IRTCPMappingGroup):
                continue

            # OK. If we have a Listener binding here already, use it -- that lets the user override
            # any choices we might make if they want to. If there's no Listener here, though, we'll
            # need to create one.
            #
            # (Note that group.bind_to() cleverly uses the same format as IRListener.bind_to().)
            group_key = group.bind_to()

            if group_key not in ir.listeners:
                # Nothing already exists, so fab one up. Use TLS if and only if a host match is specified;
                # with no host match, use TCP.
                group_host = group.get('host', None)
                protocol = "TLS" if group_host else "TCP"
                bind_address = group.get('address') or Config.envoy_bind_address
                name = f"listener-{bind_address}-{group.port}"

                ir.logger.debug("ListenerFactory: synthesizing %s listener for TCPMappingGroup on %s:%d" %
                                (protocol, bind_address, group.port))

                # The securityModel of a TCP listener is kind of a no-op at this point. We'll set it
                # to SECURE because that seems more rational than anything else. I guess.

                ir.save_listener(IRListener(
                    ir, aconf, '-internal-', name, '-internal-',
                    bind_address=bind_address,
                    port=group.port,
                    protocol=protocol,
                    securityModel="SECURE",  # See above.
                    hostBinding={
                        "namespace": {
                            "from": "SELF"
                        }
                    }
                ))
