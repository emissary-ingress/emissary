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
from typing import Any, Dict, List, Optional, Tuple, TYPE_CHECKING
from typing import cast as typecast

from os import environ

import logging
import sys

from ...ir.irhost import IRHost
from ...ir.irlistener import IRListener
from ...ir.irtcpmappinggroup import IRTCPMappingGroup

from ...utils import dump_json, parse_bool

from .v3httpfilter import V3HTTPFilter
from .v3route import V3Route, DictifiedV3Route, V3RouteVariants, v3prettyroute, hostglob_matches
from .v3tls import V3TLSContext

if TYPE_CHECKING:
    from ...ir.irhost import IRHost             # pragma: no cover
    from ...ir.irtlscontext import IRTLSContext # pragma: no cover
    from . import V3Config                      # pragma: no cover


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
# A basic asymmetry of the chain is that the filter_chain_match can only do hostname matching
# if TLS (and thus SNI) is in play, which means for our purposes that a chain _with_ TLS enabled
# is fundamentally different from a chain _without_ TLS enabled. We encapsulate that idea in
# the "type" parameter, which can be "http", "https", or "tcp" depending on how the chain will
# be used. (And yes, that implies that at the moment, you can't mix HTTP Mappings and TCP Mappings
# on the same port. Possible near-future feature.)

class V3Chain(dict):
    def __init__(self, config: 'V3Config', type: str, host: Optional[IRHost]) -> None:
        self._config = config
        self._logger = self._config.ir.logger
        self._log_debug = self._logger.isEnabledFor(logging.DEBUG)

        self.type = type

        # We can have multiple hosts here, primarily so that HTTP chains can DTRT --
        # but it would be fine to have multiple HTTPS hosts too, as long as they all
        # share a TLSContext.
        self.context: Optional[IRTLSContext]= None
        self.hosts: Dict[str, IRHost] = {}

        # It's OK if an HTTP chain has no Host.
        if host:
            self.add_host(host)

        self.routes: List[DictifiedV3Route] = []
        self.tcpmappings: List[IRTCPMappingGroup] = []

    def add_host(self, host: IRHost) -> None:
        self.hosts[host.hostname] = host

        # Don't mess with the context if we're an HTTP chain...
        if self.type.lower() == "http":
            return

        # OK, we're some type where TLS makes sense. Do the thing.
        if host.context:
            if not self.context:
                self.context = host.context
            elif self.context != host.context:
                self._config.ir.post_error("Chain context mismatch: Host %s cannot combine with %s" %
                                           (host.name, ", ".join(sorted(self.hosts.keys()))))

    def hostglobs(self) -> List[str]:
        # Get a list of host globs currently set up for this chain.
        return list(self.hosts.keys())

    def matching_hosts(self, route: V3Route) -> List[IRHost]:
        # Get a list of _IRHosts_ that the given route should be matched with.
        rv: List[IRHost] = [ host for host in self.hosts.values() if host.matches_httpgroup(route._group) ]

        return rv

    def add_route(self, route: DictifiedV3Route) -> None:
        self.routes.append(route)

    def add_tcpmapping(self, tcpmapping: IRTCPMappingGroup) -> None:
        self.tcpmappings.append(tcpmapping)

    def __str__(self) -> str:
        ctxstr = f" ctx {self.context.name}" if self.context else ""

        return "CHAIN: %s%s [ %s ]" % \
               (self.type.upper(), ctxstr, ", ".join(sorted(self.hostglobs())))


# Model an Envoy listener.
#
# In Envoy, Listeners are the top-level configuration element defining a port on which we'll
# listen; in turn, they contain filter chains which define what will be done with a connection.
#
# There is a one-to-one correspondence between an IRListener and an Envoy listener: the logic
# here is all about constructing the Envoy configuration implied by the IRListener.

