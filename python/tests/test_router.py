from typing import List, Tuple
import json

import logging

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador import Cache, IR
from ambassador.compile import Compile

def tracing_service_manifest():
    return """
---
apiVersion: getambassador.io/v2
kind: TracingService
metadata:
  name: tracing
  namespace: ambassador
spec:
  service: zipkin:9411
  driver: zipkin
  config: {}
"""

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

def _test_router(yaml, expectations={}):
    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)
    r1j = json.dumps(r1['v2'].as_dict()['static_resources']['listeners'], sort_keys=True, indent=2)
    r2j = json.dumps(r2['v2'].as_dict()['static_resources']['listeners'], sort_keys=True, indent=2)
    assert r1j == r2j

    require_no_errors(r1["ir"])
    require_no_errors(r2["ir"])
    econf = r1['v2'].as_dict()

    found_router_config = False
    for listener in econf['static_resources']['listeners']:
        # There's only one filter chain...
        filter_chains = listener['filter_chains']
        assert len(filter_chains) == 1

        # ...and one filter on that chain.
        filters = filter_chains[0]['filters']
        assert len(filters) == 1

        # The http connection manager is the only filter on the chain. Get the http filters.
        hcm = filters[0]
        assert hcm['name'] == 'envoy.filters.network.http_connection_manager'
        http_filters = hcm['typed_config']['http_filters']
        assert len(http_filters) == 2

        # Find the typed router config, and run our uexpecations over that.
        for http_filter in http_filters:
            if http_filter['name'] != 'envoy.filters.http.router':
                continue
            found_router_config = True

            # If we expect nothing, then the typed config should be missing entirely.
            if len(expectations) == 0:
                assert 'typed_config' not in http_filter
                break

            assert 'typed_config' in http_filter
            typed_config = http_filter['typed_config']
            assert typed_config['@type'] == 'type.googleapis.com/envoy.config.filter.http.router.v2.Router'
            for key, expected in expectations.items():
                assert key in typed_config
                assert typed_config[key] == expected
            break
    assert found_router_config == True

def test_suppress_envoy_headers():
    # If we do not set the config, it should not appear.
    yaml = manifests(None, [])
    _test_router(yaml, expectations={})

    # If we set the config to false, it should not appear.
    yaml = manifests(['suppress_envoy_headers: false'], [])
    _test_router(yaml, expectations={})

    # If we set the config to true, it should appear.
    yaml = manifests(['suppress_envoy_headers: true'], [])
    _test_router(yaml, expectations={'suppress_envoy_headers': True})

def test_tracing_service():
    # If we have a tracing service, we should see start_child_span
    yaml = manifests(None, []) + "\n" + tracing_service_manifest()
    _test_router(yaml, expectations={'start_child_span': True})

def test_tracing_service_and_suppress_envoy_headers():
    # If we set both suppress_envoy_headers and include a TracingService,
    # we should see both suppress_envoy_headers and the default start_child_span
    # value (True).
    yaml = manifests(['suppress_envoy_headers: true'], []) + "\n" + tracing_service_manifest()
    _test_router(yaml, expectations={'start_child_span': True, 'suppress_envoy_headers': True})