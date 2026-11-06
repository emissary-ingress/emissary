import pytest

from tests.utils import default_listener_manifests, econf_compile

SERVICE = "sharedsvc"


def _mapping(name: str, prefix: str, resolver: str = "") -> str:
    resolver_line = f"\n  resolver: {resolver}" if resolver else ""
    return f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {name}
  namespace: default
spec:
  hostname: "*"
  prefix: {prefix}
  service: {SERVICE}{resolver_line}
"""


def _clusters_for_service(econf):
    return {
        c["name"]: c
        for c in econf["static_resources"]["clusters"]
        if c["name"].startswith(f"cluster_{SERVICE}_")
    }


def _route_clusters(econf):
    found = {}
    for listener in econf["static_resources"]["listeners"]:
        for chain in listener["filter_chains"]:
            for f in chain["filters"]:
                route_config = f["typed_config"].get("route_config", {})
                for vhost in route_config.get("virtual_hosts", []):
                    for route in vhost["routes"]:
                        prefix = route["match"].get("prefix")
                        cluster = route.get("route", {}).get("cluster")
                        if prefix and cluster:
                            found[prefix] = cluster
    return found


# Mappings load sorted by name and the first one to reach a cluster name creates the
# cluster, so both name orders are covered.
@pytest.mark.compilertest
@pytest.mark.parametrize(
    "service_name,endpoint_name",
    [("svc-resolver", "endpoint-resolver"), ("a-svc-resolver", "z-endpoint-resolver")],
    ids=["endpoint-mapping-first", "service-mapping-first"],
)
def test_service_and_endpoint_resolvers_get_separate_clusters(
    service_name, endpoint_name
):
    service_prefix = "/svc/"
    endpoint_prefix = "/endpoint/"
    yaml = (
        default_listener_manifests()
        + _mapping(service_name, service_prefix)
        + _mapping(endpoint_name, endpoint_prefix, resolver="endpoint")
    )

    econf = econf_compile(yaml)

    clusters = _clusters_for_service(econf)
    routes = _route_clusters(econf)
    summary = {
        "clusters": {name: c["type"] for name, c in clusters.items()},
        "routes": {p: routes[p] for p in (service_prefix, endpoint_prefix)},
    }

    assert len(clusters) == 2, summary

    svc_cluster = clusters[routes[service_prefix]]
    ep_cluster = clusters[routes[endpoint_prefix]]
    assert svc_cluster is not ep_cluster, summary

    assert svc_cluster["type"] == "STRICT_DNS", summary
    assert "load_assignment" in svc_cluster, summary

    assert ep_cluster["type"] == "EDS", summary
    assert (
        ep_cluster["eds_cluster_config"]["service_name"] == f"k8s/default/{SERVICE}/80"
    ), summary
