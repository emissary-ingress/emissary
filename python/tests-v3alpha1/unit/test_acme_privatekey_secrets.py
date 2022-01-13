from typing import Any, Dict, List, Optional, Union, TextIO, TYPE_CHECKING

import logging
import os

import hashlib
import json

import pytest

logging.basicConfig(
    level=logging.DEBUG,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")
logger.setLevel(logging.DEBUG)

from ambassador import Config, IR
from ambassador.fetch import ResourceFetcher
from ambassador.utils import SecretHandler, SecretInfo, SavedSecret

if TYPE_CHECKING:
    from ambassador.ir import IRResource # pragma: no cover

# MemorySecretHandler is a degenerate SecretHandler that doesn't actually
# cache anything to disk. It will never load a secret that isn't already
# in the aconf.
class MemorySecretHandler (SecretHandler):
    def cache_internal(self, name: str, namespace: str,
                       tls_crt: Optional[str], tls_key: Optional[str],
                       user_key: Optional[str], root_crt: Optional[str]) -> SavedSecret:
        # This is mostly ripped from ambassador.utils.SecretHandler.cache_internal,
        # just without actually saving anything.
        tls_crt_path = None
        tls_key_path = None
        user_key_path = None
        root_crt_path = None
        cert_data = None

        # Don't save if it has neither a tls_crt or a user_key or the root_crt
        if tls_crt or user_key or root_crt:
            h = hashlib.new('sha1')

            for el in [tls_crt, tls_key, user_key]:
                if el:
                    h.update(el.encode('utf-8'))

            fp = h.hexdigest().upper()

            if tls_crt:
                tls_crt_path = f"//test-secret-{fp}.crt"

            if tls_key:
                tls_key_path = f"//test-secret-{fp}.key"

            if user_key:
                user_key_path = f"//test-secret-{fp}.user"

            if root_crt:
                root_crt_path = f"//test-secret-{fp}.root.crt"

            cert_data = {
                'tls_crt': tls_crt,
                'tls_key': tls_key,
                'user_key': user_key,
                'root_crt': root_crt,
            }

            self.logger.debug(f"saved secret {name}.{namespace}: {tls_crt_path}, {tls_key_path}, {root_crt_path}")

        return SavedSecret(name, namespace, tls_crt_path, tls_key_path, user_key_path, root_crt_path, cert_data)


def _get_ir_config(watt):
    aconf = Config(logger=logger)
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_watt(watt)
    aconf.load_all(fetcher.sorted())

    secret_handler = MemorySecretHandler(logger, "/tmp/unit-test-source-root", "/tmp/unit-test-cache-dir", "0")
    ir = IR(aconf, logger=logger, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir
    return aconf, ir


@pytest.mark.compilertest
def test_acme_privatekey_secrets():
    test_data_dir = os.path.join(
        os.path.dirname(os.path.abspath(__file__)),
        "test_general_data"
    )

    test_data_file = os.path.join(test_data_dir, "test-acme-private-key-snapshot.json")
    watt_data = open(test_data_file).read()

    aconf, ir = _get_ir_config(watt_data)

    # Remember, you'll see no log output unless the test fails!
    logger.debug("---- ACONF")
    logger.debug(json.dumps(aconf.as_dict(), indent=2, sort_keys=True))
    logger.debug("---- IR")
    logger.debug(json.dumps(ir.as_dict(), indent=2, sort_keys=True))

    assert not aconf.errors, "Wanted no errors but got:\n    %s" % "\n    ".join(aconf.errors)
