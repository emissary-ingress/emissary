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

def _test_common_http_protocol_options(yaml, expectations={}):
    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)
    r1j = json.dumps(r1['v2'].as_dict()['static_resources']['clusters'], sort_keys=True, indent=2)
    r2j = json.dumps(r2['v2'].as_dict()['static_resources']['clusters'], sort_keys=True, indent=2)
    assert r1j == r2j

    require_no_errors(r1["ir"])
    require_no_errors(r2["ir"])
    econf = r1['v2'].as_dict()
    found_cluster = False
    for cluster in econf['static_resources']['clusters']:
        if cluster['name'] == 'cluster_httpbin_default':
            found_cluster = True
            if expectations:
                assert 'common_http_protocol_options' in cluster
            else:
                assert 'common_http_protocol_options' not in cluster
            for key, expected in expectations.items():
                assert key in cluster['common_http_protocol_options']
                assert cluster['common_http_protocol_options'][key] == expected
    assert found_cluster

def test_cluster_max_connection_lifetime_ms_missing():
    # If we do not set the config, it should not appear in the Envoy conf.
    yaml = manifests(None, [])
    _test_common_http_protocol_options(yaml, expectations={})

def test_cluster_max_connection_lifetime_ms_module_only():
    # If we only set the config on the Module, it should show up.
    yaml = manifests(["cluster_max_connection_lifetime_ms: 2005"], [])
    _test_common_http_protocol_options(yaml, expectations={'max_connection_duration':'2.005s'})

def test_cluster_max_connection_lifetime_ms_mapping_only():
    # If we only set the config on the Mapping, it should show up.
    yaml = manifests(None, ["cluster_max_connection_lifetime_ms: 2005"])
    _test_common_http_protocol_options(yaml, expectations={'max_connection_duration':'2.005s'})

def test_cluster_max_connection_lifetime_ms_mapping_override():
    # If we set the config on the Module and Mapping, the Mapping value wins.
    yaml = manifests(["cluster_max_connection_lifetime_ms: 2005"], ["cluster_max_connection_lifetime_ms: 17005"])
    _test_common_http_protocol_options(yaml, expectations={'max_connection_duration':'17.005s'})

def test_cluster_idle_timeout_ms_missing():
    # If we do not set the config, it should not appear in the Envoy conf.
    yaml = manifests(None, [])
    _test_common_http_protocol_options(yaml, expectations={})

def test_cluster_idle_timeout_ms_module_only():
    # If we only set the config on the Module, it should show up.
    yaml = manifests(["cluster_idle_timeout_ms: 4005"], [])
    _test_common_http_protocol_options(yaml, expectations={'idle_timeout':'4.005s'})

def test_cluster_idle_timeout_ms_mapping_only():
    # If we only set the config on the Mapping, it should show up.
    yaml = manifests(None, ["cluster_idle_timeout_ms: 4005"])
    _test_common_http_protocol_options(yaml, expectations={'idle_timeout':'4.005s'})

def test_cluster_idle_timeout_ms_mapping_override():
    # If we set the config on the Module and Mapping, the Mapping value wins.
    yaml = manifests(["cluster_idle_timeout_ms: 4005"], ["cluster_idle_timeout_ms: 19105"])
    _test_common_http_protocol_options(yaml, expectations={'idle_timeout':'19.105s'})

def test_both_module():
    # If we set both configs on the Module, both should show up.
    yaml = manifests(["cluster_idle_timeout_ms: 4005", "cluster_max_connection_lifetime_ms: 2005"], None)
    _test_common_http_protocol_options(yaml, expectations={
        'max_connection_duration': '2.005s',
        'idle_timeout': '4.005s'
    })

def test_both_mapping():
    # If we set both configs on the Mapping, both should show up.
    yaml = manifests(None, ["cluster_idle_timeout_ms: 4005", "cluster_max_connection_lifetime_ms: 2005"])
    _test_common_http_protocol_options(yaml, expectations={
        'max_connection_duration': '2.005s',
        'idle_timeout': '4.005s'
    })

def test_both_one_module_one_mapping():
    # If we set both configs, one on a Module, one on a Mapping, both should show up.
    yaml = manifests(["cluster_idle_timeout_ms: 4005"], ["cluster_max_connection_lifetime_ms: 2005"])
    _test_common_http_protocol_options(yaml, expectations={
        'max_connection_duration': '2.005s',
        'idle_timeout': '4.005s'
    })