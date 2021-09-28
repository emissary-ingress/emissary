import logging
import json
import os
import subprocess
import requests
import socket
import tempfile
import time
from collections import namedtuple
from retry import retry

import json
import yaml

from ambassador import Cache, IR
from ambassador.compile import Compile
from ambassador.utils import NullSecretHandler
from kat.utils import namespace_manifest
from kat.harness import load_manifest
from tests.manifests import cleartext_host_manifest
from tests.kubeutils import apply_kube_artifacts

logger = logging.getLogger("ambassador")

ENVOY_PATH = os.environ.get('ENVOY_PATH', '/usr/local/bin/envoy')
# Assume that both of these are on the PATH if not explicitly set
KUBESTATUS_PATH = os.environ.get('KUBESTATUS_PATH', 'kubestatus')

SUPPORTED_ENVOY_VERSIONS = ["V2", "V3"]


def install_ambassador(namespace, single_namespace=True, envs=None, debug=None):
    """
    Install Ambassador into a given namespace. NOTE WELL that although there
    is a 'single_namespace' parameter, this function probably needs work to do
    the fully-correct thing with single_namespace False.

    :param namespace: namespace to install Ambassador in
    :param single_namespace: should we set AMBASSADOR_SINGLE_NAMESPACE? SEE NOTE ABOVE!
    :param envs: [
      {
        'name': 'ENV_NAME',
        'value': 'ENV_VALUE'
      },
      ...
      ...
    ]
    """

    if envs is None:
        envs = []

    if single_namespace:
        update_envs(envs, "AMBASSADOR_SINGLE_NAMESPACE", "true")

    if debug:
        update_envs(envs, "AMBASSADOR_DEBUG", debug)

    # Create namespace to install Ambassador
    create_namespace(namespace)

    # Create Ambassador CRDs
    apply_kube_artifacts(namespace=namespace, artifacts=load_manifest('crds'))

    # Proceed to install Ambassador now
    final_yaml = []

    serviceAccountExtra = ''
    if os.environ.get("DEV_USE_IMAGEPULLSECRET", False):
        serviceAccountExtra = """
imagePullSecrets:
- name: dev-image-pull-secret
"""

    rbac_manifest_name = 'rbac_namespace_scope' if single_namespace else 'rbac_cluster_scope'

    # Hackish fakes of actual KAT structures -- it's _far_ too much work to synthesize
    # actual KAT Nodes and Paths.
    fakeNode = namedtuple('fakeNode', [ 'namespace', 'path', 'ambassador_id' ])
    fakePath = namedtuple('fakePath', [ 'k8s' ])

    ambassador_yaml = list(yaml.safe_load_all((
        load_manifest(rbac_manifest_name) +
        load_manifest('ambassador') +
        (cleartext_host_manifest % namespace)
    ).format(
        capabilities_block="",
        envs="",
        extra_ports="",
        serviceAccountExtra=serviceAccountExtra,
        image=os.environ["AMBASSADOR_DOCKER_IMAGE"],
        self=fakeNode(
            namespace=namespace,
            ambassador_id='default',
            path=fakePath(k8s='ambassador')
        )
    )))

    for manifest in ambassador_yaml:
        kind = manifest.get('kind', None)
        metadata = manifest.get('metadata', {})
        name = metadata.get('name', None)

        if (kind == "Pod") and (name == "ambassador"):
            # Force AMBASSADOR_ID to match ours.
            #
            # XXX This is not likely to work without single_namespace=True.
            for envvar in manifest['spec']['containers'][0]['env']:
                if envvar.get('name', '') == 'AMBASSADOR_ID':
                    envvar['value'] = 'default'

            # add new envs, if any
            manifest['spec']['containers'][0]['env'].extend(envs)

    print("INSTALLING AMBASSADOR: manifests:")
    print(yaml.safe_dump_all(ambassador_yaml))

    apply_kube_artifacts(namespace=namespace, artifacts=yaml.safe_dump_all(ambassador_yaml))


def update_envs(envs, name, value):
    found = False
        
    for e in envs:
        if e['name'] == name:
            e['value'] = value
            found = True
            break

    if not found:
        envs.append({
            'name': name,
            'value': value
        })


def create_namespace(namespace):
    apply_kube_artifacts(namespace=namespace, artifacts=namespace_manifest(namespace))


def create_qotm_mapping(namespace):
    qotm_mapping = f"""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name:  qotm-mapping
  namespace: {namespace}
spec:
  host: "*"
  prefix: /qotm/
  service: qotm
"""

    apply_kube_artifacts(namespace=namespace, artifacts=qotm_mapping)

