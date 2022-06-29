import json

import pytest

from tests.utils import compile_with_cachecheck


@pytest.mark.compilertest
def test_mapping_host_star_error():
    test_yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: bad-mapping
  namespace: default
spec:
  host: "*"
  prefix: /star/
  service: star
"""

    r = compile_with_cachecheck(test_yaml, errors_ok=True)

    ir = r["ir"]

    # print(json.dumps(ir.aconf.errors, sort_keys=True, indent=4))

    errors = ir.aconf.errors["bad-mapping.default.1"]
    assert len(errors) == 1, f"Expected 1 error but got {len(errors)}"

    assert errors[0]["ok"] == False
    assert errors[0]["error"] == "host exact-match * contains *, which cannot match anything."

    for g in ir.groups.values():
        assert g.prefix != "/star/"

    # print(json.dumps(ir.as_dict(), sort_keys=True, indent=4))


@pytest.mark.compilertest
def test_mapping_host_authority_star_error():
    test_yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: bad-mapping
  namespace: default
spec:
  headers:
    ":authority": "*"
  prefix: /star/
  service: star
"""

    r = compile_with_cachecheck(test_yaml, errors_ok=True)

    ir = r["ir"]

    # print(json.dumps(ir.aconf.errors, sort_keys=True, indent=4))

    errors = ir.aconf.errors["bad-mapping.default.1"]
    assert len(errors) == 1, f"Expected 1 error but got {len(errors)}"

    assert errors[0]["ok"] == False
    assert (
        errors[0]["error"] == ":authority exact-match '*' contains *, which cannot match anything."
    )

    for g in ir.groups.values():
        assert g.prefix != "/star/"

    # print(json.dumps(ir.as_dict(), sort_keys=True, indent=4))


@pytest.mark.compilertest
def test_mapping_host_ok():
    test_yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: good-host-mapping
  namespace: default
spec:
  host: foo.example.com
  prefix: /wanted_group/
  service: star
"""

    r = compile_with_cachecheck(test_yaml, errors_ok=True)

    ir = r["ir"]

    errors = ir.aconf.errors
    assert len(errors) == 0, "Expected no errors but got %s" % (
        json.dumps(errors, sort_keys=True, indent=4)
    )

    found = 0

    for g in ir.groups.values():
        if g.prefix == "/wanted_group/":
            assert g.host == "foo.example.com"
            found += 1

    assert found == 1, "Expected 1 /wanted_group/ prefix, got %d" % found

    # print(json.dumps(ir.as_dict(), sort_keys=True, indent=4))


@pytest.mark.compilertest
def test_mapping_host_authority_ok():
    test_yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: good-host-mapping
  namespace: default
spec:
  headers:
    ":authority": foo.example.com
  prefix: /wanted_group/
  service: star
"""

    r = compile_with_cachecheck(test_yaml, errors_ok=True)

    ir = r["ir"]

    errors = ir.aconf.errors
    assert len(errors) == 0, "Expected no errors but got %s" % (
        json.dumps(errors, sort_keys=True, indent=4)
    )

    found = 0

    for g in ir.groups.values():
        if g.prefix == "/wanted_group/":
            assert g.host == "foo.example.com"
            found += 1

    assert found == 1, "Expected 1 /wanted_group/ prefix, got %d" % found

    # print(json.dumps(ir.as_dict(), sort_keys=True, indent=4))


@pytest.mark.compilertest
def test_mapping_host_authority_and_host():
    test_yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: good-host-mapping
  namespace: default
spec:
  headers:
    ":authority": bar.example.com
  host: foo.example.com
  prefix: /wanted_group/
  service: star
"""

    r = compile_with_cachecheck(test_yaml, errors_ok=True)

    ir = r["ir"]

    errors = ir.aconf.errors
    assert len(errors) == 0, "Expected no errors but got %s" % (
        json.dumps(errors, sort_keys=True, indent=4)
    )

    found = 0

    for g in ir.groups.values():
        if g.prefix == "/wanted_group/":
            assert g.host == "foo.example.com"
            found += 1

    assert found == 1, "Expected 1 /wanted_group/ prefix, got %d" % found

    # print(json.dumps(ir.as_dict(), sort_keys=True, indent=4))


@pytest.mark.compilertest
def test_mapping_hostname_ok():
    test_yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: good-hostname-mapping
  namespace: default
spec:
  hostname: "*.example.com"
  prefix: /wanted_group/
  service: star
"""

    r = compile_with_cachecheck(test_yaml, errors_ok=True)

    ir = r["ir"]

    errors = ir.aconf.errors
    assert len(errors) == 0, "Expected no errors but got %s" % (
        json.dumps(errors, sort_keys=True, indent=4)
    )

    found = 0

    for g in ir.groups.values():
        if g.prefix == "/wanted_group/":
            assert g.host == "*.example.com"
            found += 1

    assert found == 1, "Expected 1 /wanted_group/ prefix, got %d" % found

    # print(json.dumps(ir.as_dict(), sort_keys=True, indent=4))


@pytest.mark.compilertest
def test_mapping_hostname_and_host():
    test_yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: hostname-and-host-mapping
  namespace: default
spec:
  host: foo.example.com
  hostname: "*.example.com"
  prefix: /wanted_group/
  service: star
"""

    r = compile_with_cachecheck(test_yaml, errors_ok=True)

    ir = r["ir"]

    errors = ir.aconf.errors
    assert len(errors) == 0, "Expected no errors but got %s" % (
        json.dumps(errors, sort_keys=True, indent=4)
    )

    found = 0

    for g in ir.groups.values():
        if g.prefix == "/wanted_group/":
            assert g.host == "*.example.com"
            found += 1

    assert found == 1, "Expected 1 /wanted_group/ prefix, got %d" % found

    # print(json.dumps(ir.as_dict(), sort_keys=True, indent=4))


@pytest.mark.compilertest
def test_mapping_hostname_and_authority():
    test_yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: hostname-and-host-mapping
  namespace: default
spec:
  headers:
    ":authority": foo.example.com
  hostname: "*.example.com"
  prefix: /wanted_group/
  service: star
"""

    r = compile_with_cachecheck(test_yaml, errors_ok=True)

    ir = r["ir"]

    errors = ir.aconf.errors
    assert len(errors) == 0, "Expected no errors but got %s" % (
        json.dumps(errors, sort_keys=True, indent=4)
    )

    found = 0

    for g in ir.groups.values():
        if g.prefix == "/wanted_group/":
            assert g.host == "*.example.com"
            found += 1

    assert found == 1, "Expected 1 /wanted_group/ prefix, got %d" % found

    # print(json.dumps(ir.as_dict(), sort_keys=True, indent=4))


@pytest.mark.compilertest
def test_mapping_hostname_and_host_and_authority():
    test_yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: hostname-and-host-mapping
  namespace: default
spec:
  headers:
    ":authority": bar.example.com
  host: foo.example.com
  hostname: "*.example.com"
  prefix: /wanted_group/
  service: star
"""

    r = compile_with_cachecheck(test_yaml, errors_ok=True)

    ir = r["ir"]

    errors = ir.aconf.errors
    assert len(errors) == 0, "Expected no errors but got %s" % (
        json.dumps(errors, sort_keys=True, indent=4)
    )

    found = 0

    for g in ir.groups.values():
        if g.prefix == "/wanted_group/":
            assert g.host == "*.example.com"
            found += 1

    assert found == 1, "Expected 1 /wanted_group/ prefix, got %d" % found

    # print(json.dumps(ir.as_dict(), sort_keys=True, indent=4))
