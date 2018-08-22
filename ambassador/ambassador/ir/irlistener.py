from typing import Optional, TYPE_CHECKING

from ..config import Config
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR


class IRListener (IRResource):
    def __init__(self, ir: 'IR', aconf: Config,

                 service_port: int,
                 require_tls: bool,
                 use_proxy_proto: bool,
                 use_remote_address: Optional[bool]=None,

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
            use_remote_address=use_remote_address,
            **kwargs)

class ListenerFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        amod = ir.ambassador_module
        
        primary_listener = IRListener(
            ir=ir, aconf=aconf,
            service_port=amod.service_port,
            require_tls=amod.get('x_forwarded_proto_redirect', False),
            use_proxy_proto=amod.use_proxy_proto,
        )

        if 'use_remote_address' in amod:
            primary_listener.use_remote_address = amod.use_remote_address

        # If x_forwarded_proto_redirect is set, then we enable require_tls in primary listener,
        # which in turn sets require_ssl to true in envoy aconf. Once set, then all requests
        # that contain X-FORWARDED-PROTO set to https, are processes normally by envoy. In all
        # the other cases, including X-FORWARDED-PROTO set to http, a 301 redirect response to
        # https://host is sent
        if primary_listener.require_tls:
            ir.logger.debug("x_forwarded_proto_redirect is set to true, enabling 'require_tls' in listener")

        redirect_cleartext_from = None

        # Is TLS termination enabled?
        ctx = ir.get_tls_context('server')

        if ctx:
            # Yes.
            primary_listener.tls_context = ctx
            redirect_cleartext_from = ctx.get('redirect_cleartext_from')

        ir.add_listener(primary_listener)

        if redirect_cleartext_from:
            new_listener = IRListener(
                ir=ir, aconf=aconf,
                service_port=redirect_cleartext_from,
                use_proxy_proto=amod.use_proxy_proto,
                # Note: no TLS context here, this is a cleartext listener.
                # We can set require_tls True because we can let the upstream
                # tell us about that.
                require_tls=True
            )

            if 'use_remote_address' in amod:
                new_listener.use_remote_address = amod.use_remote_address

            ir.add_listener(new_listener)

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        pass
