import copy
import logging
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

def _get_ir_config(yaml):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)
    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")
    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir
    return ir


@pytest.mark.compilertest
def test_ir_mapping():
    yaml = """
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: slowsvc-slow
namespace: ambassador
prefix: /slow/
service: slowsvc
timeout_ms: 1000
docs:
  path: /endpoint
  display_name: "slow service"
  timeout_ms: 8000
"""

    conf = _get_ir_config(yaml)
    all_mappings = []
    for i in conf.groups.values():
        all_mappings = all_mappings + i.mappings

    slowsvc_mappings = [x for x in all_mappings if x['name'] == 'slowsvc-slow']
    assert(len(slowsvc_mappings) == 1)
    print(slowsvc_mappings[0].as_dict())
    assert(slowsvc_mappings[0].docs['timeout_ms'] == 8000)
