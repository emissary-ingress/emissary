from dataclasses import dataclass
from typing import Any, Dict, Optional
from unittest.mock import patch

import pytest

from tests.utils import edgestack, get_envoy_config, skip_edgestack

MAPPING = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: test
  namespace: ambassador
spec:
  prefix: /test/
  service: test
  hostname: '*'
"""

AUTH_SERVICE = """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name:  myauthservice
  namespace: default
spec:
  auth_service: someservice
  proto: grpc
  protocol_version: "v3"
"""


@dataclass(frozen=True)
class HTTPFilterResult:
    listener_name: Optional[str]
    filter_chain_name: Optional[str]
    http_filter: Dict[str, Any]
    prev_http_filter_name: Optional[str] = None
    next_http_filter_name: Optional[str] = None


def _get_go_filters(config: Dict[str, Any]) -> list[HTTPFilterResult]:
    found_filters: list[HTTPFilterResult] = []

    listener: Dict[str, Any]
    for listener in config["static_resources"]["listeners"]:
        listener_name = listener.get("name")

        filter_chain: Dict[str, Any]
        for filter_chain in listener["filter_chains"]:
            filter_chain_name = filter_chain.get("name")

            filter: Dict[str, Any]
            for filter in filter_chain["filters"]:
                http_filters = filter["typed_config"]["http_filters"]
                for i, http_filter in enumerate(filter["typed_config"]["http_filters"]):
                    if http_filter["name"] == "envoy.filters.http.golang":
                        prev_http_filter_name: Optional[str] = None
                        if i > 0:
                            prev_http_filter_name = http_filters[i - 1]["name"]

                        next_http_filter_name: Optional[str] = None
                        if i < len(http_filters) - 1:
                            next_http_filter_name = http_filters[i + 1]["name"]

                        found_filter = HTTPFilterResult(
                            listener_name=listener_name,
                            filter_chain_name=filter_chain_name,
                            http_filter=http_filter,
                            prev_http_filter_name=prev_http_filter_name,
                            next_http_filter_name=next_http_filter_name,
                        )
                        found_filters.append(found_filter)
    return found_filters


@pytest.mark.compilertest
@edgestack()
def test_gofilter_injected():
    econf = get_envoy_config(MAPPING)
    filters = _get_go_filters(econf.as_dict())
    # Two listeners - ports 8080 and 8443 - each with an HTTP and HTTPS filterchain
    assert len(filters) == 4

    errors = econf.ir.aconf.errors
    assert "ir.go_filter" not in errors

    for filter in filters:
        assert "envoy.filters.http.cors" == filter.prev_http_filter_name


@pytest.mark.compilertest
@edgestack()
def test_go_filter_injected_before_auth_service():
    econf = get_envoy_config(f"{MAPPING}\n{AUTH_SERVICE}")
    filters = _get_go_filters(econf.as_dict())
    # Two listeners - ports 8080 and 8443 - each with an HTTP and HTTPS filterchain
    assert len(filters) == 4

    errors = econf.ir.aconf.errors
    assert "ir.go_filter" not in errors

    for filter in filters:
        assert "envoy.filters.http.cors" == filter.prev_http_filter_name
        assert "envoy.filters.http.ext_authz" == filter.next_http_filter_name


@pytest.mark.compilertest
@edgestack()
def test_gofilter_missing_object_file(go_library):
    go_library.return_value = False

    econf = get_envoy_config(MAPPING)
    filters = _get_go_filters(econf.as_dict())

    assert len(filters) == 0

    errors = econf.ir.aconf.errors
    assert "ir.go_filter" in errors
    assert (
        errors["ir.go_filter"][0]["error"]
        == "/ambassador/go_filter.so not found, disabling Go filter..."
    )


@pytest.mark.compilertest
@skip_edgestack()
def test_gofilter_not_injected():
    econf = get_envoy_config(MAPPING)
    filters = _get_go_filters(econf.as_dict())

    assert len(filters) == 0

    errors = econf.ir.aconf.errors
    assert "ir.go_filter" not in errors
