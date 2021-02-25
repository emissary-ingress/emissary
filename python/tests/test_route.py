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

def _test_route(yaml, expectations={}):
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
    found_route = False
    for listener in econf['static_resources']['listeners']:
        # There's only one filter chain...
        filter_chains = listener['filter_chains']
        assert len(filter_chains) == 1

        # ...and one filter on that chain.
        filters = filter_chains[0]['filters']
        assert len(filters) == 1

        # The http connection manager is the only filter on the chain. Get its route config
        # from the one and only vhost.
        hcm = filters[0]
        assert hcm['name'] == 'envoy.filters.network.http_connection_manager'
        vhosts = hcm['typed_config']['route_config']['virtual_hosts']
        assert len(vhosts) == 1

        # Finally, find the httpbin route. Run our expecations over that.
        routes = vhosts[0]['routes']
        for r in routes:
            # Keep going until we find a real route
            if 'route' not in r:
                continue

            # Keep going until we find a prefix match for /httpbin/
            match = r['match']
            if 'prefix' not in match or match['prefix'] != '/httpbin/':
                continue

            found_route = True
            assert 'route' in r
            route = r['route']
            for key, expected in expectations.items():
                assert key in route
                assert route[key] == expected
            break

    # One of the listeners should have the route we were looking for
    assert found_route

def test_timeout_ms():
    # If we do not set the config, we should get the default 3000ms.
    yaml = manifests(None, [])
    _test_route(yaml, expectations={'timeout':'3.000s'})

def test_timeout_ms_module():
    # If we set a default on the Module, it should override the usual default of 3000ms.
    yaml = manifests(["cluster_request_timeout_ms: 4000"], [])
    _test_route(yaml, expectations={'timeout':'4.000s'})

def test_timeout_ms_mapping():
    # If we set a default on the Module, it should override the usual default of 3000ms.
    yaml = manifests(None, ["timeout_ms: 1234"])
    _test_route(yaml, expectations={'timeout':'1.234s'})

def test_timeout_ms_both():
    # If we set a default on the Module, it should override the usual default of 3000ms.
    yaml = manifests(["cluster_request_timeout_ms: 9000"], ["timeout_ms: 5001"])
    _test_route(yaml, expectations={'timeout':'5.001s'})
