import copy
import logging
import sys

import pytest
from kat.harness import EDGE_STACK
from tests.utils import econf_foreach_cluster

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)

logger = logging.getLogger("ambassador")
# logger.setLevel(logging.DEBUG)

from ambassador import Config, IR, EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler

from tests.utils import default_listener_manifests


def _get_ext_auth_config(yaml):
    for listener in yaml["static_resources"]["listeners"]:
        for filter_chain in listener["filter_chains"]:
            for f in filter_chain["filters"]:
                for http_filter in f["typed_config"]["http_filters"]:
                    if http_filter["name"] == "envoy.filters.http.ext_authz":
                        return http_filter
    return False


def _get_envoy_config(yaml, version="V3"):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(default_listener_manifests() + yaml, k8s=True)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir

    return EnvoyConfig.generate(ir, version)


@pytest.mark.compilertest
def test_irauth_grpcservice_version_v2():
    if EDGE_STACK:
        pytest.xfail("XFailing for now, custom AuthServices not supported in Edge Stack")
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name:  mycoolauthservice
  namespace: default
spec:
  auth_service: someservice
  protocol_version: "v2"
  proto: grpc
"""
    econf = _get_envoy_config(yaml, version="V2")

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    assert ext_auth_config

    assert (
        ext_auth_config["typed_config"]["grpc_service"]["envoy_grpc"]["cluster_name"]
        == "cluster_extauth_someservice_default"
    )


@pytest.mark.compilertest
def test_irauth_grpcservice_version_v3():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name:  mycoolauthservice
  namespace: default
spec:
  auth_service: someservice
  protocol_version: "v3"
  proto: grpc
"""
    econf = _get_envoy_config(yaml, version="V3")

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    assert ext_auth_config

    assert (
        ext_auth_config["typed_config"]["grpc_service"]["envoy_grpc"]["cluster_name"]
        == "cluster_extauth_someservice_default"
    )
    assert ext_auth_config["typed_config"]["transport_api_version"] == "V3"


def test_cluster_fields_v3_config():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name:  mycoolauthservice
  namespace: default
spec:
  auth_service: someservice
  protocol_version: "v3"
  proto: grpc
  stats_name: authservice
"""

    econf = _get_envoy_config(yaml, version="V3")

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    cluster_name = "cluster_extauth_someservice_default"

    assert ext_auth_config
    assert (
        ext_auth_config["typed_config"]["grpc_service"]["envoy_grpc"]["cluster_name"]
        == cluster_name
    )

    def check_fields(cluster):
        assert cluster["alt_stat_name"] == "authservice"

    econf_foreach_cluster(econf.as_dict(), check_fields, name=cluster_name)


def test_cluster_fields_v2_config():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name:  mycoolauthservice
  namespace: default
spec:
  auth_service: someservice
  protocol_version: "v3"
  proto: grpc
  stats_name: authservice
"""

    econf = _get_envoy_config(yaml, version="V2")

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    cluster_name = "cluster_extauth_someservice_default"

    assert ext_auth_config
    assert (
        ext_auth_config["typed_config"]["grpc_service"]["envoy_grpc"]["cluster_name"]
        == cluster_name
    )

    def check_fields(cluster):
        assert cluster["alt_stat_name"] == "authservice"

    econf_foreach_cluster(econf.as_dict(), check_fields, name=cluster_name)


@pytest.mark.compilertest
def test_irauth_grpcservice_version_default():
    if EDGE_STACK:
        pytest.xfail("XFailing for now, custom AuthServices not supported in Edge Stack")
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name:  mycoolauthservice
  namespace: default
spec:
  auth_service: someservice
  proto: grpc
"""
    econf = _get_envoy_config(yaml, version="V2")

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    assert ext_auth_config

    assert (
        ext_auth_config["typed_config"]["grpc_service"]["envoy_grpc"]["cluster_name"]
        == "cluster_extauth_someservice_default"
    )


@pytest.mark.compilertest
def test_irauth_grpcservice_version_default_v3():
    if EDGE_STACK:
        pytest.xfail("XFailing for now, custom AuthServices not supported in Edge Stack")
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name:  mycoolauthservice
  namespace: default
spec:
  auth_service: someservice
  proto: grpc
"""
    econf = _get_envoy_config(yaml, version="V3")

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    assert ext_auth_config

    assert (
        ext_auth_config["typed_config"]["grpc_service"]["envoy_grpc"]["cluster_name"]
        == "cluster_extauth_someservice_default"
    )
    assert ext_auth_config["typed_config"]["transport_api_version"] == "V2"
