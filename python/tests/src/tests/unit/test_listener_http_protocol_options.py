import pytest

from tests.utils import econf_compile, econf_foreach_hcm, module_and_mapping_manifests


def _test_listener_http_protocol_options(yaml, expectations={}):
    econf = econf_compile(yaml)

    # Make sure expectations pass for each HCM in the compiled config
    def check(typed_config):
        for key, expected in expectations.items():
            if expected is None:
                assert key not in typed_config["http_protocol_options"]
            else:
                assert key in typed_config["http_protocol_options"]
                assert typed_config["http_protocol_options"][key] == expected
        return True

    econf_foreach_hcm(econf, check)


@pytest.mark.compilertest
def test_emptiness():
    yaml = module_and_mapping_manifests([], [])
    _test_listener_http_protocol_options(yaml, expectations={})


@pytest.mark.compilertest
def test_proper_case_false():
    yaml = module_and_mapping_manifests(["proper_case: false"], [])
    _test_listener_http_protocol_options(yaml, expectations={})


@pytest.mark.compilertest
def test_proper_case_true():
    yaml = module_and_mapping_manifests(["proper_case: true"], [])
    _test_listener_http_protocol_options(
        yaml, expectations={"header_key_format": {"proper_case_words": {}}}
    )


@pytest.mark.compilertest
def test_proper_case_and_enable_http_10():
    yaml = module_and_mapping_manifests(["proper_case: true", "enable_http10: true"], [])
    _test_listener_http_protocol_options(
        yaml, expectations={"accept_http_10": True, "header_key_format": {"proper_case_words": {}}}
    )


@pytest.mark.compilertest
def test_allow_chunked_length_false():
    yaml = module_and_mapping_manifests(["allow_chunked_length: false"], [])
    _test_listener_http_protocol_options(yaml, expectations={"allow_chunked_length": False})


@pytest.mark.compilertest
def test_allow_chunked_length_true():
    yaml = module_and_mapping_manifests(["allow_chunked_length: true"], [])
    _test_listener_http_protocol_options(yaml, expectations={"allow_chunked_length": True})
