from typing import List, Tuple

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

def require_no_errors(ir: IR):
    assert ir.aconf.errors == {}

def require_errors(ir: IR, errors: List[Tuple[str, str]]):
    flattened_ir_errors: List[str] = []

    for key in ir.aconf.errors.keys():
        for error in ir.aconf.errors[key]:
            flattened_ir_errors.append(f"{key}: {error['error']}")

    flattened_wanted_errors: List[str] = [
        f"{key}: {error}" for key, error in errors
    ]

    assert sorted(flattened_ir_errors) == sorted(flattened_wanted_errors)

@pytest.mark.compilertest
def test_valid_forward_client_cert_details():
    yaml = """
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    forward_client_cert_details: SANITIZE_SET
"""

    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)

    require_no_errors(r1["ir"])
    require_no_errors(r2["ir"])

@pytest.mark.compilertest
def test_invalid_forward_client_cert_details():
    yaml = """
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    forward_client_cert_details: SANITIZE_INVALID
"""

    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)

    require_errors(r1["ir"], [
        ( "ambassador.default.1", "'forward_client_cert_details' may not be set to 'SANITIZE_INVALID'; it may only be set to one of: SANITIZE, FORWARD_ONLY, APPEND_FORWARD, SANITIZE_SET, ALWAYS_FORWARD_ONLY")
    ])
    require_errors(r2["ir"], [
        ( "ambassador.default.1", "'forward_client_cert_details' may not be set to 'SANITIZE_INVALID'; it may only be set to one of: SANITIZE, FORWARD_ONLY, APPEND_FORWARD, SANITIZE_SET, ALWAYS_FORWARD_ONLY")
    ])

@pytest.mark.compilertest
def test_valid_set_current_client_cert_details():
    yaml = """
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    set_current_client_cert_details:
      subject: true
      dns: true
"""

    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)

    require_no_errors(r1["ir"])
    require_no_errors(r2["ir"])

@pytest.mark.compilertest
def test_invalid_set_current_client_cert_details_key():
    yaml = """
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    set_current_client_cert_details:
      invalid: true
"""

    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)

    logger.info("R1 IR: %s", r1["ir"].as_json())

    require_errors(r1["ir"], [
        ( "ambassador.default.1", "'set_current_client_cert_details' may not contain key 'invalid'; it may only contain keys: subject, cert, chain, dns, uri")
    ])
    require_errors(r2["ir"], [
        ( "ambassador.default.1", "'set_current_client_cert_details' may not contain key 'invalid'; it may only contain keys: subject, cert, chain, dns, uri")
    ])

@pytest.mark.compilertest
def test_invalid_set_current_client_cert_details_value():
    yaml = """
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    set_current_client_cert_details:
      subject: invalid
"""

    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)

    require_errors(r1["ir"], [
        ( "ambassador.default.1", "'set_current_client_cert_details' value for key 'subject' may only be 'true' or 'false', not 'invalid'")
    ])
    require_errors(r2["ir"], [
        ( "ambassador.default.1", "'set_current_client_cert_details' value for key 'subject' may only be 'true' or 'false', not 'invalid'")
    ])
