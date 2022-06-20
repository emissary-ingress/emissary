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
from ambassador.ir.irbasemapping import normalize_service_name

yaml = '''
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
config: {}
'''

def qualify_service_name(ir: 'IR', service: str, namespace: Optional[str], rkey: Optional[str]=None) -> str:
    return normalize_service_name(ir, service, namespace, 'KubernetesTestResolver', rkey=rkey)

def test_qualify_service():
    """
    Note: This has a Go equivalent in github.com/emissary-ingress/emissary/v3/pkg/emissaryutil.  Please
    keep them in-sync.
    """
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

    assert normalize_service_name(ir, "backoffice", None, 'ConsulResolver') == "backoffice"
    assert normalize_service_name(ir, "backoffice", "default", 'ConsulResolver') == "backoffice"
    assert normalize_service_name(ir, "backoffice", "otherns", 'ConsulResolver') == "backoffice"
    assert normalize_service_name(ir, "backoffice.otherns", None, 'ConsulResolver') == "backoffice.otherns"
    assert normalize_service_name(ir, "backoffice.otherns", "default", 'ConsulResolver') == "backoffice.otherns"
    assert normalize_service_name(ir, "backoffice.otherns", "otherns", 'ConsulResolver') == "backoffice.otherns"

    assert qualify_service_name(ir, "backoffice:80", None) == "backoffice:80"
    assert qualify_service_name(ir, "backoffice:80", "default") == "backoffice:80"
    assert qualify_service_name(ir, "backoffice:80", "otherns") == "backoffice.otherns:80"
    assert qualify_service_name(ir, "backoffice.otherns:80", None) == "backoffice.otherns:80"
    assert qualify_service_name(ir, "backoffice.otherns:80", "default") == "backoffice.otherns:80"
    assert qualify_service_name(ir, "backoffice.otherns:80", "otherns") == "backoffice.otherns:80"

    assert qualify_service_name(ir, "[fe80::e022:9cff:fecc:c7c4]", None) == "[fe80::e022:9cff:fecc:c7c4]"
    assert qualify_service_name(ir, "[fe80::e022:9cff:fecc:c7c4]", "default") == "[fe80::e022:9cff:fecc:c7c4]"
    assert qualify_service_name(ir, "[fe80::e022:9cff:fecc:c7c4]", "other") == "[fe80::e022:9cff:fecc:c7c4]"
    assert qualify_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4]", None) == "https://[fe80::e022:9cff:fecc:c7c4]"
    assert qualify_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4]", "default") == "https://[fe80::e022:9cff:fecc:c7c4]"
    assert qualify_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4]", "other") == "https://[fe80::e022:9cff:fecc:c7c4]"
    assert qualify_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4]:443", None) == "https://[fe80::e022:9cff:fecc:c7c4]:443"
    assert qualify_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4]:443", "default") == "https://[fe80::e022:9cff:fecc:c7c4]:443"
    assert qualify_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4]:443", "other") == "https://[fe80::e022:9cff:fecc:c7c4]:443"
    assert qualify_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4%25zone]:443", "other") == "https://[fe80::e022:9cff:fecc:c7c4%25zone]:443"

    assert normalize_service_name(ir, "backoffice:80", None, 'ConsulResolver') == "backoffice:80"
    assert normalize_service_name(ir, "backoffice:80", "default", 'ConsulResolver') == "backoffice:80"
    assert normalize_service_name(ir, "backoffice:80", "otherns", 'ConsulResolver') == "backoffice:80"
    assert normalize_service_name(ir, "backoffice.otherns:80", None, 'ConsulResolver') == "backoffice.otherns:80"
    assert normalize_service_name(ir, "backoffice.otherns:80", "default", 'ConsulResolver') == "backoffice.otherns:80"
    assert normalize_service_name(ir, "backoffice.otherns:80", "otherns", 'ConsulResolver') == "backoffice.otherns:80"

    assert qualify_service_name(ir, "http://backoffice", None) == "http://backoffice"
    assert qualify_service_name(ir, "http://backoffice", "default") == "http://backoffice"
    assert qualify_service_name(ir, "http://backoffice", "otherns") == "http://backoffice.otherns"
    assert qualify_service_name(ir, "http://backoffice.otherns", None) == "http://backoffice.otherns"
    assert qualify_service_name(ir, "http://backoffice.otherns", "default") == "http://backoffice.otherns"
    assert qualify_service_name(ir, "http://backoffice.otherns", "otherns") == "http://backoffice.otherns"

    assert normalize_service_name(ir, "http://backoffice", None, 'ConsulResolver') == "http://backoffice"
    assert normalize_service_name(ir, "http://backoffice", "default", 'ConsulResolver') == "http://backoffice"
    assert normalize_service_name(ir, "http://backoffice", "otherns", 'ConsulResolver') == "http://backoffice"
    assert normalize_service_name(ir, "http://backoffice.otherns", None, 'ConsulResolver') == "http://backoffice.otherns"
    assert normalize_service_name(ir, "http://backoffice.otherns", "default", 'ConsulResolver') == "http://backoffice.otherns"
    assert normalize_service_name(ir, "http://backoffice.otherns", "otherns", 'ConsulResolver') == "http://backoffice.otherns"

    assert qualify_service_name(ir, "http://backoffice:80", None) == "http://backoffice:80"
    assert qualify_service_name(ir, "http://backoffice:80", "default") == "http://backoffice:80"
    assert qualify_service_name(ir, "http://backoffice:80", "otherns") == "http://backoffice.otherns:80"
    assert qualify_service_name(ir, "http://backoffice.otherns:80", None) == "http://backoffice.otherns:80"
    assert qualify_service_name(ir, "http://backoffice.otherns:80", "default") == "http://backoffice.otherns:80"
    assert qualify_service_name(ir, "http://backoffice.otherns:80", "otherns") == "http://backoffice.otherns:80"

    assert normalize_service_name(ir, "http://backoffice:80", None, 'ConsulResolver') == "http://backoffice:80"
    assert normalize_service_name(ir, "http://backoffice:80", "default", 'ConsulResolver') == "http://backoffice:80"
    assert normalize_service_name(ir, "http://backoffice:80", "otherns", 'ConsulResolver') == "http://backoffice:80"
    assert normalize_service_name(ir, "http://backoffice.otherns:80", None, 'ConsulResolver') == "http://backoffice.otherns:80"
    assert normalize_service_name(ir, "http://backoffice.otherns:80", "default", 'ConsulResolver') == "http://backoffice.otherns:80"
    assert normalize_service_name(ir, "http://backoffice.otherns:80", "otherns", 'ConsulResolver') == "http://backoffice.otherns:80"

    assert qualify_service_name(ir, "https://backoffice", None) == "https://backoffice"
    assert qualify_service_name(ir, "https://backoffice", "default") == "https://backoffice"
    assert qualify_service_name(ir, "https://backoffice", "otherns") == "https://backoffice.otherns"
    assert qualify_service_name(ir, "https://backoffice.otherns", None) == "https://backoffice.otherns"
    assert qualify_service_name(ir, "https://backoffice.otherns", "default") == "https://backoffice.otherns"
    assert qualify_service_name(ir, "https://backoffice.otherns", "otherns") == "https://backoffice.otherns"

    assert normalize_service_name(ir, "https://backoffice", None, 'ConsulResolver') == "https://backoffice"
    assert normalize_service_name(ir, "https://backoffice", "default", 'ConsulResolver') == "https://backoffice"
    assert normalize_service_name(ir, "https://backoffice", "otherns", 'ConsulResolver') == "https://backoffice"
    assert normalize_service_name(ir, "https://backoffice.otherns", None, 'ConsulResolver') == "https://backoffice.otherns"
    assert normalize_service_name(ir, "https://backoffice.otherns", "default", 'ConsulResolver') == "https://backoffice.otherns"
    assert normalize_service_name(ir, "https://backoffice.otherns", "otherns", 'ConsulResolver') == "https://backoffice.otherns"

    assert qualify_service_name(ir, "https://backoffice:443", None) == "https://backoffice:443"
    assert qualify_service_name(ir, "https://backoffice:443", "default") == "https://backoffice:443"
    assert qualify_service_name(ir, "https://backoffice:443", "otherns") == "https://backoffice.otherns:443"
    assert qualify_service_name(ir, "https://backoffice.otherns:443", None) == "https://backoffice.otherns:443"
    assert qualify_service_name(ir, "https://backoffice.otherns:443", "default") == "https://backoffice.otherns:443"
    assert qualify_service_name(ir, "https://backoffice.otherns:443", "otherns") == "https://backoffice.otherns:443"

    assert normalize_service_name(ir, "https://backoffice:443", None, 'ConsulResolver') == "https://backoffice:443"
    assert normalize_service_name(ir, "https://backoffice:443", "default", 'ConsulResolver') == "https://backoffice:443"
    assert normalize_service_name(ir, "https://backoffice:443", "otherns", 'ConsulResolver') == "https://backoffice:443"
    assert normalize_service_name(ir, "https://backoffice.otherns:443", None, 'ConsulResolver') == "https://backoffice.otherns:443"
    assert normalize_service_name(ir, "https://backoffice.otherns:443", "default", 'ConsulResolver') == "https://backoffice.otherns:443"
    assert normalize_service_name(ir, "https://backoffice.otherns:443", "otherns", 'ConsulResolver') == "https://backoffice.otherns:443"

    assert qualify_service_name(ir, "localhost", None) == "localhost"
    assert qualify_service_name(ir, "localhost", "default") == "localhost"
    assert qualify_service_name(ir, "localhost", "otherns") == "localhost"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "localhost.otherns", None) == "localhost.otherns"
    assert qualify_service_name(ir, "localhost.otherns", "default") == "localhost.otherns"
    assert qualify_service_name(ir, "localhost.otherns", "otherns") == "localhost.otherns"

    assert normalize_service_name(ir, "localhost", None, 'ConsulResolver') == "localhost"
    assert normalize_service_name(ir, "localhost", "default", 'ConsulResolver') == "localhost"
    assert normalize_service_name(ir, "localhost", "otherns", 'ConsulResolver') == "localhost"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert normalize_service_name(ir, "localhost.otherns", None, 'ConsulResolver') == "localhost.otherns"
    assert normalize_service_name(ir, "localhost.otherns", "default", 'ConsulResolver') == "localhost.otherns"
    assert normalize_service_name(ir, "localhost.otherns", "otherns", 'ConsulResolver') == "localhost.otherns"

    assert qualify_service_name(ir, "localhost:80", None) == "localhost:80"
    assert qualify_service_name(ir, "localhost:80", "default") == "localhost:80"
    assert qualify_service_name(ir, "localhost:80", "otherns") == "localhost:80"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "localhost.otherns:80", None) == "localhost.otherns:80"
    assert qualify_service_name(ir, "localhost.otherns:80", "default") == "localhost.otherns:80"
    assert qualify_service_name(ir, "localhost.otherns:80", "otherns") == "localhost.otherns:80"

    assert normalize_service_name(ir, "localhost:80", None, 'ConsulResolver') == "localhost:80"
    assert normalize_service_name(ir, "localhost:80", "default", 'ConsulResolver') == "localhost:80"
    assert normalize_service_name(ir, "localhost:80", "otherns", 'ConsulResolver') == "localhost:80"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert normalize_service_name(ir, "localhost.otherns:80", None, 'ConsulResolver') == "localhost.otherns:80"
    assert normalize_service_name(ir, "localhost.otherns:80", "default", 'ConsulResolver') == "localhost.otherns:80"
    assert normalize_service_name(ir, "localhost.otherns:80", "otherns", 'ConsulResolver') == "localhost.otherns:80"

    assert qualify_service_name(ir, "http://localhost", None) == "http://localhost"
    assert qualify_service_name(ir, "http://localhost", "default") == "http://localhost"
    assert qualify_service_name(ir, "http://localhost", "otherns") == "http://localhost"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "http://localhost.otherns", None) == "http://localhost.otherns"
    assert qualify_service_name(ir, "http://localhost.otherns", "default") == "http://localhost.otherns"
    assert qualify_service_name(ir, "http://localhost.otherns", "otherns") == "http://localhost.otherns"

    assert normalize_service_name(ir, "http://localhost", None, 'ConsulResolver') == "http://localhost"
    assert normalize_service_name(ir, "http://localhost", "default", 'ConsulResolver') == "http://localhost"
    assert normalize_service_name(ir, "http://localhost", "otherns", 'ConsulResolver') == "http://localhost"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert normalize_service_name(ir, "http://localhost.otherns", None, 'ConsulResolver') == "http://localhost.otherns"
    assert normalize_service_name(ir, "http://localhost.otherns", "default", 'ConsulResolver') == "http://localhost.otherns"
    assert normalize_service_name(ir, "http://localhost.otherns", "otherns", 'ConsulResolver') == "http://localhost.otherns"

    assert qualify_service_name(ir, "http://localhost:80", None) == "http://localhost:80"
    assert qualify_service_name(ir, "http://localhost:80", "default") == "http://localhost:80"
    assert qualify_service_name(ir, "http://localhost:80", "otherns") == "http://localhost:80"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "http://localhost.otherns:80", None) == "http://localhost.otherns:80"
    assert qualify_service_name(ir, "http://localhost.otherns:80", "default") == "http://localhost.otherns:80"
    assert qualify_service_name(ir, "http://localhost.otherns:80", "otherns") == "http://localhost.otherns:80"

    assert normalize_service_name(ir, "http://localhost:80", None, 'ConsulResolver') == "http://localhost:80"
    assert normalize_service_name(ir, "http://localhost:80", "default", 'ConsulResolver') == "http://localhost:80"
    assert normalize_service_name(ir, "http://localhost:80", "otherns", 'ConsulResolver') == "http://localhost:80"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert normalize_service_name(ir, "http://localhost.otherns:80", None, 'ConsulResolver') == "http://localhost.otherns:80"
    assert normalize_service_name(ir, "http://localhost.otherns:80", "default", 'ConsulResolver') == "http://localhost.otherns:80"
    assert normalize_service_name(ir, "http://localhost.otherns:80", "otherns", 'ConsulResolver') == "http://localhost.otherns:80"

    assert qualify_service_name(ir, "https://localhost", None) == "https://localhost"
    assert qualify_service_name(ir, "https://localhost", "default") == "https://localhost"
    assert qualify_service_name(ir, "https://localhost", "otherns") == "https://localhost"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "https://localhost.otherns", None) == "https://localhost.otherns"
    assert qualify_service_name(ir, "https://localhost.otherns", "default") == "https://localhost.otherns"
    assert qualify_service_name(ir, "https://localhost.otherns", "otherns") == "https://localhost.otherns"

    assert normalize_service_name(ir, "https://localhost", None, 'ConsulResolver') == "https://localhost"
    assert normalize_service_name(ir, "https://localhost", "default", 'ConsulResolver') == "https://localhost"
    assert normalize_service_name(ir, "https://localhost", "otherns", 'ConsulResolver') == "https://localhost"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert normalize_service_name(ir, "https://localhost.otherns", None, 'ConsulResolver') == "https://localhost.otherns"
    assert normalize_service_name(ir, "https://localhost.otherns", "default", 'ConsulResolver') == "https://localhost.otherns"
    assert normalize_service_name(ir, "https://localhost.otherns", "otherns", 'ConsulResolver') == "https://localhost.otherns"

    assert qualify_service_name(ir, "https://localhost:443", None) == "https://localhost:443"
    assert qualify_service_name(ir, "https://localhost:443", "default") == "https://localhost:443"
    assert qualify_service_name(ir, "https://localhost:443", "otherns") == "https://localhost:443"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert qualify_service_name(ir, "https://localhost.otherns:443", None) == "https://localhost.otherns:443"
    assert qualify_service_name(ir, "https://localhost.otherns:443", "default") == "https://localhost.otherns:443"
    assert qualify_service_name(ir, "https://localhost.otherns:443", "otherns") == "https://localhost.otherns:443"

    assert normalize_service_name(ir, "https://localhost:443", None, 'ConsulResolver') == "https://localhost:443"
    assert normalize_service_name(ir, "https://localhost:443", "default", 'ConsulResolver') == "https://localhost:443"
    assert normalize_service_name(ir, "https://localhost:443", "otherns", 'ConsulResolver') == "https://localhost:443"
    # It's not meaningful to actually say "localhost.otherns", but it should passed through unchanged.
    assert normalize_service_name(ir, "https://localhost.otherns:443", None, 'ConsulResolver') == "https://localhost.otherns:443"
    assert normalize_service_name(ir, "https://localhost.otherns:443", "default", 'ConsulResolver') == "https://localhost.otherns:443"
    assert normalize_service_name(ir, "https://localhost.otherns:443", "otherns", 'ConsulResolver') == "https://localhost.otherns:443"

    assert qualify_service_name(ir, "ambassador://foo.ns", "otherns") == "ambassador://foo.ns" # let's not introduce silly semantics
    assert qualify_service_name(ir, "//foo.ns:1234", "otherns") == "foo.ns:1234" # we tell people "URL-ish", actually support URL-ish
    assert qualify_service_name(ir, "foo.ns:1234", "otherns") == "foo.ns:1234"

    assert normalize_service_name(ir, "ambassador://foo.ns", "otherns", 'ConsulResolver') == "ambassador://foo.ns" # let's not introduce silly semantics
    assert normalize_service_name(ir, "//foo.ns:1234", "otherns", 'ConsulResolver') == "foo.ns:1234" # we tell people "URL-ish", actually support URL-ish
    assert normalize_service_name(ir, "foo.ns:1234", "otherns", 'ConsulResolver') == "foo.ns:1234"

    assert not ir.aconf.errors

    assert qualify_service_name(ir, "https://bad-service:443:443", "otherns") == "https://bad-service:443:443"
    assert qualify_service_name(ir, "https://bad-service:443:443", "otherns", rkey="test-rkey") == "https://bad-service:443:443"
    assert qualify_service_name(ir, "bad-service:443:443", "otherns") == "bad-service:443:443"
    assert qualify_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4:443", "otherns") == "https://[fe80::e022:9cff:fecc:c7c4:443"
    assert qualify_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4", "otherns") == "https://[fe80::e022:9cff:fecc:c7c4"
    assert qualify_service_name(ir, "https://fe80::e022:9cff:fecc:c7c4", "otherns") == "https://fe80::e022:9cff:fecc:c7c4"
    assert qualify_service_name(ir, "https://bad-service:-1", "otherns") == "https://bad-service:-1"
    assert qualify_service_name(ir, "https://bad-service:70000", "otherns") == "https://bad-service:70000"

    assert normalize_service_name(ir, "https://bad-service:443:443", "otherns", 'ConsulResolver') == "https://bad-service:443:443"
    assert normalize_service_name(ir, "https://bad-service:443:443", "otherns", 'ConsulResolver', rkey="test-rkey") == "https://bad-service:443:443"
    assert normalize_service_name(ir, "bad-service:443:443", "otherns", 'ConsulResolver') == "bad-service:443:443"
    assert normalize_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4:443", "otherns", 'ConsulResolver') == "https://[fe80::e022:9cff:fecc:c7c4:443"
    assert normalize_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4", "otherns", 'ConsulResolver') == "https://[fe80::e022:9cff:fecc:c7c4"
    assert normalize_service_name(ir, "https://fe80::e022:9cff:fecc:c7c4", "otherns", 'ConsulResolver') == "https://fe80::e022:9cff:fecc:c7c4"
    assert normalize_service_name(ir, "https://bad-service:-1", "otherns", 'ConsulResolver') == "https://bad-service:-1"
    assert normalize_service_name(ir, "https://bad-service:70000", "otherns", 'ConsulResolver') == "https://bad-service:70000"
    assert qualify_service_name(ir, "https://[fe80::e022:9cff:fecc:c7c4%zone]:443", "other") == "https://[fe80::e022:9cff:fecc:c7c4%zone]:443"

    aconf_errors = ir.aconf.errors
    assert "-global-" in aconf_errors
    errors = aconf_errors["-global-"]

    assert len(errors) == 17

    # Ugg, different versions of Python have different error messages.  Let's recognize the "Port could not be cast to
    # integer value as" to keep pytest working on peoples up-to-date laptops with Python 3.8, and let's recognize
    # "invalid literal for int() with base 10:" for the Python 3.7 in the builder container.
    assert not errors[0]["ok"]
    assert (errors[0]["error"] == "Malformed service 'https://bad-service:443:443': Port could not be cast to integer value as '443:443'" or
            errors[0]["error"] == "Malformed service 'https://bad-service:443:443': invalid literal for int() with base 10: '443:443'")

    assert not errors[1]["ok"]
    assert (errors[1]["error"] == "test-rkey: Malformed service 'https://bad-service:443:443': Port could not be cast to integer value as '443:443'" or
            errors[1]["error"] == "test-rkey: Malformed service 'https://bad-service:443:443': invalid literal for int() with base 10: '443:443'")

    assert not errors[2]["ok"]
    assert (errors[2]["error"] == "Malformed service 'bad-service:443:443': Port could not be cast to integer value as '443:443'" or
            errors[2]["error"] == "Malformed service 'bad-service:443:443': invalid literal for int() with base 10: '443:443'")

    assert not errors[3]["ok"]
    assert errors[3]["error"] == "Malformed service 'https://[fe80::e022:9cff:fecc:c7c4:443': Invalid IPv6 URL"

    assert not errors[4]["ok"]
    assert errors[4]["error"] == "Malformed service 'https://[fe80::e022:9cff:fecc:c7c4': Invalid IPv6 URL"

    assert not errors[5]["ok"]
    assert (errors[5]["error"] == "Malformed service 'https://fe80::e022:9cff:fecc:c7c4': Port could not be cast to integer value as ':e022:9cff:fecc:c7c4'" or
            errors[5]["error"] == "Malformed service 'https://fe80::e022:9cff:fecc:c7c4': invalid literal for int() with base 10: ':e022:9cff:fecc:c7c4'")

    assert not errors[6]["ok"]
    assert errors[6]["error"] == "Malformed service 'https://bad-service:-1': Port out of range 0-65535"

    assert not errors[7]["ok"]
    assert errors[7]["error"] == "Malformed service 'https://bad-service:70000': Port out of range 0-65535"

    assert not errors[8]["ok"]
    assert (errors[8]["error"] == "Malformed service 'https://bad-service:443:443': Port could not be cast to integer value as '443:443'" or
            errors[8]["error"] == "Malformed service 'https://bad-service:443:443': invalid literal for int() with base 10: '443:443'")

    assert not errors[9]["ok"]
    assert (errors[9]["error"] == "test-rkey: Malformed service 'https://bad-service:443:443': Port could not be cast to integer value as '443:443'" or
            errors[9]["error"] == "test-rkey: Malformed service 'https://bad-service:443:443': invalid literal for int() with base 10: '443:443'")

    assert not errors[10]["ok"]
    assert (errors[10]["error"] == "Malformed service 'bad-service:443:443': Port could not be cast to integer value as '443:443'" or
            errors[10]["error"] == "Malformed service 'bad-service:443:443': invalid literal for int() with base 10: '443:443'")

    assert not errors[11]["ok"]
    assert errors[11]["error"] == "Malformed service 'https://[fe80::e022:9cff:fecc:c7c4:443': Invalid IPv6 URL"

    assert not errors[12]["ok"]
    assert errors[12]["error"] == "Malformed service 'https://[fe80::e022:9cff:fecc:c7c4': Invalid IPv6 URL"

    assert not errors[13]["ok"]
    assert (errors[13]["error"] == "Malformed service 'https://fe80::e022:9cff:fecc:c7c4': Port could not be cast to integer value as ':e022:9cff:fecc:c7c4'" or
            errors[13]["error"] == "Malformed service 'https://fe80::e022:9cff:fecc:c7c4': invalid literal for int() with base 10: ':e022:9cff:fecc:c7c4'")

    assert not errors[14]["ok"]
    assert errors[14]["error"] == "Malformed service 'https://bad-service:-1': Port out of range 0-65535"

    assert not errors[15]["ok"]
    assert errors[15]["error"] == "Malformed service 'https://bad-service:70000': Port out of range 0-65535"

    assert not errors[16]["ok"]
    assert errors[16]["error"] == "Malformed service 'https://[fe80::e022:9cff:fecc:c7c4%zone]:443': Invalid percent-escape in hostname: %zo"

if __name__ == '__main__':
    pytest.main(sys.argv)
