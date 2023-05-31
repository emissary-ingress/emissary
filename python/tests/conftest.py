import os

import pytest

from ambassador.utils import parse_bool


@pytest.fixture(autouse=True)
def edgestack():
    return parse_bool(os.environ.get("EDGE_STACK", "false"))
