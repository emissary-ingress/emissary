import pytest

from tests.utils import (
    econf_compile,
    econf_foreach_hcm,
    module_and_mapping_manifests,
    zipkin_tracing_service_manifest,
)


def _test_router(yaml, expectations={}):
    econf = econf_compile(yaml)

    def check(typed_config):
        http_filters = typed_config["http_filters"]
        assert len(http_filters) == 2

        # Find the typed router config, and run our uexpecations over that.
        for http_filter in http_filters:
            if http_filter["name"] != "envoy.filters.http.router":
                continue

            # If we expect nothing, then the typed config should be missing entirely.
            if len(expectations) == 0:
                assert "typed_config" not in http_filter
                break

            assert "typed_config" in http_filter
            typed_config = http_filter["typed_config"]
            assert (
                typed_config["@type"]
                == "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
            )
            for key, expected in expectations.items():
                print("checking key %s" % key)
                assert key in typed_config
                assert typed_config[key] == expected
            break

    econf_foreach_hcm(econf, check)


@pytest.mark.compilertest
def test_suppress_envoy_headers():
    # If we do not set the config, it should not appear.
    yaml = module_and_mapping_manifests(None, [])
    _test_router(yaml, expectations={})

    # If we set the config to false, it should not appear.
    yaml = module_and_mapping_manifests(["suppress_envoy_headers: false"], [])
    _test_router(yaml, expectations={})

    # If we set the config to true, it should appear.
    yaml = module_and_mapping_manifests(["suppress_envoy_headers: true"], [])
    _test_router(yaml, expectations={"suppress_envoy_headers": True})


@pytest.mark.compilertest
def test_tracing_service():
    # If we have a tracing service, we should see start_child_span
    yaml = module_and_mapping_manifests(None, []) + "\n" + zipkin_tracing_service_manifest()
    _test_router(yaml, expectations={"start_child_span": True})


@pytest.mark.compilertest
def test_tracing_service_and_suppress_envoy_headers():
    # If we set both suppress_envoy_headers and include a TracingService,
    # we should see both suppress_envoy_headers and the default start_child_span
    # value (True).
    yaml = (
        module_and_mapping_manifests(["suppress_envoy_headers: true"], [])
        + "\n"
        + zipkin_tracing_service_manifest()
    )
    _test_router(yaml, expectations={"start_child_span": True, "suppress_envoy_headers": True})
