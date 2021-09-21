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
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler
from ambassador.ir import IRResource
from ambassador.ir.irbuffer import IRBuffer

yaml = '''
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
config:
    defaults:
        max_request_words: 1
        altered: 2
        test_resource:
            max_request_words: 3
            altered: 4
            funk: 8
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: test_mapping
hostname: "*"
prefix: /test/
service: test:9999
'''


class IRTestResource(IRBuffer):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.testresource",
                 name: str="ir.testresource",
                 kind: str="IRTestResource",
                 **kwargs) -> None:

        super().__init__(ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, **kwargs)

        self.default_class = 'test_resource'


def test_lookup():
    aconf = Config()

    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    t1 = IRBuffer(ir, aconf, rkey='-foo-', name='buffer', max_request_bytes=4096)

    t2 = IRTestResource(ir, aconf, rkey='-foo-', name='buffer', max_request_bytes=8192)

    assert t1.lookup('max_request_bytes') == 4096
    assert t1.lookup('max_request_bytes', 57) == 4096
    assert t1.lookup('max_request_bytes2', 57) == 57

    assert t1.lookup('max_request_words') == 1
    assert t1.lookup('max_request_words', 77) == 1
    assert t1.lookup('max_request_words', default_key='altered') == 2
    assert t1.lookup('max_request_words', 77, default_key='altered') == 2
    assert t1.lookup('max_request_words', default_key='altered2') == None
    assert t1.lookup('max_request_words', 77, default_key='altered2') == 77

    assert t1.lookup('max_request_words', default_class='test_resource') == 3
    assert t1.lookup('max_request_words', 77, default_class='test_resource') == 3
    assert t1.lookup('max_request_words', 77, default_class='test_resource2') == 1
    assert t1.lookup('max_request_words', default_key='altered', default_class='test_resource') == 4
    assert t1.lookup('max_request_words', 77, default_key='altered', default_class='test_resource') == 4
    assert t1.lookup('max_request_words', default_key='altered2', default_class='test_resource') == None
    assert t1.lookup('max_request_words', 77, default_key='altered2', default_class='test_resource') == 77

    assert t1.lookup('funk') == None
    assert t1.lookup('funk', 77) == 77

    assert t1.lookup('funk', default_class='test_resource') == 8
    assert t1.lookup('funk', 77, default_class='test_resource') == 8
    assert t1.lookup('funk', 77, default_class='test_resource2') == 77

    assert t2.lookup('max_request_bytes') == 8192
    assert t2.lookup('max_request_bytes', 57) == 8192
    assert t2.lookup('max_request_bytes2', 57) == 57

    assert t2.lookup('max_request_words') == 3
    assert t2.lookup('max_request_words', 77) == 3
    assert t2.lookup('max_request_words', default_key='altered') == 4
    assert t2.lookup('max_request_words', 77, default_key='altered') == 4
    assert t2.lookup('max_request_words', default_key='altered2') == None
    assert t2.lookup('max_request_words', 77, default_key='altered2') == 77

    assert t2.lookup('max_request_words', default_class='/') == 1
    assert t2.lookup('max_request_words', 77, default_class='/') == 1
    assert t2.lookup('max_request_words', 77, default_class='/2') == 1
    assert t2.lookup('max_request_words', default_key='altered', default_class='/') == 2
    assert t2.lookup('max_request_words', 77, default_key='altered', default_class='/') == 2
    assert t2.lookup('max_request_words', default_key='altered2', default_class='/') == None
    assert t2.lookup('max_request_words', 77, default_key='altered2', default_class='/') == 77

    assert t2.lookup('funk') == 8
    assert t2.lookup('funk', 77) == 8

    assert t2.lookup('funk', default_class='test_resource') == 8
    assert t2.lookup('funk', 77, default_class='test_resource') == 8
    assert t2.lookup('funk', 77, default_class='test_resource2') == 77

if __name__ == '__main__':
    pytest.main(sys.argv)
