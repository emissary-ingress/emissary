import logging
import os
import sys
import threading
import time
from typing import Optional

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)

logger = logging.getLogger("ambassador")

from ambassador.diagnostics import EnvoyStats, EnvoyStatsMgr


class EnvoyStatsMocker:
    def __init__(self) -> None:
        current_test = os.environ.get("PYTEST_CURRENT_TEST")

        assert current_test is not None, "PYTEST_CURRENT_TEST is not set??"

        self.test_dir = os.path.join(
            os.path.dirname(current_test.split("::")[0]), "test_envoy_stats_data"
        )

        self.log_idx = 0
        self.stats_idx = 0

    def fetch_log_levels(self, level: Optional[str]) -> Optional[str]:
        self.log_idx += 1
        path = os.path.join(self.test_dir, f"logging-{self.log_idx}.txt")

        try:
            return open(path, "r").read()
        except OSError:
            return None

    def fetch_envoy_stats(self) -> Optional[str]:
        self.stats_idx += 1
        path = os.path.join(self.test_dir, f"stats-{self.stats_idx}.txt")

        try:
            return open(path, "r").read()
        except OSError:
            return None

    def slow_fetch_stats(self) -> Optional[str]:
        time.sleep(5)
        return self.fetch_envoy_stats()


def test_levels():
    mocker = EnvoyStatsMocker()

    esm = EnvoyStatsMgr(
        logger, fetch_log_levels=mocker.fetch_log_levels, fetch_envoy_stats=mocker.fetch_envoy_stats
    )

    esm.update()
    assert esm.loginfo == {"all": "error"}

    # This one may be a bit more fragile than we'd like.
    esm.update()
    assert esm.loginfo == {
        "error": [
            "admin",
            "aws",
            "assert",
            "backtrace",
            "cache_filter",
            "client",
            "config",
            "connection",
            "conn_handler",
            "decompression",
            "envoy_bug",
            "ext_authz",
            "rocketmq",
            "file",
            "filter",
            "forward_proxy",
            "grpc",
            "hc",
            "health_checker",
            "http",
            "http2",
            "hystrix",
            "init",
            "io",
            "jwt",
            "kafka",
            "main",
            "misc",
            "mongo",
            "quic",
            "quic_stream",
            "pool",
            "rbac",
            "redis",
            "router",
            "runtime",
            "stats",
            "secret",
            "tap",
            "testing",
            "thrift",
            "tracing",
            "upstream",
            "udp",
            "wasm",
        ],
        "info": ["dubbo"],
        "warning": ["lua"],
    }


