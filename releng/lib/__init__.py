#!/hint/python3

import re
import subprocess
from typing import Any, List

from .gitutil import git_check_clean as git_check_clean  # Stop mypy complaining about implicit reexport
from .uiutil import run_txtcapture
from .gitutil import git_add as git_add # Stop mypy complaining about implicit reexport

# These are some regular expressions to validate and parse
# X.Y.Z[-rc.N] versions.
re_ga = re.compile(r'^([0-9]+)\.([0-9]+)\.([0-9]+)$')
vX = 1
vY = 2


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
