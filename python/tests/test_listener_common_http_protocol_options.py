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

def manifests(module_confs):
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
    return yaml

def require_no_errors(ir: IR):
    assert ir.aconf.errors == {}

def _secret_handler():
    source_root = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-source")
    cache_dir = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-cache")
    return NullSecretHandler(logger, source_root.name, cache_dir.name, "fake")

def _test_listener_common_http_protocol_options(yaml, expectations={}):
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
    http_con_mngr_conf = econf['static_resources']['listeners'][0]['filter_chains'][0]['filters'][0]['typed_config']
    if expectations:
        assert 'common_http_protocol_options' in http_con_mngr_conf
    else:
        assert 'common_http_protocol_options' not in http_con_mngr_conf
    for key, expected in expectations.items():
        assert key in http_con_mngr_conf['common_http_protocol_options']
        assert http_con_mngr_conf['common_http_protocol_options'][key] == expected

def test_headers_with_underscores_action_unset():
    yaml = manifests(None)
    _test_listener_common_http_protocol_options(yaml, expectations={})

def test_headers_with_underscores_action_reject():
    yaml = manifests(["headers_with_underscores_action: REJECT_REQUEST"])
    _test_listener_common_http_protocol_options(yaml, expectations={'headers_with_underscores_action': 'REJECT_REQUEST'})

def test_listener_idle_timeout_ms():
    yaml = manifests(["listener_idle_timeout_ms: 150000"])
    _test_listener_common_http_protocol_options(yaml, expectations={'idle_timeout': '150.000s'})

def test_all_listener_common_http_protocol_options():
    yaml = manifests(["headers_with_underscores_action: DROP_HEADER", "listener_idle_timeout_ms: 4005"])
    _test_listener_common_http_protocol_options(yaml, expectations={
        'headers_with_underscores_action': 'DROP_HEADER',
        'idle_timeout': '4.005s'
    })