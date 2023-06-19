import os
from unittest.mock import patch

import pytest


@pytest.fixture(autouse=True)
def go_library():
    with patch("ambassador.ir.irgofilter.go_library_exists") as go_library_exists:
        go_library_exists.return_value = True
        yield go_library_exists


@pytest.fixture(params=["true", "yes", "1", "True", "YES"])
def disable_go_filter(request):
    with patch("ambassador.ir.irgofilter.go_filter_disabled") as go_filter_disabled:
        go_filter_disabled.return_value = request.param
        yield go_filter_disabled


@pytest.fixture(params=["false", "no", "0"])
def enable_go_filter(request):
    with patch("ambassador.ir.irgofilter.go_filter_disabled") as go_filter_disabled:
        go_filter_disabled.return_value = request.param
        yield go_filter_disabled
