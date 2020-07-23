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
from ambassador.ir.irbasemapping import qualify_service_name

yaml = '''
---
apiVersion: getambassador.io/v1
kind: Module
name: ambassador
config: {}
'''


def test_qualify_service():
    aconf = Config()

    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir, "could not create an IR"

    assert qualify_service_name(ir, "backoffice", None) == "backoffice"
    assert qualify_service_name(ir, "backoffice", "default") == "backoffice"
    assert qualify_service_name(ir, "backoffice", "otherns") == "backoffice.otherns"
    assert qualify_service_name(ir, "backoffice.otherns", None) == "backoffice.otherns"
    assert qualify_service_name(ir, "backoffice.otherns", "default") == "backoffice.otherns"
    assert qualify_service_name(ir, "backoffice.otherns", "otherns") == "backoffice.otherns"

    assert qualify_service_name(ir, "backoffice:80", None) == "backoffice:80"
    assert qualify_service_name(ir, "backoffice:80", "default") == "backoffice:80"
    assert qualify_service_name(ir, "backoffice:80", "otherns") == "backoffice.otherns:80"
    assert qualify_service_name(ir, "backoffice.otherns:80", None) == "backoffice.otherns:80"
    assert qualify_service_name(ir, "backoffice.otherns:80", "default") == "backoffice.otherns:80"
    assert qualify_service_name(ir, "backoffice.otherns:80", "otherns") == "backoffice.otherns:80"

    assert qualify_service_name(ir, "http://backoffice", None) == "http://backoffice"
    assert qualify_service_name(ir, "http://backoffice", "default") == "http://backoffice"
    assert qualify_service_name(ir, "http://backoffice", "otherns") == "http://backoffice.otherns"
    assert qualify_service_name(ir, "http://backoffice.otherns", None) == "http://backoffice.otherns"
    assert qualify_service_name(ir, "http://backoffice.otherns", "default") == "http://backoffice.otherns"
    assert qualify_service_name(ir, "http://backoffice.otherns", "otherns") == "http://backoffice.otherns"

    assert qualify_service_name(ir, "http://backoffice:80", None) == "http://backoffice:80"
    assert qualify_service_name(ir, "http://backoffice:80", "default") == "http://backoffice:80"
    assert qualify_service_name(ir, "http://backoffice:80", "otherns") == "http://backoffice.otherns:80"
    assert qualify_service_name(ir, "http://backoffice.otherns:80", None) == "http://backoffice.otherns:80"
    assert qualify_service_name(ir, "http://backoffice.otherns:80", "default") == "http://backoffice.otherns:80"
    assert qualify_service_name(ir, "http://backoffice.otherns:80", "otherns") == "http://backoffice.otherns:80"

    assert qualify_service_name(ir, "https://backoffice", None) == "https://backoffice"
    assert qualify_service_name(ir, "https://backoffice", "default") == "https://backoffice"
    assert qualify_service_name(ir, "https://backoffice", "otherns") == "https://backoffice.otherns"
    assert qualify_service_name(ir, "https://backoffice.otherns", None) == "https://backoffice.otherns"
    assert qualify_service_name(ir, "https://backoffice.otherns", "default") == "https://backoffice.otherns"
    assert qualify_service_name(ir, "https://backoffice.otherns", "otherns") == "https://backoffice.otherns"

    assert qualify_service_name(ir, "https://backoffice:443", None) == "https://backoffice:443"
    assert qualify_service_name(ir, "https://backoffice:443", "default") == "https://backoffice:443"
    assert qualify_service_name(ir, "https://backoffice:443", "otherns") == "https://backoffice.otherns:443"
    assert qualify_service_name(ir, "https://backoffice.otherns:443", None) == "https://backoffice.otherns:443"
    assert qualify_service_name(ir, "https://backoffice.otherns:443", "default") == "https://backoffice.otherns:443"
    assert qualify_service_name(ir, "https://backoffice.otherns:443", "otherns") == "https://backoffice.otherns:443"

    assert qualify_service_name(ir, "localhost", None) == "localhost"
    assert qualify_service_name(ir, "localhost", "default") == "localhost"
    assert qualify_service_name(ir, "localhost", "otherns") == "localhost"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "localhost.otherns", None) == "localhost.otherns"
    assert qualify_service_name(ir, "localhost.otherns", "default") == "localhost.otherns"
    assert qualify_service_name(ir, "localhost.otherns", "otherns") == "localhost.otherns"

    assert qualify_service_name(ir, "localhost:80", None) == "localhost:80"
    assert qualify_service_name(ir, "localhost:80", "default") == "localhost:80"
    assert qualify_service_name(ir, "localhost:80", "otherns") == "localhost:80"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "localhost.otherns:80", None) == "localhost.otherns:80"
    assert qualify_service_name(ir, "localhost.otherns:80", "default") == "localhost.otherns:80"
    assert qualify_service_name(ir, "localhost.otherns:80", "otherns") == "localhost.otherns:80"

    assert qualify_service_name(ir, "http://localhost", None) == "http://localhost"
    assert qualify_service_name(ir, "http://localhost", "default") == "http://localhost"
    assert qualify_service_name(ir, "http://localhost", "otherns") == "http://localhost"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "http://localhost.otherns", None) == "http://localhost.otherns"
    assert qualify_service_name(ir, "http://localhost.otherns", "default") == "http://localhost.otherns"
    assert qualify_service_name(ir, "http://localhost.otherns", "otherns") == "http://localhost.otherns"

    assert qualify_service_name(ir, "http://localhost:80", None) == "http://localhost:80"
    assert qualify_service_name(ir, "http://localhost:80", "default") == "http://localhost:80"
    assert qualify_service_name(ir, "http://localhost:80", "otherns") == "http://localhost:80"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "http://localhost.otherns:80", None) == "http://localhost.otherns:80"
    assert qualify_service_name(ir, "http://localhost.otherns:80", "default") == "http://localhost.otherns:80"
    assert qualify_service_name(ir, "http://localhost.otherns:80", "otherns") == "http://localhost.otherns:80"

    assert qualify_service_name(ir, "https://localhost", None) == "https://localhost"
    assert qualify_service_name(ir, "https://localhost", "default") == "https://localhost"
    assert qualify_service_name(ir, "https://localhost", "otherns") == "https://localhost"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "https://localhost.otherns", None) == "https://localhost.otherns"
    assert qualify_service_name(ir, "https://localhost.otherns", "default") == "https://localhost.otherns"
    assert qualify_service_name(ir, "https://localhost.otherns", "otherns") == "https://localhost.otherns"

    assert qualify_service_name(ir, "https://localhost:443", None) == "https://localhost:443"
    assert qualify_service_name(ir, "https://localhost:443", "default") == "https://localhost:443"
    assert qualify_service_name(ir, "https://localhost:443", "otherns") == "https://localhost:443"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "https://localhost.otherns:443", None) == "https://localhost.otherns:443"
    assert qualify_service_name(ir, "https://localhost.otherns:443", "default") == "https://localhost.otherns:443"
    assert qualify_service_name(ir, "https://localhost.otherns:443", "otherns") == "https://localhost.otherns:443"

    assert qualify_service_name(ir, "https://bad-service:443:443", "otherns") == "https://bad-service:443:443"
    assert qualify_service_name(ir, "https://bad-service:443:443", "otherns", rkey="test-rkey") == "https://bad-service:443:443"

    errors = ir.aconf.errors
    
    assert "-global-" in errors

    errors = errors["-global-"]

    assert len(errors) == 2

    assert not errors[0]["ok"]
    assert errors[0]["error"] == "Malformed service port in https://bad-service:443:443"

    assert not errors[1]["ok"]
    assert errors[1]["error"] == "test-rkey: Malformed service port in https://bad-service:443:443"


if __name__ == '__main__':
    pytest.main(sys.argv)
