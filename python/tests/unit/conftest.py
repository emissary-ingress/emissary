import os
from unittest.mock import patch

import pytest


@pytest.fixture(autouse=True)
def go_library():
    with patch("ambassador.ir.irgofilter.go_library_exists") as go_library_exists:
        go_library_exists.return_value = True
        yield go_library_exists


@pytest.fixture()
def disable_go_filter():
    with patch("ambassador.ir.irgofilter.AMBASSADOR_DISABLE_GO_FILTER") as disable_go_filter:
        disable_go_filter.return_value = True
        yield disable_go_filter
