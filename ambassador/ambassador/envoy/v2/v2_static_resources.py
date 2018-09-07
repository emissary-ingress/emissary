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

from typing import TYPE_CHECKING, List

from ..common import EnvoyRoute

from ...ir.irlistener import IRListener

if TYPE_CHECKING:
    from . import V2Config


class V2StaticResources(dict):
    def __init__(self, config: 'V2Config') -> None:
        super().__init__()

        listeners:  List[V2Listener] = []

        for listener in config.ir.listeners:
            listeners.append(V2Listener(config, listener))

        self.update({
            'listeners': listeners
        })

    @classmethod
    def generate(cls, config: 'V2Config') -> 'V2StaticResources':
        return V2StaticResources(config)

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

        virtual_hosts = [
            {
                # 'name': ??
            }
        ]

        filters = [
            {
                'name': 'wha',
                "config": {
                  "stat_prefix": "ingress_http",
                  "route_config": {
                    "name": "local_route",
                    "virtual_hosts": [
                      {
                        "name": "local_service",
                        "domains": [
                          "*"
                        ],
                          "routes": self.parse_routes(config)
                      },
                    ],
                  },
                },
            },
        ]

        filter_chains = [
            {
                'filters': filters,
                'use_proxy_proto': listener.get('use_proxy_proto')
            }
        ]

        self.update({
            'address': {
                'socket_address': {
                    'address': '0.0.0.0',
                    'port_value': listener.service_port,
                    'protocol': 'TCP'
                }
            },
            'filter_chains': filter_chains
        })

    def parse_routes(self, config: 'V2Config'):
        routes = []

        for group in reversed(sorted(config.ir.groups.values(), key=lambda x: x['group_weight'])):
            envoy_route = EnvoyRoute(group).envoy_route

            route = {
                'match': {
                    envoy_route: group.get('prefix'),
                    'case_sensitive': group.get('case_sensitive'),
                    'headers': group.get('headers')
                },
                'route': {
                    'cors': group.get('cors') if group.get('cors', False) else group.get('cors_default'),
                    'priority': group.get('priority'),
                    'use_websocket': group.get('use_websocket'),
                    'weighted_clusters': {
                        'clusters': [
                            {
                                'name': mapping.cluster.name,
                                'weight': mapping.weight,
                                'request_headers_to_add': group.get('request_headers_to_add')
                            } for mapping in group.mappings
                        ],
                    },
                },
                'redirect': {
                    'host_redirect': group.get('host_redirect'),
                    'path_redirect': group.get('path_redirect'),
                    'prefix_rewrite': group.get('rewrite'),
                    'host_rewrite': group.get('host_rewrite'),
                    'auto_host_rewrite': group.get('auto_host_rewrite'),
                    }
                }

            # if 'shadows' in group:
            #     route['route'].update({
            #         'cluster': group.get('shadow')[0].get('name')
            #     })

            routes.append(route)

        return routes
