import pytest

from tests.utils import (
    econf_compile,
    econf_foreach_cluster,
    module_and_mapping_manifests,
)


def _test_common_http_protocol_options(yaml, expectations={}):
    econf = econf_compile(yaml)

    def check(cluster):
        if expectations:
            assert "common_http_protocol_options" in cluster
        else:
            assert "common_http_protocol_options" not in cluster
        for key, expected in expectations.items():
            print("checking key %s" % key)
            assert key in cluster["common_http_protocol_options"]
            assert cluster["common_http_protocol_options"][key] == expected

    econf_foreach_cluster(econf, check)


@pytest.mark.compilertest
def test_cluster_max_connection_lifetime_ms_missing():
    # If we do not set the config, it should not appear in the Envoy conf.
    yaml = module_and_mapping_manifests(None, [])
    _test_common_http_protocol_options(yaml, expectations={})


@pytest.mark.compilertest
def test_cluster_max_connection_lifetime_ms_module_only():
    # If we only set the config on the Module, it should show up.
    yaml = module_and_mapping_manifests(
        ["cluster_max_connection_lifetime_ms: 2005"], []
    )
    _test_common_http_protocol_options(
        yaml, expectations={"max_connection_duration": "2.005s"}
    )


@pytest.mark.compilertest
def test_cluster_max_connection_lifetime_ms_mapping_only():
    # If we only set the config on the Mapping, it should show up.
    yaml = module_and_mapping_manifests(
        None, ["cluster_max_connection_lifetime_ms: 2005"]
    )
    _test_common_http_protocol_options(
        yaml, expectations={"max_connection_duration": "2.005s"}
    )


@pytest.mark.compilertest
def test_cluster_max_connection_lifetime_ms_mapping_override():
    # If we set the config on the Module and Mapping, the Mapping value wins.
    yaml = module_and_mapping_manifests(
        ["cluster_max_connection_lifetime_ms: 2005"],
        ["cluster_max_connection_lifetime_ms: 17005"],
    )
    _test_common_http_protocol_options(
        yaml, expectations={"max_connection_duration": "17.005s"}
    )


@pytest.mark.compilertest
def test_cluster_idle_timeout_ms_missing():
    # If we do not set the config, it should not appear in the Envoy conf.
    yaml = module_and_mapping_manifests(None, [])
    _test_common_http_protocol_options(yaml, expectations={})


@pytest.mark.compilertest
def test_cluster_idle_timeout_ms_module_only():
    # If we only set the config on the Module, it should show up.
    yaml = module_and_mapping_manifests(["cluster_idle_timeout_ms: 4005"], [])
    _test_common_http_protocol_options(yaml, expectations={"idle_timeout": "4.005s"})


@pytest.mark.compilertest
def test_cluster_idle_timeout_ms_mapping_only():
    # If we only set the config on the Mapping, it should show up.
    yaml = module_and_mapping_manifests(None, ["cluster_idle_timeout_ms: 4005"])
    _test_common_http_protocol_options(yaml, expectations={"idle_timeout": "4.005s"})


@pytest.mark.compilertest
def test_cluster_idle_timeout_ms_mapping_override():
    # If we set the config on the Module and Mapping, the Mapping value wins.
    yaml = module_and_mapping_manifests(
        ["cluster_idle_timeout_ms: 4005"], ["cluster_idle_timeout_ms: 19105"]
    )
    _test_common_http_protocol_options(yaml, expectations={"idle_timeout": "19.105s"})


@pytest.mark.compilertest
def test_both_module():
    # If we set both configs on the Module, both should show up.
    yaml = module_and_mapping_manifests(
        ["cluster_idle_timeout_ms: 4005", "cluster_max_connection_lifetime_ms: 2005"],
        None,
    )
    _test_common_http_protocol_options(
        yaml,
        expectations={"max_connection_duration": "2.005s", "idle_timeout": "4.005s"},
    )


@pytest.mark.compilertest
def test_both_mapping():
    # If we set both configs on the Mapping, both should show up.
    yaml = module_and_mapping_manifests(
        None,
        ["cluster_idle_timeout_ms: 4005", "cluster_max_connection_lifetime_ms: 2005"],
    )
    _test_common_http_protocol_options(
        yaml,
        expectations={"max_connection_duration": "2.005s", "idle_timeout": "4.005s"},
    )


@pytest.mark.compilertest
def test_both_one_module_one_mapping():
    # If we set both configs, one on a Module, one on a Mapping, both should show up.
    yaml = module_and_mapping_manifests(
        ["cluster_idle_timeout_ms: 4005"], ["cluster_max_connection_lifetime_ms: 2005"]
    )
    _test_common_http_protocol_options(
        yaml,
        expectations={"max_connection_duration": "2.005s", "idle_timeout": "4.005s"},
    )
