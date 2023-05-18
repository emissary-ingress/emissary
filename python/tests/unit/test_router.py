import pytest

from tests.utils import (
    econf_compile,
    econf_foreach_hcm,
    module_and_mapping_manifests,
    zipkin_tracing_service_manifest,
)


def _test_router(yaml, expectations={}, edgestack=False):
    econf = econf_compile(yaml)

    def check(typed_config):
        http_filters = typed_config["http_filters"]

        if edgestack:
            expected_filter_names = [
                "envoy.filters.http.cors",
                "envoy.filters.http.golang",
                "envoy.filters.http.router",
            ]
        else:
            expected_filter_names = ["envoy.filters.http.cors", "envoy.filters.http.router"]

        assert [f["name"] for f in http_filters] == expected_filter_names

        http_route_filter = http_filters[-1]

        # If we expect nothing, then the typed config should be missing entirely.
        if len(expectations) == 0:
            assert "typed_config" not in http_route_filter
            return

        assert "typed_config" in http_route_filter
        typed_config = http_route_filter["typed_config"]
        assert (
            typed_config["@type"]
            == "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
        )
        for key, expected in expectations.items():
            print("checking key %s" % key)
            assert key in typed_config
            assert typed_config[key] == expected

    econf_foreach_hcm(econf, check)


@pytest.mark.compilertest
def test_suppress_envoy_headers(edgestack):
    # If we do not set the config, it should not appear.
    yaml = module_and_mapping_manifests(None, [])
    _test_router(yaml, expectations={}, edgestack=edgestack)

    # If we set the config to false, it should not appear.
    yaml = module_and_mapping_manifests(["suppress_envoy_headers: false"], [])
    _test_router(yaml, expectations={}, edgestack=edgestack)

    # If we set the config to true, it should appear.
    yaml = module_and_mapping_manifests(["suppress_envoy_headers: true"], [])
    _test_router(yaml, expectations={"suppress_envoy_headers": True}, edgestack=edgestack)


@pytest.mark.compilertest
def test_tracing_service(edgestack):
    # If we have a tracing service, we should see start_child_span
    yaml = module_and_mapping_manifests(None, []) + "\n" + zipkin_tracing_service_manifest()
    _test_router(yaml, expectations={"start_child_span": True}, edgestack=edgestack)


@pytest.mark.compilertest
def test_tracing_service_and_suppress_envoy_headers(edgestack):
    # If we set both suppress_envoy_headers and include a TracingService,
    # we should see both suppress_envoy_headers and the default start_child_span
    # value (True).
    yaml = (
        module_and_mapping_manifests(["suppress_envoy_headers: true"], [])
        + "\n"
        + zipkin_tracing_service_manifest()
    )
    _test_router(
        yaml,
        expectations={"start_child_span": True, "suppress_envoy_headers": True},
        edgestack=edgestack,
    )
