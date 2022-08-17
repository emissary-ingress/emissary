import os
from typing import TYPE_CHECKING, Optional, Tuple
from typing import cast as typecast
from urllib.parse import urlparse

from ...ir.ircluster import IRCluster
from ...ir.irlogservice import IRLogService
from ...ir.irratelimit import IRRateLimit
from ...ir.irtracing import IRTracing
from .v3cluster import V3Cluster

if TYPE_CHECKING:
    from . import V3Config  # pragma: no cover


class V3Bootstrap(dict):
    def __init__(self, config: "V3Config") -> None:
        api_version = "V3"
        super().__init__(
            **{
                "node": {
                    "cluster": config.ir.ambassador_nodename,
                    "id": "test-id",  # MUST BE test-id, see below
                },
                "static_resources": {},  # Filled in later
                "dynamic_resources": {
                    "ads_config": {
                        "api_type": "GRPC",
                        "transport_api_version": api_version,
                        "grpc_services": [{"envoy_grpc": {"cluster_name": "xds_cluster"}}],
                    },
                    "cds_config": {"ads": {}, "resource_api_version": api_version},
                    "lds_config": {"ads": {}, "resource_api_version": api_version},
                },
                "admin": dict(config.admin),
                "layered_runtime": {
                    "layers": [
                        {
                            "name": "static_layer",
                            "static_layer": {
                                "re2.max_program_size.error_level": 200,
                                # TODO(lance):
                                # the new default is that all filters are looked up using the @type which currently we exclude on a lot of
                                # our filters. This will ensure we do not break current config. We can migrate over
                                # in a minor release. see here: https://www.envoyproxy.io/docs/envoy/v1.22.0/version_history/current#minor-behavior-changes
                                # The biggest impact of this is ensuring that ambex imports all the types because we will need to import many more
                                "envoy.reloadable_features.no_extension_lookup_by_name": False,
                            },
                        }
                    ]
                },
            }
        )

        clusters = [
            {
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
                    ],
                },
            }
        ]

        if config.tracing:
            self["tracing"] = dict(config.tracing)

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

        grpcSink = os.environ.get("AMBASSADOR_GRPC_METRICS_SINK")
        if grpcSink:
            try:
                host, port = split_host_port(grpcSink)
                valid = True
            except ValueError as ex:
                config.ir.logger.error(
                    "AMBASSADOR_GRPC_METRICS_SINK value %s is invalid: %s" % (grpcSink, ex)
                )
                valid = False

            if valid:
                stats_sinks.append(
                    {
                        "name": "envoy.metrics_service",
                        "typed_config": {
                            "@type": "type.googleapis.com/envoy.config.metrics.v3.MetricsServiceConfig",
                            "transport_api_version": "V3",
                            "grpc_service": {
                                "envoy_grpc": {"cluster_name": "envoy_metrics_service"}
                            },
                        },
                    }
                )
                clusters.append(
                    {
                        "name": "envoy_metrics_service",
                        "type": "strict_dns",
                        "connect_timeout": "1s",
                        "http2_protocol_options": {},
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
                                                        "protocol": "TCP",
                                                    }
                                                }
                                            }
                                        }
                                    ]
                                }
                            ],
                        },
                    }
                )

        if config.ir.statsd["enabled"]:
            if config.ir.statsd["dogstatsd"]:
                name = "envoy.stat_sinks.dog_statsd"
                typename = "type.googleapis.com/envoy.config.metrics.v3.DogStatsdSink"
                dd_entity_id = os.environ.get("DD_ENTITY_ID", None)
                if dd_entity_id:
                    stats_tags = self.setdefault("stats_config", {}).setdefault("stats_tags", [])
                    stats_tags.append(
                        {"tag_name": "dd.internal.entity_id", "fixed_value": dd_entity_id}
                    )
            else:
                name = "envoy.stats_sinks.statsd"
                typename = "type.googleapis.com/envoy.config.metrics.v3.StatsdSink"

            stats_sinks.append(
                {
                    "name": name,
                    "typed_config": {
                        "@type": typename,
                        "address": {
                            "socket_address": {
                                "protocol": "UDP",
                                "address": config.ir.statsd["ip"],
                                "port_value": 8125,
                            }
                        },
                    },
                }
            )

            self["stats_flush_interval"] = {"seconds": config.ir.statsd["interval"]}
        self["stats_sinks"] = stats_sinks
        self["static_resources"]["clusters"] = clusters

    @classmethod
    def generate(cls, config: "V3Config") -> None:
        config.bootstrap = V3Bootstrap(config)


def split_host_port(value: str) -> Tuple[Optional[str], int]:
    parsed = urlparse("//" + value)
    return parsed.hostname, int(parsed.port or 80)