def test_stats():
    mocker = EnvoyStatsMocker()

    esm = EnvoyStatsMgr(
        logger, fetch_log_levels=mocker.fetch_log_levels, fetch_envoy_stats=mocker.fetch_envoy_stats
    )

    esm.update()
    stats = esm.get_stats()

    assert stats.created is not None
    assert stats.last_attempt is not None
    assert stats.last_update is not None

    assert stats.last_attempt >= stats.created
    assert stats.last_update > stats.last_attempt
    assert stats.update_errors == 0

    assert stats.requests == {"total": 19, "4xx": 19, "5xx": 0, "bad": 19, "ok": 0}
    assert stats.clusters["cluster_127_0_0_1_8500_ambassador"] == {
        "healthy_members": 1,
        "total_members": 1,
        "healthy_percent": 100,
        "update_attempts": 4220,
        "update_successes": 4220,
        "update_percent": 100,
        "upstream_ok": 14,
        "upstream_4xx": 14,
        "upstream_5xx": 0,
        "upstream_bad": 0,
    }

    assert stats.clusters["cluster_identity_api_jennifer_testing_sv-0"] == {
        "healthy_members": 1,
        "total_members": 1,
        "healthy_percent": None,
        "update_attempts": 4216,
        "update_successes": 4216,
        "update_percent": 100,
        "upstream_ok": 0,
        "upstream_4xx": 0,
        "upstream_5xx": 0,
        "upstream_bad": 0,
    }

    assert stats.envoy["cluster_manager"] == {
        "active_clusters": 336,
        "cds": {
            "init_fetch_timeout": 0,
            "update_attempt": 15,
            "update_failure": 0,
            "update_rejected": 0,
            "update_success": 14,
            "update_time": 1602023101467,
            "version": 11975404232982186540,
        },
        "cluster_added": 336,
        "cluster_modified": 0,
        "cluster_removed": 0,
        "cluster_updated": 0,
        "cluster_updated_via_merge": 0,
        "update_merge_cancelled": 0,
        "update_out_of_merge_window": 0,
        "warming_clusters": 0,
    }

    assert stats.envoy["control_plane"] == {
        "connected_state": 1,
        "pending_requests": 0,
        "rate_limit_enforced": 0,
    }

    assert stats.envoy["listener_manager"] == {
        "lds": {
            "init_fetch_timeout": 0,
            "update_attempt": 32,
            "update_failure": 0,
            "update_rejected": 17,
            "update_success": 14,
            "update_time": 1602023102107,
            "version": 11975404232982186540,
        },
        "listener_added": 2,
        "listener_create_failure": 0,
        "listener_create_success": 8,
        "listener_in_place_updated": 0,
        "listener_modified": 0,
        "listener_removed": 0,
        "listener_stopped": 0,
        "total_filter_chains_draining": 0,
        "total_listeners_active": 2,
        "total_listeners_draining": 0,
        "total_listeners_warming": 0,
        "workers_started": 1,
    }

    esm.update()
    stats2 = esm.get_stats()

    assert id(stats) != id(stats2)


def test_locks():
    mocker = EnvoyStatsMocker()

    esm = EnvoyStatsMgr(
        logger,
        max_live_age=3,
        max_ready_age=3,
        fetch_log_levels=mocker.fetch_log_levels,
        fetch_envoy_stats=mocker.slow_fetch_stats,
    )

    def slow_background():
        esm.update()

    def check_get_stats():
        start = time.perf_counter()
        stats = esm.get_stats()
        end = time.perf_counter()

        assert (end - start) < 0.001
        return stats

    # Start updating in the background. This will take five seconds.
    threading.Thread(target=slow_background).start()

    # At this point, we should be able to get stats very quickly, and see
    # alive but not ready.
    sys.stdout.write("1")
    sys.stdout.flush()
    stats1 = check_get_stats()
    assert stats1.is_alive()
    assert not stats1.is_ready()

    # Wait 2 seconds. We should get the _same_ stats object, and again,
    # alive but not ready.
    time.sleep(2)
    sys.stdout.write("2")
    sys.stdout.flush()
    stats2 = check_get_stats()
    assert id(stats2) == id(stats1)
    assert stats2.is_alive()
    assert not stats2.is_ready()

    # We should also see the update_lock being held.
    assert not esm.update_lock.acquire(blocking=False)

    # Wait 2 more seconds. We should get the same stats object, but it should
    # now say neither alive nor ready.
    time.sleep(2)
    sys.stdout.write("3")
    sys.stdout.flush()
    stats3 = check_get_stats()
    assert id(stats3) == id(stats1)
    assert not stats3.is_alive()
    assert not stats3.is_ready()

    # Wait 2 more seconds. At this point, we should get a new stats object,
    # and we should see alive and ready.
    time.sleep(2)
    sys.stdout.write("4")
    sys.stdout.flush()
    stats4 = check_get_stats()
    assert id(stats4) != id(stats1)
    assert stats4.is_alive()
    assert stats4.is_ready()

    # The update lock should not be held, either.
    assert esm.update_lock.acquire(blocking=False)
    esm.update_lock.release()

    # Finally, if we wait four more seconds, we should still have the same
    # stats object as last time, but we should see neither alive nor ready.
    time.sleep(4)
    sys.stdout.write("5")
    sys.stdout.flush()
    stats5 = check_get_stats()
    assert id(stats5) == id(stats4)
    assert not stats5.is_alive()
    assert not stats5.is_ready()


if __name__ == "__main__":
    pytest.main(sys.argv)
