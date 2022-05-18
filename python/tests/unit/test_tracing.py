from typing import Optional, TYPE_CHECKING

import logging
import pytest

from tests.selfsigned import TLSCerts
from tests.utils import assert_valid_envoy_config, module_and_mapping_manifests

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador import Config, IR
from ambassador.envoy import EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import SecretHandler, SecretInfo

if TYPE_CHECKING:
    from ambassador.ir.irresource import IRResource # pragma: no cover

class MockSecretHandler(SecretHandler):
    def load_secret(self, resource: 'IRResource', secret_name: str, namespace: str) -> Optional[SecretInfo]:
            return SecretInfo('fallback-self-signed-cert', 'ambassador', "mocked-fallback-secret",
                              TLSCerts["acook"].pubcert, TLSCerts["acook"].privkey, decode_b64=False)


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

    # check if custom_tags are added
    assert econf.as_dict()['static_resources']['listeners'][0]['filter_chains'][0]['filters'][0]['typed_config']['tracing'] == {
        "custom_tags": [
            {'literal': {'value': 'avalue'}, 'tag': 'ltag'},
            {'environment': {'default_value': 'efallback', 'name': 'UNKNOWN_ENV_VAR'}, 'tag': 'etag'},
            {'request_header': {'default_value': 'hfallback', 'name': 'x-does-not-exist'}, 'tag': 'htag'},
        ]
    }

    bootstrap_config, ads_config, _ = econf.split_config()
    assert "tracing" in bootstrap_config
    assert bootstrap_config["tracing"] == {
        "http": {
            "name": "envoy.lightstep",
            "typed_config": {
                "@type": "type.googleapis.com/envoy.config.trace.v3.LightstepConfig",
                "access_token_file": "/lightstep-credentials/access-token",
                "collector_cluster": "cluster_tracing_lightstep_80_ambassador",
                "propagation_modes": ["ENVOY", "TRACE_CONTEXT"]
            }
        }
    }

    ads_config.pop('@type', None)
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
                "propagation_modes": ["ENVOY", "TRACE_CONTEXT"]
            }
        }
    }

    ads_config.pop('@type', None)
    assert_valid_envoy_config(ads_config, v2=True)
    assert_valid_envoy_config(bootstrap_config, v2=True)
