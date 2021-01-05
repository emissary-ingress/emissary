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

SERVICE_NAME = 'coolsvcname'


def _get_rl_config(yaml):
    for listener in yaml['static_resources']['listeners']:
        for filter_chain in listener['filter_chains']:
            for f in filter_chain['filters']:
                for http_filter in f['typed_config']['http_filters']:
                    if http_filter['name'] == 'envoy.filters.http.ratelimit':
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



def _get_ratelimit_default_conf():
    return {
        '@type': 'type.googleapis.com/envoy.config.filter.http.rate_limit.v2.RateLimit',
        'domain': 'ambassador',
        'request_type': 'both',
        'timeout': '0.020s',
        'rate_limit_service': {
            'use_alpha': False,
            'grpc_service': {
                'envoy_grpc': {
                    'cluster_name': 'cluster_{}_default'.format(SERVICE_NAME)
                }
            }
        }
    }


def test_irratelimit_defaults():
    default_config = _get_ratelimit_default_conf()

    # Test all defaults
    yaml = """
apiVersion: ambassador/v2
kind: RateLimitService
name: myrls
service: {}
""".format(SERVICE_NAME)
    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert conf

    assert conf.get('typed_config') == default_config


def test_irratelimit_grpcsvc_version_v2():
    # Test protocol_version override
    yaml = """
---
apiVersion: ambassador/v2
kind: RateLimitService
name: myrls
service: {}
protocol_version: "v2"
""".format(SERVICE_NAME)
    config = _get_ratelimit_default_conf()
    config['rate_limit_service']['use_alpha'] = False

    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert conf

    assert conf.get('typed_config') == config


def test_irratelimit_grpcsvc_version_v2alpha():
    # Test protocol_version override
    yaml = """
---
apiVersion: ambassador/v2
kind: RateLimitService
name: myrls
service: {}
protocol_version: "v2alpha"
""".format(SERVICE_NAME)
    config = _get_ratelimit_default_conf()
    config['rate_limit_service']['use_alpha'] = True

    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert conf

    assert conf.get('typed_config') == config


def test_irratelimit_error():
    # Test error no svc name
    yaml = """
---
apiVersion: ambassador/v2
kind: RateLimitService
name: myrls
"""
    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert not conf

def test_irratelimit_overrides():

    # Test all other overrides
    config = _get_ratelimit_default_conf()
    yaml = """
---
apiVersion: ambassador/v2
kind: RateLimitService
name: myrls
service: {}
namespace: someotherns
domain: otherdomain
timeout_ms: 500
tls: rl-tls-context
protocol_version: v2
""".format(SERVICE_NAME)
    config['rate_limit_service']['grpc_service']['envoy_grpc']['cluster_name'] = 'cluster_{}_someotherns'.format(SERVICE_NAME)
    config['timeout'] = '0.500s'
    config['domain'] = 'otherdomain'

    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert conf
    assert conf.get('typed_config') == config
