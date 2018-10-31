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

from ...ir.ircluster import IRCluster

from .v1tls import V1TLSContext

if TYPE_CHECKING:
    from . import V1Config


class V1Cluster(dict):
    def __init__(self, config: 'V1Config', cluster: IRCluster) -> None:
        super().__init__()

        self["name"] = cluster.name
        self["connect_timeout_ms"] = cluster.get("timeout_ms", 3000)
        self["type"] = cluster.get("dns_type", "strict_dns")
        self["lb_type"] = cluster.get("lb_type", "round_robin")

        self["hosts"] = [ { "url": url } for url in cluster.urls ]

        if cluster.get('grpc', False):
            self["features"] = "http2"

        if 'tls_context' in cluster:
            ctx = cluster.tls_context
            host_rewrite = cluster.get('host_rewrite', None)

            envoy_ctx = V1TLSContext(ctx=ctx, host_rewrite=host_rewrite)
            self['ssl_context'] = dict(envoy_ctx)

    @classmethod
    def generate(self, config: 'V1Config') -> None:
        config.clusters = []

        for ircluster in sorted(config.ir.clusters.values(), key=lambda x: x.name):
            cluster = config.save_element('cluster', ircluster, V1Cluster(config, ircluster))
            config.clusters.append(cluster)