class V3Listener(dict):
    def __init__(self, config: 'V3Config', irlistener: IRListener) -> None:
        super().__init__()

        self.config = config
        self.bind_address = irlistener.bind_address
        self.port = irlistener.port
        self.bind_to = f"{self.bind_address}-{self.port}"

        bindstr = f"-{self.bind_address}" if (self.bind_address != "0.0.0.0") else ""
        self.name = irlistener.name or f"ambassador-listener{bindstr}-{self.port}"

        self.use_proxy_proto = False
        self.listener_filters: List[dict] = []
        self.traffic_direction: str = "UNSPECIFIED"
        self._irlistener = irlistener   # We cache the IRListener to use its match method later
        self._stats_prefix = irlistener.statsPrefix
        self._security_model: str = irlistener.securityModel
        self._l7_depth: int = irlistener.get('l7Depth', 0)
        self._insecure_only: bool = False
        self._filter_chains: List[dict] = []
        self._base_http_config: Optional[Dict[str, Any]] = None
        self._chains: Dict[str, V3Chain] = {}
        self._tls_ok: bool = False

        # It's important from a performance perspective to wrap debug log statements
        # with this check so we don't end up generating log strings (or even JSON
        # representations) that won't get logged anyway.
        self._log_debug = self.config.ir.logger.isEnabledFor(logging.DEBUG)
        if self._log_debug:
            self.config.ir.logger.debug(f"V3Listener {self.name} created -- {self._security_model}, l7Depth {self._l7_depth}")

        # If the IRListener is marked insecure-only, so are we.
        self._insecure_only = irlistener.insecure_only

        # Build out our listener filters, and figure out if we're an HTTP listener
        # in the process.
        for proto in irlistener.protocolStack:
            if proto == "HTTP":
                # Start by building our base HTTP config...
                self._base_http_config = self.base_http_config()

            if proto == "PROXY":
                # The PROXY protocol needs a listener filter.
                self.listener_filters.append({
                    'name': 'envoy.filters.listener.proxy_protocol'
                })

            if proto == "TLS":
                # TLS needs a listener filter _and_ we need to remember that this
                # listener is OK with TLS-y things like a termination context, SNI,
                # etc.
                self._tls_ok = True
                self.listener_filters.append({
                    'name': 'envoy.filters.listener.tls_inspector'
                })

            if proto == "TCP":
                # TCP doesn't require any specific listener filters, but it
                # does require stuff in the filter chains. We can go ahead and
                # tackle that here.
                for irgroup in self.config.ir.ordered_groups():
                    # Only look at TCPMappingGroups here...
                    if not isinstance(irgroup, IRTCPMappingGroup):
                        continue

                    # ...and make sure the group in question wants the same bind
                    # address that we do.
                    if irgroup.bind_to() != self.bind_to:
                        # self.config.ir.logger.debug("V3Listener %s: skip TCPMappingGroup on %s", self.bind_to, irgroup.bind_to())
                        continue

                    self.add_tcp_group(irgroup)

    def add_chain(self, chain_type: str, host: Optional[IRHost]) -> V3Chain:
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

        chain_key = chain_type
        hoststr = host.hostname if host else '(no host)'
        hostname = (host.hostname if host else None) or '*'

        if host:
            chain_key = "%s-%s" % (chain_type, hostname)

        chain = self._chains.get(chain_key)

        if chain is not None:
            if host:
                chain.add_host(host)
                if self._log_debug:
                    self.config.ir.logger.debug("      CHAIN ADD: host %s chain_key %s -- %s", hoststr, chain_key, chain)
            else:
                if self._log_debug:
                    self.config.ir.logger.debug("      CHAIN NOOP: host %s chain_key %s -- %s", hoststr, chain_key, chain)
        else:
            chain = V3Chain(self.config, chain_type, host)
            self._chains[chain_key] = chain
            if self._log_debug:
                self.config.ir.logger.debug("      CHAIN CREATE: host %s chain_key %s -- %s", hoststr, chain_key, chain)

        return chain

    def add_tcp_group(self, irgroup: IRTCPMappingGroup) -> None:
        # The TCP analog of add_chain -- it adds a chain, too, but works with a TCP
        # mapping group rather than a Host. Same deal applies with TLS: you can't do
        # host-based matching without it.

        group_host = irgroup.get('host', None)

        if self._log_debug:
            self.config.ir.logger.debug("V3Listener %s on %s: take TCPMappingGroup on %s (%s)",
                                        self.name, self.bind_to, irgroup.bind_to(), group_host or "i'*'")

        if not group_host:
            # Special case. No Host in a TCPMapping means an unconditional forward,
            # so just add this immediately as a "*" chain.
            chain = self.add_chain("tcp", None)
            chain.add_tcpmapping(irgroup)
        else:
            # What matching Hosts do we have?
            for host in sorted(self.config.ir.get_hosts(), key=lambda h: h.hostname):
                # They're asking for a hostname match here, which _cannot happen_ without
                # SNI -- so don't take any hosts that don't have a TLSContext.

                if not host.context:
                    if self._log_debug:
                        self.config.ir.logger.debug("V3Listener %s @ %s TCP %s: skip %s",
                                                    self.name, self.bind_to, group_host, host)
                    continue

                if self._log_debug:
                    self.config.ir.logger.debug("V3Listener %s @ %s TCP %s: consider %s",
                                                self.name, self.bind_to, group_host, host)

                if hostglob_matches(host.hostname, group_host):
                    chain = self.add_chain("tcp", host)
                    chain.add_tcpmapping(irgroup)

    # access_log constructs the access_log configuration for this V3Listener
    def access_log(self) -> List[dict]:
        access_log: List[dict] = []

        for al in self.config.ir.log_services.values():
            access_log_obj: Dict[str, Any] = { "common_config": al.get_common_config() }
            req_headers = []
            resp_headers = []
            trailer_headers = []

            for additional_header in al.get_additional_headers():
                if additional_header.get('during_request', True):
                    req_headers.append(additional_header.get('header_name'))
                if additional_header.get('during_response', True):
                    resp_headers.append(additional_header.get('header_name'))
                if additional_header.get('during_trailer', True):
                    trailer_headers.append(additional_header.get('header_name'))

            if al.driver == 'http':
                access_log_obj['additional_request_headers_to_log'] = req_headers
                access_log_obj['additional_response_headers_to_log'] = resp_headers
                access_log_obj['additional_response_trailers_to_log'] = trailer_headers
                access_log_obj['@type'] = 'type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.HttpGrpcAccessLogConfig'
                access_log.append({
                    "name": "envoy.access_loggers.http_grpc",
                    "typed_config": access_log_obj
                })
            else:
                # inherently TCP right now
                # tcp loggers do not support additional headers
                access_log_obj['@type'] = 'type.googleapis.com/envoy.extensions.access_loggers.grpc.v3.TcpGrpcAccessLogConfig'
                access_log.append({
                    "name": "envoy.access_loggers.tcp_grpc",
                    "typed_config": access_log_obj
                })

        # Use sane access log spec in JSON
        if self.config.ir.ambassador_module.envoy_log_type.lower() == "json":
            log_format = self.config.ir.ambassador_module.get('envoy_log_format', None)
            if log_format is None:
                log_format = {
                    'start_time': '%START_TIME%',
                    'method': '%REQ(:METHOD)%',
                    'path': '%REQ(X-ENVOY-ORIGINAL-PATH?:PATH)%',
                    'protocol': '%PROTOCOL%',
                    'response_code': '%RESPONSE_CODE%',
                    'response_flags': '%RESPONSE_FLAGS%',
                    'bytes_received': '%BYTES_RECEIVED%',
                    'bytes_sent': '%BYTES_SENT%',
                    'duration': '%DURATION%',
                    'upstream_service_time': '%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%',
                    'x_forwarded_for': '%REQ(X-FORWARDED-FOR)%',
                    'user_agent': '%REQ(USER-AGENT)%',
                    'request_id': '%REQ(X-REQUEST-ID)%',
                    'authority': '%REQ(:AUTHORITY)%',
                    'upstream_host': '%UPSTREAM_HOST%',
                    'upstream_cluster': '%UPSTREAM_CLUSTER%',
                    'upstream_local_address': '%UPSTREAM_LOCAL_ADDRESS%',
                    'downstream_local_address': '%DOWNSTREAM_LOCAL_ADDRESS%',
                    'downstream_remote_address': '%DOWNSTREAM_REMOTE_ADDRESS%',
                    'requested_server_name': '%REQUESTED_SERVER_NAME%',
                    'istio_policy_status': '%DYNAMIC_METADATA(istio.mixer:status)%',
                    'upstream_transport_failure_reason': '%UPSTREAM_TRANSPORT_FAILURE_REASON%'
                }

                tracing_config = self.config.ir.tracing
                if tracing_config and tracing_config.driver == 'envoy.tracers.datadog':
                    log_format['dd.trace_id'] = '%REQ(X-DATADOG-TRACE-ID)%'
                    log_format['dd.span_id'] = '%REQ(X-DATADOG-PARENT-ID)%'

            access_log.append({
                'name': 'envoy.access_loggers.file',
                'typed_config': {
                    '@type': 'type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog',
                    'path': self.config.ir.ambassador_module.envoy_log_path,
                    'json_format': log_format
                }
            })
        else:
            # Use a sane access log spec
            log_format = self.config.ir.ambassador_module.get('envoy_log_format', None)

            if not log_format:
                log_format = 'ACCESS [%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\"'

            if self._log_debug:
                self.config.ir.logger.debug("V3Listener: Using log_format '%s'" % log_format)
            access_log.append({
                'name': 'envoy.access_loggers.file',
                'typed_config': {
                    '@type': 'type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog',
                    'path': self.config.ir.ambassador_module.envoy_log_path,
                    'log_format': {
                        'text_format_source': {
                            'inline_string': log_format + '\n'
                        }
                    }
                }
            })

        return access_log

    # base_http_config constructs the starting configuration for this
    # V3Listener's http_connection_manager filter.
    def base_http_config(self) -> Dict[str, Any]:
        base_http_config: Dict[str, Any] = {
            'stat_prefix': self._stats_prefix,
            'access_log': self.access_log(),
            'http_filters': [],
            'normalize_path': True
        }

        # Assemble base HTTP filters
        for f in self.config.ir.filters:
            v3hf: dict = V3HTTPFilter(f, self.config)

            # V3HTTPFilter can return None to indicate that the filter config
            # should be omitted from the final envoy config. This is the
            # uncommon case, but it can happen if a filter waits utnil the
            # v3config is generated before deciding if it needs to be
            # instantiated. See IRErrorResponse for an example.
            if v3hf:
                base_http_config['http_filters'].append(v3hf)

        if 'use_remote_address' in self.config.ir.ambassador_module:
            base_http_config["use_remote_address"] = self.config.ir.ambassador_module.use_remote_address

        if 'xff_num_trusted_hops' in self.config.ir.ambassador_module:
            base_http_config["xff_num_trusted_hops"] = self.config.ir.ambassador_module.xff_num_trusted_hops

        if 'server_name' in self.config.ir.ambassador_module:
            base_http_config["server_name"] = self.config.ir.ambassador_module.server_name

        listener_idle_timeout_ms = self.config.ir.ambassador_module.get('listener_idle_timeout_ms', None)
        if listener_idle_timeout_ms:
            if 'common_http_protocol_options' in base_http_config:
                base_http_config["common_http_protocol_options"]["idle_timeout"] = "%0.3fs" % (float(listener_idle_timeout_ms) / 1000.0)
            else:
                base_http_config["common_http_protocol_options"] = { 'idle_timeout': "%0.3fs" % (float(listener_idle_timeout_ms) / 1000.0) }

        if 'headers_with_underscores_action' in self.config.ir.ambassador_module:
            if 'common_http_protocol_options' in base_http_config:
                base_http_config["common_http_protocol_options"]["headers_with_underscores_action"] = self.config.ir.ambassador_module.headers_with_underscores_action
            else:
                base_http_config["common_http_protocol_options"] = { 'headers_with_underscores_action': self.config.ir.ambassador_module.headers_with_underscores_action }

        max_request_headers_kb = self.config.ir.ambassador_module.get('max_request_headers_kb', None)
        if max_request_headers_kb:
            base_http_config["max_request_headers_kb"] = max_request_headers_kb

        if 'enable_http10' in self.config.ir.ambassador_module:
            http_options = base_http_config.setdefault("http_protocol_options", {})
            http_options['accept_http_10'] = self.config.ir.ambassador_module.enable_http10

        if 'allow_chunked_length' in self.config.ir.ambassador_module:
            if self.config.ir.ambassador_module.allow_chunked_length != None:
                http_options = base_http_config.setdefault("http_protocol_options", {})
                http_options['allow_chunked_length'] = self.config.ir.ambassador_module.allow_chunked_length

        if 'preserve_external_request_id' in self.config.ir.ambassador_module:
            base_http_config["preserve_external_request_id"] = self.config.ir.ambassador_module.preserve_external_request_id

        if 'forward_client_cert_details' in self.config.ir.ambassador_module:
            base_http_config["forward_client_cert_details"] = self.config.ir.ambassador_module.forward_client_cert_details

        if 'set_current_client_cert_details' in self.config.ir.ambassador_module:
            base_http_config["set_current_client_cert_details"] = self.config.ir.ambassador_module.set_current_client_cert_details

        if self.config.ir.tracing:
            base_http_config["generate_request_id"] = True

            base_http_config["tracing"] = {}
            self.traffic_direction = "OUTBOUND"

            req_hdrs = self.config.ir.tracing.get('tag_headers', [])

            if req_hdrs:
                base_http_config["tracing"]["custom_tags"] = []
                for hdr in req_hdrs:
                    custom_tag = {
                        "request_header": {
                            "name": hdr,
                            },
                        "tag": hdr,
                    }
                    base_http_config["tracing"]["custom_tags"].append(custom_tag)


            sampling = self.config.ir.tracing.get('sampling', {})
            if sampling:
                client_sampling = sampling.get('client', None)
                if client_sampling is not None:
                    base_http_config["tracing"]["client_sampling"] = {
                        "value": client_sampling
                    }

                random_sampling = sampling.get('random', None)
                if random_sampling is not None:
                    base_http_config["tracing"]["random_sampling"] = {
                        "value": random_sampling
                    }

                overall_sampling = sampling.get('overall', None)
                if overall_sampling is not None:
                    base_http_config["tracing"]["overall_sampling"] = {
                        "value": overall_sampling
                    }

        proper_case: bool = self.config.ir.ambassador_module['proper_case']

        # Get the list of downstream headers whose casing should be overriden
        # from the Ambassador module. We configure the upstream side of this
        # in v3cluster.py
        header_case_overrides = self.config.ir.ambassador_module.get('header_case_overrides', None)
        if header_case_overrides:
            if proper_case:
                self.config.ir.post_error(
                    "Only one of 'proper_case' or 'header_case_overrides' fields may be set on " +\
                    "the Ambassador module. Honoring proper_case and ignoring " +\
                    "header_case_overrides.")
                header_case_overrides = None
            if not isinstance(header_case_overrides, list):
                # The header_case_overrides field must be an array.
                self.config.ir.post_error("Ambassador module config 'header_case_overrides' must be an array")
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
                    self.config.ir.post_error("Skipping non-string header in 'header_case_overrides': {hdr}")
                    continue
                rules.append(hdr)

            if len(rules) == 0:
                self.config.ir.post_error(f"Could not parse any valid string headers in 'header_case_overrides': {header_case_overrides}")
            else:
                # Create custom header rules that map the lowercase version of every element in
                # `header_case_overrides` to the the respective original casing.
                #
                # For example the input array [ X-HELLO-There, X-COOL ] would create rules:
                # { 'x-hello-there': 'X-HELLO-There', 'x-cool': 'X-COOL' }. In envoy, this effectively
                # overrides the response header case by remapping the lowercased version (the default
                # casing in envoy) back to the casing provided in the config.
                custom_header_rules: Dict[str, Dict[str, dict]] = {
                    'custom': {
                        'rules': {
                            header.lower() : header for header in rules
                        }
                    }
                }
                http_options = base_http_config.setdefault("http_protocol_options", {})
                http_options["header_key_format"] = custom_header_rules

        if proper_case:
            proper_case_header: Dict[str, Dict[str, dict]] = {'header_key_format': {'proper_case_words': {}}}
            if 'http_protocol_options' in base_http_config:
                base_http_config["http_protocol_options"].update(proper_case_header)
            else:
                base_http_config["http_protocol_options"] = proper_case_header

        return base_http_config

    def finalize(self) -> None:
        if self._log_debug:
            self.config.ir.logger.debug(f"V3Listener: ==== finalize {self}")

        # OK. Assemble the high-level stuff for Envoy.
        self.address = {
            "socket_address": {
                "address": self.bind_address,
                "port_value": self.port,
                "protocol": "TCP"
            }
        }

        # Next, deal with HTTP stuff if this is an HTTP Listener.
        if self._base_http_config:
            self.compute_chains()
            self.compute_routes()
            self.finalize_http()
        else:
            # TCP is a lot simpler.
            self.finalize_tcp()

    def finalize_tcp(self) -> None:
        # Finalize a TCP listener, which amounts to walking all our TCP chains and
        # setting up Envoy configuration structures for them.
        logger = self.config.ir.logger

        for chain_key, chain in self._chains.items():
            if chain.type != "tcp":
                continue

            if self._log_debug:
                logger.debug("BUILD CHAIN %s - %s", chain_key, chain)

            for irgroup in chain.tcpmappings:
                # First up, which clusters do we need to talk to?
                clusters = [{
                    'name': mapping.cluster.envoy_name,
                    'weight': mapping.weight
                } for mapping in irgroup.mappings]

                # From that, we can sort out a basic tcp_proxy filter config.
                tcp_filter = {
                    'name': 'envoy.filters.network.tcp_proxy',
                    'typed_config': {
                        '@type': 'type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy',
                        'stat_prefix': self._stats_prefix,
                        'weighted_clusters': {
                            'clusters': clusters
                        }
                    }
                }

                # OK. Basic filter chain entry next.
                filter_chain: Dict[str, Any] = {
                    'filters': [
                        tcp_filter
                    ]
                }

                # The chain as a whole has a single matcher.
                filter_chain_match: Dict[str, Any] = {}

                chain_hosts = chain.hostglobs()

                # If we have a context...
                if chain.context:
                    # ...then we can ask for TLS.
                    filter_chain_match["transport_protocol"] = "tls"

                    # Note that we're modifying the filter_chain itself here, not
                    # filter_chain_match.
                    envoy_ctx = V3TLSContext(chain.context)

                    filter_chain['transport_socket'] = {
                        'name': 'envoy.transport_sockets.tls',
                        'typed_config': {
                            '@type': 'type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext',
                            **envoy_ctx
                        }
                    }

                # We do server-name matching whether or not we have TLS, just to help
                # make sure that we don't have two chains with an empty filter_match
                # criterion (since Envoy will reject such a configuration).

                if len(chain_hosts) > 0:
                    filter_chain_match['server_names'] = chain_hosts

                # Once all of that is done, hook in the match...
                filter_chain['filter_chain_match'] = filter_chain_match

                # ...and stick this chain into our filter.
                self._filter_chains.append(filter_chain)

    def compute_chains(self) -> None:
        # Compute the set of chains we need, HTTP version. The core here is matching
        # up Hosts with this Listener, and creating a chain for each Host.

        self.config.ir.logger.debug("V3Listener %s: checking hosts for %s", self.name, self)

        for host in sorted(self.config.ir.get_hosts(), key=lambda h: h.hostname):
            if self._log_debug:
                self.config.ir.logger.debug("  consider %s", host)

            # First up: drop this host if nothing matches at all.
            if not self._irlistener.matches_host(host):
                # Bzzzt.
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
                if self._log_debug:
                    self.config.ir.logger.debug("      drop %s, insecure-only port mismatch", host.name)

                continue

            # OK, we can't drop it for that, so we need to check the actions.

            security_model = self._security_model
            secure_action = host.secure_action
            insecure_action = host.insecure_action

            # If the Listener's securityModel is SECURE, but this host has a secure_action
            # of Reject (or empty), we'll skip this host, because the only requests this
            # Listener can ever produce will be rejected. In any other case, we'll set up an
            # HTTPS chain for this Host, as long as we think TLS is OK.

            will_reject_secure = ((not secure_action) or (secure_action == "Reject"))
            if self._tls_ok and (not ((security_model == "SECURE") and will_reject_secure)):
                if self._log_debug:
                    self.config.ir.logger.debug("      take SECURE %s", host)

                self.add_chain("https", host)

            # Same idea on the insecure side: only skip the Host if the Listener's securityModel
            # is INSECURE but the Host's insecure_action is Reject.

            if not ((security_model == "INSECURE") and (insecure_action == "Reject")):
                if self._log_debug:
                    self.config.ir.logger.debug("      take INSECURE %s", host)

                self.add_chain("http", host)

    def compute_routes(self) -> None:
        # Compute the set of valid HTTP routes for _each chain_ in this Listener.
        #
        # Note that a route using XFP can match _any_ chain, whether HTTP or HTTPS.

        logger = self.config.ir.logger

        for chain_key, chain in self._chains.items():
            # Only look at HTTP(S) chains.
            if (chain.type != "http") and (chain.type != "https"):
                continue

            if self._log_debug:
                logger.debug("MATCH CHAIN %s - %s", chain_key, chain)

            # Remember whether we found an ACME route.
            found_acme = False

            # The data structure we're walking here is config.route_variants rather than
            # config.routes. There's a one-to-one correspondence between the two, but we use the
            # V3RouteVariants to lazily cache some of the work that we're doing across chains.
            for rv in self.config.route_variants:
                if self._log_debug:
                    logger.debug("  CHECK ROUTE: %s", v3prettyroute(dict(rv.route)))

                matching_hosts = chain.matching_hosts(rv.route)

                if self._log_debug:
                    logger.debug("    = matching_hosts %s", ", ".join([ h.hostname for h in matching_hosts ]))

                if not matching_hosts:
                    if self._log_debug:
                        logger.debug(f"    drop outright: no hosts match {sorted(rv.route['_host_constraints'])}")
                    continue

                for host in matching_hosts:
                    # For each host, we need to look at things for the secure world as well
                    # as the insecure world, depending on what the action is exactly (and note
                    # that, yes, we can have an action of None for an insecure_only listener).
                    #
                    # "candidates" is host, matcher, action, V3RouteVariants
                    candidates: List[Tuple[IRHost, str, str, V3RouteVariants]] = []
                    hostname = host.hostname

                    if (host.secure_action is not None) and (self._security_model != "INSECURE"):
                        # We have a secure action, and we're willing to believe that at least some of
                        # our requests will be secure.
                        matcher = 'always' if (self._security_model == 'SECURE') else 'xfp-https'

                        candidates.append(( host, matcher, 'Route', rv ))

                    if (host.insecure_action is not None) and (self._security_model != "SECURE"):
                        # We have an insecure action, and we're willing to believe that at least some of
                        # our requests will be insecure.
                        matcher = 'always' if (self._security_model == 'INSECURE') else 'xfp-http'
                        action = host.insecure_action

                        candidates.append(( host, matcher, action, rv ))

                    for host, matcher, action, rv in candidates:
                        route_precedence = rv.route.get('_precedence', None)
                        extra_info = ""

                        if rv.route["match"].get("prefix", None) == "/.well-known/acme-challenge/":
                            # We need to be sure to route ACME challenges, no matter what else is going
                            # on (this is the infamous ACME hole-puncher mentioned everywhere).
                            extra_info = " (force Route for ACME challenge)"
                            action = "Route"
                            found_acme = True
                        elif (self.config.ir.edge_stack_allowed and
                                (route_precedence == -1000000) and
                                (rv.route["match"].get("safe_regex", {}).get("regex", None) == "^/$")):
                            extra_info = " (force Route for fallback Mapping)"
                            action = "Route"

                        if action != 'Reject':
                            # Worth noting here that "Route" really means "do what the V3Route really
                            # says", which might be a host redirect. When we talk about "Redirect", we
                            # really mean "redirect to HTTPS" specifically.

                            if self._log_debug:
                                logger.debug("      %s - %s: accept on %s %s%s",
                                             matcher, action, self.name, hostname, extra_info)

                            variant = dict(rv.get_variant(matcher, action.lower()))
                            variant["_host_constraints"] = set([ hostname ])
                            chain.add_route(variant)
                        else:
                            if self._log_debug:
                                logger.debug("      %s - %s: drop from %s %s%s",
                                             matcher, action, self.name, hostname, extra_info)

            # If we're on Edge Stack and we don't already have an ACME route, add one.
            if self.config.ir.edge_stack_allowed and not found_acme:
                # The target cluster doesn't actually matter -- the auth service grabs the
                # challenge and does the right thing. But we do need a cluster that actually
                # exists, so use the sidecar cluster.

                if not self.config.ir.sidecar_cluster_name:
                    # Uh whut? how is Edge Stack running exactly?
                    raise Exception("Edge Stack claims to be running, but we have no sidecar cluster??")

                if self._log_debug:
                    logger.debug("      punching a hole for ACME")

                # Make sure to include _host_constraints in here for now.
                #
                # XXX This is needed only because we're dictifying the V3Route too early.

                chain.routes.insert(0, {
                    "_host_constraints": set(),
                    "match": {
                        "case_sensitive": True,
                        "prefix": "/.well-known/acme-challenge/"
                    },
                    "route": {
                        "cluster": self.config.ir.sidecar_cluster_name,
                        "prefix_rewrite": "/.well-known/acme-challenge/",
                        "timeout": "3.000s"
                    }
                })

            if self._log_debug:
                for route in chain.routes:
                    logger.debug("  CHAIN ROUTE: %s" % v3prettyroute(route))

    def finalize_http(self) -> None:
        # Finalize everything HTTP. Like the TCP side of the world, this is about walking
        # chains and generating Envoy config.
        #
        # All of our HTTP chains get collapsed into a single chain with (likely) multiple
        # domains here.

        filter_chains: Dict[str, Dict[str, Any]] = {}

        for chain_key, chain in self._chains.items():
            if self._log_debug:
                self._irlistener.logger.debug("FHTTP %s / %s / %s", self, chain_key, chain)

            filter_chain: Optional[Dict[str, Any]] = None

            if chain.type == "http":
                # All HTTP chains get collapsed into one here, using domains to separate them.
                # This works because we don't need to offer TLS certs (we can't anyway), and
                # because of that, SNI (and thus filter server_names matches) aren't things.
                chain_key = "http"

                filter_chain = filter_chains.get(chain_key, None)

                if not filter_chain:
                    if self._log_debug:
                        self._irlistener.logger.debug("FHTTP   create filter_chain %s / empty match", chain_key)
                    filter_chain = {
                        "filter_chain_match": {},
                        "_vhosts": {}
                    }

                    filter_chains[chain_key] = filter_chain
                else:
                    if self._log_debug:
                        self._irlistener.logger.debug("FHTTP   use filter_chain %s: vhosts %d", chain_key, len(filter_chain["_vhosts"]))
            elif chain.type == "https":
                # Since chain_key is a dictionary key in its own right, we can't already
                # have a matching chain for this.

                filter_chain = {
                    "_vhosts": {}
                }
                filter_chain_match: Dict[str, Any] = {}

                chain_hosts = chain.hostglobs()

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
                # So, if "*" is present at all in our chain_hosts, we can't match server_names
                # at all.

                if (len(chain_hosts) > 0) and ("*" not in chain_hosts):
                    filter_chain_match['server_names'] = chain_hosts

                # Likewise, an HTTPS chain will ask for TLS.
                filter_chain_match["transport_protocol"] = "tls"

                if chain.context:
                    # ...uh. How could we not have a context if we're doing TLS?
                    # Note that we're modifying the filter_chain itself here, not
                    # filter_chain_match.
                    envoy_ctx = V3TLSContext(chain.context)

                    filter_chain['transport_socket'] = {
                        'name': 'envoy.transport_sockets.tls',
                        'typed_config': {
                            '@type': 'type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext',
                            **envoy_ctx
                        }
                    }

                # Finally, stash the match in the chain...
                filter_chain["filter_chain_match"] = filter_chain_match

                # ...and save it.
                filter_chains[chain_key] = filter_chain
            else:
                # The chain type is neither HTTP nor HTTPS -- must be a TCP chain. Skip it.
                continue

            # OK, we have the filter_chain variable set -- build the Envoy virtual_hosts for it.

            for host in chain.hosts.values():
                # Make certain that no internal keys from the route make it into the Envoy
                # configuration.
                routes = []

                for r in chain.routes:
                    routes.append({ k: v for k, v in r.items() if k[0] != '_' })

                # Do we - somehow - already have a vhost for this hostname? (This should
                # be "impossible".)

                vhost: Dict[str, Any] = filter_chain["_vhosts"].get(host.hostname, None)

                if not vhost:
                    vhost = {
                        "name": f"{self.name}-{host.hostname}",
                        "domains": [ host.hostname ],
                        "routes": []
                    }

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
            del(filter_chain["_vhosts"])

            # Finish up config for this filter chain...
            if parse_bool(self.config.ir.ambassador_module.get("strip_matching_host_port", "false")):
                http_config["strip_matching_host_port"] = True

            if parse_bool(self.config.ir.ambassador_module.get("merge_slashes", "false")):
                http_config["merge_slashes"] = True

            if parse_bool(self.config.ir.ambassador_module.get("reject_requests_with_escaped_slashes", "false")):
                http_config["path_with_escaped_slashes_action"] = "REJECT_REQUEST"

            filter_chain["filters"] = [
                {
                    "name": "envoy.filters.network.http_connection_manager",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
                        **http_config
                    }
                }
            ]

            # ...and save it.
            self._filter_chains.append(filter_chain)

    def as_dict(self) -> dict:
        listener = {
            "name": self.name,
            "address": self.address,
            "filter_chains": self._filter_chains,
            "traffic_direction": self.traffic_direction
        }
        # We only want to add the buffer limit setting to the listener if specified in the module.
        # Otherwise, we want to leave it unset and allow Envoys Default 1MiB setting.
        if 'buffer_limit_bytes' in self.config.ir.ambassador_module and self.config.ir.ambassador_module.buffer_limit_bytes != None:
            odict["per_connection_buffer_limit_bytes"] = self.config.ir.ambassador_module.buffer_limit_bytes

        if self.listener_filters:
            odict["listener_filters"] = self.listener_filters

        return odict

    def pretty(self) -> dict:
        return {
            "name": self.name,
            "bind_address": self.bind_address,
            "port": self.port,
            "chains": self._chains,
        }

    def __str__(self) -> str:
        return "<V3Listener %s %s on %s:%d [%s]>" % (
            "HTTP" if self._base_http_config else "TCP",
            self.name, self.bind_address, self.port, self._security_model
        )

    @classmethod
    def generate(cls, config: 'V3Config') -> None:
        config.listeners = []
        logger = config.ir.logger

        for key in config.ir.listeners.keys():
            irlistener = config.ir.listeners[key]
            v3listener = V3Listener(config, irlistener)
            v3listener.finalize()

            config.ir.logger.info(f"V3Listener: ==== GENERATED {v3listener}")

            if v3listener._log_debug:
                for k in sorted(v3listener._chains.keys()):
                    chain = v3listener._chains[k]
                    config.ir.logger.debug("    %s", chain)

                    for r in chain.routes:
                        config.ir.logger.debug("      %s", v3prettyroute(r))

            # Does this listener have any filter chains?
            if v3listener._filter_chains:
                config.listeners.append(v3listener)
            else:
                irlistener.post_error("No matching Hosts found, disabling!")
