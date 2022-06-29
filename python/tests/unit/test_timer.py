import sys
import time

import pexpect
import pytest

from ambassador.utils import Timer

epsilon = 0.0001


def feq(x: float, y: float) -> bool:
    return abs(x - y) <= epsilon


def test_Timer():
    t1 = Timer("test1")
    t2 = Timer("test2")

    assert not t1.running, "t1 must not be running"
    assert not t2.running, "t2 must not be running"

    t1.start(100)
    t2.start(100)

    assert t1.running, "t1 must be running"
    assert feq(t1.starttime, 100), "t1.starttime must be 100, got {t1.starttime}"
    assert t1.cycles == 0

    assert t2.running, "t2 must be running"
    assert t2.starttime == 100
    assert t2.cycles == 0

    a2 = t2.stop(110)
    assert feq(a2, 10), f"t2.stop() must be 10, got {a2}"
    assert not t2.running, "t2 must not be running"
    assert feq(t2.starttime, 100), "t2.starttime must be 100, got {t2.starttime}"
    assert t2.cycles == 1

    assert feq(t2.accumulated, 10), f"t2.accumulated must be 10, got {t2.accumulated}"
    assert feq(t2.minimum, 10), f"t2.minimum must be 10, got {t2.minimum}"
    assert feq(t2.maximum, 10), f"t2.maximum must be 10, got {t2.maximum}"
    assert feq(t2.average, 10), f"t2.average must be 10, got {t2.average}"

    t2.start(120)
    assert t2.running, "t2 must be running"
    assert feq(t2.starttime, 120), "t2.starttime must be 120, got {t2.starttime}"
    assert t2.cycles == 1

    a2 = t2.stop(140)
    assert feq(a2, 30), f"t2.stop() must be 30, got {a2}"
    assert not t2.running, "t2 must not be running"
    assert t2.starttime == 120
    assert t2.cycles == 2

    assert feq(t2.accumulated, 30), f"t2.accumulated must be 30, got {t2.accumulated}"
    assert feq(t2.minimum, 10), f"t2.minimum must be 10, got {t2.minimum}"
    assert feq(t2.maximum, 20), f"t2.maximum must be 20, got {t2.maximum}"
    assert feq(t2.average, 15), f"t2.average must be 15, got {t2.average}"

    with t2:
        assert t2.running, "t2 must be running"
        # Don't assert t2.starttime() here, since we can't have set it.
        assert t2.cycles == 2

        t2.faketime(6)

    assert feq(t2.accumulated, 36), f"t2.stop() must be 36, got {t2.accumulated}"
    assert not t2.running, "t2 must not be running"
    # Don't assert t2.starttime() here, since we can't have set it.
    assert t2.cycles == 3

    assert feq(t2.accumulated, 36), f"t2.accumulated must be 36, got {t2.accumulated}"
    assert feq(t2.minimum, 6), f"t2.minimum must be 6, got {t2.minimum}"
    assert feq(t2.maximum, 20), f"t2.maximum must be 20, got {t2.maximum}"
    assert feq(t2.average, 12), f"t2.average must be 12, got {t2.average}"

    a1 = t1.stop(300)
    assert feq(a1, 200), f"t1.stop() must be 200, got {a1}"
    assert not t1.running, "t1 must not be running"
    assert t1.starttime == 100
    assert t1.cycles == 1

    assert feq(t1.accumulated, 200), f"t1.accumulated must be 200, got {t1.accumulated}"
    assert feq(t1.minimum, 200), f"t1.minimum must be 200, got {t1.minimum}"
    assert feq(t1.maximum, 200), f"t1.maximum must be 200, got {t1.maximum}"
    assert feq(t1.average, 200), f"t1.average must be 200, got {t1.average}"

    # Test calling stop twice...
    a1 = t1.stop(300)
    assert feq(a1, 200), f"t1.stop() must be 200, got {a1}"
    assert not t1.running, "t1 must not be running"
    assert t1.starttime == 100
    assert t1.cycles == 1

    assert feq(t1.accumulated, 200), f"t1.accumulated must be 200, got {t1.accumulated}"
    assert feq(t1.minimum, 200), f"t1.minimum must be 200, got {t1.minimum}"
    assert feq(t1.maximum, 200), f"t1.maximum must be 200, got {t1.maximum}"
    assert feq(t1.average, 200), f"t1.average must be 200, got {t1.average}"

    # Test calling start twice...
    t1.start(400)
    assert t1.running, "t1 must be running"
    assert feq(t1.starttime, 400), "t1.starttime must be 400, got {t1.starttime}"
    assert t1.cycles == 1

    t1.start(500)
    assert t1.running, "t1 must be running"
    assert feq(t1.starttime, 500), "t1.starttime must be 500, got {t1.starttime}"
    assert t1.cycles == 1

    a1 = t1.stop(600)
    assert feq(a1, 300), f"t1.stop() must be 300, got {a1}"
    assert not t1.running, "t1 must not be running"
    assert t1.starttime == 500
    assert t1.cycles == 2

    assert feq(t1.accumulated, 300), f"t1.accumulated must be 300, got {t1.accumulated}"
    assert feq(t1.minimum, 100), f"t1.minimum must be 200, got {t1.minimum}"
    assert feq(t1.maximum, 200), f"t1.maximum must be 300, got {t1.maximum}"
    assert feq(t1.average, 150), f"t1.average must be 150, got {t1.average}"


if __name__ == "__main__":
    pytest.main(sys.argv)
