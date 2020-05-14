from typing import Dict, Optional, Tuple, TYPE_CHECKING

import copy
import json

from ..config import Config
from .irresource import IRResource
from .irtlscontext import IRTLSContext

if TYPE_CHECKING:
    from .ir import IR


class IRListener (IRResource):
    """
    IRListener is Ambassador's concept of a listener.

    Note that an IRListener is a _very different beast_ than an Envoy listener.
    An IRListener is pretty straightforward right now: a port, whether TLS is
    required, whether we're doing Proxy protocol.

    NOTE WELL: at present, all IRListeners are considered to be equals --
    specifically, every Mapping is assumed to belong to every IRListener. That
    may change later, but that's why you don't see Mappings associated with an
    IRListener when creating the IRListener.

    An Envoy listener, by contrast, is something more akin to an environment
    definition. See V2Listener for more.
    """

    def __init__(self, ir: 'IR', aconf: Config,
                 service_port: int,
                 # require_tls: bool,
                 use_proxy_proto: bool,
                 redirect_listener: bool = False,

                 rkey: str="ir.listener",
                 kind: str="IRListener",
                 name: str="ir.listener",
                 **kwargs) -> None:
        # print("IRListener __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            service_port=service_port,
            # require_tls=require_tls,
            use_proxy_proto=use_proxy_proto,
            **kwargs)

        self.redirect_listener: bool = redirect_listener
        # self.require_tls: bool = require_tls

    def pretty(self) -> str:
        ctx = self.get('context', None)
        ctx_name = '-none-' if not ctx else ctx.name

        return "<Listener %s for %s:%d, ctx %s, secure %s, insecure %s/%s>" % \
               (self.name, self.hostname, self.service_port, ctx_name,
                self.secure_action, self.insecure_action, self.insecure_addl_port)


class ListenerFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        amod = ir.ambassador_module

        # An IRListener roughly corresponds to something partway between an Envoy
        # FilterChain and an Envoy VirtualHost -- it's a single domain entry (which
        # could be a wildcard) that can have routes and such associated with it.
        #
        # A single IRListener can require TLS, or not. If TLS is around, it can
        # require a specific SNI host. Since it contains VirtualHosts, it can also
        # do things like require the PROXY protocol, and we can use an IRListener
        # to say "any host without TLS", then do things on a per-host basis within
        # it -- but the guts of all _that_ are down in V2Listener, since it's very
        # highly Envoy-specific.
        #
        # Port-based TCPMappings also happen down in V2Listener at the moment. This
        # means that if you try to do a port-based TCPMapping for a port that you
        # also try to have an IRListener on, that won't work. That's OK for now.
        #
        # The way this works goes like this:
        #
        # 1. Collect all our TLSContexts with their host entries.
        # 2. Walk our Hosts and figure out which port(s) each needs to listen on.
        #    If a Host has TLS info, pull the corresponding TLSContext from the set
        #    of leftover TLSContexts.
        # 3. If any TLSContexts are left when we're done with Hosts, walk over those
        #    and treat them like a Host that's asking for secure routing, using the
        #    global redirect_cleartext_from setting to decide what to do for insecure
        #    routing.
        #
        # So. First build our set of TLSContexts.
        unused_contexts: Dict[str, IRTLSContext] = {}

        for ctx in ir.get_tls_contexts():
            if ctx.is_active:
                ctx_hosts = ctx.get('hosts', [])

                if ctx_hosts:
                    # This is a termination context.
                    for hostname in ctx_hosts:
                        extant_context = unused_contexts.get(hostname, None)

                        if extant_context:
                            ir.post_error("TLSContext %s claims hostname %s, which was already claimed by %s" %
                                          (ctx.name, hostname, extant_context.name))
                            continue

                        unused_contexts[hostname] = ctx

        # Next, start with an empty set of listeners...
        listeners: Dict[str, IRListener] = {}

        cls.dump_info(ir, "AT START", listeners, unused_contexts)

        # OK. Walk hosts.
        hosts = ir.get_hosts() or []

        for host in hosts:
            ir.logger.debug(f"ListenerFactory: consider Host {host.pretty()}")

            hostname = host.hostname
            request_policy = host.get('requestPolicy', {})
            insecure_policy = request_policy.get('insecure', {})
            insecure_action = insecure_policy.get('action', 'Redirect')
            insecure_addl_port = insecure_policy.get('additionalPort', None)

            # The presence of a TLSContext matching our hostname is good enough
            # to go on here, so let's see if there is one.
            ctx = unused_contexts.get(hostname, None)

            # Let's also check to see if the host has a context defined. If it
            # does, check for mismatches.
            if host.context:
                if ctx:
                    if ctx != host.context:
                        # Huh. This is actually "impossible" but let's complain about it
                        # anyway.
                        ir.post_error("Host %s and mismatched TLSContext %s both claim hostname %s?" %
                                      (host.name, ctx.name, hostname))
                        # Skip this Host, something weird is going on.
                        continue

                    # Force additionalPort to 8080 if it's not set at all.
                    if insecure_addl_port is None:
                        ir.logger.info(f"ListenerFactory: Host {hostname} has TLS active, defaulting additionalPort to 8080")
                        insecure_addl_port = 8080
                else:
                    # Huh. This is actually a different kind of "impossible".
                    ctx = host.context
                    ir.post_error("Host %s contains unsaved TLSContext %s?" %
                                  (host.name, ctx.name))
                    # DON'T skip this Host. This "can't happen" but it clearly did.

            # OK, once here, either ctx is not None, or this Host isn't interested in
            # TLS termination.

            if ctx:
                ir.logger.info(f"ListenerFactory: Host {hostname} terminating TLS with context {ctx.name}")

                # We could check for the secure action here, but we're only supporting
                # 'route' right now.

            # So. At this point, we know the hostname, the TLSContext, the secure action,
            # the insecure action, and any additional insecure port. Save everything.

            listener = IRListener(
                ir=ir, aconf=aconf, location=host.location,
                service_port=amod.service_port,
                hostname=hostname,
                # require_tls=amod.get('x_forwarded_proto_redirect', False),
                use_proxy_proto=amod.use_proxy_proto,
                context=ctx,
                secure_action='Route',
                insecure_action=insecure_action,
                insecure_addl_port=insecure_addl_port
            )

            # Do we somehow have a collision on the hostname?
            extant_listener = listeners.get(hostname, None)

            if extant_listener:
                # Uh whut.
                ir.post_error("Hostname %s is defined by both Host %s and Host %s?" %
                              (hostname, extant_listener.name, listener.name))
                continue

            # OK, so far so good. Save what we have so far...
            listeners[hostname] = listener

            # ...make sure we don't try to use this hostname's TLSContext again...
            unused_contexts.pop(hostname, None)

        cls.dump_info(ir, "AFTER HOSTS", listeners, unused_contexts)

        # Walk the remaining unused contexts, if any, and turn them into listeners too.
        for hostname, ctx in unused_contexts.items():
            insecure_action = 'Reject'
            insecure_addl_port = None

            redirect_cleartext_from = ctx.get('redirect_cleartext_from', None)

            if ir.edge_stack_allowed and ctx.is_fallback:
                # If this is the fallback context in Edge Stack, force redirection:
                # this way the fallback context will listen on both ports, to make
                # things easier on the user.
                redirect_cleartext_from = 8080

            if redirect_cleartext_from is not None:
                insecure_action = 'Redirect'
                insecure_addl_port = redirect_cleartext_from

            listener = IRListener(
                ir=ir, aconf=aconf, location=ctx.location,
                service_port=amod.service_port,
                hostname=hostname,
                # require_tls=amod.get('x_forwarded_proto_redirect', False),
                use_proxy_proto=amod.use_proxy_proto,
                context=ctx,
                secure_action='Route',
                insecure_action=insecure_action,
                insecure_addl_port=insecure_addl_port
            )

            listeners[hostname] = listener

        unused_contexts = {}

        cls.dump_info(ir, "AFTER CONTEXTS", listeners, unused_contexts)

        # If we have no listeners, that implies that we had no Hosts _and_ no termination contexts,
        # so let's synthesize a fallback listener. We'll default to using Route as the insecure action
        # (which means accepting either TLS or cleartext), but x_forwarded_proto_redirect can override
        # that.

        xfp_redirect = amod.get('x_forwarded_proto_redirect', False)
        insecure_action = "Redirect" if xfp_redirect else "Route"

        if not listeners:
            listeners['*'] = IRListener(
                ir=ir, aconf=aconf, location=amod.location,
                service_port=amod.service_port,
                hostname='*',
                # require_tls=amod.get('x_forwarded_proto_redirect', False),
                use_proxy_proto=amod.use_proxy_proto,
                context=None,
                secure_action='Route',
                insecure_action=insecure_action,
                insecure_addl_port=None
            )

        cls.dump_info(ir, "AFTER FALLBACK", listeners, unused_contexts)

        # OK. Now that all that's taken care of, add these listeners to the IR.
        for hostname, listener in listeners.items():
            ir.add_listener(listener)

    @classmethod
    def dump_info(cls, ir, what, listeners, unused_contexts):
        ir.logger.debug(f"ListenerFactory: {what}")

        pretty_listeners = {k: v.pretty() for k, v in listeners.items()}
        ir.logger.debug(f"listeners: {json.dumps(pretty_listeners, sort_keys=True, indent=4)}")

        pretty_contexts = {k: v.pretty() for k, v in unused_contexts.items()}
        ir.logger.debug(f"unused_contexts: {json.dumps(pretty_contexts, sort_keys=True, indent=4)}")

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        pass
