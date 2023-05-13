import logging

import pytest

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

SERVICE_NAME = "coolsvcname"


def _get_rl_config(yaml):
    for listener in yaml["static_resources"]["listeners"]:
        for filter_chain in listener["filter_chains"]:
            for f in filter_chain["filters"]:
                for http_filter in f["typed_config"]["http_filters"]:
                    if http_filter["name"] == "envoy.filters.http.ratelimit":
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


def _get_ratelimit_default_conf():
    return {
        "@type": "type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit",
        "domain": "ambassador",
        "request_type": "both",
        "timeout": "0.020s",
        "failure_mode_deny": False,
        "rate_limited_as_resource_exhausted": False,
        "rate_limit_service": {
            "transport_api_version": "V3",
            "grpc_service": {
                "envoy_grpc": {"cluster_name": "cluster_{}_default".format(SERVICE_NAME)}
            },
        },
    }


@pytest.mark.compilertest
def test_irratelimit_defaults():
    # Test all defaults
    yaml = """
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: myrls
  namespace: default
spec:
  service: {}
""".format(
        SERVICE_NAME
    )

    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert conf == False

    errors = econf.ir.aconf.errors
    assert "ir.ratelimit" in errors
    assert (
        errors["ir.ratelimit"][0]["error"]
        == 'RateLimitService: protocol_version v2 is unsupported, protocol_version must be "v3"'
    )


@pytest.mark.compilertest
def test_irratelimit_grpcsvc_version_v3():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: myrls
  namespace: default
spec:
  service: {}
  protocol_version: "v3"
""".format(
        SERVICE_NAME
    )

    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert conf
    assert conf.get("typed_config") == _get_ratelimit_default_conf()

    assert "ir.ratelimit" not in econf.ir.aconf.errors


@pytest.mark.compilertest
def test_irratelimit_cluster_fields():
    stats_name = "ratelimitservice"

    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: myrls
  namespace: default
spec:
  service: {}
  protocol_version: "v3"
  stats_name: {}
""".format(
        SERVICE_NAME, stats_name
    )

    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert conf
    assert conf.get("typed_config") == _get_ratelimit_default_conf()

    assert "ir.ratelimit" not in econf.ir.aconf.errors

    def check_fields(cluster):
        assert cluster["alt_stat_name"] == stats_name

    econf_foreach_cluster(
        econf.as_dict(), check_fields, name="cluster_{}_default".format(SERVICE_NAME)
    )


@pytest.mark.compilertest
def test_irratelimit_grpcsvc_version_v2():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: myrls
  namespace: default
spec:
  service: {}
  protocol_version: "v2"
""".format(
        SERVICE_NAME
    )

    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert conf == False

    errors = econf.ir.aconf.errors
    assert "ir.ratelimit" in errors
    assert (
        errors["ir.ratelimit"][0]["error"]
        == 'RateLimitService: protocol_version v2 is unsupported, protocol_version must be "v3"'
    )


@pytest.mark.compilertest
def test_irratelimit_error():
    """Test error no valid spec with service name"""

    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: myrls
  namespace: default
spec: {}
"""

    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert not conf

    errors = econf.ir.aconf.errors
    assert "ir.ratelimit" in errors
    assert errors["ir.ratelimit"][0]["error"] == "service is required in RateLimitService"


@pytest.mark.compilertest
def test_irratelimit_overrides():
    """Test that default are properly overriden"""

    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: RateLimitService
metadata:
  name: myrls
  namespace: someotherns
spec:
  service: {}
  domain: otherdomain
  timeout_ms: 500
  tls: rl-tls-context
  protocol_version: v3
  failure_mode_deny: True
  rate_limited_as_resource_exhausted: True
""".format(
        SERVICE_NAME
    )

    config = _get_ratelimit_default_conf()
    config["rate_limit_service"]["grpc_service"]["envoy_grpc"][
        "cluster_name"
    ] = "cluster_{}_someotherns".format(SERVICE_NAME)
    config["timeout"] = "0.500s"
    config["domain"] = "otherdomain"
    config["failure_mode_deny"] = True
    config["rate_limited_as_resource_exhausted"] = True

    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert conf
    assert conf.get("typed_config") == config

    errors = econf.ir.aconf.errors
    assert "ir.ratelimit" not in errors