def create_httpbin_mapping(namespace):
    httpbin_mapping = f"""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name:  httpbin-mapping
  namespace: {namespace}
spec:
  host: "*"
  prefix: /httpbin/
  rewrite: /
  service: httpbin
"""

    apply_kube_artifacts(namespace=namespace, artifacts=httpbin_mapping)


def get_code_with_retry(req, headers={}):
    for attempts in range(10):
        try:
            resp = requests.get(req, headers=headers, timeout=10)
            if resp.status_code < 500:
                return resp.status_code
            print(f"get_code_with_retry: 5xx code {resp.status_code}, retrying...")
        except requests.exceptions.ConnectionError as e:
            print(f"get_code_with_retry: ConnectionError {e}, attempt {attempts+1}")
        except socket.timeout as e:
            print(f"get_code_with_retry: socket.timeout {e}, attempt {attempts+1}")
        except Exception as e:
            print(f"get_code_with_retry: generic exception {e}, attempt {attempts+1}")
        time.sleep(5)
    return 503

def zipkin_tracing_service_manifest():
    return """
---
apiVersion: getambassador.io/v2
kind: TracingService
metadata:
  name: tracing
  namespace: ambassador
spec:
  service: zipkin:9411
  driver: zipkin
  config: {}
"""

def module_and_mapping_manifests(module_confs, mapping_confs):
    yaml = """
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:"""
    if module_confs:
        for module_conf in module_confs:
            yaml = yaml + """
    {}
""".format(module_conf)
    else:
        yaml = yaml + " {}\n"

    yaml = yaml + """
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
metadata:
  name: ambassador
  namespace: default
spec:
  host: "*"
  prefix: /httpbin/
  service: httpbin"""
    if mapping_confs:
        for mapping_conf in mapping_confs:
            yaml = yaml + """
  {}""".format(mapping_conf)
    return yaml

def _require_no_errors(ir: IR):
    assert ir.aconf.errors == {}

def _secret_handler():
    source_root = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-source")
    cache_dir = tempfile.TemporaryDirectory(prefix="null-secret-", suffix="-cache")
    return NullSecretHandler(logger, source_root.name, cache_dir.name, "fake")

def econf_compile(yaml, envoy_version="V2"):
    # Compile with and without a cache. Neither should produce errors.
    cache = Cache(logger)
    secret_handler = _secret_handler()
    r1 = Compile(logger, yaml, k8s=True, secret_handler=secret_handler, envoy_version=envoy_version)
    r2 = Compile(logger, yaml, k8s=True, secret_handler=secret_handler, cache=cache,
            envoy_version=envoy_version)
    _require_no_errors(r1["ir"])
    _require_no_errors(r2["ir"])

    # Both should produce equal Envoy config as sorted json.
    r1j = json.dumps(r1[envoy_version.lower()].as_dict(), sort_keys=True, indent=2)
    r2j = json.dumps(r2[envoy_version.lower()].as_dict(), sort_keys=True, indent=2)
    assert r1j == r2j

    # Now we can return the Envoy config as a dictionary
    return r1[envoy_version.lower()].as_dict()

def econf_foreach_hcm(econf, fn, envoy_version='V2', chain_count=2):
    found_hcm = False
    for listener in econf['static_resources']['listeners']:
        # We need a specific number of filter chains. Normally it's 2,
        # since the compiler tests don't generally supply Listeners or Hosts,
        # so we get secure and insecure chains.
        filter_chains = listener['filter_chains']
        assert len(filter_chains) == chain_count

        for chain in filter_chains:
            # We expect one filter on that chain (the http_connection_manager).
            filters = chain['filters']
            assert len(filters) == 1

            # The http connection manager is the only filter on the chain from the one and only vhost.
            hcm = filters[0]
            assert hcm['name'] == 'envoy.filters.network.http_connection_manager'
            typed_config = hcm['typed_config']
            envoy_version_type_map = {
                'V3': 'type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager',
                'V2': 'type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager',
            }
            assert typed_config['@type'] == envoy_version_type_map[envoy_version], "bad type: %s" % typed_config['@type']

            found_hcm = True

            r = fn(typed_config)
            if not r:
                break

    assert found_hcm

def econf_foreach_cluster(econf, fn, name='cluster_httpbin_default'):
    for cluster in econf['static_resources']['clusters']:
        if cluster['name'] != name:
            continue

        found_cluster = True
        r = fn(cluster)
        if not r:
            break
    assert found_cluster

def assert_valid_envoy_config(config_dict):
    with tempfile.NamedTemporaryFile() as temp:
        temp.write(bytes(json.dumps(config_dict), encoding = 'utf-8'))
        temp.flush()
        f_name = temp.name
        cmd = [ENVOY_PATH, '--config-path', f_name, '--mode', 'validate']
        v_encoded = subprocess.check_output(cmd, stderr=subprocess.STDOUT)