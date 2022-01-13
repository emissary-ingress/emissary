from typing import Optional

import pytest

def pytest_addoption(parser):
    parser.addoption("--letter-range", action="store", default="all", choices=["ah","ip","qz","all"])


letter_range: Optional[str] = None
def pytest_configure(config):
    global letter_range
    letter_range = config.getoption('--letter-range')

    print(f"pytest selecting tests in letter range {letter_range}")
