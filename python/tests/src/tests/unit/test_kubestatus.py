import logging
import types

import pytest

from ambassador_diag.diagd import KubeStatusNoMappings, status_resource_name


@pytest.mark.parametrize(
    "key,namespace,expected",
    [
        ("minimal-host.emissary", "emissary", "minimal-host"),
        ("host.example.com.emissary", "emissary", "host.example.com"),
        ("bare-name", "emissary", "bare-name"),
    ],
)
def test_status_resource_name(key, namespace, expected):
    """Strips only the namespace suffix, so dotted names survive."""
    assert status_resource_name(key, namespace) == expected


class FakePool:
    """Records submit() calls instead of running kubestatus."""

    def __init__(self):
        self.calls = []

    def submit(self, fn, *args):
        self.calls.append(args)
        return types.SimpleNamespace(add_done_callback=lambda cb: None)


def _kubestatus() -> KubeStatusNoMappings:
    ks = KubeStatusNoMappings(types.SimpleNamespace(logger=logging.getLogger("ambassador")))
    ks.pool = FakePool()
    return ks


def _reconfigure(ks, name, text):
    # Same order as diagd's _load_ir: mark live, prune, then post.
    ks.mark_live("Host", name, "default")
    ks.prune()
    ks.post("Host", name, "default", text)


def test_unchanged_host_status_is_posted_once():
    """Live Hosts keep their cached status, so a repeat reconfigure skips the write."""
    ks = _kubestatus()

    _reconfigure(ks, "minimal-host", '{"state": "Ready"}')
    _reconfigure(ks, "minimal-host", '{"state": "Ready"}')
    assert len(ks.pool.calls) == 1

    _reconfigure(ks, "minimal-host", '{"state": "Error"}')
    assert len(ks.pool.calls) == 2


def test_mapping_status_still_ignored():
    """KubeStatusNoMappings must still drop Mapping updates."""
    ks = _kubestatus()

    ks.mark_live("Mapping", "m", "default")
    ks.post("Mapping", "m", "default", "{}")
    assert ks.pool.calls == []
