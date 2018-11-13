from typing import TYPE_CHECKING

from .v2cluster import V2Cluster

if TYPE_CHECKING:
    from . import V2Config


class V2Bootstrap(dict):
    def __init__(self, config: 'V2Config') -> None:
        super().__init__(**{
            "node": {
                "cluster": config.ir.ambassador_nodename,
                "id": "test-id"     # MUST BE test-id, see below
            },
            "static_resources": {}, # Filled in later
            "dynamic_resources": {
                "ads_config": {
                    "api_type": "GRPC",
                    "grpc_services": [ {
                        "envoy_grpc": {
                            "cluster_name": "xds_cluster"
                        }
                    } ]
                },
                "cds_config": { "ads": {} },
                "lds_config": { "ads": {} }
            },
            "admin": dict(config.admin)
        })

        clusters = [{
            "name": "xds_cluster",
            "connect_timeout": "1s",
            "hosts": [ {
                "socket_address": {
                    "address": "127.0.0.1",
                    "port_value": 18000
                }
            } ],
            "http2_protocol_options": {}
        }]

        if config.tracing:
            self['tracing'] = dict(config.tracing)
            clusters.append(V2Cluster(config, self['tracing']['cluster']))

        if config.ratelimit:
            self['rate_limit_service'] = dict(config.ratelimit)
            clusters.append(V2Cluster(config, self['rate_limit_service']['cluster']))

        self['static_resources']['clusters'] = clusters

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.bootstrap = V2Bootstrap(config)
