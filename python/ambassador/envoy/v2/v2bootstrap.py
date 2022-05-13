from typing import TYPE_CHECKING
from typing import cast as typecast
from typing import Tuple, Optional

import os
from urllib.parse import urlparse

from ...ir.ircluster import IRCluster
from ...ir.irlogservice import IRLogService
from ...ir.irratelimit import IRRateLimit
from ...ir.irtracing import IRTracing

from .v2cluster import V2Cluster

if TYPE_CHECKING:
    from . import V2Config # pragma: no cover


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
                            'envoy.deprecated_features.allow_deprecated_gzip_http_filter': True,
                            # We haven't yet told users that we'll be deprecating `regex_type: unsafe`.
                            'envoy.deprecated_features:envoy.api.v2.route.RouteMatch.regex': True,         # HTTP path
                            'envoy.deprecated_features:envoy.api.v2.route.HeaderMatcher.regex_match': True, # HTTP header,
                            'envoy.reloadable_features.enable_deprecated_v2_api': True,
                            # Envoy 1.14.1 disabled the use of lowercase string matcher for headers matching in HTTP-based.
                            # Following setting toggled it to be consistent with old behavior.
                            # AuthenticationTest (v0) is a good example that expects the old behavior.
                            'envoy.reloadable_features.ext_authz_http_service_enable_case_sensitive_string_matcher': False,
                            # enable all deprecated features for v2 since we're also EOL-ing it in
                            # edigissary
                            'envoy.features.enable_all_deprecated_feature': True
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
                "cluster_name": "cluster_xds_cluster",
                "endpoints": [
                    {
                        "lb_endpoints": [
                            {
                                "endpoint": {
                                    "address": {
                                        "pipe": {
                                            "path": "/tmp/ambex.sock"
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

        stats_sinks = []

        if config.ir.statsd['enabled']:
            if config.ir.statsd['dogstatsd']:
                name = 'envoy.stat_sinks.dog_statsd'
                typename = 'type.googleapis.com/envoy.config.metrics.v2.DogStatsdSink'
                dd_entity_id = os.environ.get('DD_ENTITY_ID', None)
                if dd_entity_id:
                    stats_tags = self.setdefault('stats_config', {}).setdefault('stats_tags', [])
                    stats_tags.append({
                        'tag_name': 'dd.internal.entity_id',
                        'fixed_value': dd_entity_id
                    })
            else:
                name = 'envoy.stats_sinks.statsd'
                typename = 'type.googleapis.com/envoy.config.metrics.v2.StatsdSink'

            stats_sinks.append({
                'name': name,
                'typed_config': {
                    '@type': typename,
                    'address': {
                        'socket_address': {
                            'protocol': 'UDP',
                            'address': config.ir.statsd['ip'],
                            'port_value': 8125
                        }
                    }
                }
            })

            self['stats_flush_interval'] = {
                'seconds': config.ir.statsd['interval']
            }

        grpcSink = os.environ.get("AMBASSADOR_GRPC_METRICS_SINK")
        if grpcSink:
            try:
                host, port = split_host_port(grpcSink)
                valid = True
            except ValueError as ex:
                config.ir.logger.error("AMBASSADOR_GRPC_METRICS_SINK value %s is invalid: %s" % (grpcSink, ex))
                valid = False

            if valid:
                stats_sinks.append({
                    'name': "envoy.metrics_service",
                    'typed_config': {
                        '@type': 'type.googleapis.com/envoy.config.metrics.v2.MetricsServiceConfig',
                        'grpc_service': {
                            'envoy_grpc': {
                                'cluster_name': 'envoy_metrics_service'
                            }
                        }
                    }
                })

                clusters.append({
                    "name": "envoy_metrics_service",
                    "type": "strict_dns",
                    "connect_timeout": "1s",
                    "http2_protocol_options": {},
                    "lb_policy": "ROUND_ROBIN",
                    "load_assignment": {
                        "cluster_name": "envoy_metrics_service",
                        "endpoints": [
                            {
                                "lb_endpoints": [
                                    {
                                        "endpoint": {
                                            "address": {
                                                "socket_address": {
                                                    "address": host,
                                                    "port_value": port,
                                                    "protocol": "TCP"
                                                }
                                            }
                                        }
                                    }
                                ]
                            }
                        ]
                    }
                })

        if stats_sinks:
            self['stats_sinks'] = stats_sinks
        self['static_resources']['clusters'] = clusters

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.bootstrap = V2Bootstrap(config)


def split_host_port(value: str) -> Tuple[Optional[str], int]:
    parsed = urlparse("//"+value)
    return parsed.hostname, int(parsed.port or 80)
