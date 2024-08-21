# Copyright 2021 Datawire. All rights reserved.
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

import os
from typing import TYPE_CHECKING, List

from ...utils import parse_bool

if TYPE_CHECKING:
    from . import V3Config  # pragma: no cover

# The defaults can be changed by using those 2 env vars:
# AMBASSADOR_READY_PORT: Port number (default 8006)
# AMBASSADOR_READY_LOG: true/false (default false)
ambassador_ready_port = int(os.getenv("AMBASSADOR_READY_PORT", "8006"))
if ambassador_ready_port not in range(1, 32767):
    ambassador_ready_port = 8006
ambassador_ready_log = parse_bool(os.getenv("AMBASSADOR_READY_LOG", "false"))
ambassador_ready_ip = (
    "::1" if os.getenv("AMBASSADOR_HEALTHCHECK_IP_FAMILY", "ANY") == "IPV6_ONLY" else "127.0.0.1"
)


class V3Ready(dict):
    @classmethod
    def generate(cls, config: "V3Config") -> None:
        # Inject the ready listener to the list of listeners
        config.ir.logger.info(
            "V3Ready: ==== listen on %s:%s" % (ambassador_ready_ip, ambassador_ready_port)
        )

        typed_config = {
            "@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
            "stat_prefix": "ready_http",
            "route_config": {"name": "local_route"},
            "http_filters": [
                {
                    "name": "envoy.filters.http.health_check",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.filters.http.health_check.v3.HealthCheck",
                        "pass_through_mode": False,
                        "headers": [{"name": ":path", "exact_match": "/ready"}],
                    },
                },
                {"name": "envoy.filters.http.router"},
            ],
        }
        if ambassador_ready_log:
            typed_config["access_log"] = cls.access_log(config)

        # required for test_max_request_header.py
        max_request_headers_kb = config.ir.ambassador_module.get("max_request_headers_kb", None)
        if max_request_headers_kb:
            typed_config["max_request_headers_kb"] = max_request_headers_kb

        listener_idle_timeout_ms = config.ir.ambassador_module.get("listener_idle_timeout_ms", None)
        if listener_idle_timeout_ms:
            typed_config["common_http_protocol_options"] = {
                "idle_timeout": "%0.3fs" % (float(listener_idle_timeout_ms) / 1000.0)
            }

        ready_listener = {
            "name": "ambassador-listener-ready-%s-%s"
            % (ambassador_ready_ip, ambassador_ready_port),
            "address": {
                "socket_address": {
                    "address": ambassador_ready_ip,
                    "port_value": ambassador_ready_port,
                    "protocol": "TCP",
                }
            },
            "filter_chains": [
                {
                    "filter_chain_match": {},
                    "filters": [
                        {
                            "name": "envoy.filters.network.http_connection_manager",
                            "typed_config": typed_config,
                        }
                    ],
                }
            ],
        }

        # required for test_buffer_limit_bytes.py
        per_connection_buffer_limit_bytes = config.ir.ambassador_module.get(
            "buffer_limit_bytes", None
        )
        if per_connection_buffer_limit_bytes:
            ready_listener["per_connection_buffer_limit_bytes"] = per_connection_buffer_limit_bytes

        config.static_resources["listeners"].append(ready_listener)

    # access_log constructs the access_log configuration for this V3Listener
    @classmethod
    def access_log(cls, config: "V3Config") -> List[dict]:
        access_log: List[dict] = []

        # Use sane access log spec in JSON
        if config.ir.ambassador_module.envoy_log_type.lower() == "json":
            log_format = config.ir.ambassador_module.get("envoy_log_format", None)
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

                tracing_config = config.ir.tracing
                if tracing_config and tracing_config.driver == "envoy.tracers.datadog":
                    log_format["dd.trace_id"] = "%REQ(X-DATADOG-TRACE-ID)%"
                    log_format["dd.span_id"] = "%REQ(X-DATADOG-PARENT-ID)%"
            access_log.append(
                {
                    "name": "envoy.access_loggers.file",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
                        "path": config.ir.ambassador_module.envoy_log_path,
                        "json_format": log_format,
                    },
                }
            )
        # Use sane access log spec in Typed JSON
        elif config.ir.ambassador_module.envoy_log_type.lower() == "typed_json":
            log_format = config.ir.ambassador_module.get("envoy_log_format", None)
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

                tracing_config = config.ir.tracing
                if tracing_config and tracing_config.driver == "envoy.tracers.datadog":
                    log_format["dd.trace_id"] = "%REQ(X-DATADOG-TRACE-ID)%"
                    log_format["dd.span_id"] = "%REQ(X-DATADOG-PARENT-ID)%"

            access_log.append(
                {
                    "name": "envoy.access_loggers.file",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
                        "path": config.ir.ambassador_module.envoy_log_path,
                        "typed_json_format": log_format,
                    },
                }
            )
        else:
            # Use a sane access log spec
            log_format = config.ir.ambassador_module.get("envoy_log_format", None)

            if not log_format:
                log_format = 'ACCESS [%START_TIME%] "%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% "%REQ(X-FORWARDED-FOR)%" "%REQ(USER-AGENT)%" "%REQ(X-REQUEST-ID)%" "%REQ(:AUTHORITY)%" "%UPSTREAM_HOST%"'

            access_log.append(
                {
                    "name": "envoy.access_loggers.file",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog",
                        "path": config.ir.ambassador_module.envoy_log_path,
                        "log_format": {"text_format_source": {"inline_string": log_format + "\n"}},
                    },
                }
            )

        return access_log
