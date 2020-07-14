from typing import Optional

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
from ambassador.config import ResourceFetcher
from ambassador.utils import NullSecretHandler
from ambassador.ir import IRResource
from ambassador.ir.irbuffer import IRBuffer

yaml = '''
---
apiVersion: getambassador.io/v1
kind: Module
name: ambassador
config:
    defaults:
        max_request_words: 1
        altered: 2
        test_resource:
            max_request_words: 3
            altered: 4
---
apiVersion: getambassador.io/v1
kind: Mapping
name: test_mapping
prefix: /test/
service: test:9999
'''


def test_resourcefetcher_handle_k8s_service():
    aconf = Config()

    fetcher = ResourceFetcher(logger, aconf)

    # Test no metadata key
    svc = {}
    result = fetcher.handle_k8s_service(svc)
    assert result == None

    svc["metadata"] = {
        "name": "testservice",
        "annotations": {
            "foo": "bar"
        }
    }
    # Test no ambassador annotation
    result = fetcher.handle_k8s_service(svc)
    assert result == ('testservice.default', [])

    # Test empty annotation
    svc["metadata"]["annotations"]["getambassador.io/config"] = {}
    result = fetcher.handle_k8s_service(svc)
    assert result == ('testservice.default', [])

    # Test valid annotation
    svc["metadata"]["annotations"]["getambassador.io/config"] = """apiVersion: getambassador.io/v1
kind: Mapping
name: test_mapping
prefix: /test/
service: test:9999"""
    result = fetcher.handle_k8s_service(svc)
    expected = {
        '_force_validation': True,
        'apiVersion': 'getambassador.io/v1',
        'kind': 'Mapping',
        'name': 'test_mapping',
        'prefix': '/test/',
        'service': 'test:9999',
        'namespace': 'default'
    }
    assert result == ('testservice.default', [expected])

if __name__ == '__main__':
    pytest.main(sys.argv)
