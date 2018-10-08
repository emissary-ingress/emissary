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

import urllib
from typing import List, TYPE_CHECKING

from ...ir.ircluster import IRCluster

from .v2tls import V2TLSContext

if TYPE_CHECKING:
    from . import V2Config


class V2Cluster(dict):
    def __init__(self, config: 'V2Config', cluster: IRCluster) -> None:
        super().__init__()

        fields = {
            'name': cluster.name,
            'type': cluster.type.upper(),
            'lb_policy': cluster.lb_type.upper(),
            'connect_timeout': "3s",
            'load_assignment': {
                'cluster_name': cluster.name,
                'endpoints': [
                    {
                        'lb_endpoints': self.get_endpoints(cluster)
                    }
                ]
            }
        }

        if 'tls_context' in cluster:
            fields['tls_context'] = {
                'common_tls_context': {}
            }

        self.update(fields)
        return

        self["name"] = cluster.name
        self["connect_timeout_ms"] = cluster.get("timeout_ms", 3000)
        self["type"] = cluster.get("dns_type", "strict_dns")
        self["lb_type"] = cluster.get("lb_type", "round_robin")

        self["hosts"] = [ { "url": url } for url in cluster.urls ]

        if cluster.get('features', []):
            self["features"] = cluster.features

        if cluster.get('breakers', {}):
            pass
            # brk = cluster.breakers
            #
            # self["circuit_breakers"] = {
            #     "default": {
            #         "max_connections": {{ brk.max_connections or 1024 }},
            #         "max_pending_requests": {{ brk.max_pending or 1024 }},
            #         "max_requests": {{ brk.max_requests or 1024 }},
            #         "max_retries": {{ brk.max_retries or 3 }}
            #     }
            # }

        if cluster.get('outlier', {}):
            pass
            # outlier = cluster.outlier
            #
            # self["outlier_detection"] = {
            #     "consecutive_5xx": outlier.consecutive_5xx or 5
            #     "max_ejection_percent": outlier.max_ejection or 100
            #     "interval_ms": outlier.interval_ms or 3000
            # }

        if 'tls_context' in cluster:
            ctx = cluster.tls_context
            host_rewrite = cluster.get('host_rewrite', None)

            envoy_ctx = V2TLSContext(ctx=ctx, host_rewrite=host_rewrite)

            if envoy_ctx:
                self['ssl_context'] = dict(envoy_ctx)

    def get_endpoints(self, cluster: IRCluster):
        result = []
        for u in cluster.urls:
            p = urllib.parse.urlparse(u)
            address = {
                'address': p.hostname,
                'port_value': int(p.port)
            }
            if p.scheme:
                address['protocol'] = p.scheme.upper()
            result.append({'endpoint': {'address': {'socket_address': address}}})
        return result

    @classmethod
    def generate(self, config: 'V2Config') -> None:
        config.clusters = []

        for ircluster in sorted(config.ir.clusters.values(), key=lambda x: x.name):
            cluster = config.save_element('cluster', ircluster, V2Cluster(config, ircluster))
            config.clusters.append(cluster)

