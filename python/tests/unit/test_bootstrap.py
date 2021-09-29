import os
from tests.utils import assert_valid_envoy_config, econf_compile, module_and_mapping_manifests

import pytest

def _test_bootstrap(yaml, v2, expectations={}):
    # Compile an envoy config
    if v2:
        econf = econf_compile(yaml, envoy_version="V2")
    else:
        econf = econf_compile(yaml)

    # Get just the bootstrap config...
    bootstrap = econf['bootstrap']

    # ...and make sure that Envoy thinks it is valid (it doesn't like the @type field)
    bootstrap.pop('@type', None)
    assert_valid_envoy_config(bootstrap, v2=v2)

    for key, expected in expectations.items():
        if expected is None:
            assert key not in bootstrap
        else:
            import json
            assert key in bootstrap
            assert bootstrap[key] == expected


def _test_dd_entity_id(val, expected, v2):
    # Setup by setting dd / statsd vars
    os.environ["STATSD_ENABLED"] = "true"
    os.environ["STATSD_HOST"] = "0.0.0.0"
    os.environ["DOGSTATSD"] = "true"
    if val:
        os.environ["DD_ENTITY_ID"] = val

    # Run the bootstrap test. We don't need any special yaml
    # since all of this behavior is driven by env vars.
    yaml = module_and_mapping_manifests(None, [])
    _test_bootstrap(yaml, v2, expectations={'stats_config': expected})

    # Teardown by removing dd / statsd vars
    del os.environ["STATSD_ENABLED"]
    del os.environ["STATSD_HOST"]
    del os.environ["DOGSTATSD"]
    if val:
        del os.environ["DD_ENTITY_ID"]

def test_dd_entity_id_missing_v3():
    # If we do not set the env var, then stats config should be missing.
    _test_dd_entity_id(None, None, False)

def test_dd_entity_id_empty_v3():
    # If we set the env var to the empty string, the stats config should be missing.
    _test_dd_entity_id("", None, False)

def test_dd_entity_id_set_v3():
    # If we set the env var, then it should show up the config.
    _test_dd_entity_id("my.cool.1234.entity-id", {
        'stats_tags': [
            {
                'tag_name':'dd.internal.entity_id',
                'fixed_value':'my.cool.1234.entity-id'
            }
        ]
    }, False)

def test_dd_entity_id_set_typical_v3():
    # If we set the env var to a typical pod UID, then it should show up int he config.
    _test_dd_entity_id("1fb8f8d8-00b3-44ef-bc8b-3659e4a3c2bd", {
        'stats_tags': [
            {
                'tag_name':'dd.internal.entity_id',
                'fixed_value':'1fb8f8d8-00b3-44ef-bc8b-3659e4a3c2bd'
            }
        ]
    }, False)


def test_dd_entity_id_missing_v2():
    # If we do not set the env var, then stats config should be missing.
    _test_dd_entity_id(None, None, True)

def test_dd_entity_id_empty_v2():
    # If we set the env var to the empty string, the stats config should be missing.
    _test_dd_entity_id("", None, True)

def test_dd_entity_id_set_v2():
    # If we set the env var, then it should show up the config.
    _test_dd_entity_id("my.cool.1234.entity-id", {
        'stats_tags': [
            {
                'tag_name':'dd.internal.entity_id',
                'fixed_value':'my.cool.1234.entity-id'
            }
        ]
    }, True)

def test_dd_entity_id_set_typical_v2():
    # If we set the env var to a typical pod UID, then it should show up int he config.
    _test_dd_entity_id("1fb8f8d8-00b3-44ef-bc8b-3659e4a3c2bd", {
        'stats_tags': [
            {
                'tag_name':'dd.internal.entity_id',
                'fixed_value':'1fb8f8d8-00b3-44ef-bc8b-3659e4a3c2bd'
            }
        ]
    }, True)
