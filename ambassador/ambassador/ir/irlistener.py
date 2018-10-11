from typing import Optional, Tuple, TYPE_CHECKING

from ..config import Config
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR


class IRListener (IRResource):
    @staticmethod
    def helper_contexts(res: IRResource, k: str) -> Tuple[str, dict]:
        return k, { ctx_key: ctx.as_dict() for ctx_key, ctx in res[k].items() }

    def __init__(self, ir: 'IR', aconf: Config,

                 service_port: int,
                 require_tls: bool,
                 use_proxy_proto: bool,

                 rkey: str="ir.listener",
                 kind: str="IRListener",
                 name: str="ir.listener",
                 **kwargs) -> None:
        # print("IRListener __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            service_port=service_port,
            require_tls=require_tls,
            use_proxy_proto=use_proxy_proto,
            **kwargs)

        self.redirect_listener: bool = False
        self.add_dict_helper('tls_contexts', IRListener.helper_contexts)

class ListenerFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        amod = ir.ambassador_module
        
        primary_listener = IRListener(
            ir=ir, aconf=aconf, location=amod.location,
            service_port=amod.service_port,
            require_tls=amod.get('x_forwarded_proto_redirect', False),
            use_proxy_proto=amod.use_proxy_proto
        )

        # If x_forwarded_proto_redirect is set, then we enable require_tls in primary listener,
        # which in turn sets require_ssl to true in envoy aconf. Once set, then all requests
        # that contain X-FORWARDED-PROTO set to https, are processes normally by envoy. In all
        # the other cases, including X-FORWARDED-PROTO set to http, a 301 redirect response to
        # https://host is sent
        if primary_listener.require_tls:
            ir.logger.debug("x_forwarded_proto_redirect is set to true, enabling 'require_tls' in listener")

        redirect_cleartext_from = None

        # What do we know about TLS?
        # XXX This will have to change as we mess more with arbitrary contexts.
        contexts = {}
        ctx_location = amod.location

        override_source = bool(amod.location == '--internal--')

        for ctxname in [ 'server', 'client' ]:
            ctx = ir.get_tls_context(ctxname)

            if not ctx:
                continue

            # ir.logger.debug("primary listener: ctx %s: %s" % (ctxname, ctx.as_json()))

            if ctx.enabled:
                contexts[ctxname] = ctx

                if override_source:
                    primary_listener.sourced_by(ctx)
                    override_source = False

                # XXX Should we be making sure that this is a termination context somehow??
                if 'redirect_cleartext_from' in ctx:
                    redirect_cleartext_from = ctx.redirect_cleartext_from

                    if 'location' in ctx:
                        ctx_location = ctx.location

                    # ir.logger.debug("primary listener: ctx %s sets redirect_cleartext_from %s" %
                    #                 (ctxname, redirect_cleartext_from))

        if contexts:
            primary_listener['tls_contexts'] = contexts

        if 'use_remote_address' in amod:
            primary_listener.use_remote_address = amod.use_remote_address

        ir.add_listener(primary_listener)

        if redirect_cleartext_from:
            new_listener = IRListener(
                ir=ir, aconf=aconf, location=ctx_location,
                service_port=redirect_cleartext_from,
                use_proxy_proto=amod.use_proxy_proto,
                # Note: no TLS context here, this is a cleartext listener.
                # We can set require_tls True because we can let the upstream
                # tell us about that.
                require_tls=True
            )

            new_listener.redirect_listener = True

            if 'use_remote_address' in amod:
                new_listener.use_remote_address = amod.use_remote_address

            ir.add_listener(new_listener)

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        pass
