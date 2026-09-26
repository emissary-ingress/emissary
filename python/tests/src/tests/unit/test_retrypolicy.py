import pytest

from tests.utils import (
    compile_with_cachecheck,
    default_listener_manifests,
    econf_compile,
    econf_foreach_hcm,
)


def _mapping_yaml(retry_policy=None):
    yaml = (
        default_listener_manifests()
        + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: httpbin
  namespace: default
spec:
  hostname: "*"
  prefix: /httpbin/
  service: httpbin
"""
    )

    if retry_policy:
        yaml += "  retry_policy:\n" + retry_policy

    return yaml


def _module_yaml(retry_policy):
    return (
        _mapping_yaml()
        + """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    retry_policy:
"""
        + retry_policy
    )


def _route_retry_policy(yaml):
    """Compile yaml and return the retry_policy of the /httpbin/ route."""

    econf = econf_compile(yaml)
    found = []

    def check(typed_config):
        for vhost in typed_config["route_config"]["virtual_hosts"]:
            for route in vhost["routes"]:
                if "route" not in route:
                    continue

                if route["match"].get("prefix", None) != "/httpbin/":
                    continue

                found.append(route["route"].get("retry_policy", None))

    econf_foreach_hcm(econf, check)

    assert found, "no /httpbin/ route was generated"

    return found[0]


def _errors_string(result):
    ir = result["ir"]
    return " ".join(str(e) for errors in ir.aconf.errors.values() for e in errors)


@pytest.mark.compilertest
def test_no_retry_back_off():
    # Without retry_back_off, the route's retry_policy is unchanged, and Envoy applies its own
    # back-off defaults.
    retry_policy = _route_retry_policy(
        _mapping_yaml(
            """
    retry_on: "5xx"
    num_retries: 4
"""
        )
    )

    assert retry_policy == {"retry_on": "5xx", "num_retries": 4}


@pytest.mark.compilertest
def test_retry_back_off_mapping():
    retry_policy = _route_retry_policy(
        _mapping_yaml(
            """
    retry_on: "5xx"
    num_retries: 4
    retry_back_off:
      base_interval: "0.5s"
      max_interval: "5s"
"""
        )
    )

    assert retry_policy == {
        "retry_on": "5xx",
        "num_retries": 4,
        "retry_back_off": {"base_interval": "0.5s", "max_interval": "5s"},
    }


@pytest.mark.compilertest
def test_retry_back_off_without_max_interval():
    # max_interval is optional: Envoy defaults it to ten times base_interval.
    retry_policy = _route_retry_policy(
        _mapping_yaml(
            """
    retry_on: "5xx"
    retry_back_off:
      base_interval: "0.025s"
"""
        )
    )

    assert retry_policy == {
        "retry_on": "5xx",
        "retry_back_off": {"base_interval": "0.025s"},
    }


@pytest.mark.compilertest
def test_retry_back_off_module():
    # A retry_policy on the Module applies to Mappings that don't set their own.
    retry_policy = _route_retry_policy(
        _module_yaml(
            """
      retry_on: "gateway-error"
      num_retries: 2
      retry_back_off:
        base_interval: "1s"
        max_interval: "10s"
"""
        )
    )

    assert retry_policy == {
        "retry_on": "gateway-error",
        "num_retries": 2,
        "retry_back_off": {"base_interval": "1s", "max_interval": "10s"},
    }


@pytest.mark.compilertest
def test_retry_back_off_requires_base_interval():
    result = compile_with_cachecheck(
        _mapping_yaml(
            """
    retry_on: "5xx"
    retry_back_off:
      max_interval: "10s"
"""
        ),
        errors_ok=True,
    )

    assert "retry_back_off requires base_interval" in _errors_string(result)


@pytest.mark.compilertest
def test_retry_back_off_rejects_other_units():
    # Envoy wants a number of seconds, so "25ms" must not be handed to it.
    result = compile_with_cachecheck(
        _mapping_yaml(
            """
    retry_on: "5xx"
    retry_back_off:
      base_interval: "25ms"
"""
        ),
        errors_ok=True,
    )

    assert (
        'retry_back_off base_interval "25ms" is not a number of seconds'
        in _errors_string(result)
    )


@pytest.mark.compilertest
def test_retry_back_off_rejects_zero_base_interval():
    result = compile_with_cachecheck(
        _mapping_yaml(
            """
    retry_on: "5xx"
    retry_back_off:
      base_interval: "0s"
"""
        ),
        errors_ok=True,
    )

    assert (
        'retry_back_off base_interval "0s" must be greater than zero'
        in _errors_string(result)
    )


@pytest.mark.compilertest
def test_retry_back_off_rejects_max_interval_below_base_interval():
    result = compile_with_cachecheck(
        _mapping_yaml(
            """
    retry_on: "5xx"
    retry_back_off:
      base_interval: "2s"
      max_interval: "1s"
"""
        ),
        errors_ok=True,
    )

    assert (
        'retry_back_off max_interval "1s" must not be less than base_interval "2s"'
        in _errors_string(result)
    )
