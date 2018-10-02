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

from typing import List, TYPE_CHECKING
from typing import cast as typecast

import json

from ..common import EnvoyRoute
from ...ir.irlistener import IRListener
# from ...ir.irmapping import IRMapping
from ...ir.irfilter import IRFilter

# from .v2tls import V2TLSContext
# from .v2ratelimit import V2RateLimits

if TYPE_CHECKING:
    from . import V2Config


# XXX This is probably going to go away!
class V2Filter(dict):
    def __init__(self, filter: IRFilter) -> None:
        super().__init__()

        self['name'] = filter.name
        self['config'] = filter.config_dict()

        if filter.get('type', None):
           self['type'] = filter.type


class V2Listener(dict):
    def __init__(self, config: 'V2Config', listener: IRListener) -> None:
        super().__init__()

        # TODO: tls contexts
        # if 'tls_contexts' in listener:
        #     ssl_context = {}
        #     found_some = False
        #     for ctx_name, ctx in listener.tls_contexts.items():
        #         for key in ["cert_chain_file", "private_key_file",
        #                     "alpn_protocols", "cacert_chain_file"]:
        #             if key in ctx:
        #                 ssl_context[key] = ctx[key]
        #                 found_some = True
        #
        #         if "cert_required" in ctx:
        #             ssl_context["require_client_certificate"] = ctx["cert_required"]
        #             found_some = True
        #
        #     if found_some:
        #         self['ssl_context'] = ssl_context

        # TODO: rate limit
        # if "rate_limits" in group:
        #     route["rate_limits"] = group.rate_limits

        self.update({
            'name': listener.name,
            'address': {
                'socket_address': {
                    'address': '0.0.0.0',
                    'port_value': listener.service_port,
                    'protocol': 'TCP'
                }
            },
            'filter_chains': [
                {
                    'filters': [
                        {
                            'name': 'envoy.http_connection_manager',
                            'config': {
                                'stat_prefix': 'ingress_http',
                                'access_log': [
                                    {
                                        'name': 'envoy.file_access_log',
                                        'config': {
                                            'path': '/dev/fd/1',
                                            'format': 'ACCESS [%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\"\n'
                                        }
                                    }
                                ],
                                'http_filters': [
                                    {
                                        'name': 'envoy.router'
                                    }
                                ],
                                'route_config': {
                                    'virtual_hosts': [
                                        {
                                            'name': 'backend',
                                            'domains': [
                                                '*'
                                            ],
                                            'routes': config.routes
                                        }
                                    ]
                                }
                            }
                        }
                    ],
                    'use_proxy_proto': listener.get('use_proxy_proto')
                }
            ]
        })

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.listeners = []

        for irlistener in config.ir.listeners:
            listener = config.save_element('listener', irlistener, V2Listener(config, irlistener))
            config.listeners.append(listener)
