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

from typing import TYPE_CHECKING

from .v2listener import V2Listener

if TYPE_CHECKING:
    from . import V2Config # pragma: no cover


class V2Ready(dict):

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        # Inject the ready listener to the list of listeners if enabled
        rport = config.ir.aconf.module_lookup('ambassador', 'ready_port', -1)
        if rport <= 0:
            return

        ready_listener = {
            'name': 'ambassador-listener-ready-%s' % rport,
            'address': {
                'socket_address': {
                    'address': '0.0.0.0', # Todo: Change this to 127.0.0.1 or make it a parameter
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
                                '@type': 'type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager',
                                "access_log": [
                                    {
                                        "name": "envoy.access_loggers.file",
                                        "typed_config": {
                                            "@type": "type.googleapis.com/envoy.config.accesslog.v2.FileAccessLog",
                                            "format": "ACCESS [%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\"\n",
                                            "path": "/dev/fd/1"
                                        }
                                    }
                                ],
                                'stat_prefix': 'ready_http',
                                'route_config': {
                                    'name': 'local_route'
                                },
                                'http_filters': [
                                    {
                                        'name': 'envoy.filters.http.health_check',
                                        'typed_config': {
                                            '@type': 'type.googleapis.com/envoy.config.filter.http.health_check.v2.HealthCheck',
                                            'pass_through_mode': False,
                                            'headers': [
                                                {
                                                    'name': ':path',
                                                    'exact_match': '/ready'
                                                }
                                            ]
                                        }
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
