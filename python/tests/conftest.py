import os
from unittest.mock import patch

import pytest

from ambassador.utils import parse_bool


@pytest.fixture(autouse=True)
def go_library():
    with patch("ambassador.ir.irgofilter.go_library_exists") as go_library_exists:
        go_library_exists.return_value = True
        yield go_library_exists


@pytest.fixture(autouse=True)
def edgestack():
    return parse_bool(os.environ.get("EDGE_STACK", "false"))
