#!/hint/python3

import re
import subprocess
from typing import Any, List

from .uiutil import run_txtcapture

# These are some regular expressions to validate and parse
# X.Y.Z[-rc.N] versions.
re_rc = re.compile(r'^([0-9]+)\.([0-9]+)\.([0-9]+)-rc\.([0-9]+)$')
re_ga = re.compile(r'^([0-9]+)\.([0-9]+)\.([0-9]+)$')
vX = 1
vY = 2
vZ = 3
vN = 4


def base_version(release_version: str) -> str:
    """Given 'X.Y.Z[-rc.N]', return 'X.Y'."""
    return build_version(release_version).rsplit(sep='.', maxsplit=1)[0]


def build_version(release_version: str) -> str:
    """Given 'X.Y.Z[-rc.N]', return 'X.Y.Z'."""
    return release_version.split('-')[0]


def assert_eq(actual: Any, expected: Any) -> None:
    """`assert_eq(a, b)` is like `assert a == b`, but has a useful error
    message when they're not equal.

    """
    if actual != expected:
        raise AssertionError(f"wanted '{expected}', got '{actual}'")


def get_is_private() -> bool:
    """Return whether we're in a "private" Git checkout, for doing
    embargoed work.

    """
    remote_names = run_txtcapture(['git', 'remote']).split()
    remote_urls: List[str] = []
    for remote_name in remote_names:
        remote_urls += run_txtcapture(['git', 'remote', 'get-url', '--all', remote_name]).split()
    return 'private' in "\n".join(remote_urls)


def aes_branchname() -> str:
    """aes_branchname() returns the apro.git branchname for this vX.Y.Z
    release; this function is codification of convention.

    """
    # Convention: Just allow whatever branch we're currently checked
    # out at; enforce a convention in `rel-00-sanity-check`.
    return run_txtcapture(['git', 'rev-parse', '--abbrev-ref', 'HEAD'])


def plugin_branchname() -> str:
    """plugin_branchname() returns the apro-example-plugin.git branchname
    for this vX.Y.Z release; this function is codification of
    convention.

    """
    # Convention: Use the same branchname as in apro.git.
    return aes_branchname()


def oss_branchname() -> str:
    """oss_branchname() returns the ambassador.git branchname for this
    vX.Y.Z release; this function is codification of convention.

    """
    # Convention: Use the same branchname as in apro.git.
    return aes_branchname()
