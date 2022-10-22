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

from ambassador import IR, Config, EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler
from tests.utils import default_listener_manifests


def _assert_ext_auth_disabled(route):
    assert route
    per_filter_config = route.get("typed_per_filter_config")
    assert per_filter_config.get("envoy.filters.http.ext_authz")
    assert per_filter_config.get("envoy.filters.http.ext_authz").get("disabled") == True


def _get_ext_auth_config(yaml):
    for listener in yaml["static_resources"]["listeners"]:
        for filter_chain in listener["filter_chains"]:
            for f in filter_chain["filters"]:
                for http_filter in f["typed_config"]["http_filters"]:
                    if http_filter["name"] == "envoy.filters.http.ext_authz":
                        return http_filter
    return False


def _get_envoy_config(yaml):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(default_listener_manifests() + yaml, k8s=True)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir

    return EnvoyConfig.generate(ir)


@pytest.mark.compilertest
def test_irauth_grpcservice_version_v2():
    """Test to ensure that setting protocol_version to will cause an error"""

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

    econf = _get_envoy_config(yaml)

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    assert ext_auth_config == False

    errors = econf.ir.aconf.errors["mycoolauthservice.default.1"]
    assert (
        errors[0]["error"]
        == 'AuthService: protocol_version v2 is unsupported, protocol_version must be "v3"'
    )


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

    econf = _get_envoy_config(yaml)

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    assert ext_auth_config
    assert (
        ext_auth_config["typed_config"]["grpc_service"]["envoy_grpc"]["cluster_name"]
        == "cluster_extauth_someservice_default"
    )
    assert ext_auth_config["typed_config"]["transport_api_version"] == "V3"

    assert "mycoolauthservice.default.1" not in econf.ir.aconf.errors


def test_cluster_fields():
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

    econf = _get_envoy_config(yaml)

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

    econf = _get_envoy_config(yaml)

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    assert ext_auth_config == False

    errors = econf.ir.aconf.errors["mycoolauthservice.default.1"]
    assert (
        errors[0]["error"]
        == 'AuthService: protocol_version v2 is unsupported, protocol_version must be "v3"'
    )


@pytest.mark.compilertest
def test_basic_http_redirect_with_no_authservice():
    """Test that http --> https redirect route exists when no AuthService is provided
    and verify that the typed_per_filter_config is NOT included
    """

    yaml = """
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: ambassador
  namespace: default
spec:
  hostname: "*"
  prefix: /httpbin/
  service: httpbin
    """
    econf = _get_envoy_config(yaml)

    for rv in econf.route_variants:
        if rv.route.get("match").get("prefix") == "/httpbin/":
            xfp_http_redirect = rv.variants.get("xfp-http-redirect")
            assert xfp_http_redirect
            assert "redirect" in xfp_http_redirect
            assert "typed_per_filter_config" not in xfp_http_redirect


@pytest.mark.compilertest
def test_redirects_disables_ext_authz():
    """Test that the ext_authz is disabled on envoy redirect routes
    for https_redirects and host_redirects. This is to ensure that the
    redirect occurs before making any calls to the ext_authz service
    """

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
  protocol_version: v3
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: ambassador
  namespace: default
spec:
  hostname: "*"
  prefix: /httpbin/
  service: httpbin
  host_redirect: true
    """
    econf = _get_envoy_config(yaml)

    # check https_redirect variant route
    for rv in econf.route_variants:
        if rv.route.get("match").get("prefix") == "/httpbin/":
            xfp_http_redirect = rv.variants.get("xfp-http-redirect")
            assert xfp_http_redirect
            assert "redirect" in xfp_http_redirect
            _assert_ext_auth_disabled(xfp_http_redirect)

    # check host_redirect route
    for route in econf.routes:
        if route.get("match").get("prefix") == "/httpbin/":
            redirect = route.get("redirect")
            assert redirect
            assert redirect.get("host_redirect") == "httpbin"
            _assert_ext_auth_disabled(route)
