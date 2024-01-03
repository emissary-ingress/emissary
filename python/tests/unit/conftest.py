import os
from unittest.mock import patch

import pytest


@pytest.fixture(autouse=True)
def go_library():
    with patch("ambassador.ir.irgofilter.go_library_exists") as go_library_exists:
        go_library_exists.return_value = True
        yield go_library_exists
