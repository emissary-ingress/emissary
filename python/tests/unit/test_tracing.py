from typing import Optional, TYPE_CHECKING

import logging
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

from ambassador import Config, IR
from ambassador.envoy import EnvoyConfig
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


def _get_envoy_config(yaml, version="V2"):

    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(default_listener_manifests() + yaml, k8s=True)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir
    return EnvoyConfig.generate(ir, version)


def lightstep_tracing_service_manifest():
    return """
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
metadata:
  name: tracing
  namespace: ambassador
spec:
  service: lightstep:80
  driver: lightstep
  config:
    access_token_file: /lightstep-credentials/access-token
    propagation_modes: ["ENVOY", "TRACE_CONTEXT"]
"""


@pytest.mark.compilertest
def test_tracing_config_v3():
    aconf = Config()

    yaml = module_and_mapping_manifests(None, []) + "\n" + lightstep_tracing_service_manifest()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml, k8s=True)

    aconf.load_all(fetcher.sorted())

    secret_handler = MockSecretHandler(logger, "mockery", "/tmp/ambassador/snapshots", "v1")
    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir

    econf = EnvoyConfig.generate(ir, "V3")

    bootstrap_config, ads_config, _ = econf.split_config()
    assert "tracing" in bootstrap_config
    assert bootstrap_config["tracing"] == {
        "http": {
            "name": "envoy.lightstep",
            "typed_config": {
                "@type": "type.googleapis.com/envoy.config.trace.v3.LightstepConfig",
                "access_token_file": "/lightstep-credentials/access-token",
                "collector_cluster": "cluster_tracing_lightstep_80_ambassador",
                "propagation_modes": ["ENVOY", "TRACE_CONTEXT"],
            },
        }
    }

    ads_config.pop("@type", None)
    assert_valid_envoy_config(ads_config)
    assert_valid_envoy_config(bootstrap_config)


@pytest.mark.compilertest
def test_tracing_config_v2():
    aconf = Config()

    yaml = module_and_mapping_manifests(None, []) + "\n" + lightstep_tracing_service_manifest()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml, k8s=True)

    aconf.load_all(fetcher.sorted())

    secret_handler = MockSecretHandler(logger, "mockery", "/tmp/ambassador/snapshots", "v1")
    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir

    econf = EnvoyConfig.generate(ir, "V2")

    bootstrap_config, ads_config, _ = econf.split_config()
    assert "tracing" in bootstrap_config
    assert bootstrap_config["tracing"] == {
        "http": {
            "name": "envoy.lightstep",
            "typed_config": {
                "@type": "type.googleapis.com/envoy.config.trace.v2.LightstepConfig",
                "access_token_file": "/lightstep-credentials/access-token",
                "collector_cluster": "cluster_tracing_lightstep_80_ambassador",
                "propagation_modes": ["ENVOY", "TRACE_CONTEXT"],
            },
        }
    }

    ads_config.pop("@type", None)
    assert_valid_envoy_config(ads_config, v2=True)
    assert_valid_envoy_config(bootstrap_config, v2=True)


@pytest.mark.compilertest
def test_tracing_zipkin_defaults_v3_config():

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

    econf = _get_envoy_config(yaml, version="V3")

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


def test_tracing_zipkin_defaults_v2_config():

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

    econf = _get_envoy_config(yaml, version="V2")

    bootstrap_config, _, _ = econf.split_config()
    assert "tracing" in bootstrap_config

    assert bootstrap_config["tracing"] == {
        "http": {
            "name": "envoy.zipkin",
            "typed_config": {
                "@type": "type.googleapis.com/envoy.config.trace.v2.ZipkinConfig",
                "collector_endpoint": "/api/v2/spans",
                "collector_endpoint_version": "HTTP_JSON",
                "trace_id_128bit": True,
                "collector_cluster": "cluster_tracing_zipkin_test_9411_default",
            },
        }
    }


@pytest.mark.compilertest
def test_tracing_cluster_fields_v2_config():

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

    econf = _get_envoy_config(yaml, version="V2")

    bootstrap_config, _, _ = econf.split_config()
    assert "tracing" in bootstrap_config

    cluster_name = "cluster_tracing_zipkin_test_9411_default"
    assert bootstrap_config["tracing"] == {
        "http": {
            "name": "envoy.zipkin",
            "typed_config": {
                "@type": "type.googleapis.com/envoy.config.trace.v2.ZipkinConfig",
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
def test_tracing_cluster_fields_v3_config():

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

    econf = _get_envoy_config(yaml, version="V3")

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
