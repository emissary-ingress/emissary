import pytest

from tests.utils import compile_with_cachecheck, default_listener_manifests


def _base_yaml():
    return (
        default_listener_manifests()
        + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: ambassador
  namespace: default
spec:
  hostname: "*"
  prefix: /httpbin/
  service: httpbin
"""
    )


def _errors_string(result):
    ir = result["ir"]
    return " ".join(str(e) for errors in ir.aconf.errors.values() for e in errors)


@pytest.mark.compilertest
def test_module_error_response_overrides_logs_error():
    """error_response_overrides on the Module should log an error in Emissary 4."""
    yaml = (
        _base_yaml()
        + """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    error_response_overrides:
    - on_status_code: 404
      body:
        text_format: "not found"
"""
    )
    result = compile_with_cachecheck(yaml, errors_ok=True)
    all_errors = _errors_string(result)
    assert "error_response_overrides" in all_errors, f"expected error not found; errors: {all_errors!r}"
    assert "not supported" in all_errors


@pytest.mark.compilertest
def test_mapping_error_response_overrides_logs_error():
    """error_response_overrides on a Mapping should log an error in Emissary 4."""
    yaml = (
        default_listener_manifests()
        + """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config: {}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: ambassador
  namespace: default
spec:
  hostname: "*"
  prefix: /httpbin/
  service: httpbin
  error_response_overrides:
  - on_status_code: 503
    body:
      text_format: "unavailable"
"""
    )
    result = compile_with_cachecheck(yaml, errors_ok=True)
    all_errors = _errors_string(result)
    assert "error_response_overrides" in all_errors, f"expected error not found; errors: {all_errors!r}"
    assert "not supported" in all_errors

