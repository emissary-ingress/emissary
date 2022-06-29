from typing import Optional, Tuple

import hashlib
import io
import json
import logging
import os

import pytest

from ambassador import Config, IR
from ambassador.fetch import ResourceFetcher
from ambassador.utils import SecretHandler, SavedSecret

# MemorySecretHandler is a degenerate SecretHandler that doesn't actually
# cache anything to disk. It will never load a secret that isn't already
# in the aconf.
class MemorySecretHandler(SecretHandler):
    def cache_internal(
        self,
        name: str,
        namespace: str,
        tls_crt: Optional[str],
        tls_key: Optional[str],
        user_key: Optional[str],
        root_crt: Optional[str],
    ) -> SavedSecret:
        # This is mostly ripped from ambassador.utils.SecretHandler.cache_internal,
        # just without actually saving anything.
        tls_crt_path = None
        tls_key_path = None
        user_key_path = None
        root_crt_path = None
        cert_data = None

        # Don't save if it has neither a tls_crt or a user_key or the root_crt
        if tls_crt or user_key or root_crt:
            h = hashlib.new("sha1")

            for el in [tls_crt, tls_key, user_key]:
                if el:
                    h.update(el.encode("utf-8"))

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
                "tls_crt": tls_crt,
                "tls_key": tls_key,
                "user_key": user_key,
                "root_crt": root_crt,
            }

            self.logger.debug(
                f"saved secret {name}.{namespace}: {tls_crt_path}, {tls_key_path}, {root_crt_path}"
            )

        return SavedSecret(
            name, namespace, tls_crt_path, tls_key_path, user_key_path, root_crt_path, cert_data
        )


def _get_config_and_ir(logger: logging.Logger, watt: str) -> Tuple[Config, IR]:
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_watt(watt)
    aconf.load_all(fetcher.sorted())

    secret_handler = MemorySecretHandler(
        logger, "/tmp/unit-test-source-root", "/tmp/unit-test-cache-dir", "0"
    )
    ir = IR(aconf, logger=logger, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir
    return aconf, ir


def _get_errors(caplog: pytest.LogCaptureFixture, logger_name: str, watt_data_filename: str):
    watt_data = open(watt_data_filename).read()

    aconf, ir = _get_config_and_ir(logging.getLogger(logger_name), watt_data)

    log_errors = [
        rec for rec in caplog.record_tuples if rec[0] == logger_name and rec[1] > logging.INFO
    ]

    aconf_errors = aconf.errors
    if "-global-" in aconf_errors:
        # We expect some global errors related to us not being a real Emissary instance, such as
        # "Pod labels are not mounted in the container".  Ignore those.
        del aconf_errors["-global-"]

    return log_errors, aconf_errors


@pytest.mark.compilertest
def test_acme_privatekey_secrets(caplog: pytest.LogCaptureFixture):
    caplog.set_level(logging.DEBUG)

    nl = "\n"
    tab = "\t"

    # What this test is really about is ensuring that test-acme-private-key-snapshot.json doesn't
    # emit any errors.  But, in order to validate the test itself and ensure that the test is
    # checking for errors in the correct place, we'll also run against a bad version of that file
    # and check that we *do* see errors.

    badsnap_log_errors, badsnap_aconf_errors = _get_errors(
        caplog,
        "test_acme_privatekey_secrets-bad",
        os.path.join(
            os.path.dirname(os.path.abspath(__file__)),
            "test_general_data",
            "test-acme-private-key-snapshot-bad.json",
        ),
    )
    assert badsnap_log_errors
    assert not badsnap_aconf_errors, "Wanted no aconf errors but got:%s" % "".join(
        [f"{nl}    {err}" for err in badsnap_aconf_errors]
    )

    goodsnap_log_errors, goodsnap_aconf_errors = _get_errors(
        caplog,
        "test_acme_privatekey_secrets",
        os.path.join(
            os.path.dirname(os.path.abspath(__file__)),
            "test_general_data",
            "test-acme-private-key-snapshot.json",
        ),
    )
    assert not goodsnap_log_errors, "Wanted no logged errors bug got:%s" % "".join(
        [
            f"{nl}    {logging.getLevelName(rec[1])}{tab}{rec[0]}:{rec[2]}"
            for rec in goodsnap_log_errors
        ]
    )
    assert not goodsnap_aconf_errors, "Wanted no aconf errors but got:%s" % "".join(
        [f"{nl}    {err}" for err in goodsnap_aconf_errors]
    )
