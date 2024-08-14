import importlib
import logging
import os

import pytest

logger = logging.getLogger("ambassador")

import ambassador.envoy.v3.v3ready  # noqa: E402
from ambassador import IR, Config, EnvoyConfig  # noqa: E402
from ambassador.utils import NullSecretHandler  # noqa: E402


def _get_envoy_config() -> EnvoyConfig:
    """Helper function for getting an envoy config with correctly initialized ready listener"""
    aconf = Config()

    secret_handler = NullSecretHandler(logger, None, None, "0")
    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)
    assert ir

    # This is required in order to reload module-level variables that depend on os.environ.
    # Otherwise the module variables can have stale values.
    importlib.reload(ambassador.envoy.v3.v3ready)
    return EnvoyConfig.generate(ir)


def _validate_ready_listener_config(
    econf: EnvoyConfig, expectedPort: int, readyLogEnabled: bool
):
    """
    Helper function that fails if the envoy config does not match an expected ready listener config
    """
    conf = econf.as_dict()
    readyListener = conf["static_resources"]["listeners"][0]
    assert readyListener["address"]["socket_address"]["address"] == "127.0.0.1"
    assert readyListener["address"]["socket_address"]["port_value"] == expectedPort
    assert (
        readyListener["name"] == "ambassador-listener-ready-127.0.0.1-%d" % expectedPort
    )

    filterTypedConfig = readyListener["filter_chains"][0]["filters"][0]["typed_config"]
    assert (
        filterTypedConfig["http_filters"][0]["typed_config"]["headers"][0][
            "exact_match"
        ]
        == "/ready"
    )
    if readyLogEnabled:
        assert "access_log" in filterTypedConfig
    else:
        assert "access_log" not in filterTypedConfig


@pytest.mark.compilertest
def test_ready_listener_custom():
    """
    Test to ensure that the ready endpoint is configurable using the AMBASSADOR_READY_PORT and
    AMBASSADOR_READY_LOG environment variables.
    """
    os.environ["AMBASSADOR_READY_PORT"] = str(8010)
    os.environ["AMBASSADOR_READY_LOG"] = str(True)
    econf = _get_envoy_config()
    _validate_ready_listener_config(econf, 8010, True)


@pytest.mark.compilertest
def test_ready_listener_default():
    """Test to ensure that the ready endpoint is configured with sensible defaults."""
    # Unset environment variables that could have been set in an earlier test
    os.environ.pop("AMBASSADOR_READY_PORT", None)
    os.environ.pop("AMBASSADOR_READY_LOG", None)
    econf = _get_envoy_config()
    _validate_ready_listener_config(econf, 8006, False)


@pytest.mark.compilertest
def test_ready_listener_high_port():
    """
    Test to ensure that the ready endpoint falls back to the default port if a port above 32767 is
    used.
    """
    os.environ["AMBASSADOR_READY_PORT"] = str(32767)
    os.environ.pop("AMBASSADOR_READY_LOG", None)
    econf = _get_envoy_config()
    _validate_ready_listener_config(econf, 8006, False)


@pytest.mark.compilertest
def test_ready_listener_invalid_logs():
    """
    Test to ensure that invalid values passed to AMBASSADOR_READY_LOG are handled gracefully.
    """
    os.environ.pop("AMBASSADOR_READY_PORT")
    os.environ["AMBASSADOR_READY_LOG"] = "0.6666666666666666"
    econf = _get_envoy_config()
    _validate_ready_listener_config(econf, 8006, False)


@pytest.mark.compilertest
def test_ready_listener_invalid_port():
    """
    Test to ensure that an error is raised when an invalid port string is passed to
    AMBASSADOR_READY_PORT.
    """
    os.environ["AMBASSADOR_READY_PORT"] = "420/1337.abcd"
    os.environ.pop("AMBASSADOR_READY_LOG", None)
    with pytest.raises(ValueError):
        econf = _get_envoy_config()
        _validate_ready_listener_config(econf, 0, False)


@pytest.mark.compilertest
def test_ready_listener_zero_port():
    """
    Test to ensure that the ready endpoint falls back to the default port if a zero port is used.
    """
    os.environ["AMBASSADOR_READY_PORT"] = str(0)
    os.environ.pop("AMBASSADOR_READY_LOG", None)
    econf = _get_envoy_config()
    _validate_ready_listener_config(econf, 8006, False)
