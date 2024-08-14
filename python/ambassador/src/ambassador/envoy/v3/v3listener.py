# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License
import logging
from typing import TYPE_CHECKING, Any, Dict, List, Literal, Optional, Set, Tuple, Union
from typing import cast as typecast

from ...ir.irhost import IRHost
from ...ir.irlistener import IRListener
from ...ir.irtcpmappinggroup import IRTCPMappingGroup
from ...utils import parse_bool
from .v3route import (
    DictifiedV3Route,
    V3Route,
    V3RouteVariants,
    hostglob_matches,
    v3prettyroute,
)
from .v3tls import V3TLSContext

if TYPE_CHECKING:
    from ...ir.irhost import IRHost  # pragma: no cover
    from ...ir.irtlscontext import IRTLSContext  # pragma: no cover
    from . import V3Config  # pragma: no cover


# Model an Envoy filter chain.
#
# In Envoy, Listeners contain filter chains, which define the basic processing on a connection.
# Filters include things like the HTTP connection manager, which handles the HTTP protocol, and
# the TCP proxy filter, which does L4 routing.
#
# An Envoy chain doesn't have a "type": it's just an ordered set of filters. However, it _does_
# have a filter_chain_match which specifies what input connections will be processed, and it
# also can have a TLS context to say which certificate to serve if a connection is to be
# processed by the chain.
#
# A basic asymmetry of the chain is that the filter_chain_match can only do hostname matching if SNI
# is available (i.e. we're terminating TLS), which means for our purposes that a chain _with_ TLS
# enabled is fundamentally different from a chain _without_ TLS enabled.  Whether a chain has TLS
# enabled can be checked with the truthiness of `chain.context`.


class V3Chain:
    _config: "V3Config"
    _logger: logging.Logger
    _log_debug: bool

    context: Optional["IRTLSContext"]
    hosts: Dict[str, Union[IRHost, IRTCPMappingGroup]]
    # unique set of sni names to match on chain if terminating TLS
    server_names: Set[str]
    # routes is keyed on a per virtual_host.domain and with routes only matching a vhost
    routes: Dict[str, List[DictifiedV3Route]]

    def __init__(self, config: "V3Config", context: Optional["IRTLSContext"]) -> None:
        self._config = config
        self._logger = self._config.ir.logger
        self._log_debug = self._logger.isEnabledFor(logging.DEBUG)

        self.context = context
        self.hosts = {}
        self.server_names = set([])
        self.routes = {}

    def add_tcphost(self, tcpmapping: IRTCPMappingGroup) -> None:
        if self._log_debug:
            self._logger.debug(
                f"      CHAIN UPDATE: add TCP host: hostname={repr(tcpmapping.get('host'))}"
            )

        if len(self.hosts) > 0:
            # If we have SNI, then each tcp host gets its own Filter Chain, so we should never have more than 1
            # entry in self.hosts; if we don't have SNI then a single FilterChain with no filter_chain_match
            # takes over the entire chain and so we still should not have more than 1 self.hosts then either.
            # We process TCPMappings first so in theory there never should be a `Host` here and if there was
            # another TCPMapping for this FilterChain then it would be a duplicate and the first one wins.
            other = next(iter(self.hosts.values()))
            other_type = (
                "TCPMapping" if isinstance(other, IRTCPMappingGroup) else "Host"
            )
            tcpmapping.post_error(
                f"TCPMapping {tcpmapping.name}: discarding because it conflicts with {other_type} {other.name}"
            )
            return

        hostname = tcpmapping.get("host", "*")

        if self.context:
            self.server_names.add(hostname)

        self.hosts[hostname] = tcpmapping

    def add_httphost(self, host: IRHost) -> None:
        if self._log_debug:
            self._logger.debug(
                f"      CHAIN UPDATE: add HTTP virtual host: hostname={repr(host.hostname)}"
            )

        error_prefix = "TLS Host" if self.context else "Cleartext Host"

        # we need to make sure this chain isn't already owned by TCPMapping
        for other in self.hosts.values():
            # if a TCPMapping is already claiming this filter_chain then we give it precedence and will drop this http host
            # This can happen if a user configures it incorrectly or if a user is using a Host to grab the TLSContext for a TCPMapping.
            # In the latter scenario, we recommend having a TCPMapping fetch its TLS settings directly from a TLSContext
            # rather than indirectly through a Host (legacy). In the former we give TCPMapping precedence on the conflicts.
            if isinstance(other, IRTCPMappingGroup):
                host.post_error(
                    f"{error_prefix} {host.name}: discarding because it conflicts with TCPMapping {other.name}"
                )
                return

        if self.context:
            if not host.context:
                host.post_error(
                    f"{error_prefix} {host.name}: discarding because host is missing TLSContext"
                )
                return

            # In most TLS scenarios a single Host will translate into a single Filter Chain and virtual host. However,
            # when a user wants to allow clients to access the same dns hostname (example.com) on multiple ports like the
            # standard https port of 443 and a non standard port like 8500. Then Hosts can be merged together on a single
            # Filter Chain with multiple virtual hosts. However, we can only group them if they are using the same TLS Contexts
            # because if they were not we wouldn't know which settings to take from which hosts.
            if (
                self.context.name != host.context.name
                or self.context.namespace != host.context.namespace
            ):
                host.post_error(
                    f"{error_prefix} {host.name}: discarding because mismatching TLSContext between Hosts matching on dns hostname={host.sni}"
                )
                return

            self.server_names.add(host.sni)

        self.routes[host.hostname] = []
        self.hosts[host.hostname] = host

    def hostglobs(self) -> List[str]:
        # Get a list of host globs currently set up for this chain.
        return list(self.hosts.keys())

    def matching_hosts(self, route: V3Route) -> List[IRHost]:
        # Get a list of _IRHosts_ that the given route should be matched with.
        rv: List[IRHost] = []
        for host in self.hosts.values():
            if isinstance(host, IRHost) and host.matches_httpgroup(route._group):
                rv.append(host)
        return rv

    def add_route(self, virtual_host: str, route: DictifiedV3Route) -> None:
        """
        add_route will add the route to the matching virtual host. If the
        virtual_host doesn't already exist in routes then we initialize an
        empty list and append the route
        """
        if virtual_host not in self.routes:
            self.routes[virtual_host] = []

        self.routes[virtual_host].append(route)

    def __str__(self) -> str:
        # ctxstr = f" ctx {self.context.name}" if self.context else ""

        return f"CHAIN: tls={bool(self.context)} hostglobs={repr(sorted(self.hostglobs()))}"


