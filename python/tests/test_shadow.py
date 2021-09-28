import json
import logging
import time
import sys
from typing import Optional

import pytest
import requests

from utils import assert_valid_envoy_config

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador import Config, IR
from ambassador.ir.irerrorresponse import IRErrorResponse
from ambassador.envoy import EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler, SecretHandler, SecretInfo


CRT = '''
-----BEGIN CERTIFICATE-----
MIIF7zCCA9egAwIBAgIUBj+Xwyen6cj/bUWUsvYl+kP1m6MwDQYJKoZIhvcNAQEL
BQAwgYYxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJOWTENMAsGA1UEBwwEY2l0eTER
MA8GA1UECgwIZGF0YXdpcmUxFDASBgNVBAsMC2VuZ2luZWVyaW5nMQ4wDAYDVQQD
DAVhY29vazEiMCAGCSqGSIb3DQEJARYTZGV2bnVsbEBkYXRhd2lyZS5pbzAeFw0y
MTAxMjgxOTQ2NTBaFw0zMTAxMjYxOTQ2NTBaMIGGMQswCQYDVQQGEwJVUzELMAkG
A1UECAwCTlkxDTALBgNVBAcMBGNpdHkxETAPBgNVBAoMCGRhdGF3aXJlMRQwEgYD
VQQLDAtlbmdpbmVlcmluZzEOMAwGA1UEAwwFYWNvb2sxIjAgBgkqhkiG9w0BCQEW
E2Rldm51bGxAZGF0YXdpcmUuaW8wggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIK
AoICAQDR8KsgN3WrcsLtJ9gzXF4oCeEk920LSdbET0elyak1XAyi/SKDRow4VhBp
dbrF763j0e6e7d3qoRK48kCyZWoi3RRCfp3o4ZpmAi1sByrMY2SXEAQ2bg8Z2njn
H6m7zIK9ZNK+ovF9FZk7V7lytMVLROyKTz9tAcTlsWz2bBmpRStEAramHmcjGJc7
1hSalPY4UKfU7U2J6fGu0AVqxWyf0bJdyCjQcbhO/FfZc0ZDJpdyP1S1UcL77BGy
JSSrrwS6Xb9oSMaUcl9EEiFGKuEle5VNDRoPPWF9B8Rnj6kn+7eQWA8u7FBcGKAK
JH7orfLYrzCIDYSgnF9fJpw4AwZkgFiEz/sjj6tNZ0m8LE/uqxAwWHC7LmpaQJrd
UiW38q0TtMNKOCaUQ3Tn7zNRyEYPXJEJTc00ZmkwIgELLL6aZnNuNdeYXWODVV6H
KBxI9X0OvYb3eDV023gLXsDyrNgQmXjKEU0rgL6Iw1lH8UyImr2XezqEidvDgCfv
JUQKRw/oU92I3SFaLmN2uC4hX8+zp7oBJhOAxtp0LHJbeGxsfTDBwxwlY84A4Yqw
y0dnC/T7mof7ugW9GrYgobiFiI3iOEeVoFVrqurEMj2ek+af+N19ZWxOqiqjVwjG
qqNP18CmERe0hMWlibeMQ598u5AXw39mKjwGSx33KUBNchglZwIDAQABo1MwUTAd
BgNVHQ4EFgQUPpghohVuKCxZf3828hfRt01kLyYwHwYDVR0jBBgwFoAUPpghohVu
KCxZf3828hfRt01kLyYwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOC
AgEAPCiWBuEgkUCx6+t4pZ/3uCcQ155MYtPRTY+UZdZ8dfyZfYzmdyF7A9x6yDBY
3yQj1Jyd1BV8zfmsN7O+3aRSOMPhadzmW5Gk3m6Rwcn2R6Cepg/cMw58ODHbePNd
zsEndFOQ+YA6UJ5G8aTpyMOqcLjD0Uw7wfGV8ZoY55fNT7EzKQAaqNhZHHoI1pNX
heOPOzUikWc+SYPsfHwSU0FJl4QO6HZaO4xJtOZIbTD/uWStG8ZUq1lE5LbrVbaH
Lece/A084SI7dBkagHve6xLtLBd4bOxiXDyPgD3oIIJWcHEGZBzp99npX4jGr0Z/
CbyfixtGVWRRIhnu1AKBZ3TL9FRCujIrYplzaFbEUiLeIO1sUT1AmS4eUjTdLF55
+HcsxpMU6O2XEng/bw2rbdzQNUKCsgwCEcCfY5GTcrzX9dHQeVeRVXVLBwS08SUg
73ZEklr62w8XoXsjik5AZ30cDCTe3FGJ+O6ziBIv1NHVM9+TUkfH4mTCxbvnRJsP
4WmBh2ZqgNdCJfPjJzPB5wLqq6heVH1hwp7o2oNJcj/XirKt2KN1i8Hyl+5qpS5s
ipaO1QrTqrs3G1dea3L47NK+oRlOYJ01CLVV40xvS7ZT9Tz5dXnmhL4rSB0vYGlA
TYF3xLsQoUdmx8dQuiKUJ8NsaJkpj6QV9Bi5/LzIVvhB4Rw=
-----END CERTIFICATE-----
'''

