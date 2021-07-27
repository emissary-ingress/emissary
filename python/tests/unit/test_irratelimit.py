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
# logger.setLevel(logging.DEBUG)

from ambassador import Config, IR, EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler

from tests.utils import default_listener_manifests

SERVICE_NAME = 'coolsvcname'


def _get_rl_config(yaml):
    for listener in yaml['static_resources']['listeners']:
        for filter_chain in listener['filter_chains']:
            for f in filter_chain['filters']:
                for http_filter in f['typed_config']['http_filters']:
                    if http_filter['name'] == 'envoy.filters.http.ratelimit':
                        return http_filter
    return False


def _get_envoy_config(yaml, version='V2'):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(default_listener_manifests() + yaml, k8s=True)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir
    return EnvoyConfig.generate(ir, version)


def _get_ratelimit_default_conf_v3():
    return {
        '@type': 'type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit',
        'domain': 'ambassador',
        'request_type': 'both',
        'timeout': '0.020s',
        'rate_limit_service': {
            'transport_api_version': 'V2',
            'grpc_service': {
                'envoy_grpc': {
                    'cluster_name': 'cluster_{}_default'.format(SERVICE_NAME)
                }
            }
        }
    }


def _get_ratelimit_default_conf_v2():
    return {
        '@type': 'type.googleapis.com/envoy.config.filter.http.rate_limit.v2.RateLimit',
        'domain': 'ambassador',
        'request_type': 'both',
        'timeout': '0.020s',
        'rate_limit_service': {
            'grpc_service': {
                'envoy_grpc': {
                    'cluster_name': 'cluster_{}_default'.format(SERVICE_NAME)
                }
            }
        }
    }


@pytest.mark.compilertest
def test_irratelimit_defaultsv3():
    default_config = _get_ratelimit_default_conf_v3()

    # Test all defaults
    yaml = """
apiVersion: getambassador.io/v2
kind: RateLimitService
metadata:
  name: myrls
  namespace: default
spec:
  service: {}
""".format(SERVICE_NAME)
    econf = _get_envoy_config(yaml, version='V3')
    conf = _get_rl_config(econf.as_dict())

    assert conf

    assert conf.get('typed_config') == default_config


@pytest.mark.compilertest
def test_irratelimit_defaults():
    default_config = _get_ratelimit_default_conf_v2()

    # Test all defaults
    yaml = """
apiVersion: getambassador.io/v2
kind: RateLimitService
metadata:
  name: myrls
  namespace: default
spec:
  service: {}
""".format(SERVICE_NAME)
    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert conf

    assert conf.get('typed_config') == default_config


@pytest.mark.compilertest
def test_irratelimit_grpcsvc_version_v3():
    # Test protocol_version override
    yaml = """
---
apiVersion: getambassador.io/v2
kind: RateLimitService
metadata:
  name: myrls
  namespace: default
spec:
  service: {}
  protocol_version: "v3"
""".format(SERVICE_NAME)
    config = _get_ratelimit_default_conf_v3()
    config['rate_limit_service']['transport_api_version'] = 'V3'

    econf = _get_envoy_config(yaml, version='V3')
    conf = _get_rl_config(econf.as_dict())

    assert conf

    assert conf.get('typed_config') == config


@pytest.mark.compilertest
def test_irratelimit_grpcsvc_version_v2():
    # Test protocol_version override
    yaml = """
---
apiVersion: getambassador.io/v2
kind: RateLimitService
metadata:
  name: myrls
  namespace: default
spec:
  service: {}
  protocol_version: "v2"
""".format(SERVICE_NAME)
    config = _get_ratelimit_default_conf_v2()
    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert conf

    assert conf.get('typed_config') == config


@pytest.mark.compilertest
def test_irratelimit_error():
    # Test error no svc name
    yaml = """
---
apiVersion: getambassador.io/v2
kind: RateLimitService
metadata:
  name: myrls
  namespace: default
spec: {}
"""
    econf = _get_envoy_config(yaml)
    conf = _get_rl_config(econf.as_dict())

    assert not conf


@pytest.mark.compilertest
def test_irratelimit_error_v3():
    # Test error no svc name
    yaml = """
---
apiVersion: getambassador.io/v2
kind: RateLimitService
metadata:
  name: myrls
  namespace: default
spec: {}
"""
    econf = _get_envoy_config(yaml, version='V3')
    conf = _get_rl_config(econf.as_dict())

    assert not conf


@pytest.mark.compilertest
def test_irratelimit_overrides():

    # Test all other overrides
    config = _get_ratelimit_default_conf_v2()
    yaml = """
---
apiVersion: getambassador.io/v2
kind: RateLimitService
metadata:
  name: myrls
  namespace: someotherns
spec:
  service: {}
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


@pytest.mark.compilertest
def test_irratelimit_overrides_v3():

    # Test all other overrides
    config = _get_ratelimit_default_conf_v3()
    yaml = """
---
apiVersion: getambassador.io/v2
kind: RateLimitService
metadata:
  name: myrls
  namespace: someotherns
spec:
  service: {}
  domain: otherdomain
  timeout_ms: 500
  tls: rl-tls-context
  protocol_version: v2
""".format(SERVICE_NAME)
    config['rate_limit_service']['grpc_service']['envoy_grpc']['cluster_name'] = 'cluster_{}_someotherns'.format(SERVICE_NAME)
    config['timeout'] = '0.500s'
    config['domain'] = 'otherdomain'

    econf = _get_envoy_config(yaml, version='V3')
    conf = _get_rl_config(econf.as_dict())

    assert conf
    assert conf.get('typed_config') == config