def tlscontext_for_tcpmapping(
    irgroup: IRTCPMappingGroup, config: "V3Config"
) -> Optional["IRTLSContext"]:
    group_host = irgroup.get("host")
    if not group_host:
        return None

    # We can pair directly with a 'TLSContext', or get a TLS config through a 'Host'.
    #
    # Give 'Hosts' precedence.  Why?  IDK, it felt right.

    # First, Hosts:

    for irhost in sorted(config.ir.get_hosts(), key=lambda h: h.hostname):
        if irhost.context and hostglob_matches(irhost.hostname, group_host):
            return irhost.context

    # Second, TLSContexts:

    for context in config.ir.get_tls_contexts():
        for context_host in context.get("hosts") or []:
            # Note: this is *not* glob matching.
            if context_host == group_host:
                return context

    return None


# Model an Envoy listener.
#
# In Envoy, Listeners are the top-level configuration element defining a port on which we'll
# listen; in turn, they contain filter chains which define what will be done with a connection.
#
# There is a one-to-one correspondence between an IRListener and an Envoy listener: the logic
# here is all about constructing the Envoy configuration implied by the IRListener.


class V3Listener:
    config: "V3Config"
    _irlistener: IRListener

    @property
    def http3_enabled(self) -> bool:
        return self._irlistener.http3_enabled

    @property
    def socket_protocol(self) -> Literal["TCP", "UDP"]:
        return self._irlistener.socket_protocol

    @property
    def bind_address(self) -> str:
        return self._irlistener.bind_address

    @property
    def port(self) -> int:
        return self._irlistener.port

    @property
    def bind_to(self) -> str:
        return self._irlistener.bind_to()

    @property
    def _stats_prefix(self) -> str:
        return self._irlistener.statsPrefix

    @property
    def _security_model(self) -> Literal["XFP", "SECURE", "INSECURE"]:
        return self._irlistener.securityModel

    @property
    def _l7_depth(self) -> int:
        return self._irlistener.get("l7Depth", 0)

    @property
    def _insecure_only(self) -> bool:
        return self._irlistener.insecure_only

    @property
    def per_connection_buffer_limit_bytes(self) -> Optional[int]:
        return self.config.ir.ambassador_module.get("buffer_limit_bytes", None)

    def __init__(self, config: "V3Config", irlistener: IRListener) -> None:
        super().__init__()

        self.config = config
        self._irlistener = (
            irlistener  # We cache the IRListener to use its match method later
        )

        bindstr = (
            f"-{irlistener.socket_protocol.lower()}-{self.bind_address}"
            if (self.bind_address != "0.0.0.0")
            else ""
        )
        self.name = irlistener.name or f"ambassador-listener{bindstr}-{self.port}"

        self.listener_filters: List[dict] = []
        self.traffic_direction: str = "UNSPECIFIED"
        self._filter_chains: List[dict] = []
        self._base_http_config: Optional[Dict[str, Any]] = None
        self._chains: Dict[str, V3Chain] = {}
        self._tls_ok: bool = False

        # It's important from a performance perspective to wrap debug log statements
        # with this check so we don't end up generating log strings (or even JSON
        # representations) that won't get logged anyway.
        self._log_debug = self.config.ir.logger.isEnabledFor(logging.DEBUG)
        if self._log_debug:
            self.config.ir.logger.debug(
                f"V3Listener {self.name}: created: port={self.port} security_model={self._security_model} l7depth={self._l7_depth}"
            )

        # Build out our listener filters, and figure out if we're an HTTP listener
        # in the process.
        for proto in irlistener.protocolStack:
            if proto == "HTTP":
                # Start by building our base HTTP config...
                self._base_http_config = self.base_http_config()

            if proto == "PROXY":
                # The PROXY protocol needs a listener filter.
                self.listener_filters.append(
                    {"name": "envoy.filters.listener.proxy_protocol"}
                )

            if proto == "TLS":
                # TLS needs a listener filter _and_ we need to remember that this
                # listener is OK with TLS-y things like a termination context, SNI,
                # etc.
                self._tls_ok = True

                ## When UDP we assume it is http/3 listener and configured for quic which has TLS built into the protocol
                ## therefore, we only need to add this when socket_protocol is TCP
                if self.isProtocolTCP():
                    self.listener_filters.append(
                        {"name": "envoy.filters.listener.tls_inspector"}
                    )

            if proto == "TCP":
                # Nothing to do.
                pass

    def add_chain(
        self,
        chain_type: Literal["tcp", "http", "https"],
        context: Optional["IRTLSContext"],
        hostname: str,
        sni: str,
    ) -> V3Chain:
        # Add a chain for a specific Host to this listener, while dealing with the fundamental
        # asymmetry that filter_chain_match can - and should - use SNI whenever the chain has
        # TLS available, but that's simply not available for chains without TLS.
        #
        # The pratical upshot is that we can generate _only one_ HTTP chain, but we can have
        # HTTPS and TCP chains for specfic hostnames. HOWEVER, we still track HTTP chains by
        # hostname, because we can - and do - separate HTTP chains into specific domains.
        #
        # But wait, I hear you cry, why don't we have a separate domain data structure??! The
        # answer is just that it would needlessly add nesting to all our loops and such (this
        # is also why there's no vhost data structure).

        if chain_type == "http":
            assert not context
        if chain_type == "https":
            assert context

        hostname = hostname or "*"

        chain_key = f"tls-{sni}" if context else "cleartext"

        if chain_type == "http":
            chain_key += f"-{hostname}"

        chain = self._chains.get(chain_key)
        verb = "REUSED" if chain else "CREATE"
        if chain is None:
            chain = V3Chain(self.config, context)
            self._chains[chain_key] = chain

        if self._log_debug:
            self.config.ir.logger.debug(
                f"      CHAIN {verb}: tls={bool(context)} host={repr(hostname)} sni={repr(sni)} => chains[{repr(chain_key)}]={repr(chain)}"
            )

        return chain

    def json_helper(self) -> Any:
        log_format = self.config.ir.ambassador_module.get("envoy_log_format", None)
        if log_format is None:
            log_format = {
                "start_time": "%START_TIME%",
                "method": "%REQ(:METHOD)%",
                "path": "%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%",
                "protocol": "%PROTOCOL%",
                "response_code": "%RESPONSE_CODE%",
                "response_flags": "%RESPONSE_FLAGS%",
                "bytes_received": "%BYTES_RECEIVED%",
                "bytes_sent": "%BYTES_SENT%",
                "duration": "%DURATION%",
                "upstream_service_time": "%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%",
                "x_forwarded_for": "%REQ(X-FORWARDED-FOR)%",
                "user_agent": "%REQ(USER-AGENT)%",
                "request_id": "%REQ(X-REQUEST-ID)%",
                "authority": "%REQ(:AUTHORITY)%",
                "upstream_host": "%UPSTREAM_HOST%",
                "upstream_cluster": "%UPSTREAM_CLUSTER%",
                "upstream_local_address": "%UPSTREAM_LOCAL_ADDRESS%",
                "downstream_local_address": "%DOWNSTREAM_LOCAL_ADDRESS%",
                "downstream_remote_address": "%DOWNSTREAM_REMOTE_ADDRESS%",
                "requested_server_name": "%REQUESTED_SERVER_NAME%",
                "istio_policy_status": "%DYNAMIC_METADATA(istio.mixer:status)%",
                "upstream_transport_failure_reason": "%UPSTREAM_TRANSPORT_FAILURE_REASON%",
            }

            tracing_config = self.config.ir.tracing
            if tracing_config and tracing_config.driver == "envoy.tracers.datadog":
                log_format["dd.trace_id"] = "%REQ(X-DATADOG-TRACE-ID)%"
                log_format["dd.span_id"] = "%REQ(X-DATADOG-PARENT-ID)%"
        return log_format

    # access_log constructs the access_log configuration for this V3Listener
    def access_log(self) -> List[dict]:
        access_log: List[dict] = []

        for al in self.config.ir.log_services.values():
            access_log_obj: Dict[str, Any] = {"common_config": al.get_common_config()}
            req_headers = []
            resp_headers = []
            trailer_headers = []

            for additional_header in al.get_additional_headers():
                if additional_header.get("during_request", True):
                    req_headers.append(additional_header.get("header_name"))
                if additional_header.get("during_response", True):
                    resp_headers.append(additional_header.get("header_name"))
                if additional_header.get("during_trailer", True):
                    trailer_headers.append(additional_header.get("header_name"))

            if al.driver == "http":
                access_log_obj["additional_request_headers_to_log"] = req_headers
                access_log_obj["additional_response_headers_to_log"] = resp_headers
                access_log_obj["additional_response_trailers_to_log"] = trailer_headers
                access_log_obj["@type"] = (
                    "type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.HttpGrpcAccessLogConfig"
                )
                access_log.append(
                    {
                        "name": "envoy.access_loggers.http_grpc",
                        "typed_config": access_log_obj,
                    }
                )
            else:
                # inherently TCP right now
                # tcp loggers do not support additional headers
                access_log_obj["@type"] = (
                    "type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.TcpGrpcAccessLogConfig"
                )
                access_log.append(
                    {
                        "name": "envoy.access_loggers.tcp_grpc",
                        "typed_config": access_log_obj,
                    }
                )

        # Use sane access log spec in JSON
        if self.config.ir.ambassador_module.envoy_log_type.lower() == "json":
            log_format = V3Listener.json_helper(self)
            access_log.append(
                {
                    "name": "envoy.access_loggers.file",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
                        "path": self.config.ir.ambassador_module.envoy_log_path,
                        "json_format": log_format,
                    },
                }
            )

        # Use sane access log spec in Typed JSON
        elif self.config.ir.ambassador_module.envoy_log_type.lower() == "typed_json":
            log_format = V3Listener.json_helper(self)
            access_log.append(
                {
                    "name": "envoy.access_loggers.file",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
                        "path": self.config.ir.ambassador_module.envoy_log_path,
                        "typed_json_format": log_format,
                    },
                }
            )
        else:
            # Use a sane access log spec
            log_format = self.config.ir.ambassador_module.get("envoy_log_format", None)

            if not log_format:
                log_format = 'ACCESS [%START_TIME%] "%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% "%REQ(X-FORWARDED-FOR)%" "%REQ(USER-AGENT)%" "%REQ(X-REQUEST-ID)%" "%REQ(:AUTHORITY)%" "%UPSTREAM_HOST%"'

            if self._log_debug:
                self.config.ir.logger.debug(
                    "V3Listener: Using log_format '%s'" % log_format
                )
            access_log.append(
                {
                    "name": "envoy.access_loggers.file",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
                        "path": self.config.ir.ambassador_module.envoy_log_path,
                        "log_format": {
                            "text_format_source": {"inline_string": log_format + "\n"}
                        },
                    },
                }
            )

        return access_log

    # base_http_config constructs the starting configuration for this
    # V3Listener's http_connection_manager filter.
    def base_http_config(self) -> Dict[str, Any]:
        base_http_config: Dict[str, Any] = {
            "stat_prefix": self._stats_prefix,
            "access_log": self.access_log(),
            "http_filters": [],
            "normalize_path": True,
        }

        # Instructs the HTTP Connection Mananger to support http/3. This is required for both TCP and UDP Listeners
        if self.http3_enabled:
            base_http_config["http3_protocol_options"] = {}
            if self.isProtocolUDP():
                base_http_config["codec_type"] = "HTTP3"

        # Assemble base HTTP filters
        from .v3httpfilter import V3HTTPFilter

        for f in self.config.ir.filters:
            v3hf: dict = V3HTTPFilter(f, self.config)

            # V3HTTPFilter can return None to indicate that the filter config
            # should be omitted from the final envoy config. This is the
            # uncommon case, but it can happen if a filter waits utnil the
            # v3config is generated before deciding if it needs to be
            # instantiated. See IRErrorResponse for an example.
            if v3hf:
                base_http_config["http_filters"].append(v3hf)

        if "use_remote_address" in self.config.ir.ambassador_module:
            base_http_config["use_remote_address"] = (
                self.config.ir.ambassador_module.use_remote_address
            )

        if self._l7_depth > 0:
            base_http_config["xff_num_trusted_hops"] = self._l7_depth

        if "server_name" in self.config.ir.ambassador_module:
            base_http_config["server_name"] = (
                self.config.ir.ambassador_module.server_name
            )

        listener_idle_timeout_ms = self.config.ir.ambassador_module.get(
            "listener_idle_timeout_ms", None
        )
        if listener_idle_timeout_ms:
            if "common_http_protocol_options" in base_http_config:
                base_http_config["common_http_protocol_options"]["idle_timeout"] = (
                    "%0.3fs" % (float(listener_idle_timeout_ms) / 1000.0)
                )
            else:
                base_http_config["common_http_protocol_options"] = {
                    "idle_timeout": "%0.3fs"
                    % (float(listener_idle_timeout_ms) / 1000.0)
                }

        if "headers_with_underscores_action" in self.config.ir.ambassador_module:
            if "common_http_protocol_options" in base_http_config:
                base_http_config["common_http_protocol_options"][
                    "headers_with_underscores_action"
                ] = self.config.ir.ambassador_module.headers_with_underscores_action
            else:
                base_http_config["common_http_protocol_options"] = {
                    "headers_with_underscores_action": self.config.ir.ambassador_module.headers_with_underscores_action
                }

        max_request_headers_kb = self.config.ir.ambassador_module.get(
            "max_request_headers_kb", None
        )
        if max_request_headers_kb:
            base_http_config["max_request_headers_kb"] = max_request_headers_kb

        if "enable_http10" in self.config.ir.ambassador_module:
            http_options = base_http_config.setdefault("http_protocol_options", {})
            http_options["accept_http_10"] = (
                self.config.ir.ambassador_module.enable_http10
            )

        if "allow_chunked_length" in self.config.ir.ambassador_module:
            if self.config.ir.ambassador_module.allow_chunked_length is not None:
                http_options = base_http_config.setdefault("http_protocol_options", {})
                http_options["allow_chunked_length"] = (
                    self.config.ir.ambassador_module.allow_chunked_length
                )

        if "preserve_external_request_id" in self.config.ir.ambassador_module:
            base_http_config["preserve_external_request_id"] = (
                self.config.ir.ambassador_module.preserve_external_request_id
            )

        if "forward_client_cert_details" in self.config.ir.ambassador_module:
            base_http_config["forward_client_cert_details"] = (
                self.config.ir.ambassador_module.forward_client_cert_details
            )

        if "set_current_client_cert_details" in self.config.ir.ambassador_module:
            base_http_config["set_current_client_cert_details"] = (
                self.config.ir.ambassador_module.set_current_client_cert_details
            )

        if self.config.ir.tracing:
            base_http_config["generate_request_id"] = True

            base_http_config["tracing"] = {}
            self.traffic_direction = "OUTBOUND"

            custom_tags = self.config.ir.tracing.get("custom_tags", [])
            if custom_tags:
                base_http_config["tracing"]["custom_tags"] = custom_tags

            sampling = self.config.ir.tracing.get("sampling", {})
            if sampling:
                client_sampling = sampling.get("client", None)
                if client_sampling is not None:
                    base_http_config["tracing"]["client_sampling"] = {
                        "value": client_sampling
                    }

                random_sampling = sampling.get("random", None)
                if random_sampling is not None:
                    base_http_config["tracing"]["random_sampling"] = {
                        "value": random_sampling
                    }

                overall_sampling = sampling.get("overall", None)
                if overall_sampling is not None:
                    base_http_config["tracing"]["overall_sampling"] = {
                        "value": overall_sampling
                    }

        proper_case: bool = self.config.ir.ambassador_module["proper_case"]

        # Get the list of downstream headers whose casing should be overriden
        # from the Ambassador module. We configure the upstream side of this
        # in v3cluster.py
        header_case_overrides = self.config.ir.ambassador_module.get(
            "header_case_overrides", None
        )
        if header_case_overrides:
            if proper_case:
                self.config.ir.post_error(
                    "Only one of 'proper_case' or 'header_case_overrides' fields may be set on "
                    + "the Ambassador module. Honoring proper_case and ignoring "
                    + "header_case_overrides."
                )
                header_case_overrides = None
            if not isinstance(header_case_overrides, list):
                # The header_case_overrides field must be an array.
                self.config.ir.post_error(
                    "Ambassador module config 'header_case_overrides' must be an array"
                )
                header_case_overrides = None
            elif len(header_case_overrides) == 0:
                # Allow an empty list to mean "do nothing".
                header_case_overrides = None

        if header_case_overrides:
            # We have this config validation here because the Ambassador module is
            # still an untyped config. That is, we aren't yet using a CRD or a
            # python schema to constrain the configuration that can be present.
            rules = []
            for hdr in header_case_overrides:
                if not isinstance(hdr, str):
                    self.config.ir.post_error(
                        "Skipping non-string header in 'header_case_overrides': {hdr}"
                    )
                    continue
                rules.append(hdr)

            if len(rules) == 0:
                self.config.ir.post_error(
                    f"Could not parse any valid string headers in 'header_case_overrides': {header_case_overrides}"
                )
            else:
                # Create custom header rules that map the lowercase version of every element in
                # `header_case_overrides` to the the respective original casing.
                #
                # For example the input array [ X-HELLO-There, X-COOL ] would create rules:
                # { 'x-hello-there': 'X-HELLO-There', 'x-cool': 'X-COOL' }. In envoy, this effectively
                # overrides the response header case by remapping the lowercased version (the default
                # casing in envoy) back to the casing provided in the config.
                custom_header_rules: Dict[str, Dict[str, dict]] = {
                    "custom": {"rules": {header.lower(): header for header in rules}}
                }
                http_options = base_http_config.setdefault("http_protocol_options", {})
                http_options["header_key_format"] = custom_header_rules

        if proper_case:
            proper_case_header: Dict[str, Dict[str, dict]] = {
                "header_key_format": {"proper_case_words": {}}
            }
            if "http_protocol_options" in base_http_config:
                base_http_config["http_protocol_options"].update(proper_case_header)
            else:
                base_http_config["http_protocol_options"] = proper_case_header

        return base_http_config

    def finalize(self) -> None:
        if self._log_debug:
            self.config.ir.logger.debug(
                f"V3Listener {self}: finalize ============================"
            )

        # We do TCP chains before HTTP chains so that TCPMappings have precedence over Hosts.  This
        # is important because 2.x releases prior to 2.4 required you to create a Host for the
        # TCPMapping to steal the TLS termination config from (so TCPMapping users coming from 2.3
        # will _very likely_ have "conflicting" Hosts and TCPMappings), and also didn't support
        # TCPMappings and Hosts on the same Listener (so 2.3 didn't see these as "conflicts").  But
        # now that we do support them together on the same Listener, we do see them as conflicts,
        # and so we keep compatibility with 2.3 by saying "in the event of a conflict, TCPMappings
        # have precedence over Hosts."
        self.compute_tcpchains()
        self.finalize_tcp()

        if self._base_http_config:
            self.compute_httpchains()
            self.compute_http_routes()
            self.finalize_http()

    def finalize_tcp(self) -> None:
        # Finalize a TCP listener, which amounts to walking all our TCP chains and
        # setting up Envoy configuration structures for them.

        self.config.ir.logger.debug("  finalize_tcp")

        for chain_key, chain in self._chains.items():
            if self._log_debug:
                self.config.ir.logger.debug(
                    f"    build chain[{repr(chain_key)}]={chain}"
                )

            for irgroup in chain.hosts.values():
                if not isinstance(irgroup, IRTCPMappingGroup):
                    continue

                # First up, which clusters do we need to talk to?
                clusters = [
                    {"name": mapping.cluster.envoy_name, "weight": mapping._weight}
                    for mapping in irgroup.mappings
                ]

                # From that, we can sort out a basic tcp_proxy filter config.
                tcp_filter = {
                    "name": "envoy.filters.network.tcp_proxy",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy",
                        "stat_prefix": self._stats_prefix,
                        "weighted_clusters": {"clusters": clusters},
                    },
                }

                # OK. Basic filter chain entry next.
                filter_chain: Dict[str, Any] = {
                    "name": f"tcphost-{irgroup.name}",
                    "filters": [tcp_filter],
                }

                filter_chain_match: Dict[str, Any] = {}

                if chain.context:
                    filter_chain_match["transport_protocol"] = "tls"

                    # Note that we're modifying the filter_chain itself here, not
                    # filter_chain_match.
                    envoy_ctx = V3TLSContext(chain.context)

                    filter_chain["transport_socket"] = {
                        "name": "envoy.transport_sockets.tls",
                        "typed_config": {
                            "@type": "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext",
                            **envoy_ctx,
                        },
                    }

                # We do server-name matching whether or not we have TLS, just to help
                # make sure that we don't have two chains with an empty filter_match
                # criterion (since Envoy will reject such a configuration).
                server_names = list(chain.server_names)

                if len(server_names) > 0 and ("*" not in server_names):
                    filter_chain_match["server_names"] = server_names

                filter_chain["filter_chain_match"] = filter_chain_match

                self._filter_chains.append(filter_chain)

    def compute_tcpchains(self) -> None:
        self.config.ir.logger.debug("  compute_tcpchains")

        for irgroup in self.config.ir.ordered_groups():
            if not isinstance(irgroup, IRTCPMappingGroup):
                continue

            if self._log_debug:
                self.config.ir.logger.debug(f"    consider {irgroup}")

            if irgroup.bind_to() != self.bind_to:
                self.config.ir.logger.debug("      reject")
                continue

            self.config.ir.logger.debug("      accept")

            # Add a chain, same as we do in compute_httpchains, just for a 'TCPMappingGroup' rather
            # than for a 'Host'.  Same deal applies with TLS: you can't do host-based matching
            # without it.

            group_host = irgroup.get("host", None)
            if not group_host:  # cleartext
                # Special case. No host (aka hostname) in a TCPMapping means an unconditional forward,
                # so just add this immediately as a "*" chain.
                self.add_chain("tcp", None, "*", "*").add_tcphost(irgroup)
            else:  # TLS/SNI
                context = tlscontext_for_tcpmapping(irgroup, self.config)
                if not context:
                    irgroup.post_error("No matching TLSContext found, disabling!")
                    continue

                # group_host comes from `TCPMapping.host` which is expected to be a valid dns hostname
                # without a port so no need to parse out a port
                sni = group_host
                self.add_chain("tcp", context, group_host, sni).add_tcphost(irgroup)

    def compute_httpchains(self) -> None:
        # Compute the set of chains we need, HTTP version. The core here is matching
        # up Hosts with this Listener, and creating a chain for each Host.

        self.config.ir.logger.debug("  compute_httpchains")

        for host in sorted(self.config.ir.get_hosts(), key=lambda h: h.hostname):
            if self._log_debug:
                self.config.ir.logger.debug(f"    consider {host}")

            # First up: drop this host if nothing matches at all.
            if not self._irlistener.matches_host(host):
                self.config.ir.logger.debug("      reject: hostglobs don't match")
                continue

            # OK, if we're still here, then it's a question of matching the Listener's
            # SecurityModel with the Host's requestPolicy. It happens that it's actually
            # pretty hard to reject things at this level.
            #
            # First up, if the Listener is marked insecure-only, but the Listener's port
            # doesn't match the Host's insecure_addl_port, don't take this Host: this
            # Listener was synthesized to handle some other Host. (This is a corner case that
            # will become less and less likely as more people hop on the Listener bandwagon.
            # Also, remember that Hosts don't specify bind addresses, so only the port matters
            # here.)

            if self._insecure_only and (self.port != host.insecure_addl_port):
                self.config.ir.logger.debug("      reject: insecure-only port mismatch")
                continue

            # OK, we can't drop it for that, so we need to check the actions.

            # If the Listener's securityModel is SECURE, but this host has a secure_action
            # of Reject (or empty), we'll skip this host, because the only requests this
            # Listener can ever produce will be rejected. In any other case, we'll set up an
            # HTTPS chain for this Host, as long as we think TLS is OK.
            host_will_reject_secure = (not host.secure_action) or (
                host.secure_action == "Reject"
            )
            if (
                self._tls_ok
                and host.context
                and (
                    not ((self._security_model == "SECURE") and host_will_reject_secure)
                )
            ):
                self.config.ir.logger.debug("      accept SECURE")
                self.add_chain(
                    "https", host.context, host.hostname, host.sni
                ).add_httphost(host)

            # Same idea on the insecure side: only skip the Host if the Listener's securityModel
            # is INSECURE but the Host's insecure_action is Reject.
            if not (
                (self._security_model == "INSECURE")
                and (host.insecure_action == "Reject")
            ):
                self.config.ir.logger.debug("      accept INSECURE")
                self.add_chain("http", None, host.hostname, host.sni).add_httphost(host)

    def compute_http_routes(self) -> None:
        # Compute the set of valid HTTP routes for _each chain_ in this Listener.
        #
        # Note that a route using XFP can match _any_ chain, whether HTTP or HTTPS.

        self.config.ir.logger.debug("  compute_routes")

        for chain_key, chain in self._chains.items():
            if self._log_debug:
                self.config.ir.logger.debug(
                    f"    consider chain[{repr(chain_key)}]={chain}"
                )

            # Only look at HTTP(S) chains.
            if not any(isinstance(h, IRHost) for h in chain.hosts.values()):
                self.config.ir.logger.debug("      reject: is non-HTTP")
                continue

            # Remember whether we found an ACME route.
            # found_acme = False

            # The data structure we're walking here is config.route_variants rather than
            # config.routes. There's a one-to-one correspondence between the two, but we use the
            # V3RouteVariants to lazily cache some of the work that we're doing across chains.
            for rv in self.config.route_variants:
                if self._log_debug:
                    self.config.ir.logger.debug(
                        f"        consider route {v3prettyroute(dict(rv.route))}"
                    )

                matching_hosts = chain.matching_hosts(rv.route)

                if self._log_debug:
                    self.config.ir.logger.debug(
                        f"          matching_hosts={[h.hostname for h in matching_hosts]}"
                    )

                if not matching_hosts:
                    if self._log_debug:
                        self.config.ir.logger.debug(
                            f"          reject: no hosts match {sorted(rv.route['_host_constraints'])}"
                        )
                    continue

                for host in matching_hosts:
                    # For each host, we need to look at things for the secure world as well
                    # as the insecure world, depending on what the action is exactly (and note
                    # that, yes, we can have an action of None for an insecure_only listener).
                    #
                    # "candidates" is a list of tuples (host, matcher, action, V3RouteVariants)
                    candidates: List[Tuple[IRHost, str, str, V3RouteVariants]] = []
                    hostname = host.hostname

                    if self._log_debug:
                        self.config.ir.logger.debug(f"          host={hostname}")

                    if (host.secure_action is not None) and (
                        self._security_model != "INSECURE"
                    ):
                        # We have a secure action, and we're willing to believe that at least some of
                        # our requests will be secure.
                        matcher = (
                            "always"
                            if (self._security_model == "SECURE")
                            else "xfp-https"
                        )

                        candidates.append((host, matcher, "Route", rv))

                    if (host.insecure_action is not None) and (
                        self._security_model != "SECURE"
                    ):
                        # We have an insecure action, and we're willing to believe that at least some of
                        # our requests will be insecure.
                        matcher = (
                            "always"
                            if (self._security_model == "INSECURE")
                            else "xfp-http"
                        )
                        action = host.insecure_action

                        candidates.append((host, matcher, action, rv))

                    for host, matcher, action, rv in candidates:
                        # route_precedence = rv.route.get("_precedence", None)
                        extra_info = ""

                        if (
                            rv.route["match"].get("prefix", None)
                            == "/.well-known/acme-challenge/"
                        ):
                            # We need to be sure to route ACME challenges, no matter what else is going
                            # on (this is the infamous ACME hole-puncher mentioned everywhere).
                            extra_info = " (force Route for ACME challenge)"
                            action = "Route"
                            # found_acme = True
                        # elif (
                        #     self.config.ir.edge_stack_allowed
                        #     and (route_precedence == -1000000)
                        #     and (
                        #         rv.route["match"].get("safe_regex", {}).get("regex", None) == "^/$"
                        #     )
                        # ):
                        #     extra_info = " (force Route for fallback Mapping)"
                        #     action = "Route"

                        if action != "Reject":
                            # Worth noting here that "Route" really means "do what the V3Route really
                            # says", which might be a host redirect. When we talk about "Redirect", we
                            # really mean "redirect to HTTPS" specifically.

                            if self._log_debug:
                                self.config.ir.logger.debug(
                                    f"          route: accept matcher={matcher} action={action} {extra_info}"
                                )

                            variant = dict(rv.get_variant(matcher, action.lower()))
                            variant["_host_constraints"] = set([hostname])
                            # virtual_host domains are key by the hostname for :authority header matching
                            chain.add_route(hostname, variant)
                        else:
                            if self._log_debug:
                                self.config.ir.logger.debug(
                                    f"          route: reject matcher={matcher} action={action} {extra_info}"
                                )

            # # If we're on Edge Stack and we don't already have an ACME route, add one.
            # if self.config.ir.edge_stack_allowed and not found_acme:
            #     # This route is needed to trigger an ExtAuthz request for the AuthService.
            #     # The auth service grabs the challenge and does the right thing.
            #     # Rather than try to route to some existing cluster we can just return a
            #     # direct response. What we return doesn't really matter but
            #     # to match existing Edge Stack behavior we return a 404 response.

            #     self.config.ir.logger.debug("      punching a hole for ACME")

            #     # we need to make sure the acme route is added to every virtual host domain
            #     # so we must insert the route into each unique domains list of routes
            #     for hostname in chain.hosts:
            #         # Make sure to include _host_constraints in here for now so it can be
            #         # applied to the correct vhost during future proccessing
            #         chain.routes[hostname].insert(
            #             0,
            #             {
            #                 "_host_constraints": set(),
            #                 "match": {
            #                     "case_sensitive": True,
            #                     "prefix": "/.well-known/acme-challenge/",
            #                 },
            #                 "direct_response": {"status": 404},
            #             },
            #         )

            if self._log_debug:
                for hostname in chain.hosts:
                    for route in chain.routes[hostname]:
                        self.config.ir.logger.debug(
                            f"  CHAIN ROUTE: vhost={hostname} {v3prettyroute(route)}"
                        )

    def finalize_http(self) -> None:
        # Finalize everything HTTP. Like the TCP side of the world, this is about walking
        # chains and generating Envoy config.
        #
        # All of our HTTP chains get collapsed into a single chain with (likely) multiple
        # domains here.

        self.config.ir.logger.debug("  finalize_http")

        filter_chains: Dict[str, Dict[str, Any]] = {}

        for chain_key, chain in self._chains.items():
            if not any(isinstance(h, IRHost) for h in chain.hosts.values()):
                continue

            if self._log_debug:
                self._irlistener.logger.debug(
                    f"    build chain[{repr(chain_key)}]={chain}"
                )

            filter_chain: Optional[Dict[str, Any]] = None

            if not chain.context:  # cleartext
                # http/3 is built on quic which has TLS built-in. This means that our UDP Listener will only ever need routes
                # that match the TLS Filter chain and will diverge from the TCP listener in that it will not support redirect
                # therefore, we can exclude duplicating the filterchain and routes so hitting this endpoint using non-tls http will fail
                if self.isProtocolUDP() and self.http3_enabled:
                    continue

                if self._log_debug:
                    self._irlistener.logger.debug(
                        f"      cleartext for hostglobs={chain.hostglobs()}"
                    )
                # All HTTP chains get collapsed into one here, using domains to separate them.
                # This works because we don't need to offer TLS certs (we can't anyway), and
                # because of that, SNI (and thus filter server_names matches) aren't things.
                chain_key = "http"

                filter_chain = filter_chains.get(chain_key, None)

                if not filter_chain:
                    if self._log_debug:
                        self._irlistener.logger.debug(
                            "FHTTP   create filter_chain %s / empty match", chain_key
                        )
                    filter_chain = {
                        "name": "httphost-shared",
                        "filter_chain_match": {},
                        "_vhosts": {},
                    }

                    filter_chains[chain_key] = filter_chain
                else:
                    if self._log_debug:
                        self._irlistener.logger.debug(
                            "FHTTP   use filter_chain %s: vhosts %d",
                            chain_key,
                            len(filter_chain["_vhosts"]),
                        )
            else:  # TLS/SNI
                # Since chain_key is a dictionary key in its own right, we can't already
                # have a matching chain for this.

                filter_chain = {
                    "name": f"httpshost-{next(iter(chain.hosts.values())).name}",
                    "_vhosts": {},
                }
                filter_chain_match: Dict[str, Any] = {}

                server_names = list(chain.server_names)

                if self._log_debug:
                    chain_hosts = chain.hostglobs()
                    self._irlistener.logger.debug(
                        f"      tls for hostglobs={chain_hosts} matched with server_names={server_names}"
                    )

                # Set up the server_names part of the match, if we have any names.
                #
                # Note that "*" is _not allowed_ in server_names, though e.g. "*.example.com"
                # is. So we need to filter out the "*" itself... which is ugly, because
                #
                # server_names: [ "*", "foo.example.com" ]
                #
                # is very different from
                #
                # server_names: [ "foo.example.com" ]
                #
                # So, if "*" is present at all in our server_names, we can't match server_names
                # at all.

                if (len(server_names) > 0) and ("*" not in server_names):
                    filter_chain_match["server_names"] = server_names

                # Likewise, an HTTPS chain will ask for TLS or QUIC (when udp)
                filter_chain_match["transport_protocol"] = (
                    "quic" if self.isProtocolUDP() and self.http3_enabled else "tls"
                )

                envoy_ctx = V3TLSContext(chain.context)

                envoy_tls_config = {
                    "name": "envoy.transport_sockets.tls",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext",
                        **envoy_ctx,
                    },
                }

                if self.isProtocolUDP():
                    envoy_tls_config = {
                        "name": "envoy.transport_sockets.quic",
                        "typed_config": {
                            "@type": "type.googleapis.com/envoy.extensions.transport_sockets.quic.v3.QuicDownstreamTransport",
                            "downstream_tls_context": {**envoy_ctx},
                        },
                    }

                filter_chain["transport_socket"] = envoy_tls_config

                # Finally, stash the match in the chain...
                filter_chain["filter_chain_match"] = filter_chain_match

                # ...and save it.
                filter_chains[chain_key] = filter_chain

            # OK, we have the filter_chain variable set -- build the Envoy virtual_hosts for it.

            for host in chain.hosts.values():
                if not isinstance(host, IRHost):
                    continue

                if self._log_debug:
                    self._irlistener.logger.debug(
                        f"      adding vhost {repr(host.hostname)}"
                    )

                # Make certain that no internal keys from the route make it into the Envoy
                # configuration.
                routes = []

                for r in chain.routes[host.hostname]:
                    routes.append({k: v for k, v in r.items() if k[0] != "_"})

                # Do we - somehow - already have a vhost for this hostname? (This should
                # be "impossible".)

                vhost: Dict[str, Any] = filter_chain["_vhosts"].get(host.hostname, None)

                if not vhost:
                    vhost = {
                        "name": f"{self.name}-{host.hostname}",
                        "response_headers_to_add": [],
                        "domains": [host.hostname],
                        "routes": [],
                    }

                    if self.http3_enabled and (self.socket_protocol == "TCP"):
                        # Setting the alternative service header, tells the client to use the alternate location for future requests.
                        # Clients such as chrome/firefox, etc... require this to instruct it to start speaking http/3 with the server.
                        #
                        # Additional reading on alt-svc header: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Alt-Svc
                        #
                        # The default sets the max-age in seconds to be 1 day and supports clients that speak h3 & h3-29 specifications
                        alt_svc_hdr = {
                            "key": "alt-svc",
                            "value": 'h3=":443"; ma=86400, h3-29=":443"; ma=86400',
                        }

                        vhost["response_headers_to_add"].append({"header": alt_svc_hdr})
                    else:
                        del vhost["response_headers_to_add"]

                    filter_chain["_vhosts"][host.hostname] = vhost

                vhost["routes"] += routes

        # Once that's all done, walk the filter_chains dict...
        for fc_key, filter_chain in filter_chains.items():
            # ...set up our HTTP config...
            http_config = dict(typecast(dict, self._base_http_config))

            # ...and unfold our vhosts dict into a list for Envoy.
            http_config["route_config"] = {
                "virtual_hosts": list(filter_chain["_vhosts"].values())
            }

            # Now that we've saved our vhosts as a list, drop the dict version.
            del filter_chain["_vhosts"]

            # Finish up config for this filter chain...
            if parse_bool(
                self.config.ir.ambassador_module.get(
                    "strip_matching_host_port", "false"
                )
            ):
                http_config["strip_matching_host_port"] = True

            if parse_bool(
                self.config.ir.ambassador_module.get("merge_slashes", "false")
            ):
                http_config["merge_slashes"] = True

            if parse_bool(
                self.config.ir.ambassador_module.get(
                    "reject_requests_with_escaped_slashes", "false"
                )
            ):
                http_config["path_with_escaped_slashes_action"] = "REJECT_REQUEST"

            filter_chain["filters"] = [
                {
                    "name": "envoy.filters.network.http_connection_manager",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                        **http_config,
                    },
                }
            ]

            # ...and save it.
            self._filter_chains.append(filter_chain)

    def as_dict(self) -> dict:
        listener: dict = {
            "name": self.name,
            "address": {
                "socket_address": {
                    "address": self.bind_address,
                    "port_value": self.port,
                    "protocol": self.socket_protocol,  ## "TCP" or "UDP"
                }
            },
            "filter_chains": self._filter_chains,
            "traffic_direction": self.traffic_direction,
        }

        if self.isProtocolUDP():
            listener["udp_listener_config"] = {
                "quic_options": {},
                "downstream_socket_config": {"prefer_gro": True},
            }

        # We only want to add the buffer limit setting to the listener if specified in the module.
        # Otherwise, we want to leave it unset and allow Envoys Default 1MiB setting.
        if self.per_connection_buffer_limit_bytes:
            listener["per_connection_buffer_limit_bytes"] = (
                self.per_connection_buffer_limit_bytes
            )

        if self.listener_filters:
            listener["listener_filters"] = self.listener_filters

        return listener

    def __str__(self) -> str:
        return "<V3Listener %s %s on %s:%d [%s]>" % (
            "HTTP" if self._base_http_config else "TCP",
            self.name,
            self.bind_address,
            self.port,
            self._security_model,
        )

    def isProtocolTCP(self) -> bool:
        """Whether the listener is configured to use the TCP protocol or not?"""
        return self.socket_protocol == "TCP"

    def isProtocolUDP(self) -> bool:
        """Whether the listener is configured to use the UDP protocol or not?"""
        return self.socket_protocol == "UDP"

    @classmethod
    def generate(cls, config: "V3Config") -> None:
        config.listeners = []

        for key in config.ir.listeners.keys():
            irlistener = config.ir.listeners[key]
            v3listener = V3Listener(config, irlistener)
            v3listener.finalize()

            config.ir.logger.info(
                f"V3Listener {v3listener}: generated ==========================="
            )
            if config.ir.logger.isEnabledFor(logging.DEBUG):
                if v3listener._log_debug:
                    for k in sorted(v3listener._chains.keys()):
                        chain = v3listener._chains[k]
                        config.ir.logger.debug(f"  chain {chain}")
                        for hostname in chain.hosts:
                            config.ir.logger.debug(f"    host {hostname}")
                            routes = chain.routes.get(hostname, [])
                            for r in routes:
                                config.ir.logger.debug(
                                    f"      route {v3prettyroute(r)}"
                                )

            # Does this listener have any filter chains?
            if v3listener._filter_chains:
                config.listeners.append(v3listener)
            else:
                irlistener.post_error("No matching Hosts/TCPMappings found, disabling!")
