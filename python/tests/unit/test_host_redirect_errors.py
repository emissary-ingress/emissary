from typing import Any, Dict, List, Tuple

import difflib
import json
import logging
import os
import random
import sys
import yaml

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador import Cache, IR
from ambassador.utils import NullSecretHandler
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
def test_hr_good_1():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: mapping-1
    namespace: default
spec:
    hostname: "*"
    prefix: /
    service: svc1
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: mapping-2
    namespace: default
spec:
    hostname: "*"
    prefix: /
    service: svc2
"""

    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)

    logger.info("R1 IR: %s", r1["ir"].as_json())

    require_no_errors(r1["ir"])
    require_no_errors(r2["ir"])

@pytest.mark.compilertest
def test_hr_error_1():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: mapping-1
    namespace: default
spec:
    hostname: "*"
    prefix: /
    service: svc1
    host_redirect: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: mapping-2
    namespace: default
spec:
    hostname: "*"
    prefix: /
    service: svc2
    host_redirect: true
"""

    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)

    # XXX Why are these showing up tagged with "mapping-1.default.1" rather than "mapping-2.default.1"?
    require_errors(r1["ir"], [
        ( "mapping-1.default.1", "cannot accept mapping-2 as second host_redirect after mapping-1")
    ])

    require_errors(r2["ir"], [
        ( "mapping-1.default.1", "cannot accept mapping-2 as second host_redirect after mapping-1")
    ])

@pytest.mark.compilertest
def test_hr_error_2():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: mapping-1
    namespace: default
spec:
    hostname: "*"
    prefix: /
    service: svc1
    host_redirect: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: mapping-2
    namespace: default
spec:
    hostname: "*"
    prefix: /
    service: svc2
"""

    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)

    # XXX Why are these showing up as "-global-"?
    require_errors(r1["ir"], [
        ( "-global-", "cannot accept mapping-2 without host_redirect after mapping-1 with host_redirect")
    ])

    require_errors(r2["ir"], [
        ( "-global-", "cannot accept mapping-2 without host_redirect after mapping-1 with host_redirect")
    ])

@pytest.mark.compilertest
def test_hr_error_3():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: mapping-1
    namespace: default
spec:
    hostname: "*"
    prefix: /
    service: svc1
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: mapping-2
    namespace: default
spec:
    hostname: "*"
    prefix: /
    service: svc2
    host_redirect: true
"""

    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)

    # XXX Why are these showing up tagged with "mapping-1.default.1" rather than "mapping-2.default.1"?
    require_errors(r1["ir"], [
        ( "mapping-1.default.1", "cannot accept mapping-2 with host_redirect after mappings without host_redirect (eg mapping-1)")
    ])

    require_errors(r2["ir"], [
        ( "mapping-1.default.1", "cannot accept mapping-2 with host_redirect after mappings without host_redirect (eg mapping-1)")
    ])

@pytest.mark.compilertest
def test_hr_error_4():
    yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: mapping-1
    namespace: default
spec:
    hostname: "*"
    prefix: /svc1
    service: svc1
    host_redirect: true
    path_redirect: /path/
    prefix_redirect: /prefix/
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: mapping-2
    namespace: default
spec:
    hostname: "*"
    prefix: /svc2
    service: svc2
    host_redirect: true
    path_redirect: /path/
    regex_redirect:
      pattern: /regex/
      substitution: /substitution/
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
    name: mapping-3
    namespace: default
spec:
    hostname: "*"
    prefix: /svc3
    service: svc3
    host_redirect: true
    prefix_redirect: /prefix/
    regex_redirect:
      pattern: /regex/
      substitution: /substitution/
"""

    cache = Cache(logger)
    r1 = Compile(logger, yaml, k8s=True)
    r2 = Compile(logger, yaml, k8s=True, cache=cache)

    for r in [ r1, r2 ]:
        require_errors(r["ir"], [
            ( "mapping-1.default.1", "Cannot specify both path_redirect and prefix_redirect. Using path_redirect and ignoring prefix_redirect."),
            ( "mapping-2.default.1", "Cannot specify both path_redirect and regex_redirect. Using path_redirect and ignoring regex_redirect."),
            ( "mapping-3.default.1", "Cannot specify both prefix_redirect and regex_redirect. Using prefix_redirect and ignoring regex_redirect.")
        ])
