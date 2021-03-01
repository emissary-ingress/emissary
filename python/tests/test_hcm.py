from typing import List, Tuple

import json
import logging
import pytest
import tempfile

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador import Cache, IR
from ambassador.compile import Compile
from ambassador.utils import NullSecretHandler

def manifests(module_confs, mapping_confs):
    yaml = """
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:"""
    if module_confs:
        for module_conf in module_confs:
            yaml = yaml + """
    {}
""".format(module_conf)
    else:
        yaml = yaml + " {}\n"

    yaml = yaml + """
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: ambassador
  namespace: default
spec:
  prefix: /httpbin/
  service: httpbin"""
    if mapping_confs:
        for mapping_conf in mapping_confs:
            yaml = yaml + """
  {}""".format(mapping_conf)
    return yaml

def require_no_errors(ir: IR):
    assert ir.aconf.errors == {}

def _secret_handler():
    source_root = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-source")
    cache_dir = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-cache")
    return NullSecretHandler(logger, source_root.name, cache_dir.name, "fake")

def _test_hcm(yaml, expectations={}):
    cache = Cache(logger)
    secret_handler = _secret_handler()
    r1 = Compile(logger, yaml, k8s=True, secret_handler=secret_handler)
    r2 = Compile(logger, yaml, k8s=True, secret_handler=secret_handler, cache=cache)
    r1j = json.dumps(r1['v2'].as_dict(), sort_keys=True, indent=2)
    r2j = json.dumps(r2['v2'].as_dict(), sort_keys=True, indent=2)
    assert r1j == r2j

    require_no_errors(r1["ir"])
    require_no_errors(r2["ir"])
    econf = r1['v2'].as_dict()

    # Get the https listener. The http listener just serves redirects.
    found_hcm = False
    for listener in econf['static_resources']['listeners']:
        # There's only one filter chain...
        filter_chains = listener['filter_chains']
        assert len(filter_chains) == 1

        # ...and one filter on that chain.
        filters = filter_chains[0]['filters']
        assert len(filters) == 1

        # The http connection manager is the only filter on the chain.
        # from the one and only vhost.
        hcm = filters[0]
        assert hcm['name'] == 'envoy.filters.network.http_connection_manager'
        typed_config = hcm['typed_config']
        assert typed_config['@type'] == 'type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager'

        found_hcm = True
        for key, expected in expectations.items():
            if expected is None:
                assert key not in typed_config
            else:
                assert key in typed_config
                assert typed_config[key] == expected
        break

    # One of the listeners should have the hcm we were looking for
    assert found_hcm

def test_strip_matching_host_port_missing():
    # If we do not set the config, it should be missing (noted in this test as None).
    yaml = manifests(None, [])
    _test_hcm(yaml, expectations={'strip_matching_host_port': None})

def test_strip_matching_host_port_module_false():
    # If we set the config to false, it should be missing (noted in this test as None).
    yaml = manifests(['strip_matching_host_port: false'], [])
    _test_hcm(yaml, expectations={'strip_matching_host_port': None})

def test_strip_matching_host_port_module_true():
    # If we set the config to true, it should show up as true.
    yaml = manifests(['strip_matching_host_port: true'], [])
    _test_hcm(yaml, expectations={'strip_matching_host_port': True})
