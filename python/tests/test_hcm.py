from utils import econf_compile, econf_foreach_hcm, module_and_mapping_manifests

import pytest

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

def test_strip_matching_host_port_missing():
    # If we do not set the config, it should be missing (noted in this test as None).
    yaml = module_and_mapping_manifests(None, [])
    _test_hcm(yaml, expectations={'strip_matching_host_port': None})

def test_strip_matching_host_port_module_false():
    # If we set the config to false, it should be missing (noted in this test as None).
    yaml = module_and_mapping_manifests(['strip_matching_host_port: false'], [])
    _test_hcm(yaml, expectations={'strip_matching_host_port': None})

def test_strip_matching_host_port_module_true():
    # If we set the config to true, it should show up as true.
    yaml = module_and_mapping_manifests(['strip_matching_host_port: true'], [])
    _test_hcm(yaml, expectations={'strip_matching_host_port': True})
