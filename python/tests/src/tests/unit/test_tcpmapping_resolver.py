"""Regression tests for https://github.com/emissary-ingress/emissary/issues/3330

A TCPMapping that uses a KubernetesEndpointResolver (explicitly, or via the
Ambassador Module's default resolver) must produce an EDS cluster that routes to
the Service's endpoint IPs, not a STRICT_DNS cluster pointed at the ClusterIP.
"""

import pytest

from tests.utils import compile_with_cachecheck, default_tcp_listener_manifest


ENDPOINT_RESOLVER = """
---
apiVersion: getambassador.io/v3alpha1
kind: KubernetesEndpointResolver
metadata:
  name: my-endpoint
  namespace: default
spec: {}
"""


def _tcpmapping(service: str, resolver: str = "") -> str:
    resolver_line = f"  resolver: {resolver}\n" if resolver else ""
    return f"""
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
metadata:
  name: mysql
  namespace: default
spec:
  port: 8443
  service: {service}
{resolver_line}"""


def _module(resolver: str) -> str:
    return f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    resolver: {resolver}
"""


def _find_cluster(yaml: str, name: str) -> dict:
    compiled = compile_with_cachecheck(yaml)
    clusters = compiled["xds"].as_dict()["static_resources"]["clusters"]
    matches = [c for c in clusters if c["name"] == name]
    assert len(matches) == 1, f"expected exactly one cluster named {name}, got {clusters}"
    return matches[0]


@pytest.mark.compilertest
def test_tcpmapping_explicit_endpoint_resolver():
    yaml = (
        default_tcp_listener_manifest()
        + ENDPOINT_RESOLVER
        + _tcpmapping("mysql:3306", resolver="my-endpoint")
    )
    cluster = _find_cluster(yaml, "cluster_mysql_3306_default")

    assert cluster["type"] == "EDS"
    assert cluster["eds_cluster_config"]["service_name"] == "k8s/default/mysql/3306"
    assert "load_assignment" not in cluster


@pytest.mark.compilertest
def test_tcpmapping_builtin_endpoint_resolver():
    # "endpoint" is an implicitly-defined KubernetesEndpointResolver.
    yaml = default_tcp_listener_manifest() + _tcpmapping("mysql:3306", resolver="endpoint")
    cluster = _find_cluster(yaml, "cluster_mysql_3306_default")

    assert cluster["type"] == "EDS"
    assert cluster["eds_cluster_config"]["service_name"] == "k8s/default/mysql/3306"


@pytest.mark.compilertest
def test_tcpmapping_module_default_endpoint_resolver():
    yaml = (
        default_tcp_listener_manifest()
        + ENDPOINT_RESOLVER
        + _module("my-endpoint")
        + _tcpmapping("mysql:3306")
    )
    cluster = _find_cluster(yaml, "cluster_mysql_3306_default")

    assert cluster["type"] == "EDS"
    assert cluster["eds_cluster_config"]["service_name"] == "k8s/default/mysql/3306"


@pytest.mark.compilertest
def test_tcpmapping_endpoint_resolver_cross_namespace():
    yaml = (
        default_tcp_listener_manifest()
        + ENDPOINT_RESOLVER
        + _tcpmapping("mysql.other:3306", resolver="my-endpoint")
    )
    cluster = _find_cluster(yaml, "cluster_mysql_other_3306_default")

    assert cluster["type"] == "EDS"
    assert cluster["eds_cluster_config"]["service_name"] == "k8s/other/mysql/3306"


@pytest.mark.compilertest
def test_tcpmapping_default_service_resolver():
    yaml = default_tcp_listener_manifest() + _tcpmapping("mysql:3306")
    cluster = _find_cluster(yaml, "cluster_mysql_3306_default")

    assert cluster["type"] == "STRICT_DNS"
    assert "eds_cluster_config" not in cluster
    addr = cluster["load_assignment"]["endpoints"][0]["lb_endpoints"][0]["endpoint"]["address"]
    assert addr["socket_address"]["address"] == "mysql"
    assert addr["socket_address"]["port_value"] == 3306
