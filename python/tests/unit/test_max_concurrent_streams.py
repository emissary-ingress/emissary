import logging

import pytest

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
def test_max_concurrent_requests_v3():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    max_concurrent_requests: 96
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: ambassador
  namespace: default
spec:
  hostname: "*"
  prefix: /test/
  service: test:9999
"""
    print("hey ----------- i am here")
    econf = _get_envoy_config(yaml)
    expected = 96
    key_found = False

    conf = econf.as_dict()

    for listener in conf["static_resources"]["listeners"]:
        for filter_chain in listener["filter_chains"]:
            for f in filter_chain["filters"]:
                max_concurrent_requests = f["typed_config"].get("max_concurrent_requests", None)
                assert (
                    max_concurrent_requests is not None
                ), f"max_concurrent_requests not found on typed_config: {f['typed_config']}"

                print(f"Found max_concurrent_requests = {max_concurrent_requests}")
                key_found = True
                assert expected == int(
                    max_concurrent_requests
                ), "max_concurrent_requests must equal the value set on the ambassador Module"
    assert key_found, "max_concurrent_requests must be found in the envoy config"
