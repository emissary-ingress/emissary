from typing import Optional

import logging
import os
import sys

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador import Config, IR
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler
from ambassador.ir import IRResource
from ambassador.ir.irbuffer import IRBuffer

yaml = '''
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name: test_mapping
host: "*"
prefix: /test/
service: ${TEST_SERVICE}:9999
'''


def test_envvar_expansion():
    os.environ["TEST_SERVICE"] = "foo"

    aconf = Config()

    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)

    aconf.load_all(fetcher.sorted())

    mappings = aconf.config["mappings"]
    test_mapping = mappings["test_mapping"]

    assert test_mapping.service == "foo:9999"


if __name__ == '__main__':
    pytest.main(sys.argv)
