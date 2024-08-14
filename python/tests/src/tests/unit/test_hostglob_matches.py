import logging
import sys

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)

logger = logging.getLogger("ambassador")

from ambassador.ir.irutils import hostglob_matches  # noqa: E402


@pytest.mark.compilertest
def test_hostglob_matches():
    for v1, v2, wanted_result in [
        ("a.example.com", "a.example.com", True),
        ("a.example.com", "b.example.com", False),
        ("*", "foo.example.com", True),
        ("*.example.com", "a.example.com", True),
        ("*example.com", "b.example.com", True),
        # This is never OK: the "*" can't match a bare ".".
        ("*example.com", ".example.com", False),
        # This is OK, because DNS allows names to end with a "."
        ("foo.example*", "foo.example.com.", True),
        # This is never OK: the "*" cannot match an empty string.
        ("*example.com", "example.com", False),
        ("*ple.com", "b.example.com", True),
        ("*.example.com", "a.example.org", False),
        ("*example.com", "a.example.org", False),
        ("*ple.com", "a.example.org", False),
        ("a.example.*", "a.example.com", True),
        ("a.example*", "a.example.com", True),
        ("a.exa*", "a.example.com", True),
        ("a.example.*", "a.example.org", True),
        ("a.example.*", "b.example.com", False),
        ("a.example*", "b.example.com", False),
        ("a.exa*", "b.example.com", False),
        # '*' has to appear at the beginning or the end, not in the middle.
        ("a.*.com", "a.example.com", False),
        # Various DNS glob situations disagree about whether "*" can cross subdomain
        # boundaries. We follow what Envoy does, which is to allow crossing.
        ("*.com", "a.example.com", True),
        ("*.com", "a.example.org", False),
        ("*.example.com", "*.example.com", True),
        # This looks wrong but it's OK: both match e.g. foo.example.com.
        ("*example.com", "*.example.com", True),
        # These are ugly corner cases, but they should still work!
        ("*.example.com", "a.example.*", True),
        ("*.example.com", "a.b.example.*", True),
        ("*.example.baz.com", "a.b.example.*", True),
        ("*.foo.bar", "baz.zing.*", True),
        # The Host.hostname is overloaded in that it determines the SNI for TLS and the
        # virtual host name for :authority header matching for HTTP. These are valid
        # scenarios that users try when using non-standard ports so we make sure they work.
        ("*.local:8500", "quote.local", False),
        ("*.local:8500", "quote.local:8500", True),
        ("*", "quote.local:8500", True),
        ("quote.*", "quote.local:8500", True),
        ("quote.*", "*.local:8500", True),
        ("quote.com:8500", "quote.com:8500", True),
    ]:
        assert (
            hostglob_matches(v1, v2) == wanted_result
        ), f"1. {v1} ~ {v2} != {wanted_result}"
        assert (
            hostglob_matches(v2, v1) == wanted_result
        ), f"2. {v2} ~ {v1} != {wanted_result}"


if __name__ == "__main__":
    pytest.main(sys.argv)
