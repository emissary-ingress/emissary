import pytest

from tests.utils import (
    SUPPORTED_ENVOY_VERSIONS,
    econf_compile,
    econf_foreach_cluster,
    module_and_mapping_manifests,
)


# Tests if `setting` exists within the cluster config and has `expected` as the value for that setting
# Use `exists` to test if you expect a setting to not exist
def _test_cluster_setting(yaml, setting, expected, exists=True, envoy_version="V2"):
    econf = econf_compile(yaml, envoy_version=envoy_version)

    def check(cluster):
        if exists:
            assert setting in cluster
            assert cluster[setting] == expected
        else:
            assert setting not in cluster

    econf_foreach_cluster(econf, check)


# Tests a setting in a cluster that has it's own fields. Example: common_http_protocol_options has multiple subfields
def _test_cluster_subfields(yaml, setting, expectations={}, exists=True, envoy_version="V2"):
    econf = econf_compile(yaml, envoy_version=envoy_version)

    def check(cluster):
        if exists:
            assert setting in cluster
        else:
            assert setting not in cluster
        for key, expected in expectations.items():
            print("Checking key: {} for the {} setting in Envoy cluster".format(key, setting))
            assert key in cluster[setting]
            assert cluster[setting][key] == expected

    econf_foreach_cluster(econf, check)


# Test dns_type setting in Mapping
@pytest.mark.compilertest
def test_logical_dns_type():
    yaml = module_and_mapping_manifests(None, ["dns_type: logical_dns"])
    for v in SUPPORTED_ENVOY_VERSIONS:
        # The dns type is listed as just "type"
        _test_cluster_setting(
            yaml, setting="type", expected="LOGICAL_DNS", exists=True, envoy_version=v
        )


@pytest.mark.compilertest
def test_strict_dns_type():
    # Make sure we can configure strict dns as well even though it's the default
    yaml = module_and_mapping_manifests(None, ["dns_type: strict_dns"])
    for v in SUPPORTED_ENVOY_VERSIONS:
        # The dns type is listed as just "type"
        _test_cluster_setting(
            yaml, setting="type", expected="STRICT_DNS", exists=True, envoy_version=v
        )


@pytest.mark.compilertest
def test_dns_type_wrong():
    # Ensure we fallback to strict_dns as the setting when an invalid string is passed
    # This is preferable to invalid config and an error is logged
    yaml = module_and_mapping_manifests(None, ["dns_type: something_new"])
    for v in SUPPORTED_ENVOY_VERSIONS:
        # The dns type is listed as just "type"
        _test_cluster_setting(
            yaml, setting="type", expected="STRICT_DNS", exists=True, envoy_version=v
        )


@pytest.mark.compilertest
def test_logical_dns_type_endpoints():
    # Ensure we use endpoint discovery instead of this value when using the endpoint resolver.
    yaml = module_and_mapping_manifests(None, ["dns_type: logical_dns", "resolver: endpoint"])
    for v in SUPPORTED_ENVOY_VERSIONS:
        # The dns type is listed as just "type"
        _test_cluster_setting(yaml, setting="type", expected="EDS", exists=True, envoy_version=v)


@pytest.mark.compilertest
def test_dns_ttl_module():
    # Test configuring the respect_dns_ttl generates an Envoy config
    yaml = module_and_mapping_manifests(None, ["respect_dns_ttl: true"])
    for v in SUPPORTED_ENVOY_VERSIONS:
        # The dns type is listed as just "type"
        _test_cluster_setting(
            yaml, setting="respect_dns_ttl", expected=True, exists=True, envoy_version=v
        )


@pytest.mark.compilertest
def test_dns_ttl_mapping():
    # Test dns_ttl is not configured when not applied in the Mapping
    yaml = module_and_mapping_manifests(None, None)
    for v in SUPPORTED_ENVOY_VERSIONS:
        # The dns type is listed as just "type"
        _test_cluster_setting(
            yaml, setting="respect_dns_ttl", expected=False, exists=False, envoy_version=v
        )
