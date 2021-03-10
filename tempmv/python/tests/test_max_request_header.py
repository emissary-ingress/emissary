import logging

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador import Config, IR, EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler


def _get_envoy_config(yaml):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir

    return EnvoyConfig.generate(ir, "V2")


def test_set_max_request_header():
    yaml = """
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
config:
  max_request_headers_kb: 96
---
apiVersion: getambassador.io/v2
kind: Mapping
name: ambassador
prefix: /test/
service: test:9999
"""
    econf = _get_envoy_config(yaml)
    expected = 96
    key_found = False

    conf = econf.as_dict()

    for listener in conf['static_resources']['listeners']:
        for filter_chain in listener['filter_chains']:
            for f in filter_chain['filters']:
                max_req_headers = f['typed_config'].get('max_request_headers_kb', None)
                assert max_req_headers is not None, \
                        f"max_request_headers_kb not found on typed_config: {f['typed_config']}"

                print(f"Found max_req_headers = {max_req_headers}")
                key_found = True
                assert expected == int(max_req_headers), \
                        "max_request_headers_kb must equal the value set on the ambassador Module"
    assert key_found, 'max_request_headers_kb must be found in the envoy config'
