import pytest

from tests.utils import econf_compile, econf_foreach_hcm, module_and_mapping_manifests


def _test_listener_common_http_protocol_options(yaml, expectations={}):
    # Compile an envoy config
    econf = econf_compile(yaml)

    # Make sure expectations pass for each HCM in the compiled config
    def check(typed_config):
        for key, expected in expectations.items():
            if expected is None:
                assert key not in typed_config["common_http_protocol_options"]
            else:
                assert key in typed_config["common_http_protocol_options"]
                assert typed_config["common_http_protocol_options"][key] == expected
        return True

    econf_foreach_hcm(econf, check)


@pytest.mark.compilertest
def test_headers_with_underscores_action_unset():
    yaml = module_and_mapping_manifests(None, [])
    _test_listener_common_http_protocol_options(yaml, expectations={})


@pytest.mark.compilertest
def test_headers_with_underscores_action_reject():
    yaml = module_and_mapping_manifests(["headers_with_underscores_action: REJECT_REQUEST"], [])
    _test_listener_common_http_protocol_options(
        yaml, expectations={"headers_with_underscores_action": "REJECT_REQUEST"}
    )


@pytest.mark.compilertest
def test_listener_idle_timeout_ms():
    yaml = module_and_mapping_manifests(["listener_idle_timeout_ms: 150000"], [])
    _test_listener_common_http_protocol_options(yaml, expectations={"idle_timeout": "150.000s"})


@pytest.mark.compilertest
def test_all_listener_common_http_protocol_options():
    yaml = module_and_mapping_manifests(
        ["headers_with_underscores_action: DROP_HEADER", "listener_idle_timeout_ms: 4005"], []
    )
    _test_listener_common_http_protocol_options(
        yaml,
        expectations={"headers_with_underscores_action": "DROP_HEADER", "idle_timeout": "4.005s"},
    )