KEY = '''
-----BEGIN PRIVATE KEY-----
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
NU56R894ytDCPGO6gcVCix8bhdSn/R0=
-----END PRIVATE KEY-----
'''


class MockSecretHandler(SecretHandler):
    def load_secret(self, resource: 'IRResource', secret_name: str, namespace: str) -> Optional[SecretInfo]:
            return SecretInfo('fallback-self-signed-cert', 'ambassador', "mocked-fallback-secret",
                              CRT, KEY, decode_b64=False)


def get_mirrored_config(ads_config):
    for l in ads_config.get('static_resources', {}).get('listeners'):
        for fc in l.get('filter_chains'):
            for f in fc.get('filters'):
                for vh in f['typed_config']['route_config']['virtual_hosts']:
                    for r in vh.get('routes'):
                        if r['match']['prefix'] == '/httpbin/':
                            return r
    return None


@pytest.mark.compilertest
def test_shadow_v3():
    aconf = Config()

    yaml = '''
---
apiVersion: getambassador.io/v2
kind: Mapping
name: httpbin-mapping
service: httpbin
host: "*"
prefix: /httpbin/
---
apiVersion: getambassador.io/v2
kind: Mapping
name: httpbin-mapping-shadow
service: httpbin-shadow
host: "*"
prefix: /httpbin/
shadow: true
weight: 10
'''
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)

    aconf.load_all(fetcher.sorted())


    secret_handler = MockSecretHandler(logger, "mockery", "/tmp/ambassador/snapshots", "v1")
    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir

    econf = EnvoyConfig.generate(ir, "V3")

    bootstrap_config, ads_config, _ = econf.split_config()
    ads_config.pop('@type', None)

    mirrored_config = get_mirrored_config(ads_config)
    assert 'request_mirror_policies' in mirrored_config['route']
    assert len(mirrored_config['route']['request_mirror_policies']) == 1
    mirror_policy = mirrored_config['route']['request_mirror_policies'][0]
    assert mirror_policy['cluster'] == 'cluster_shadow_httpbin_shadow_default'
    assert mirror_policy['runtime_fraction']['default_value']['numerator'] == 10
    assert mirror_policy['runtime_fraction']['default_value']['denominator'] == 'HUNDRED'
    assert_valid_envoy_config(ads_config)
    assert_valid_envoy_config(bootstrap_config)


@pytest.mark.compilertest
def test_shadow_v2():
    aconf = Config()

    yaml = '''
---
apiVersion: getambassador.io/v2
kind: Mapping
name: httpbin-mapping
service: httpbin
host: "*"
prefix: /httpbin/
---
apiVersion: getambassador.io/v2
kind: Mapping
name: httpbin-mapping-shadow
service: httpbin-shadow
host: "*"
prefix: /httpbin/
shadow: true
weight: 10
'''
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml)

    aconf.load_all(fetcher.sorted())


    secret_handler = MockSecretHandler(logger, "mockery", "/tmp/ambassador/snapshots", "v1")
    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir

    econf = EnvoyConfig.generate(ir, "V2")

    bootstrap_config, ads_config, _ = econf.split_config()
    ads_config.pop('@type', None)

    mirrored_config = get_mirrored_config(ads_config)
    assert 'request_mirror_policy' in mirrored_config['route']
    mirror_policy = mirrored_config['route']['request_mirror_policy']
    assert mirror_policy['cluster'] == 'cluster_shadow_httpbin_shadow_default'
    assert mirror_policy['runtime_fraction']['default_value']['numerator'] == 10
    assert mirror_policy['runtime_fraction']['default_value']['denominator'] == 'HUNDRED'
    assert_valid_envoy_config(ads_config)
    assert_valid_envoy_config(bootstrap_config)
