import logging

import pytest

from ambassador_diag.diagd import ReconfigStats


def assert_checks(r: ReconfigStats, when: float, want_check: bool, want_timers: bool) -> None:
    got_check = r.needs_check(when)
    assert got_check == want_check, f"{when}: wanted check {want_check}, got {got_check}"

    got_timers = r.needs_timers(when)
    assert got_timers == want_timers, f"{when}: wanted timers {want_timers}, got {got_timers}"


def test_reconfig_stats():
    logging.basicConfig(
        level=logging.DEBUG,
        format="%(asctime)s ffs %(levelname)s: %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S",
    )

    logger = logging.getLogger("ffs")
    logger.setLevel(logging.DEBUG)

    r = ReconfigStats(
        logger,
        max_incr_between_checks=5,
        max_time_between_checks=20,
        max_config_between_timers=2,
        max_time_between_timers=10,
    )

    r.dump()

    r.mark("complete", 10)
    assert_checks(r, 11, False, False)
    assert_checks(r, 12, False, False)
    r.mark("incremental", 10)
    assert_checks(r, 14, False, True)  # Need timers from outstanding
    assert_checks(r, 18, False, True)
    r.mark_timers_logged(20)
    assert_checks(r, 20, False, False)
    r.mark("diag", 21)
    assert_checks(r, 22, False, False)
    assert_checks(r, 30, False, False)
    assert_checks(r, 32, True, True)
    r.mark_checked(False, 32)

    r.mark("incremental", 33)
    assert_checks(r, 34, False, True)
    r.mark_timers_logged(34)
    r.mark("incremental", 35)
    assert_checks(r, 36, False, False)
    r.mark("incremental", 37)
    assert_checks(r, 38, False, True)
    r.mark_timers_logged(38)
    r.mark("incremental", 39)
    assert_checks(r, 40, False, False)
    r.mark("incremental", 41)
    assert_checks(r, 42, True, True)
    r.mark_timers_logged(42)
    r.mark_checked(True, 42)
    r.mark("incremental", 43)
    assert_checks(r, 44, False, False)
    r.mark("incremental", 45)
    assert_checks(r, 46, False, True)
    r.mark("incremental", 47)
    assert_checks(r, 48, False, True)
    r.mark("incremental", 49)
    assert_checks(r, 50, False, True)
    r.mark("incremental", 51)
    assert_checks(r, 52, True, True)
    r.mark_checked(True, 52)
    assert_checks(r, 55, False, True)

    assert_checks(r, 74, False, True)

    assert_checks(r, 84, False, True)

    r.mark("complete", 100)
    assert_checks(r, 101, False, True)
    r.mark("incremental", 102)
    r.mark_timers_logged(102)
    assert_checks(r, 103, False, False)
    r.mark("incremental", 104)
    assert_checks(r, 105, False, False)

    r.dump()

    assert r.counts["incremental"] == 13
    assert r.counts["complete"] == 2
    assert r.incrementals_outstanding == 2
    assert r.checks == 3
    assert r.errors == 1


if __name__ == "__main__":
    import sys

    pytest.main(sys.argv)
