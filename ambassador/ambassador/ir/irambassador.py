from typing import Any, ClassVar, Dict, List, Optional, TYPE_CHECKING

from ..constants import Constants

from ..config import Config

from .irresource import IRResource
from .irhttpmapping import IRHTTPMapping
from .irtls import IRAmbassadorTLS
from .irtlscontext import IRTLSContext
from .ircors import IRCORS
from .irretrypolicy import IRRetryPolicy
from .irbuffer import IRBuffer
from .irgzip import IRGzip
from .irfilter import IRFilter

if TYPE_CHECKING:
    from .ir import IR


class IRAmbassador (IRResource):
    AModTransparentKeys: ClassVar = [
        'admin_port',
        'auth_enabled',
        'circuit_breakers',
        'default_label_domain',
        'default_labels',
        'diag_port',
        'diagnostics',
        'enable_ipv6',
        'enable_ipv4',
        'liveness_probe',
        'load_balancer',
        'readiness_probe',
        'resolver',
        'server_name',
        'service_port',
        'statsd',
        'use_proxy_proto',
        'use_remote_address',
        'x_forwarded_proto_redirect',
        'xff_num_trusted_hops',
        'enable_http10'
    ]

    service_port: int
    diag_port: int

    # Set up the default probes and such.
    default_liveness_probe: ClassVar[Dict[str, str]] = {
        "prefix": "/ambassador/v0/check_alive",
        "rewrite": "/ambassador/v0/check_alive",
    }

    default_readiness_probe: ClassVar[Dict[str, str]] = {
        "prefix": "/ambassador/v0/check_ready",
        "rewrite": "/ambassador/v0/check_ready",
    }

    default_diagnostics: ClassVar[Dict[str, str]] = {
        "prefix": "/ambassador/v0/",
        "rewrite": "/ambassador/v0/",
    }

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.ambassador",
                 kind: str="IRAmbassador",
                 name: str="ir.ambassador",
                 use_remote_address: bool=True,
                 **kwargs) -> None:
        # print("IRAmbassador __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            service_port=Constants.SERVICE_PORT_HTTP,
            admin_port=Constants.ADMIN_PORT,
            diag_port=Constants.DIAG_PORT,
            auth_enabled=None,
            enable_ipv6=False,
            enable_ipv4=True,
            liveness_probe={"enabled": True},
            readiness_probe={"enabled": True},
            diagnostics={"enabled": True},
            use_proxy_proto=False,
            enable_http10=False,
            use_remote_address=use_remote_address,
            x_forwarded_proto_redirect=False,
            load_balancer=None,
            circuit_breakers=None,
            xff_num_trusted_hops=0,
            server_name="envoy",
            **kwargs
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # We're interested in the 'ambassador' module from the Config, if any...
        amod = aconf.get_module("ambassador")

        # Is there a TLS module in the Ambassador module?
        if amod:
            self.sourced_by(amod)
            self.referenced_by(amod)

            amod_tls = amod.get('tls', None)

            if amod_tls:
                # XXX What a hack. IRAmbassadorTLS.from_resource() should be able to make
                # this painless.
                new_args = dict(amod_tls)
                new_rkey = new_args.pop('rkey', amod.rkey)
                new_kind = new_args.pop('kind', 'Module')
                new_name = new_args.pop('name', 'tls-from-ambassador-module')
                new_location = new_args.pop('location', amod.location)

                # Overwrite any existing TLS module.
                ir.tls_module = IRAmbassadorTLS(ir, aconf,
                                                rkey=new_rkey,
                                                kind=new_kind,
                                                name=new_name,
                                                location=new_location,
                                                **new_args)

                # ir.logger.debug("IRAmbassador saving TLS module: %s" % ir.tls_module.as_json())

        if ir.tls_module:
            self.logger.debug("final TLS module: %s" % ir.tls_module.as_json())

            # Stash a sane rkey and location for contexts we create.
            ctx_rkey = ir.tls_module.get('rkey', self.rkey)
            ctx_location = ir.tls_module.get('location', self.location)

            # The TLS module 'server' and 'client' blocks are actually a _single_ TLSContext
            # to Ambassador.

            server = ir.tls_module.pop('server', None)
            client = ir.tls_module.pop('client', None)

            if server and server.get('enabled', True):
                # We have a server half. Excellent.

                ctx = IRTLSContext.from_legacy(ir, 'server', ctx_rkey, ctx_location,
                                               cert=server, termination=True, validation_ca=client)

                if ctx.is_active():
                    ir.save_tls_context(ctx)

            # Other blocks in the TLS module weren't ever really documented, so I seriously doubt
            # that they're a factor... but, weirdly, we have a test for them...

            for legacy_name, legacy_ctx in ir.tls_module.as_dict().items():
                if (legacy_name.startswith('_') or
                    (legacy_name == 'name') or
                    (legacy_name == 'location') or
                    (legacy_name == 'kind') or
                    (legacy_name == 'enabled')):
                    continue

                ctx = IRTLSContext.from_legacy(ir, legacy_name, ctx_rkey, ctx_location,
                                               cert=legacy_ctx, termination=False, validation_ca=None)

                if ctx.is_active():
                    ir.save_tls_context(ctx)

        # Finally, check TLSContext resources to see if we should enable TLS termination.
        for ctx in ir.get_tls_contexts():
            if ctx.get('hosts', None):
                # This is a termination context
                self.logger.debug("TLSContext %s is a termination context, enabling TLS termination" % ctx.name)
                self.service_port = Constants.SERVICE_PORT_HTTPS

                if ctx.get('ca_cert', None):
                    # Client-side TLS is enabled.
                    self.logger.debug("TLSContext %s enables client certs!" % ctx.name)

        # After that, check for port definitions, probes, etc., and copy them in
        # as we find them.
        for key in IRAmbassador.AModTransparentKeys:
            if amod and (key in amod):
                # Yes. It overrides the default.
                self[key] = amod[key]

        # If we don't have a default label domain, force it to 'ambassador'.
        if not self.get('default_label_domain'):
            self.default_label_domain = 'ambassador'

        # Likewise, if we have no default labels, force an empty dict (it makes life easier
        # on other modules).
        if not self.get('default_labels'):
            self.default_labels: Dict[str, Any] = {}

        # Next up: diag port & services.
        diag_port = aconf.module_lookup('ambassador', 'diag_port', Constants.DIAG_PORT)
        diag_service = "127.0.0.1:%d" % diag_port

        for name, cur, dflt in [
            ("liveness",    self.liveness_probe,  IRAmbassador.default_liveness_probe),
            ("readiness",   self.readiness_probe, IRAmbassador.default_readiness_probe),
            ("diagnostics", self.diagnostics,     IRAmbassador.default_diagnostics)
        ]:
            if cur and cur.get("enabled", False):
                if not cur.get('prefix', None):
                    cur['prefix'] = dflt['prefix']

                if not cur.get('rewrite', None):
                    cur['rewrite'] = dflt['rewrite']

                if not cur.get('service', None):
                    cur['service'] = diag_service

        if amod and ('enable_grpc_http11_bridge' in amod):
            self.grpc_http11_bridge = IRFilter(ir=ir, aconf=aconf,
                                               kind='ir.grpc_http1_bridge',
                                               name='grpc_http1_bridge',
                                               config=dict())
            self.grpc_http11_bridge.sourced_by(amod)
            ir.save_filter(self.grpc_http11_bridge)

        if amod and ('enable_grpc_web' in amod):
            self.grpc_web = IRFilter(ir=ir, aconf=aconf, kind='ir.grpc_web', name='grpc_web', config=dict())
            self.grpc_web.sourced_by(amod)
            ir.save_filter(self.grpc_web)

        if amod and ('lua_scripts' in amod):
            self.lua_scripts = IRFilter(ir=ir, aconf=aconf, kind='ir.lua_scripts', name='lua_scripts',
                                        config={'inline_code': amod.lua_scripts})
            self.lua_scripts.sourced_by(amod)
            ir.save_filter(self.lua_scripts)

        # Gzip.
        if amod and ('gzip' in amod):
            self.gzip = IRGzip(ir=ir, aconf=aconf, location=self.location, **amod.gzip)

            if self.gzip:
                ir.save_filter(self.gzip)
            else:
                return False


         # Buffer.
        if amod and ('buffer' in amod):
            self.buffer = IRBuffer(ir=ir, aconf=aconf, location=self.location, **amod.buffer)

            if self.buffer:
                ir.save_filter(self.buffer)
            else:
                return False

        # Finally, default CORS stuff.
        if amod and ('cors' in amod):
            self.cors = IRCORS(ir=ir, aconf=aconf, location=self.location, **amod.cors)

            if self.cors:
                self.cors.referenced_by(self)
            else:
                return False

        if amod and ('retry_policy' in amod):
            self.retry_policy = IRRetryPolicy(ir=ir, aconf=aconf, location=self.location, **amod.retry_policy)

            if self.retry_policy:
                self.retry_policy.referenced_by(self)
            else:
                return False

        if self.get('load_balancer', None) is not None:
            if not IRHTTPMapping.validate_load_balancer(self['load_balancer']):
                self.post_error("Invalid load_balancer specified: {}".format(self['load_balancer']))
                return False

        if self.get('circuit_breakers', None) is not None:
            if not IRHTTPMapping.validate_circuit_breakers(self['circuit_breakers']):
                self.post_error("Invalid circuit_breakers specified: {}".format(self['circuit_breakers']))
                return False

        return True

    def add_mappings(self, ir: 'IR', aconf: Config):
        for name, cur in [
            ( "liveness",    self.liveness_probe ),
            ( "readiness",   self.readiness_probe ),
            ( "diagnostics", self.diagnostics )
        ]:
            if cur and cur.get("enabled", False):
                name = "internal_%s_probe_mapping" % name

                mapping = IRHTTPMapping(ir, aconf, rkey=self.rkey, name=name, location=self.location,
                                        timeout_ms=10000, **cur)
                mapping.referenced_by(self)
                ir.add_mapping(aconf, mapping)

    def get_default_label_domain(self) -> str:
        return self.default_label_domain

    def get_default_labels(self, domain: Optional[str]=None) -> Optional[List]:
        if not domain:
            domain = self.get_default_label_domain()

        domain_info = self.default_labels.get(domain, {})

        self.logger.debug("default_labels info for %s: %s" % (domain, domain_info))

        return domain_info.get('defaults')

    def get_default_label_prefix(self, domain: Optional[str]=None) -> Optional[List]:
        if not domain:
            domain = self.get_default_label_domain()

        domain_info = self.default_labels.get(domain, {})
        return domain_info.get('label_prefix')

