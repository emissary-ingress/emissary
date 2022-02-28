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

from typing import List, TYPE_CHECKING

from .v3listener import V3Listener

if TYPE_CHECKING:
    from . import V3Config # pragma: no cover


class V3Ready(dict):

    @classmethod
    def generate(cls, config: 'V3Config') -> None:
        # Inject the ready listener to the list of listeners if enabled
        rport = config.ir.aconf.module_lookup('ambassador', 'ready_port', -1)
        if rport <= 0:
            config.ir.logger.info(f"V3Ready: ==== disabled")
            return

        rip = config.ir.aconf.module_lookup('ambassador', 'ready_ip', '127.0.0.1')
        rlog = config.ir.aconf.module_lookup('ambassador', 'ready_log', True)

        config.ir.logger.info(f"V3Ready: ==== listen on %s:%s" % (rip, rport))

        typed_config = {
            '@type': 'type.googleapis.com/envoy.extensions.filters.http.health_check.v3.HealthCheck',
            'pass_through_mode': False,
            'headers': [
                {
                    'name': ':path',
                    'exact_match': '/ready'
                }
            ]
        }
        if rlog:
            typed_config['access_log'] = cls.access_log(config)

        ready_listener = {
            'name': 'ambassador-listener-ready-%s-%s' % (rip, rport),
            'address': {
                'socket_address': {
                    'address': rip,
                    'port_value': rport,
                    'protocol': 'TCP'
                }
            },
            'filter_chains': [
                {
                    'filters': [
                        {
                            'name': 'envoy.filters.network.http_connection_manager',
                            'typed_config': {
                                '@type': 'type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager',
                                'stat_prefix': 'ready_http',
                                'route_config': {
                                    'name': 'local_route'
                                },
                                'http_filters': [
                                    {
                                        'name': 'envoy.filters.http.health_check',
                                        'typed_config': typed_config
                                    },
                                    {
                                        'name': 'envoy.filters.http.router'
                                    }
                                ]
                            }
                        }
                    ]
                }
            ]
        }

        config.static_resources['listeners'].append(ready_listener)

    # access_log constructs the access_log configuration for this V3Listener
    @classmethod
    def access_log(cls, config: 'V3Config') -> List[dict]:
        access_log: List[dict] = []

        # Use sane access log spec in JSON
        if config.ir.ambassador_module.envoy_log_type.lower() == "json":
            log_format = config.ir.ambassador_module.get('envoy_log_format', None)
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

                tracing_config = config.ir.tracing
                if tracing_config and tracing_config.driver == 'envoy.tracers.datadog':
                    log_format['dd.trace_id'] = '%REQ(X-DATADOG-TRACE-ID)%'
                    log_format['dd.span_id'] = '%REQ(X-DATADOG-PARENT-ID)%'

            access_log.append({
                'name': 'envoy.access_loggers.file',
                'typed_config': {
                    '@type': 'type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog',
                    'path': config.ir.ambassador_module.envoy_log_path,
                    'json_format': log_format
                }
            })
        else:
            # Use a sane access log spec
            log_format = config.ir.ambassador_module.get('envoy_log_format', None)

            if not log_format:
                log_format = 'ACCESS [%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\"'

            access_log.append({
                'name': 'envoy.access_loggers.file',
                'typed_config': {
                    '@type': 'type.googleapis.com/envoy.extensions.access_loggers.file.v3.FileAccessLog',
                    'path': config.ir.ambassador_module.envoy_log_path,
                    'log_format': {
                        'text_format_source': {
                            'inline_string': log_format + '\n'
                        }
                    }
                }
            })

        return access_log
