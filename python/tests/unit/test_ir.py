import logging
from dataclasses import dataclass
from typing import Optional
from unittest.mock import MagicMock
from unittest.mock import patch
import pytest

from tests.utils import (
    Compile,
    default_http3_listener_manifest,
    default_listener_manifests,
    default_tcp_listener_manifest,
    default_udp_listener_manifest,
    logger,
    generate_istio_cert_delta,
    generate_istio_saved_secret
)

from ambassador.ir import IR
from ambassador.config import Config
from test_acme_privatekey_secrets import MemorySecretHandler


def http3_quick_start_manifests():
    return default_listener_manifests() + default_http3_listener_manifest()


class TestIR:
    def test_http3_enabled(self, caplog):
        caplog.set_level(logging.WARNING, logger="ambassador")

        @dataclass
        class TestCase:
            name: str
            inputYaml: str
            expected: dict[str, bool]
            expectedLog: Optional[str] = None

        testcases = [
            TestCase(
                "quick-start",
                default_listener_manifests(),
                {"tcp-0.0.0.0-8080": False, "tcp-0.0.0.0-8443": False},
            ),
            TestCase(
                "quick-start-with_http3",
                http3_quick_start_manifests(),
                {"tcp-0.0.0.0-8080": False, "tcp-0.0.0.0-8443": True, "udp-0.0.0.0-8443": True},
            ),
            TestCase("http3-only", default_http3_listener_manifest(), {"udp-0.0.0.0-8443": True}),
            TestCase("raw-udp", default_udp_listener_manifest(), {}),
            TestCase("raw-tcp", default_tcp_listener_manifest(), {"tcp-0.0.0.0-8443": False}),
        ]

        for case in testcases:
            compiled_ir = Compile(logger, case.inputYaml, k8s=True)
            result_ir = compiled_ir["ir"]

            listeners = result_ir.listeners

            assert len(case.expected.items()) == len(listeners)

            for listener_id, http3_enabled in case.expected.items():
                listener = listeners.get(listener_id, None)
                assert listener is not None
                assert listener.http3_enabled == http3_enabled

            if case.expectedLog != None:
                assert case.expectedLog in caplog.text


    @pytest.mark.parametrize("name, cache_entry, deltas, expected", [
        (
            "cached-secret-with-deltas",
            generate_istio_saved_secret(),
            [generate_istio_cert_delta()],
            {"config_type": "incremental", "reset_cache": True, "invalidate_groups_for": []}
        ),
        (
            "cached-secret-without-deltas",
            generate_istio_saved_secret(),
            None,
            {"config_type": "complete", "reset_cache": True, "invalidate_groups_for": []}
        ),
        (
            "null-cache-with-deltas",
            None,
            [generate_istio_cert_delta()],
            {"config_type": "complete", "reset_cache": True, "invalidate_groups_for": []}
        ),
        (
            "null-cache-without-deltas",
            None,
            None,
            {"config_type": "complete", "reset_cache": True, "invalidate_groups_for": []}
        )
        
        ]
    )
    def test_check_deltas(self, name, cache_entry, deltas, expected, caplog):
        caplog.set_level(logging.DEBUG, logger="ambassador")

        with patch("ambassador.cache.Cache") as cache:
            cache.return_value = [cache_entry] if cache_entry is not None else None
            fetcher = MagicMock()
            fetcher.deltas = deltas
            aconfig = Config()
            secret_handler = MemorySecretHandler(logger, "/tmp/unit-test-source-root", "/tmp/unit-test-cache-dir", "0")
            ir = IR(aconf=aconfig, file_checker=lambda path: True, secret_handler=secret_handler)
            config_type, reset_cache, invalidate_groups_for = ir.check_deltas(logger=logger, fetcher=fetcher, cache=cache)

            if (cache_entry and deltas) is not None:
                cache.dump.assert_called()
                assert config_type == expected["config_type"]
                assert reset_cache == expected["reset_cache"]
                assert invalidate_groups_for == expected["invalidate_groups_for"]
