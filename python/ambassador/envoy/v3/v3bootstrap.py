from typing import TYPE_CHECKING
from typing import cast as typecast

import os

from ...ir.ircluster import IRCluster
from ...ir.irlogservice import IRLogService
from ...ir.irratelimit import IRRateLimit
from ...ir.irtracing import IRTracing

from .v3cluster import V3Cluster

if TYPE_CHECKING:
    from . import V3Config # pragma: no cover


class V3Bootstrap(dict):
    def __init__(self, config: 'V3Config') -> None:
        api_version = "V3"
        super().__init__(**{
            "node": {
                "cluster": config.ir.ambassador_nodename,
                "id": "test-id"         # MUST BE test-id, see below
            },
            "static_resources": {},     # Filled in later
            "dynamic_resources": {
                "ads_config": {
                    "api_type": "GRPC",
                    "transport_api_version": api_version,
                    "grpc_services": [ {
                        "envoy_grpc": {
                            "cluster_name": "xds_cluster"
                        }
                    } ]
                },
                "cds_config": {
                    "ads": {},
                    "resource_api_version": api_version
                },
                "lds_config": {
                    "ads": {},
                    "resource_api_version": api_version
                }
            },
            "admin": dict(config.admin),
            'layered_runtime': {
                'layers': [
                    {
                        'name': 'static_layer',
                        'static_layer': {
                            # Envoy 1.14.1 disabled the use of lowercase string matcher for headers matching in HTTP-based.
                            # Following setting toggled it to be consistent with old behavior.
                            # AuthenticationTest (v0) is a good example that expects the old behavior.
                            'envoy.reloadable_features.ext_authz_http_service_enable_case_sensitive_string_matcher': False,
                            're2.max_program_size.error_level': 200,
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
                                            # this should be kept in-sync with entrypoint.sh `ambex --ads-listen-address=...`
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
            clusters.append(V3Cluster(config, typecast(IRCluster, tracing.cluster)))

        if config.ir.log_services.values():
            for als in config.ir.log_services.values():
                log_service = typecast(IRLogService, als)
                assert log_service.cluster
                clusters.append(V3Cluster(config, typecast(IRCluster, log_service.cluster)))

        # if config.ratelimit:
        #     self['rate_limit_service'] = dict(config.ratelimit)
        #
        #     ratelimit = typecast(IRRateLimit, config.ir.ratelimit)
        #
        #     assert ratelimit.cluster
        #     clusters.append(V3Cluster(config, ratelimit.cluster))

        stats_sinks = []

        if config.ir.statsd['enabled']:
            if config.ir.statsd['dogstatsd']:
                name = 'envoy.stat_sinks.dog_statsd'
                typename = 'type.googleapis.com/envoy.config.metrics.v3.DogStatsdSink'
                dd_entity_id = os.environ.get('DD_ENTITY_ID', None)
                if dd_entity_id:
                    stats_tags = self.setdefault('stats_config', {}).setdefault('stats_tags', [])
                    stats_tags.append({
                        'tag_name': 'dd.internal.entity_id',
                        'fixed_value': dd_entity_id
                    })
            else:
                name = 'envoy.stats_sinks.statsd'
                typename = 'type.googleapis.com/envoy.config.metrics.v3.StatsdSink'

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
                        '@type': 'type.googleapis.com/envoy.config.metrics.v3.MetricsServiceConfig',
                        'grpc_service': {
                            'envoy_grpc': {
                                'cluster_name': 'envoy_metrics_service'
                            }
                        },
                        'transport_api_version': 'v3'
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
    def generate(cls, config: 'V3Config') -> None:
        config.bootstrap = V3Bootstrap(config)


def split_host_port(value):
    parts = value.split(":")
    if len(parts) == 1:
        host = parts[0]
        port = 80
    elif len(parts) == 2:
        host = parts[0]
        port = int(parts[1])
    else:
        raise ValueError("too many colons")
    return host, port
