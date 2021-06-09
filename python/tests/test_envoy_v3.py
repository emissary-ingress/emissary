import copy
import logging
import sys
import json
import os
from typing import Optional
from utils import assert_valid_envoy_config

import pytest

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador import Config, IR, EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler, SecretHandler, SecretInfo


testfolder = os.path.join(os.path.dirname(os.path.abspath(__file__)), 'test_envoy_v3')
testdata = [ f.name for f in os.scandir(testfolder) if f.is_dir() ]


class MockSecretHandler(SecretHandler):
    def load_secret(self, resource: 'IRResource', secret_name: str, namespace: str) -> Optional[SecretInfo]:
        if ((secret_name == "fallback-self-signed-cert") and
            (namespace == Config.ambassador_namespace)):
            return SecretInfo(secret_name, namespace, "mocked-fallback-secret",
                              "-fallback-cert-", "-fallback-key-", decode_b64=False)


def teardown_function(function):
    os.environ['AMBASSADOR_ID'] = 'default'
    os.environ['AMBASSADOR_NAMESPACE'] = 'default'
    Config.ambassador_id = 'default'
    Config.ambassador_namespace = 'default'


def _test_compiler(test_name, envoy_api_version="V2"):
    test_path = os.path.join(testfolder, test_name)
    basename = os.path.basename(test_path)

    with open(os.path.join(test_path, 'bootstrap-ads.json'), 'r') as f:
        expected_bootstrap = json.loads(f.read())
    node_name = expected_bootstrap.get('node', {}).get('cluster', None)
    assert node_name
    namespace = node_name.replace(test_name + '-', '', 1)

    with open(os.path.join(test_path, 'snapshot.yaml'), 'r') as f:
        watt = f.read()

    Config.ambassador_id = basename
    Config.ambassador_namespace = namespace
    os.environ['AMBASSADOR_ID'] = basename
    os.environ['AMBASSADOR_NAMESPACE'] = namespace
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_watt(watt)

    aconf.load_all(fetcher.sorted())

    secret_handler = MockSecretHandler(logger, "mockery", "/tmp/ambassador/snapshots", "v1")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)
    expected_econf_file = 'econf.json'
    if ir.edge_stack_allowed:
        expected_econf_file = 'econf-aes.json'
    with open(os.path.join(test_path, expected_econf_file), 'r') as f:
        expected_econf = json.loads(f.read())

    assert ir
    econf = EnvoyConfig.generate(ir, version=envoy_api_version)
    bootstrap_config, ads_config, _ = econf.split_config()
    ads_config.pop('@type', None)

    assert bootstrap_config == expected_bootstrap
    assert ads_config == expected_econf
    assert_valid_envoy_config(ads_config)
    assert_valid_envoy_config(bootstrap_config)

# @pytest.mark.compilertest
# @pytest.mark.parametrize("test_name", testdata)
# def test_compiler_v3(test_name):
#     _test_compiler(test_name, envoy_api_version="V3")