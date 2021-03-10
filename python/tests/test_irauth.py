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

from ambassador import Config, IR, EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler


def _get_ext_auth_config(yaml):
    for listener in yaml['static_resources']['listeners']:
        for filter_chain in listener['filter_chains']:
            for f in filter_chain['filters']:
                for http_filter in f['typed_config']['http_filters']:
                    if http_filter['name'] == 'envoy.filters.http.ext_authz':
                        return http_filter
    return False


def _get_envoy_config(yaml):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir

    return EnvoyConfig.generate(ir, "V2")


def test_irauth_grpcservice_version_v2():
    yaml = """
---
apiVersion: ambassador/v2
kind: AuthService
name:  mycoolauthservice
auth_service: someservice
protocol_version: "v2"
proto: grpc
"""
    econf = _get_envoy_config(yaml)

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    assert ext_auth_config

    assert ext_auth_config['typed_config']['grpc_service']['envoy_grpc']['cluster_name'] == 'cluster_extauth_someservice_default'
    assert not ext_auth_config['typed_config']['use_alpha']


def test_irauth_grpcservice_version_v2alpha():
    yaml = """
---
apiVersion: ambassador/v2
kind: AuthService
name:  mycoolauthservice
auth_service: someservice
protocol_version: "v2alpha"
proto: grpc
"""
    econf = _get_envoy_config(yaml)

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    assert ext_auth_config

    assert ext_auth_config['typed_config']['grpc_service']['envoy_grpc']['cluster_name'] == 'cluster_extauth_someservice_default'
    assert ext_auth_config['typed_config']['use_alpha']


def test_irauth_grpcservice_version_default():
    yaml = """
---
apiVersion: ambassador/v2
kind: AuthService
name:  mycoolauthservice
auth_service: someservice
proto: grpc
"""
    econf = _get_envoy_config(yaml)

    conf = econf.as_dict()
    ext_auth_config = _get_ext_auth_config(conf)

    assert ext_auth_config

    assert ext_auth_config['typed_config']['grpc_service']['envoy_grpc']['cluster_name'] == 'cluster_extauth_someservice_default'
    assert not ext_auth_config['typed_config']['use_alpha']
