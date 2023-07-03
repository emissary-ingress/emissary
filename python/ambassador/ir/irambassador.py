from typing import TYPE_CHECKING, Any, ClassVar, Dict, List, Optional

from ..config import Config
from ..constants import Constants
from .irbasemapping import IRBaseMapping
from .irbuffer import IRBuffer
from .ircors import IRCORS
from .irfilter import IRFilter
from .irgzip import IRGzip
from .irhttpmapping import IRHTTPMapping
from .iripallowdeny import IRIPAllowDeny
from .irresource import IRResource
from .irretrypolicy import IRRetryPolicy

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


class IRAmbassador(IRResource):
    # All the AModTransparentKeys are copied from the incoming Ambassador resource
    # into the IRAmbassador object partway through IRAmbassador.finalize().
    #
    # PLEASE KEEP THIS LIST SORTED.

    AModTransparentKeys: ClassVar = [
        "add_linkerd_headers",
        "admin_port",
        "auth_enabled",
        "allow_chunked_length",
        "buffer_limit_bytes",
        "circuit_breakers",
        "cluster_idle_timeout_ms",
        "cluster_max_connection_lifetime_ms",
        "cluster_request_timeout_ms",
        "debug_mode",
        # Do not include defaults, that's handled manually in setup.
        "default_label_domain",
        "default_labels",
        "diagnostics",
        "enable_http10",
        "enable_ipv4",
        "enable_ipv6",
        "envoy_log_format",
        "envoy_log_path",
        "envoy_log_type",
        "forward_client_cert_details",
        # Do not include envoy_validation_timeout; we let finalize() type-check it.
        # Do not include ip_allow or ip_deny; we let finalize() type-check them.
        "headers_with_underscores_action",
        "keepalive",
        "listener_idle_timeout_ms",
        "liveness_probe",
        "load_balancer",
        "max_request_headers_kb",
        "merge_slashes",
        "reject_requests_with_escaped_slashes",
        "preserve_external_request_id",
        "proper_case",
        "prune_unreachable_routes",
        "readiness_probe",
        "regex_max_size",
        "regex_type",
        "resolver",
        "error_response_overrides",
        "header_case_overrides",
        "server_name",
        "service_port",
        "set_current_client_cert_details",
        "statsd",
        "strip_matching_host_port",
        "suppress_envoy_headers",
        "use_ambassador_namespace_for_service_resolution",
        "use_proxy_proto",
        "use_remote_address",
        "x_forwarded_proto_redirect",
        "xff_num_trusted_hops",
    ]

    service_port: int
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

    # Set up the default Envoy validation timeout. This is deliberately chosen to be very large
    # because the consequences of this timeout tripping are very bad. Ambassador basically ceases
    # to function. It is far better to slow down as our configurations grow and give users a
    # leading indicator that there is a scaling issue that needs to be dealt with than to
    # suddenly and mysteriously stop functioning the day their configuration happens to become
    # large enough to exceed this threshold.
    default_validation_timeout: ClassVar[int] = 60

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.ambassador",
        kind: str = "IRAmbassador",
        name: str = "ir.ambassador",
        use_remote_address: bool = True,
        **kwargs,
    ) -> None:
        # print("IRAmbassador __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir,
            aconf=aconf,
            rkey=rkey,
            kind=kind,
            name=name,
            service_port=Constants.SERVICE_PORT_HTTP,
            admin_port=Constants.ADMIN_PORT,
            auth_enabled=None,
            enable_ipv6=False,
            envoy_log_type="text",
            envoy_log_path="/dev/fd/1",
            envoy_log_format=None,
            envoy_validation_timeout=IRAmbassador.default_validation_timeout,
            enable_ipv4=True,
            listener_idle_timeout_ms=None,
            liveness_probe={"enabled": True},
            readiness_probe={"enabled": True},
            diagnostics={"enabled": True},  # TODO(lukeshu): In getambassador.io/v3alpha2, change
            # the default to {"enabled": False}.  See the related
            # comment in crd_module.go.
            use_proxy_proto=False,
            enable_http10=False,
            proper_case=False,
            prune_unreachable_routes=True,  # default True; can be updated in finalize()
            use_remote_address=use_remote_address,
            x_forwarded_proto_redirect=False,
            load_balancer=None,
            circuit_breakers=None,
            xff_num_trusted_hops=0,
            use_ambassador_namespace_for_service_resolution=False,
            server_name="envoy",
            debug_mode=False,
            preserve_external_request_id=False,
            max_request_headers_kb=None,
            **kwargs,
        )

        self.ip_allow_deny: Optional[IRIPAllowDeny] = None
        self._finalized = False

    def setup(self, ir: "IR", aconf: Config) -> bool:
        # The heavy lifting here is mostly in the finalize() method, so that when we do fallback
        # lookups for TLS configuration stuff, the defaults are present in the Ambassador module.
        #
        # Of course, that means that we have to copy the defaults in here.

        # We're interested in the 'ambassador' module from the Config, if any...
        amod = aconf.get_module("ambassador")

        if amod and "defaults" in amod:
            self["defaults"] = amod["defaults"]

        return True

    def finalize(self, ir: "IR", aconf: Config) -> bool:
        self._finalized = True

        # Check TLSContext resources to see if we should enable TLS termination.
        to_delete = []

        for ctx_name, ctx in ir.tls_contexts.items():
            if not ctx.resolve():
                # Welllll this ain't good.
                ctx.set_active(False)
                to_delete.append(ctx_name)
            elif ctx.get("hosts", None):
                # This is a termination context
                self.logger.debug(
                    "TLSContext %s is a termination context, enabling TLS termination" % ctx.name
                )
                self.service_port = Constants.SERVICE_PORT_HTTPS

                if ctx.get("ca_cert", None):
                    # Client-side TLS is enabled.
                    self.logger.debug("TLSContext %s enables client certs!" % ctx.name)

        for ctx_name in to_delete:
            del ir.tls_contexts[ctx_name]

        # After that, walk the AModTransparentKeys and copy all those things from the
        # input into our IRAmbassador.
        #
        # Some of these will get overridden later, and some things not in AModTransparentKeys
        # get handled manually below.
        amod = aconf.get_module("ambassador")

        if amod:
            for key in IRAmbassador.AModTransparentKeys:
                if key in amod:
                    # Override the default here.
                    self[key] = amod[key]

            # If we have an envoy_validation_timeout...
            if "envoy_validation_timeout" in amod:
                # ...then set our timeout from it.
                try:
                    self.envoy_validation_timeout = int(amod["envoy_validation_timeout"])
                except ValueError:
                    self.post_error("envoy_validation_timeout must be an integer number of seconds")

        # If we don't have a default label domain, force it to 'ambassador'.
        if not self.get("default_label_domain"):
            self.default_label_domain = "ambassador"

        # Likewise, if we have no default labels, force an empty dict (it makes life easier
        # on other modules).
        if not self.get("default_labels"):
            self.default_labels: Dict[str, Any] = {}

        # Next up: diag port & services.
        diag_service = "127.0.0.1:%d" % Constants.DIAG_PORT

        for name, cur, dflt in [
            ("liveness", self.liveness_probe, IRAmbassador.default_liveness_probe),
            ("readiness", self.readiness_probe, IRAmbassador.default_readiness_probe),
            ("diagnostics", self.diagnostics, IRAmbassador.default_diagnostics),
        ]:
            if cur and cur.get("enabled", False):
                if not cur.get("prefix", None):
                    cur["prefix"] = dflt["prefix"]

                if not cur.get("rewrite", None):
                    cur["rewrite"] = dflt["rewrite"]

                if not cur.get("service", None):
                    cur["service"] = diag_service

        if amod and ("enable_grpc_http11_bridge" in amod):
            self.grpc_http11_bridge = IRFilter(
                ir=ir,
                aconf=aconf,
                kind="ir.grpc_http1_bridge",
                name="grpc_http1_bridge",
                config=dict(),
            )
            self.grpc_http11_bridge.sourced_by(amod)
            ir.save_filter(self.grpc_http11_bridge)

        if amod and ("enable_grpc_web" in amod):
            self.grpc_web = IRFilter(
                ir=ir, aconf=aconf, kind="ir.grpc_web", name="grpc_web", config=dict()
            )
            self.grpc_web.sourced_by(amod)
            ir.save_filter(self.grpc_web)

        if amod and (grpc_stats := amod.get("grpc_stats")) is not None:
            # grpc_stats = { 'all_methods': False} if amod.grpc_stats is None else amod.grpc_stats
            # default config with safe values
            config: Dict[str, Any] = {"enable_upstream_stats": False}

            # Only one of config['individual_method_stats_allowlist'] or
            # config['stats_for_all_methods'] can be set.
            if "services" in grpc_stats:
                config["individual_method_stats_allowlist"] = {"services": grpc_stats["services"]}
            else:
                config["stats_for_all_methods"] = bool(grpc_stats.get("all_methods", False))

            if "upstream_stats" in grpc_stats:
                config["enable_upstream_stats"] = bool(grpc_stats["upstream_stats"])

            self.grpc_stats = IRFilter(
                ir=ir, aconf=aconf, kind="ir.grpc_stats", name="grpc_stats", config=config
            )
            self.grpc_stats.sourced_by(amod)
            ir.save_filter(self.grpc_stats)

        if amod and ("lua_scripts" in amod):
            self.lua_scripts = IRFilter(
                ir=ir,
                aconf=aconf,
                kind="ir.lua_scripts",
                name="lua_scripts",
                config={"inline_code": amod.lua_scripts},
            )
            self.lua_scripts.sourced_by(amod)
            ir.save_filter(self.lua_scripts)

        # Gzip.
        if amod and ("gzip" in amod):
            self.gzip = IRGzip(ir=ir, aconf=aconf, location=self.location, **amod.gzip)

            if self.gzip:
                ir.save_filter(self.gzip)
            else:
                return False

        # Buffer.
        if amod and ("buffer" in amod):
            self.buffer = IRBuffer(ir=ir, aconf=aconf, location=self.location, **amod.buffer)

            if self.buffer:
                ir.save_filter(self.buffer)
            else:
                return False

        if amod and ("keepalive" in amod):
            self.keepalive = amod["keepalive"]

        # Finally, default CORS stuff.
        if amod and ("cors" in amod):
            self.cors = IRCORS(ir=ir, aconf=aconf, location=self.location, **amod.cors)

            if self.cors:
                self.cors.referenced_by(self)
            else:
                return False

        if amod and ("retry_policy" in amod):
            self.retry_policy = IRRetryPolicy(
                ir=ir, aconf=aconf, location=self.location, **amod.retry_policy
            )

            if self.retry_policy:
                self.retry_policy.referenced_by(self)
            else:
                return False

        if amod:
            if "ip_allow" in amod:
                self.handle_ip_allow_deny(allow=True, principals=amod.ip_allow)

            if "ip_deny" in amod:
                self.handle_ip_allow_deny(allow=False, principals=amod.ip_deny)

            if self.ip_allow_deny is not None:
                ir.save_filter(self.ip_allow_deny)

                # Clear this so it doesn't get duplicated when we dump the
                # Ambassador module.
                self.ip_allow_deny = None

        if self.get("load_balancer", None) is not None:
            if not IRHTTPMapping.validate_load_balancer(self["load_balancer"]):
                self.post_error("Invalid load_balancer specified: {}".format(self["load_balancer"]))
                return False

        if self.get("circuit_breakers", None) is not None:
            if not IRBaseMapping.validate_circuit_breakers(self.ir, self["circuit_breakers"]):
                self.post_error(
                    "Invalid circuit_breakers specified: {}".format(self["circuit_breakers"])
                )
                return False

        if self.get("envoy_log_type") == "text":
            if self.get("envoy_log_format", None) is not None and not isinstance(
                self.get("envoy_log_format"), str
            ):
                self.post_error(
                    "envoy_log_type 'text' requires a string in envoy_log_format: {}, invalidating...".format(
                        self.get("envoy_log_format")
                    )
                )
                self["envoy_log_format"] = ""
                return False
        elif self.get("envoy_log_type") == "json":
            if self.get("envoy_log_format", None) is not None and not isinstance(
                self.get("envoy_log_format"), dict
            ):
                self.post_error(
                    "envoy_log_type 'json' requires a dictionary in envoy_log_format: {}, invalidating...".format(
                        self.get("envoy_log_format")
                    )
                )
                self["envoy_log_format"] = {}
                return False
        else:
            self.post_error(
                "Invalid log_type specified: {}. Supported: json, text".format(
                    self.get("envoy_log_type")
                )
            )
            return False

        if self.get("forward_client_cert_details") is not None:
            # https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/network/http_connection_manager/v3/http_connection_manager.proto#envoy-v3-api-enum-extensions-filters-network-http-connection-manager-v3-httpconnectionmanager-forwardclientcertdetails
            valid_values = (
                "SANITIZE",
                "FORWARD_ONLY",
                "APPEND_FORWARD",
                "SANITIZE_SET",
                "ALWAYS_FORWARD_ONLY",
            )

            value = self.get("forward_client_cert_details")
            if value not in valid_values:
                self.post_error(
                    "'forward_client_cert_details' may not be set to '{}'; it may only be set to one of: {}".format(
                        value, ", ".join(valid_values)
                    )
                )
                return False

        cert_details = self.get("set_current_client_cert_details")
        if cert_details:
            # https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/network/http_connection_manager/v3/http_connection_manager.proto#envoy-v3-api-msg-extensions-filters-network-http-connection-manager-v3-httpconnectionmanager-setcurrentclientcertdetails
            valid_keys = ("subject", "cert", "chain", "dns", "uri")

            for k, v in cert_details.items():
                if k not in valid_keys:
                    self.post_error(
                        "'set_current_client_cert_details' may not contain key '{}'; it may only contain keys: {}".format(
                            k, ", ".join(valid_keys)
                        )
                    )
                    return False

                if v not in (True, False):
                    self.post_error(
                        "'set_current_client_cert_details' value for key '{}' may only be 'true' or 'false', not '{}'".format(
                            k, v
                        )
                    )
                    return False

        return True

    def add_mappings(self, ir: "IR", aconf: Config):
        for name, cur in [
            ("liveness", self.liveness_probe),
            ("readiness", self.readiness_probe),
            ("diagnostics", self.diagnostics),
        ]:
            if cur and cur.get("enabled", False):
                name = "internal_%s_probe_mapping" % name
                cache_key = "InternalMapping-v2-%s-default" % name

                mapping = ir.cache_fetch(cache_key)

                if mapping is not None:
                    # Cache hit. We know a priori that anything in the cache under a Mapping
                    # key must be an IRBaseMapping, but let's assert that rather than casting.
                    assert isinstance(mapping, IRBaseMapping)
                else:
                    mapping = IRHTTPMapping(
                        ir,
                        aconf,
                        kind="InternalMapping",
                        rkey=self.rkey,
                        name=name,
                        location=self.location,
                        timeout_ms=10000,
                        hostname="*",
                        **cur,
                    )
                    mapping.referenced_by(self)

                ir.add_mapping(aconf, mapping)

        # if ir.edge_stack_allowed:
        #     if self.diagnostics and self.diagnostics.get("enabled", False):
        #         ir.logger.debug("adding mappings for Edge Policy Console")
        #         edge_stack_response_header = {"x-content-type-options": "nosniff"}
        #         mapping = IRHTTPMapping(ir, aconf, rkey=self.rkey, location=self.location,
        #                                 name="edgestack-direct-mapping",
        #                                 metadata_labels={"ambassador_diag_class": "private"},
        #                                 prefix="/edge_stack/",
        #                                 rewrite="/edge_stack_ui/edge_stack/",
        #                                 service="127.0.0.1:8500",
        #                                 precedence=1000000,
        #                                 timeout_ms=60000,
        #                                 hostname="*",
        #                                 add_response_headers=edge_stack_response_header)
        #         mapping.referenced_by(self)
        #         ir.add_mapping(aconf, mapping)

        #         mapping = IRHTTPMapping(ir, aconf, rkey=self.rkey, location=self.location,
        #                                 name="edgestack-fallback-mapping",
        #                                 metadata_labels={"ambassador_diag_class": "private"},
        #                                 prefix="^/$", prefix_regex=True,
        #                                 rewrite="/edge_stack_ui/",
        #                                 service="127.0.0.1:8500",
        #                                 precedence=-1000000,
        #                                 timeout_ms=60000,
        #                                 hostname="*",
        #                                 add_response_headers=edge_stack_response_header)
        #         mapping.referenced_by(self)
        #         ir.add_mapping(aconf, mapping)
        #     else:
        #         ir.logger.debug("diagnostics disabled, skipping mapping for Edge Policy Console")

    def get_default_label_domain(self) -> str:
        return self.default_label_domain

    def get_default_labels(self, domain: Optional[str] = None) -> Optional[List]:
        if not domain:
            domain = self.get_default_label_domain()

        domain_info = self.default_labels.get(domain, {})

        self.logger.debug("default_labels info for %s: %s" % (domain, domain_info))

        return domain_info.get("defaults")

    def handle_ip_allow_deny(self, allow: bool, principals: List[str]) -> None:
        """
        Handle IP Allow/Deny. "allow" here states whether this is an
        allow rule (True) or a deny rule (False); "principals" is a list
        of IP addresses or CIDR ranges to allow or deny.

        Only one of ip_allow or ip_deny can be set, so it's an error to
        call this twice (even if "allow" is the same for both calls).

        :param allow: True for an ALLOW rule, False for a DENY rule
        :param principals: list of IP addresses or CIDR ranges to match
        """

        if self.get("ip_allow_deny") is not None:
            self.post_error("ip_allow and ip_deny may not both be set")
            return

        ipa = IRIPAllowDeny(
            self.ir,
            self.ir.aconf,
            rkey=self.rkey,
            parent=self,
            action="ALLOW" if allow else "DENY",
            principals=principals,
        )

        if ipa:
            self["ip_allow_deny"] = ipa
