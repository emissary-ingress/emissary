from typing import Any, Dict, List

import pytest

from tests.utils import compile_with_cachecheck, default_listener_manifests, econf_foreach_hcm


def _mapping_yaml(name: str, prefix: str, extra: str) -> str:
    return f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {name}
  namespace: default
spec:
  hostname: "*"
  prefix: {prefix}
  service: httpbin
{extra}
"""


def _routes_for_prefix(econf: Dict[str, Any], prefix: str) -> List[Dict[str, Any]]:
    routes: List[Dict[str, Any]] = []

    def check(typed_config):
        for vhost in typed_config["route_config"]["virtual_hosts"]:
            for route in vhost["routes"]:
                if route["match"].get("prefix") == prefix:
                    routes.append(route)

    econf_foreach_hcm(econf, check)
    return routes


def _single_route(econf: Dict[str, Any], prefix: str) -> Dict[str, Any]:
    routes = _routes_for_prefix(econf, prefix)
    assert routes, f"no route for {prefix}"

    # The compiler tests get an insecure and a secure chain, which should
    # agree about the headers to add.
    for route in routes[1:]:
        assert route.get("request_headers_to_add") == routes[0].get("request_headers_to_add")
        assert route.get("response_headers_to_add") == routes[0].get("response_headers_to_add")

    return routes[0]


@pytest.mark.compilertest
def test_added_headers_append_action():
    yaml = default_listener_manifests() + _mapping_yaml(
        "added-headers",
        "/httpbin/",
        """
  add_request_headers:
    x-shorthand: shorthand
    x-append-unset:
      value: unset
    x-append-false:
      value: replaced
      append: false
    x-append-if-exists:
      value: appended
      append_action: APPEND_IF_EXISTS_OR_ADD
    x-add-if-absent:
      value: absent
      append_action: ADD_IF_ABSENT
  add_response_headers:
    x-overwrite-or-add:
      value: overwritten
      append_action: OVERWRITE_IF_EXISTS_OR_ADD
    x-overwrite-if-exists:
      value: overwritten-if-exists
      append_action: OVERWRITE_IF_EXISTS
    x-both:
      value: both
      append: true
      append_action: ADD_IF_ABSENT
""",
    )

    r = compile_with_cachecheck(yaml)
    route = _single_route(r["xds"].as_dict(), "/httpbin/")

    assert route["request_headers_to_add"] == [
        {"header": {"key": "x-shorthand", "value": "shorthand"}, "append": True},
        {"header": {"key": "x-append-unset", "value": "unset"}, "append": True},
        {"header": {"key": "x-append-false", "value": "replaced"}, "append": False},
        {
            "header": {"key": "x-append-if-exists", "value": "appended"},
            "append_action": "APPEND_IF_EXISTS_OR_ADD",
        },
        {
            "header": {"key": "x-add-if-absent", "value": "absent"},
            "append_action": "ADD_IF_ABSENT",
        },
    ]

    # Envoy rejects a header that sets both append and a non-default
    # append_action, so append_action must take the place of append.
    assert route["response_headers_to_add"] == [
        {
            "header": {"key": "x-overwrite-or-add", "value": "overwritten"},
            "append_action": "OVERWRITE_IF_EXISTS_OR_ADD",
        },
        {
            "header": {"key": "x-overwrite-if-exists", "value": "overwritten-if-exists"},
            "append_action": "OVERWRITE_IF_EXISTS",
        },
        {
            "header": {"key": "x-both", "value": "both"},
            "append_action": "ADD_IF_ABSENT",
        },
    ]


@pytest.mark.compilertest
@pytest.mark.parametrize("field", ["add_request_headers", "add_response_headers"])
def test_added_headers_invalid_append_action(field: str):
    yaml = (
        default_listener_manifests()
        + _mapping_yaml(
            "bad-append-action",
            "/bad/",
            f"""
  {field}:
    x-bad:
      value: bad
      append_action: REPLACE
""",
        )
        + _mapping_yaml(
            "good-append-action",
            "/good/",
            f"""
  {field}:
    x-good:
      value: good
      append_action: ADD_IF_ABSENT
""",
        )
    )

    r = compile_with_cachecheck(yaml, errors_ok=True)

    errors = r["ir"].aconf.errors
    assert list(errors.keys()) == ["bad-append-action.default.1"]
    assert [error["error"] for error in errors["bad-append-action.default.1"]] == [
        f"{field}: header x-bad: invalid append_action 'REPLACE', must be one of "
        + "APPEND_IF_EXISTS_OR_ADD, ADD_IF_ABSENT, OVERWRITE_IF_EXISTS_OR_ADD, "
        + "OVERWRITE_IF_EXISTS, invalidating mapping"
    ]

    econf = r["xds"].as_dict()

    # The bad Mapping must be dropped rather than handed to Envoy, which would
    # reject the whole configuration, and the good one must be unaffected.
    assert _routes_for_prefix(econf, "/bad/") == []

    envoy_field = field.replace("add_", "", 1) + "_to_add"
    good_route = _single_route(econf, "/good/")
    assert good_route[envoy_field] == [
        {"header": {"key": "x-good", "value": "good"}, "append_action": "ADD_IF_ABSENT"},
    ]


@pytest.mark.compilertest
@pytest.mark.parametrize("field", ["add_request_headers", "add_response_headers"])
def test_added_headers_missing_value(field: str):
    yaml = default_listener_manifests() + _mapping_yaml(
        "missing-value",
        "/httpbin/",
        f"""
  {field}:
    x-no-value-append:
      append: true
    x-no-value-action:
      append_action: ADD_IF_ABSENT
    x-kept:
      value: kept
      append_action: ADD_IF_ABSENT
""",
    )

    r = compile_with_cachecheck(yaml)

    # A header with no value is skipped, whichever form it takes; the rest stay.
    envoy_field = field.replace("add_", "", 1) + "_to_add"
    route = _single_route(r["xds"].as_dict(), "/httpbin/")
    assert route[envoy_field] == [
        {"header": {"key": "x-kept", "value": "kept"}, "append_action": "ADD_IF_ABSENT"},
    ]
