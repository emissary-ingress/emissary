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
from typing import TYPE_CHECKING, Dict, List, Union

from ...cache import Cacheable
from ...config import Config
from ...ir.ircluster import IRCluster
from .v3tls import V3TLSContext

if TYPE_CHECKING:
    from . import V3Config  # pragma: no cover


class V3Cluster(Cacheable):
    def __init__(self, config: "V3Config", cluster: IRCluster) -> None:
        super().__init__()

        dns_lookup_family = "V4_ONLY"

        if cluster.enable_ipv6:
            if cluster.enable_ipv4:
                dns_lookup_family = "AUTO"
            else:
                dns_lookup_family = "V6_ONLY"

        # We must not use cluster.name in the envoy config, since it may be too long
        # to pass envoy's cluster name length constraint, currently 60 characters.
        #
        # Instead, we must have generated an appropriate envoy_name during IR finalize.
        # In practice, the envoy_name is a short-form of cluster.name with the first
        # 40 characters followed by `-n` where `n` is an incremental value, one for
        # every cluster whose name contains the same first 40 characters.
        assert cluster.envoy_name
        assert len(cluster.envoy_name) <= 60

        cmap_entry = cluster.clustermap_entry()
        if cmap_entry["kind"] == "KubernetesServiceResolver":
            ctype = cluster.type.upper()
            # For now we are only allowing Logical_dns for the cluster since it is similar enough to strict_dns that we dont need any other config changes
            # It should be easy to add the other dns_types here in the future if we decide to support them
            if ctype not in ["STRICT_DNS", "LOGICAL_DNS"]:
                cluster.ir.logger.warning(
                    "dns_type %s, is an invalid type. Options are STRICT_DNS or LOGICAL_DNS. Using default of STRICT_DNS"
                    % (ctype)
                )
                ctype = "STRICT_DNS"
        else:
            ctype = "EDS"

        fields = {
            "name": cluster.envoy_name,
            "type": ctype,
            "lb_policy": cluster.lb_type.upper(),
            "connect_timeout": "%0.3fs" % (float(cluster.connect_timeout_ms) / 1000.0),
            "dns_lookup_family": dns_lookup_family,
        }

        if cluster.get("stats_name", ""):
            fields["alt_stat_name"] = cluster.stats_name

        if cluster.respect_dns_ttl:
            fields["respect_dns_ttl"] = cluster.respect_dns_ttl

        if ctype == "EDS":
            fields["eds_cluster_config"] = {
                "eds_config": {
                    "ads": {},
                    # Envoy may default to an older API version if we are not explicit about V3 here.
                    "resource_api_version": "V3",
                },
                "service_name": cmap_entry["endpoint_path"],
            }
        else:
            fields["load_assignment"] = {
                "cluster_name": cluster.envoy_name,
                "endpoints": [{"lb_endpoints": self.get_endpoints(cluster)}],
            }

        if cluster.cluster_idle_timeout_ms:
            cluster_idle_timeout_ms = cluster.cluster_idle_timeout_ms
        else:
            cluster_idle_timeout_ms = cluster.ir.ambassador_module.get(
                "cluster_idle_timeout_ms", None
            )
        if cluster_idle_timeout_ms:
            common_http_options = self.setdefault("common_http_protocol_options", {})
            common_http_options["idle_timeout"] = "%0.3fs" % (
                float(cluster_idle_timeout_ms) / 1000.0
            )

        if cluster.cluster_max_connection_lifetime_ms:
            cluster_max_connection_lifetime_ms = cluster.cluster_max_connection_lifetime_ms
        else:
            cluster_max_connection_lifetime_ms = cluster.ir.ambassador_module.get(
                "cluster_max_connection_lifetime_ms", None
            )
        if cluster_max_connection_lifetime_ms:
            common_http_options = self.setdefault("common_http_protocol_options", {})
            common_http_options["max_connection_duration"] = "%0.3fs" % (
                float(cluster_max_connection_lifetime_ms) / 1000.0
            )

        circuit_breakers = self.get_circuit_breakers(cluster)
        if circuit_breakers is not None:
            fields["circuit_breakers"] = circuit_breakers

        # If this cluster is using http2 for grpc, set http2_protocol_options
        # Otherwise, check for http1-specific configuration.
        if cluster.get("grpc", False):
            self["http2_protocol_options"] = {}
        else:
            proper_case: bool = cluster.ir.ambassador_module["proper_case"]

            # Get the list of upstream headers whose casing should be overriden
            # from the Ambassador module. We configure the downstream side of this
            # in v3listener.py
            header_case_overrides = cluster.ir.ambassador_module.get("header_case_overrides", None)
            if (
                header_case_overrides
                and not proper_case
                and isinstance(header_case_overrides, list)
            ):
                # We have this config validation here because the Ambassador module is
                # still an untyped config. That is, we aren't yet using a CRD or a
                # python schema to constrain the configuration that can be present.
                rules = []
                for hdr in header_case_overrides:
                    if not isinstance(hdr, str):
                        continue
                    rules.append(hdr)
                if len(rules) > 0:
                    custom_header_rules: Dict[str, Dict[str, dict]] = {
                        "custom": {"rules": {header.lower(): header for header in rules}}
                    }
                    http_options = self.setdefault("http_protocol_options", {})
                    http_options["header_key_format"] = custom_header_rules

        ctx = cluster.get("tls_context", None)

        if ctx is not None:
            # If this is a null TLS Context (_ambassador_enabled is True), then we at need to specify a
            # minimal `tls_context` to enable HTTPS origination. This means that we type envoy_ctx just
            # as a plain ol' dict, because it's a royal pain to try to use a V3TLSContext (which is a
            # subclass of dict anyway) for this degenerate case.
            #
            # XXX That's a silly reason to not do proper typing.
            envoy_ctx: dict

            if ctx.get("_ambassador_enabled", False):
                envoy_ctx = {"common_tls_context": {}}
            else:
                envoy_ctx = V3TLSContext(ctx=ctx, host_rewrite=cluster.get("host_rewrite", None))
            if envoy_ctx:
                fields["transport_socket"] = {
                    "name": "envoy.transport_sockets.tls",
                    "typed_config": {
                        "@type": "type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext",
                        **envoy_ctx,
                    },
                }

        keepalive = cluster.get("keepalive", None)
        # in case of empty keepalive for service, we can try to fallback to default
        if keepalive is None:
            if cluster.ir.ambassador_module and cluster.ir.ambassador_module.get("keepalive", None):
                keepalive = cluster.ir.ambassador_module["keepalive"]

        if keepalive is not None:
            keepalive_options = {}
            keepalive_time = keepalive.get("time", None)
            keepalive_interval = keepalive.get("interval", None)
            keepalive_probes = keepalive.get("probes", None)

            if keepalive_time is not None:
                keepalive_options["keepalive_time"] = keepalive_time
            if keepalive_interval is not None:
                keepalive_options["keepalive_interval"] = keepalive_interval
            if keepalive_probes is not None:
                keepalive_options["keepalive_probes"] = keepalive_probes

            fields["upstream_connection_options"] = {"tcp_keepalive": keepalive_options}

        self.update(fields)

    def get_endpoints(self, cluster: IRCluster):
        result = []

        targetlist = cluster.get("targets", [])

        if len(targetlist) > 0:
            for target in targetlist:
                address = {
                    "address": target["ip"],
                    "port_value": target["port"],
                    "protocol": "TCP",  # Yes, really. Envoy uses the TLS context to determine whether to originate TLS.
                }
                result.append({"endpoint": {"address": {"socket_address": address}}})
        else:
            for u in cluster.urls:
                p = urllib.parse.urlparse(u)
                address = {"address": p.hostname, "port_value": int(p.port)}
                if p.scheme:
                    address["protocol"] = p.scheme.upper()
                result.append({"endpoint": {"address": {"socket_address": address}}})
        return result

    def get_circuit_breakers(self, cluster: IRCluster):
        cluster_circuit_breakers = cluster.get("circuit_breakers", None)
        if cluster_circuit_breakers is None:
            return None

        circuit_breakers: Dict[str, List[Dict[str, Union[str, int]]]] = {"thresholds": []}

        for circuit_breaker in cluster_circuit_breakers:
            threshold = {}
            if "priority" in circuit_breaker:
                threshold["priority"] = circuit_breaker.get("priority").upper()
            else:
                threshold["priority"] = "DEFAULT"

            digit_fields = [
                "max_connections",
                "max_pending_requests",
                "max_requests",
                "max_retries",
            ]
            for field in digit_fields:
                if field in circuit_breaker:
                    threshold[field] = int(circuit_breaker.get(field))

            if len(threshold) > 0:
                circuit_breakers["thresholds"].append(threshold)

        return circuit_breakers

    @classmethod
    def generate(self, config: "V3Config") -> None:
        cluster: "V3Cluster"

        config.clusters = []
        config.clustermap = {}

        # Sort by the envoy cluster name (x.envoy_name), not the symbolic IR cluster name (x.name)
        for ircluster in sorted(config.ir.clusters.values(), key=lambda x: x.envoy_name):
            # XXX This magic format is duplicated for now in ir.py.
            cache_key = f"V3-{ircluster.cache_key}"
            cached_cluster = config.cache[cache_key]

            if cached_cluster is None:
                # Cache miss.
                cluster = config.save_element("cluster", ircluster, V3Cluster(config, ircluster))

                # Cheat a bit and force the route's cache key.
                cluster.cache_key = cache_key

                # Not all IRClusters are cached yet -- e.g. AuthService and RateLimitService
                # don't participate in the cache yet. Don't try to cache this V3Cluster if
                # we won't be able to link it correctly.
                cached_ircluster = config.cache[ircluster.cache_key]

                if cached_ircluster is not None:
                    config.cache.add(cluster)
                    config.cache.link(ircluster, cluster)
                    config.cache.dump("V2Cluster synth %s", cache_key)
            else:
                # Cache hit. We know a priori that it's a V3Cluster, but let's assert
                # that rather than casting.
                assert isinstance(cached_cluster, V3Cluster)
                cluster = cached_cluster

            config.clusters.append(cluster)
            config.clustermap[ircluster.envoy_name] = ircluster.clustermap_entry()
