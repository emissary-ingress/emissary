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

    # All the AModTransparentKeys are copied from the incoming Ambassador resource
    # into the IRAmbassador object partway through IRAmbassador.finalize().
    AModTransparentKeys: ClassVar = [
        'add_linkerd_headers',
        'admin_port',
        'auth_enabled',
        'circuit_breakers',
        'default_label_domain',
        'default_labels',
        # Do not include defaults, that's handled manually in setup.
        'diag_port',
        'diagnostics',
        'enable_http10',
        'enable_ipv6',
        'envoy_log_type',
        'envoy_log_path',
        'envoy_log_format',
        'enable_ipv4',
        'cluster_idle_timeout_ms',
        'listener_idle_timeout_ms',
        'liveness_probe',
        'load_balancer',
        'keepalive',
        'proper_case',
        'readiness_probe',
        'regex_max_size',
        'regex_type',
        'resolver',
        'debug_mode',
        'server_name',
        'service_port',
        'statsd',
        'use_proxy_proto',
        'use_remote_address',
        'x_forwarded_proto_redirect',
        'xff_num_trusted_hops',
        'use_ambassador_namespace_for_service_resolution'
    ]

    service_port: int
    diag_port: int
    default_label_domain: str

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
            envoy_log_type="text",
            envoy_log_path="/dev/fd/1",
            envoy_log_format=None,
            enable_ipv4=True,
            listener_idle_timeout_ms=None,
            liveness_probe={"enabled": True},
            readiness_probe={"enabled": True},
            diagnostics={"enabled": True},
            use_proxy_proto=False,
            enable_http10=False,
            proper_case=False,
            use_remote_address=use_remote_address,
            x_forwarded_proto_redirect=False,
            load_balancer=None,
            circuit_breakers=None,
            xff_num_trusted_hops=0,
            use_ambassador_namespace_for_service_resolution=False,
            server_name="envoy",
            debug_mode=False,
            **kwargs
        )

        self._finalized = False

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # The heavy lifting here is mostly in the finalize() method, so that when we do fallback
        # lookups for TLS configuration stuff, the defaults are present in the Ambassador module.
        #
        # Of course, that means that we have to copy the defaults in here.

        # We're interested in the 'ambassador' module from the Config, if any...
        amod = aconf.get_module("ambassador")

        if amod and 'defaults' in amod:
            self['defaults'] = amod['defaults']

        return True

    def finalize(self, ir: 'IR', aconf: Config) -> bool:
        self._finalized = True

        # Check TLSContext resources to see if we should enable TLS termination.
        to_delete = []

        for ctx_name, ctx in ir.tls_contexts.items():
            if not ctx.resolve():
                # Welllll this ain't good.
                ctx.set_active(False)
                to_delete.append(ctx_name)
            elif ctx.get('hosts', None):
                # This is a termination context
                self.logger.debug("TLSContext %s is a termination context, enabling TLS termination" % ctx.name)
                self.service_port = Constants.SERVICE_PORT_HTTPS

                if ctx.get('ca_cert', None):
                    # Client-side TLS is enabled.
                    self.logger.debug("TLSContext %s enables client certs!" % ctx.name)

        for ctx_name in to_delete:
            del(ir.tls_contexts[ctx_name])

        # After that, walk the AModTransparentKeys and copy all those things from the
        # input into our IRAmbassador.
        #
        # Some of these will get overridden later, and some things not in AModTransparentKeys
        # get handled manually below.
        amod = aconf.get_module("ambassador")

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

        if amod and ('keepalive' in amod):
            self.keepalive = amod['keepalive']

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
            if not IRHTTPMapping.validate_circuit_breakers(self.ir, self['circuit_breakers']):
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

        if ir.edge_stack_allowed:
            if self.diagnostics and self.diagnostics.get("enabled", False):
                ir.logger.info("adding mappings for Edge Policy Console")
                mapping = IRHTTPMapping(ir, aconf, rkey=self.rkey, location=self.location,
                                        name="edgestack-direct-mapping",
                                        metadata_labels={"ambassador_diag_class": "private"},
                                        prefix="/edge_stack/",
                                        rewrite="/edge_stack_ui/edge_stack/",
                                        service="127.0.0.1:8500",
                                        precedence=1000000,
                                        timeout_ms=60000)
                mapping.referenced_by(self)
                ir.add_mapping(aconf, mapping)

                mapping = IRHTTPMapping(ir, aconf, rkey=self.rkey, location=self.location,
                                        name="edgestack-fallback-mapping",
                                        metadata_labels={"ambassador_diag_class": "private"},
                                        prefix="^/$", prefix_regex=True,
                                        rewrite="/edge_stack_ui/",
                                        service="127.0.0.1:8500",
                                        precedence=-1000000,
                                        timeout_ms=60000)
                mapping.referenced_by(self)
                ir.add_mapping(aconf, mapping)
            else:
                ir.logger.info("diagnostics disabled, skipping mapping for Edge Policy Console")

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
