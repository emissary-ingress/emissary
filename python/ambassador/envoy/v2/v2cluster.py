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
from typing import Dict, List, Union, TYPE_CHECKING

from ...ir.ircluster import IRCluster

from .v2tls import V2TLSContext

if TYPE_CHECKING:
    from . import V2Config


class V2Cluster(dict):
    def __init__(self, config: 'V2Config', cluster: IRCluster) -> None:
        super().__init__()

        dns_lookup_family = 'V4_ONLY'

        if cluster.enable_ipv6:
            if cluster.enable_ipv4:
                dns_lookup_family = 'AUTO'
            else:
                dns_lookup_family = 'V6_ONLY'

        fields = {
            'name': cluster.name,
            'type': cluster.type.upper(),
            'lb_policy': cluster.lb_type.upper(),
            'connect_timeout':"%0.3fs" % (float(cluster.connect_timeout_ms) / 1000.0),
            'load_assignment': {
                'cluster_name': cluster.name,
                'endpoints': [
                    {
                        'lb_endpoints': self.get_endpoints(cluster)
                    }
                ]
            },
            'dns_lookup_family': dns_lookup_family
        }

        if cluster.cluster_idle_timeout_ms:
            cluster_idle_timeout_ms = cluster.cluster_idle_timeout_ms
        else:
            cluster_idle_timeout_ms = cluster.ir.ambassador_module.get('cluster_idle_timeout_ms', None)
        if cluster_idle_timeout_ms:
            fields['common_http_protocol_options'] = {
                'idle_timeout': "%0.3fs" % (float(cluster_idle_timeout_ms) / 1000.0)
            }

        circuit_breakers = self.get_circuit_breakers(cluster)
        if circuit_breakers is not None:
            fields['circuit_breakers'] = circuit_breakers

        if cluster.get('grpc', False):
            self["http2_protocol_options"] = {}

        ctx = cluster.get('tls_context', None)

        if ctx is not None:
            # If this is a null TLS Context (_ambassador_enabled is True), then we at need to specify a
            # minimal `tls_context` to enable HTTPS origination.

            if ctx.get('_ambassador_enabled', False):
                envoy_ctx = {
                    'common_tls_context': {}
                }
            else:
                envoy_ctx = V2TLSContext(ctx=ctx, host_rewrite=cluster.get('host_rewrite', None))

            if envoy_ctx:
                fields['transport_socket'] = {
                    'name': 'envoy.transport_sockets.tls',
                    'typed_config': {
                        '@type': 'type.googleapis.com/envoy.api.v2.auth.UpstreamTlsContext',
                        **envoy_ctx
                    }
                }


        keepalive = cluster.get('keepalive', None)
        #in case of empty keepalive for service, we can try to fallback to default
        if keepalive is None:
            if cluster.ir.ambassador_module and cluster.ir.ambassador_module.get('keepalive', None):
                keepalive = cluster.ir.ambassador_module['keepalive']

        if keepalive is not None:
            keepalive_options = {}    
            keepalive_time = keepalive.get('time', None) 
            keepalive_interval = keepalive.get('interval', None) 
            keepalive_probes = keepalive.get('probes', None) 
            
            if keepalive_time is not None:
                keepalive_options['keepalive_time'] = keepalive_time
            if keepalive_interval is not None:
                keepalive_options['keepalive_interval'] = keepalive_interval
            if keepalive_probes is not None:
                keepalive_options['keepalive_probes'] = keepalive_probes
                
            fields['upstream_connection_options'] = {'tcp_keepalive' : keepalive_options }

        self.update(fields)

    def get_endpoints(self, cluster: IRCluster):
        result = []

        targetlist = cluster.get('targets', [])

        if len(targetlist) > 0:
            for target in targetlist:
                address = {
                    'address': target['ip'],
                    'port_value': target['port'],
                    'protocol': 'TCP'  # Yes, really. Envoy uses the TLS context to determine whether to originate TLS.
                }
                result.append({'endpoint': {'address': {'socket_address': address}}})
        else:
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

    def get_circuit_breakers(self, cluster: IRCluster):
        cluster_circuit_breakers = cluster.get('circuit_breakers', None)
        if cluster_circuit_breakers is None:
            return None

        circuit_breakers: Dict[str, List[Dict[str, Union[str, int]]]] = {
            'thresholds': []
        }

        for circuit_breaker in cluster_circuit_breakers:
            threshold = {}
            if 'priority' in circuit_breaker:
                threshold['priority'] = circuit_breaker.get('priority').upper()
            else:
                threshold['priority'] = 'DEFAULT'

            digit_fields = ['max_connections', 'max_pending_requests', 'max_requests', 'max_retries']
            for field in digit_fields:
                if field in circuit_breaker:
                    threshold[field] = int(circuit_breaker.get(field))

            if len(threshold) > 0:
                circuit_breakers['thresholds'].append(threshold)

        return circuit_breakers

    @classmethod
    def generate(self, config: 'V2Config') -> None:
        config.clusters = []

        for ircluster in sorted(config.ir.clusters.values(), key=lambda x: x.name):
            cluster = config.save_element('cluster', ircluster, V2Cluster(config, ircluster))
            config.clusters.append(cluster)
