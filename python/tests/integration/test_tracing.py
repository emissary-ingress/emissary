import logging
from pathlib import Path
from typing import TYPE_CHECKING, Optional

import pytest

from tests.selfsigned import TLSCerts
from tests.utils import (
    assert_valid_envoy_config,
    econf_foreach_cluster,
    module_and_mapping_manifests,
)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)

logger = logging.getLogger("ambassador")

from ambassador import IR, Config, EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler, SecretHandler, SecretInfo
from tests.utils import default_listener_manifests

if TYPE_CHECKING:
    from ambassador.ir.irresource import IRResource  # pragma: no cover


class MockSecretHandler(SecretHandler):
    def load_secret(
        self, resource: "IRResource", secret_name: str, namespace: str
    ) -> Optional[SecretInfo]:
        return SecretInfo(
            "fallback-self-signed-cert",
            "ambassador",
            "mocked-fallback-secret",
            TLSCerts["acook"].pubcert,
            TLSCerts["acook"].privkey,
            decode_b64=False,
        )


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
def test_tracing_config_v3(tmp_path: Path):
    aconf = Config()

    yaml = (
        module_and_mapping_manifests(None, [])
        + "\n"
        + """
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
metadata:
    name: myts
    namespace: default
spec:
    service: zipkin-test:9411
    driver: zipkin
    custom_tags:
    - tag: ltag
      literal:
        value: avalue
    - tag: etag
      environment:
        name: UNKNOWN_ENV_VAR
        default_value: efallback
    - tag: htag
      request_header:
        name: x-does-not-exist
        default_value: hfallback
"""
    )

    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml, k8s=True)

    aconf.load_all(fetcher.sorted())

    secret_handler = MockSecretHandler(
        logger, "mockery", str(tmp_path / "ambassador" / "snapshots"), "v1"
    )
    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir

    econf = EnvoyConfig.generate(ir)

    # check if custom_tags are added
    assert econf.as_dict()["static_resources"]["listeners"][0]["filter_chains"][0]["filters"][0][
        "typed_config"
    ]["tracing"] == {
        "custom_tags": [
            {"literal": {"value": "avalue"}, "tag": "ltag"},
            {
                "environment": {"default_value": "efallback", "name": "UNKNOWN_ENV_VAR"},
                "tag": "etag",
            },
            {
                "request_header": {"default_value": "hfallback", "name": "x-does-not-exist"},
                "tag": "htag",
            },
        ]
    }

    bootstrap_config, ads_config, _ = econf.split_config()
    assert "tracing" in bootstrap_config
    assert bootstrap_config["tracing"] == {
        "http": {
            "name": "envoy.zipkin",
            "typed_config": {
                "@type": "type.googleapis.com/envoy.config.trace.v3.ZipkinConfig",
                "collector_endpoint": "/api/v2/spans",
                "collector_endpoint_version": "HTTP_JSON",
                "trace_id_128bit": True,
                "collector_cluster": "cluster_tracing_zipkin_test_9411_default",
            },
        }
    }

    ads_config.pop("@type", None)
    assert_valid_envoy_config(ads_config, extra_dirs=[str(tmp_path / "ambassador" / "snapshots")])
    assert_valid_envoy_config(
        bootstrap_config, extra_dirs=[str(tmp_path / "ambassador" / "snapshots")]
    )


@pytest.mark.compilertest
def test_tracing_zipkin_defaults():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
metadata:
    name: myts
    namespace: default
spec:
    service: zipkin-test:9411
    driver: zipkin
"""

    econf = _get_envoy_config(yaml)

    bootstrap_config, _, _ = econf.split_config()
    assert "tracing" in bootstrap_config

    assert bootstrap_config["tracing"] == {
        "http": {
            "name": "envoy.zipkin",
            "typed_config": {
                "@type": "type.googleapis.com/envoy.config.trace.v3.ZipkinConfig",
                "collector_endpoint": "/api/v2/spans",
                "collector_endpoint_version": "HTTP_JSON",
                "trace_id_128bit": True,
                "collector_cluster": "cluster_tracing_zipkin_test_9411_default",
            },
        }
    }


@pytest.mark.compilertest
def test_tracing_cluster_fields():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
metadata:
    name: myts
    namespace: default
spec:
    service: zipkin-test:9411
    driver: zipkin
    stats_name: tracingservice
"""

    econf = _get_envoy_config(yaml)

    bootstrap_config, _, _ = econf.split_config()
    assert "tracing" in bootstrap_config

    cluster_name = "cluster_tracing_zipkin_test_9411_default"
    assert bootstrap_config["tracing"] == {
        "http": {
            "name": "envoy.zipkin",
            "typed_config": {
                "@type": "type.googleapis.com/envoy.config.trace.v3.ZipkinConfig",
                "collector_endpoint": "/api/v2/spans",
                "collector_endpoint_version": "HTTP_JSON",
                "trace_id_128bit": True,
                "collector_cluster": cluster_name,
            },
        }
    }

    def check_fields(cluster):
        assert cluster["alt_stat_name"] == "tracingservice"

    econf_foreach_cluster(econf.as_dict(), check_fields, name=cluster_name)


@pytest.mark.compilertest
def test_tracing_zipkin_invalid_collector_version():
    """test to ensure that providing an improper value will result in an error and the tracer not included"""

    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
metadata:
    name: myts
    namespace: default
spec:
    service: zipkin-test:9411
    driver: zipkin
    config:
        collector_endpoint_version: "HTTP_JSON_V1"
"""

    econf = _get_envoy_config(yaml)

    bootstrap_config, _, _ = econf.split_config()
    assert "tracing" not in bootstrap_config


@pytest.mark.compilertest
def test_lightstep_not_supported(tmp_path: Path):
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
metadata:
  name: tracing
  namespace: ambassador
spec:
  service: lightstep:80
  driver: lightstep
  custom_tags:
  - tag: ltag
    literal:
      value: avalue
  - tag: etag
    environment:
      name: UNKNOWN_ENV_VAR
      default_value: efallback
  - tag: htag
    request_header:
      name: x-does-not-exist
      default_value: hfallback
  config:
    access_token_file: /lightstep-credentials/access-token
    propagation_modes: ["ENVOY", "TRACE_CONTEXT"]
"""
    econf = _get_envoy_config(yaml)
    assert "ir.tracing" in econf.ir.aconf.errors

    tracing_error = econf.ir.aconf.errors["ir.tracing"][0]["error"]
    assert "'lightstep' driver is no longer supported" in tracing_error

    bootstrap_config, _, _ = econf.split_config()
    assert "tracing" not in bootstrap_config
