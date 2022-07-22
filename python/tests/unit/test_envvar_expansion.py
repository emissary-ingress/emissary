import logging
import os
import sys
from typing import Optional

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)

logger = logging.getLogger("ambassador")

from ambassador import IR, Config
from ambassador.fetch import ResourceFetcher
from ambassador.ir import IRResource
from ambassador.ir.irbuffer import IRBuffer
from ambassador.utils import NullSecretHandler

yaml = """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: test_mapping
hostname: "*"
prefix: /test/
service: ${TEST_SERVICE}:9999
"""


def test_envvar_expansion():
    os.environ["TEST_SERVICE"] = "foo"

    aconf = Config()

    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)

    aconf.load_all(fetcher.sorted())

    mappings = aconf.config["mappings"]
    test_mapping = mappings["test_mapping"]

    assert test_mapping.service == "foo:9999"


if __name__ == "__main__":
    pytest.main(sys.argv)
