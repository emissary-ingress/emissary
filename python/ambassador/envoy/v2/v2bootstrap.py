from typing import TYPE_CHECKING
from typing import cast as typecast

from ...ir.ircluster import IRCluster
from ...ir.irlogservice import IRLogService
from ...ir.irratelimit import IRRateLimit
from ...ir.irtracing import IRTracing

from .v2cluster import V2Cluster

if TYPE_CHECKING:
    from . import V2Config


class V2Bootstrap(dict):
    def __init__(self, config: 'V2Config') -> None:
        super().__init__(**{
            "node": {
                "cluster": config.ir.ambassador_nodename,
                "id": "test-id"         # MUST BE test-id, see below
            },
            "static_resources": {},     # Filled in later
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
            "admin": dict(config.admin),
            'layered_runtime': {
                'layers': [
                    {
                        'name': 'static_layer',
                        'static_layer': {
                            # For now, we enable the deprecated & disallowed_by_default "HTTP_JSON_V1" Zipkin
                            # collector_endpoint_version because it repesents the Zipkin v1 API, while the
                            # non-deprecated options HTTP_JSON and HTTP_PROTO are the Zipkin v2 API; switching
                            # top one of them would change how Envoy talks to the outside world.
                            'envoy.deprecated_features:envoy.config.trace.v2.ZipkinConfig.HTTP_JSON_V1': True,
                            # Give our users more time to migrate to v2; we've said that we'll continue
                            # supporting both for a while even after we change the default.
                            'envoy.deprecated_features:envoy.config.filter.http.ext_authz.v2.ExtAuthz.use_alpha': True,
                            # We haven't yet told users that we'll be deprecating `regex_type: unsafe`.
                            'envoy.deprecated_features:envoy.api.v2.route.RouteMatch.regex': True,         # HTTP path
                            'envoy.deprecated_features:envoy.api.v2.route.HeaderMatcher.regex_match': True, # HTTP header,
                            # Envoy 1.14.1 disabled the use of lowercase string matcher for headers matching in HTTP-based.
                            # Following setting toggled it to be consistent with old behavior.
                            # AuthenticationTest (v0) is a good example that expects the old behavior. 
                            'envoy.reloadable_features.ext_authz_http_service_enable_case_sensitive_string_matcher': False
                        }
                    }
                ]
            }
        })

        clusters = [{
            "name": "xds_cluster",
            "connect_timeout": "1s",
            "dns_lookup_family": "V4_ONLY",
            "http2_protocol_options": {},
            "lb_policy": "ROUND_ROBIN",
            "load_assignment": {
                "cluster_name": "cluster_127_0_0_1_8003",
                "endpoints": [
                    {
                        "lb_endpoints": [
                            {
                                "endpoint": {
                                    "address": {
                                        "socket_address": {
                                            "address": "127.0.0.1",
                                            "port_value": 8003,
                                            "protocol": "TCP"
                                        }
                                    }
                                }
                            }
                        ]
                    }
                ]
            }
        }]

        if config.tracing:
            self['tracing'] = dict(config.tracing)

            tracing = typecast(IRTracing, config.ir.tracing)

            assert tracing.cluster
            clusters.append(V2Cluster(config, typecast(IRCluster, tracing.cluster)))

        if config.ir.log_services.values():
            for als in config.ir.log_services.values():
                log_service = typecast(IRLogService, als)
                assert log_service.cluster
                clusters.append(V2Cluster(config, typecast(IRCluster, log_service.cluster)))

        # if config.ratelimit:
        #     self['rate_limit_service'] = dict(config.ratelimit)
        #
        #     ratelimit = typecast(IRRateLimit, config.ir.ratelimit)
        #
        #     assert ratelimit.cluster
        #     clusters.append(V2Cluster(config, ratelimit.cluster))

        if config.ir.statsd['enabled']:
            name = 'envoy.dog_statsd' if config.ir.statsd['dogstatsd'] else 'envoy.statsd'
            self['stats_sinks'] = [
                {
                    'name': name,
                    'config': {
                        'address': {
                            'socket_address': {
                                'protocol': 'UDP',
                                'address': config.ir.statsd['ip'],
                                'port_value': 8125
                            }
                        }
                    }
                }
            ]

            self['stats_flush_interval'] = {
                'seconds': config.ir.statsd['interval']
            }

        self['static_resources']['clusters'] = clusters

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.bootstrap = V2Bootstrap(config)
