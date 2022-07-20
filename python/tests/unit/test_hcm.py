import pytest

from tests.utils import econf_compile, econf_foreach_hcm, module_and_mapping_manifests


def _test_hcm(yaml, expectations={}):
    # Compile an envoy config
    econf = econf_compile(yaml)

    # Make sure expectations pass for each HCM in the compiled config
    def check(typed_config):
        for key, expected in expectations.items():
            if expected is None:
                assert key not in typed_config
            else:
                assert key in typed_config
                assert typed_config[key] == expected
        return True

    econf_foreach_hcm(econf, check)


@pytest.mark.compilertest
def test_strip_matching_host_port_missing():
    # If we do not set the config, it should be missing (noted in this test as None).
    yaml = module_and_mapping_manifests(None, [])
    _test_hcm(yaml, expectations={"strip_matching_host_port": None})


@pytest.mark.compilertest
def test_strip_matching_host_port_module_false():
    # If we set the config to false, it should be missing (noted in this test as None).
    yaml = module_and_mapping_manifests(["strip_matching_host_port: false"], [])
    _test_hcm(yaml, expectations={"strip_matching_host_port": None})


@pytest.mark.compilertest
def test_strip_matching_host_port_module_true():
    # If we set the config to true, it should show up as true.
    yaml = module_and_mapping_manifests(["strip_matching_host_port: true"], [])
    _test_hcm(yaml, expectations={"strip_matching_host_port": True})


@pytest.mark.compilertest
def test_merge_slashes_missing():
    # If we do not set the config, it should be missing (noted in this test as None).
    yaml = module_and_mapping_manifests(None, [])
    _test_hcm(yaml, expectations={"merge_slashes": None})


@pytest.mark.compilertest
def test_merge_slashes_module_false():
    # If we set the config to false, it should be missing (noted in this test as None).
    yaml = module_and_mapping_manifests(["merge_slashes: false"], [])
    _test_hcm(yaml, expectations={"merge_slashes": None})


@pytest.mark.compilertest
def test_merge_slashes_module_true():
    # If we set the config to true, it should show up as true.
    yaml = module_and_mapping_manifests(["merge_slashes: true"], [])
    _test_hcm(yaml, expectations={"merge_slashes": True})


@pytest.mark.compilertest
def test_reject_requests_with_escaped_slashes_missing():
    # If we set the config to false, the action should be missing.
    yaml = module_and_mapping_manifests(None, [])
    _test_hcm(yaml, expectations={"path_with_escaped_slashes_action": None})


@pytest.mark.compilertest
def test_reject_requests_with_escaped_slashes_false():
    # If we set the config to false, the action should be missing.
    yaml = module_and_mapping_manifests(["reject_requests_with_escaped_slashes: false"], [])
    _test_hcm(yaml, expectations={"path_with_escaped_slashes_action": None})


@pytest.mark.compilertest
def test_reject_requests_with_escaped_slashes_true():
    # If we set the config to true, the action should show up as "REJECT_REQUEST".
    yaml = module_and_mapping_manifests(["reject_requests_with_escaped_slashes: true"], [])
    _test_hcm(yaml, expectations={"path_with_escaped_slashes_action": "REJECT_REQUEST"})


@pytest.mark.compilertest
def test_preserve_external_request_id_missing():
    # If we do not set the config, it should be false
    yaml = module_and_mapping_manifests(None, [])
    _test_hcm(yaml, expectations={"preserve_external_request_id": False})


@pytest.mark.compilertest
def test_preserve_external_request_id_module_false():
    # If we set the config to false, it should be false
    yaml = module_and_mapping_manifests(["preserve_external_request_id: false"], [])
    _test_hcm(yaml, expectations={"preserve_external_request_id": False})


@pytest.mark.compilertest
def test_preserve_external_request_id_module_true():
    # If we set the config to true, it should show up as true.
    yaml = module_and_mapping_manifests(["preserve_external_request_id: true"], [])
    _test_hcm(yaml, expectations={"preserve_external_request_id": True})
