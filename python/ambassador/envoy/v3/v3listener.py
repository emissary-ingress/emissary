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
from typing import Any, Dict, List, Optional, Tuple, Union, TYPE_CHECKING
from typing import cast as typecast

from os import environ

import logging

from ...ir.irlistener import IRListener
from ...ir.irtcpmappinggroup import IRTCPMappingGroup

from ...utils import dump_json, parse_bool

from .v3httpfilter import V3HTTPFilter
from .v3route import DictifiedV3Route, v3prettyroute
from .v3tls import V3TLSContext
from .v3virtualhost import V3VirtualHost

if TYPE_CHECKING:
    from ...ir.irtlscontext import IRTLSContext # pragma: no cover
    from . import V3Config                      # pragma: no cover


class V3Listener(dict):
    def __init__(self, config: 'V3Config', irlistener: IRListener) -> None:
        super().__init__()

        self.config = config
        self.bind_address = irlistener.bind_address
        self.port = irlistener.port

        bindstr = f"-{self.bind_address}" if (self.bind_address != "0.0.0.0") else ""
        self.name = f"ambassador-listener{bindstr}-{self.port}"

        self.use_proxy_proto = False
        self.listener_filters: List[dict] = []
        self.traffic_direction: str = "UNSPECIFIED"

        self._security_model: str = irlistener.securityModel
        self._l7_depth: int = irlistener.get('l7Depth', 0)
        self._base_http_config: Optional[Dict[str, Any]] = None

        # It's important from a performance perspective to wrap debug log statements
        # with this check so we don't end up generating log strings (or even JSON
        # representations) that won't get logged anyway.
        self._log_debug = self.config.ir.logger.isEnabledFor(logging.DEBUG)
        if self._log_debug:
            self.config.ir.logger.debug(f"V3Listener {self.name} created -- {self._security_model}, l7Depth {self._l7_depth}")

        # Start by building our base HTTP config...
        self._base_http_config = self.base_http_config()

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
                    'format': log_format + '\n'
                }
            })

        return access_log

    # base_http_config constructs the starting configuration for this
    # V3Listener's http_connection_manager filter.
    def base_http_config(self) -> Dict[str, Any]:
        base_http_config: Dict[str, Any] = {
            'stat_prefix': 'ingress_http',
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
            self.config.ir.logger.debug(f"V3Listener finalize {self.pretty()}")

        # OK. Assemble the high-level stuff for Envoy.
        self.address = {
            "socket_address": {
                "address": self.bind_address,
                "port_value": self.port,
                "protocol": "TCP"
            }
        }

        self.filter_chains: List[dict] = []

    def as_dict(self) -> dict:
        return {
            "name": self.name,
            "address": self.address,
            "filter_chains": self.filter_chains,
            "listener_filters": self.listener_filters,
            "traffic_direction": self.traffic_direction
        }

    def pretty(self) -> dict:
        return {
            "name": self.name,
            "bind_address": self.bind_address,
            "port": self.port,
            # "use_proxy_proto": self.use_proxy_proto
        }

    def __str__(self) -> str:
        return "<V3Listener %s %s on %s:%d [%s]>" % (
            "HTTP" if self._base_http_config else "TCP",
            self.name, self.bind_address, self.port, self._security_model
        )

    @classmethod
    def dump_listeners(cls, logger, listeners_by_port) -> None:
        pretty = { k: v.pretty() for k, v in listeners_by_port.items() }

        logger.debug(f"V3Listeners: {dump_json(pretty, pretty=True)}")

    @classmethod
    def generate(cls, config: 'V3Config') -> None:
        config.listeners = []
        logger = config.ir.logger

        for key in sorted(config.ir.listeners.keys()):
            irlistener = config.ir.listeners[key]
            v3listener = V3Listener(config, irlistener)
            v3listener.finalize()

            config.listeners.append(v3listener)

            if v3listener._log_debug:
                config.ir.logger.debug(f"V3Listener generated: {v3listener.pretty()}")
