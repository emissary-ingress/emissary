import logging

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

from tests.utils import default_listener_manifests

def _get_envoy_config(yaml, version='V3'):
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(default_listener_manifests() + yaml, k8s=True)
    aconf.load_all(fetcher.sorted())
    secret_handler = NullSecretHandler(logger, None, None, "0")
    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)

    assert ir

    econf = EnvoyConfig.generate(ir, version)
    assert econf, "could not create an econf"

    return econf


@pytest.mark.compilertest
def test_setting_buffer_limit():
    yaml = """
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    buffer_limit_bytes: 5242880
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: ambassador
  namespace: default
spec:
  prefix: /test/
  hostname: "*"
  service: test:9999
"""
    econf = _get_envoy_config(yaml, version='V2')
    expected = 5242880
    key_found = False

    conf = econf.as_dict()

    for listener in conf['static_resources']['listeners']:
        per_connection_buffer_limit_bytes = listener.get('per_connection_buffer_limit_bytes', None)
        assert per_connection_buffer_limit_bytes is not None, \
            f"per_connection_buffer_limit_bytes not found on listener: {listener.name}"
        print(f"Found per_connection_buffer_limit_bytes = {per_connection_buffer_limit_bytes}")
        key_found = True
        assert expected == int(per_connection_buffer_limit_bytes), \
            "per_connection_buffer_limit_bytes must equal the value set on the ambassador Module"
    assert key_found, 'per_connection_buffer_limit_bytes must be found in the envoy config'


@pytest.mark.compilertest
def test_setting_buffer_limit_V3():
    yaml = """
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    buffer_limit_bytes: 5242880
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: ambassador
  namespace: default
spec:
  prefix: /test/
  hostname: "*"
  service: test:9999


"""
    econf = _get_envoy_config(yaml)
    expected = 5242880
    key_found = False

    conf = econf.as_dict()

    for listener in conf['static_resources']['listeners']:
        per_connection_buffer_limit_bytes = listener.get('per_connection_buffer_limit_bytes', None)
        assert per_connection_buffer_limit_bytes is not None, \
            f"per_connection_buffer_limit_bytes not found on listener: {listener.name}"
        print(f"Found per_connection_buffer_limit_bytes = {per_connection_buffer_limit_bytes}")
        key_found = True
        assert expected == int(per_connection_buffer_limit_bytes), \
            "per_connection_buffer_limit_bytes must equal the value set on the ambassador Module"
    assert key_found, 'per_connection_buffer_limit_bytes must be found in the envoy config'

# Tests that the default value of per_connection_buffer_limit_bytes is disabled when there is not Module config for it.
@pytest.mark.compilertest
def test_default_buffer_limit():
    yaml = """
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: ambassador
  namespace: default
spec:
  prefix: /test/
  hostname: "*"
  service: test:9999
"""
    econf = _get_envoy_config(yaml, version='V2')

    conf = econf.as_dict()

    for listener in conf['static_resources']['listeners']:
        per_connection_buffer_limit_bytes = listener.get('per_connection_buffer_limit_bytes', None)
        assert per_connection_buffer_limit_bytes is None, \
            f"per_connection_buffer_limit_bytes found on listener (should not exist unless configured in the module): {listener.name}"


@pytest.mark.compilertest
def test_default_buffer_limit_V3():
    yaml = """
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: ambassador
  namespace: default
spec:
  prefix: /test/
  hostname: "*"
  service: test:9999
"""
    econf = _get_envoy_config(yaml)

    conf = econf.as_dict()

    for listener in conf['static_resources']['listeners']:
        per_connection_buffer_limit_bytes = listener.get('per_connection_buffer_limit_bytes', None)
        assert per_connection_buffer_limit_bytes is None, \
            f"per_connection_buffer_limit_bytes found on listener (should not exist unless configured in the module): {listener.name}" 

# Tests that the default value of per_connection_buffer_limit_bytes is disabled when there is not Module config for it (and that there are no issues when we dont make a listener).
@pytest.mark.compilertest
def test_buffer_limit_no_listener():
    yaml = """
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: ambassador
  namespace: default
spec:
  prefix: /test/
  hostname: "*"
  service: test:9999
"""
    econf = _get_envoy_config(yaml, version='V2')

    conf = econf.as_dict()

    for listener in conf['static_resources']['listeners']:
        per_connection_buffer_limit_bytes = listener.get('per_connection_buffer_limit_bytes', None)
        assert per_connection_buffer_limit_bytes is None, \
            f"per_connection_buffer_limit_bytes found on listener (should not exist unless configured in the module): {listener.name}"


@pytest.mark.compilertest
def test_buffer_limit_no_listener_V3():
    yaml = """
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: ambassador
  namespace: default
spec:
  prefix: /test/
  hostname: "*"
  service: test:9999
"""
    econf = _get_envoy_config(yaml)

    conf = econf.as_dict()

    for listener in conf['static_resources']['listeners']:
        per_connection_buffer_limit_bytes = listener.get('per_connection_buffer_limit_bytes', None)
        assert per_connection_buffer_limit_bytes is None, \
            f"per_connection_buffer_limit_bytes found on listener (should not exist unless configured in the module): {listener.name}" 