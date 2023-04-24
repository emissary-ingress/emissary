import os

from tests.utils import assert_valid_envoy_config, econf_compile, module_and_mapping_manifests


def _test_bootstrap(yaml, expectations={}):
    # Compile an envoy config
    econf = econf_compile(yaml)

    # Get just the bootstrap config...
    bootstrap = econf["bootstrap"]

    # ...and make sure that Envoy thinks it is valid (it doesn't like the @type field)
    bootstrap.pop("@type", None)
    assert_valid_envoy_config(bootstrap)

    for key, expected in expectations.items():
        if expected is None:
            assert key not in bootstrap
        else:
            pass

            assert key in bootstrap
            assert bootstrap[key] == expected


def _test_dd_entity_id(val, expected):
    # Setup by setting dd / statsd vars
    os.environ["STATSD_ENABLED"] = "true"
    os.environ["STATSD_HOST"] = "0.0.0.0"
    os.environ["DOGSTATSD"] = "true"
    if val:
        os.environ["DD_ENTITY_ID"] = val

    # Run the bootstrap test. We don't need any special yaml
    # since all of this behavior is driven by env vars.
    yaml = module_and_mapping_manifests(None, [])
    _test_bootstrap(yaml, expectations={"stats_config": expected})

    # Teardown by removing dd / statsd vars
    del os.environ["STATSD_ENABLED"]
    del os.environ["STATSD_HOST"]
    del os.environ["DOGSTATSD"]
    if val:
        del os.environ["DD_ENTITY_ID"]


def test_dd_entity_id_missing():
    # If we do not set the env var, then stats config should be missing.
    _test_dd_entity_id(None, None)


def test_dd_entity_id_empty():
    # If we set the env var to the empty string, the stats config should be missing.
    _test_dd_entity_id("", None)


def test_dd_entity_id_set():
    # If we set the env var, then it should show up the config.
    _test_dd_entity_id(
        "my.cool.1234.entity-id",
        {
            "stats_tags": [
                {"tag_name": "dd.internal.entity_id", "fixed_value": "my.cool.1234.entity-id"}
            ]
        },
    )


def test_dd_entity_id_set_typical():
    # If we set the env var to a typical pod UID, then it should show up int he config.
    _test_dd_entity_id(
        "1fb8f8d8-00b3-44ef-bc8b-3659e4a3c2bd",
        {
            "stats_tags": [
                {
                    "tag_name": "dd.internal.entity_id",
                    "fixed_value": "1fb8f8d8-00b3-44ef-bc8b-3659e4a3c2bd",
                }
            ]
        },
    )
