from typing import Optional, TYPE_CHECKING

import logging
from pathlib import Path

import pytest

from tests.selfsigned import TLSCerts
from tests.utils import assert_valid_envoy_config

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")

from ambassador import Config, IR
from ambassador.envoy import EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import SecretHandler, SecretInfo

if TYPE_CHECKING:
    from ambassador.ir.irresource import IRResource # pragma: no cover

class MockSecretHandler(SecretHandler):
    def load_secret(self, resource: 'IRResource', secret_name: str, namespace: str) -> Optional[SecretInfo]:
            return SecretInfo('fallback-self-signed-cert', 'ambassador', "mocked-fallback-secret",
                              TLSCerts["acook"].pubcert, TLSCerts["acook"].privkey, decode_b64=False)


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
def test_shadow(tmp_path: Path):
    aconf = Config()

    yaml = '''
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-listener-8080
  namespace: default
spec:
  port: 8080
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: httpbin-mapping
  namespace: default
spec:
  service: httpbin
  hostname: "*"
  prefix: /httpbin/
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: httpbin-mapping-shadow
  namespace: default
spec:
  service: httpbin-shadow
  hostname: "*"
  prefix: /httpbin/
  shadow: true
  weight: 10
'''
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml, k8s=True)

    aconf.load_all(fetcher.sorted())


    secret_handler = MockSecretHandler(logger, "mockery", str(tmp_path/"ambassador"/"snapshots"), "v1")
    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir

    econf = EnvoyConfig.generate(ir)

    bootstrap_config, ads_config, _ = econf.split_config()
    ads_config.pop('@type', None)

    mirrored_config = get_mirrored_config(ads_config)
    assert 'request_mirror_policies' in mirrored_config['route']
    assert len(mirrored_config['route']['request_mirror_policies']) == 1
    mirror_policy = mirrored_config['route']['request_mirror_policies'][0]
    assert mirror_policy['cluster'] == 'cluster_shadow_httpbin_shadow_default'
    assert mirror_policy['runtime_fraction']['default_value']['numerator'] == 10
    assert mirror_policy['runtime_fraction']['default_value']['denominator'] == 'HUNDRED'
    assert_valid_envoy_config(ads_config, extra_dirs=[str(tmp_path/"ambassador"/"snapshots")])
    assert_valid_envoy_config(bootstrap_config, extra_dirs=[str(tmp_path/"ambassador"/"snapshots")])
