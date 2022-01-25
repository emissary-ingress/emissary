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

import tests.integration.manifests as integration_manifests
from kat.utils import namespace_manifest
from tests.manifests import cleartext_host_manifest
from tests.kubeutils import apply_kube_artifacts
from tests.runutils import run_and_assert

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
    apply_kube_artifacts(namespace='emissary-system', artifacts=integration_manifests.CRDmanifests)

    print("Wait for apiext to be running...")
    run_and_assert(['tools/bin/kubectl', 'wait', '--timeout=90s', '--for=condition=available', 'deploy', 'emissary-apiext', '-n', 'emissary-system'])

    # Proceed to install Ambassador now
    final_yaml = []

    rbac_manifest_name = 'rbac_namespace_scope' if single_namespace else 'rbac_cluster_scope'

    # Hackish fakes of actual KAT structures -- it's _far_ too much work to synthesize
    # actual KAT Nodes and Paths.
    fakeNode = namedtuple('fakeNode', [ 'namespace', 'path', 'ambassador_id' ])
    fakePath = namedtuple('fakePath', [ 'k8s' ])

    ambassador_yaml = list(yaml.safe_load_all(
        integration_manifests.format(
            "\n".join([
                integration_manifests.load(rbac_manifest_name),
                integration_manifests.load('ambassador'),
                (cleartext_host_manifest % namespace),
            ]),
            capabilities_block="",
            envs="",
            extra_ports="",
            self=fakeNode(
                namespace=namespace,
                ambassador_id='default',
                path=fakePath(k8s='ambassador')
            ),
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

    # print("INSTALLING AMBASSADOR: manifests:")
    # print(yaml.safe_dump_all(ambassador_yaml))

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
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name:  qotm-mapping
  namespace: {namespace}
spec:
  hostname: "*"
  prefix: /qotm/
  service: qotm
"""

    apply_kube_artifacts(namespace=namespace, artifacts=qotm_mapping)

def create_httpbin_mapping(namespace):
    httpbin_mapping = f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name:  httpbin-mapping
  namespace: {namespace}
spec:
  hostname: "*"
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
apiVersion: getambassador.io/v3alpha1
kind: TracingService
metadata:
  name: tracing
  namespace: ambassador
spec:
  service: zipkin:9411
  driver: zipkin
  config: {}
"""

def default_listener_manifests():
    return """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: listener-8080
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
kind: Listener
metadata:
  name: listener-8443
  namespace: default
spec:
  port: 8443
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
"""

def module_and_mapping_manifests(module_confs, mapping_confs):
    yaml = default_listener_manifests() + """
---
apiVersion: getambassador.io/v3alpha1
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
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: ambassador
  namespace: default
spec:
  hostname: "*"
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

def compile_with_cachecheck(yaml, envoy_version="V3", errors_ok=False):
    # Compile with and without a cache. Neither should produce errors.
    cache = Cache(logger)
    secret_handler = _secret_handler()
    r1 = Compile(logger, yaml, k8s=True, secret_handler=secret_handler, envoy_version=envoy_version)
    r2 = Compile(logger, yaml, k8s=True, secret_handler=secret_handler, cache=cache,
            envoy_version=envoy_version)

    if not errors_ok:
        _require_no_errors(r1["ir"])
        _require_no_errors(r2["ir"])

    # Both should produce equal Envoy config as sorted json.
    r1j = json.dumps(r1[envoy_version.lower()].as_dict(), sort_keys=True, indent=2)
    r2j = json.dumps(r2[envoy_version.lower()].as_dict(), sort_keys=True, indent=2)
    assert r1j == r2j

    # All good.
    return r1

EnvoyFilterInfo = namedtuple('EnvoyFilterInfo', [ 'name', 'type' ])

EnvoyHCMInfo = {
    "V2": EnvoyFilterInfo(
        name="envoy.filters.network.http_connection_manager",
        type="type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager"
    ),
    "V3": EnvoyFilterInfo(
        name="envoy.filters.network.http_connection_manager",
        type="type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager"
    ),
}

EnvoyTCPInfo = {
    "V2": EnvoyFilterInfo(
        name="envoy.filters.network.tcp_proxy",
        type="type.googleapis.com/envoy.config.filter.network.tcp_proxy.v2.TcpProxy"
    ),
    "V3": EnvoyFilterInfo(
        name="envoy.filters.network.tcp_proxy",
        type="type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy"
    ),
}

def econf_compile(yaml, envoy_version="V2"):
    compiled = compile_with_cachecheck(yaml, envoy_version=envoy_version)
    return compiled[envoy_version.lower()].as_dict()

def econf_foreach_listener(econf, fn, envoy_version='V3', listener_count=1):
    listeners = econf['static_resources']['listeners']

    wanted_plural = "" if (listener_count == 1) else "s"
    assert len(listeners) == listener_count, f"Expected {listener_count} listener{wanted_plural}, got {len(listeners)}"

    for listener in listeners:
        fn(listener, envoy_version)

def econf_foreach_listener_chain(listener, fn, chain_count=2, need_name=None, need_type=None, dump_info=None):
    # We need a specific number of filter chains. Normally it's 2,
    # since the compiler tests don't generally supply Listeners or Hosts,
    # so we get secure and insecure chains.
    filter_chains = listener['filter_chains']

    if dump_info:
        dump_info(filter_chains)

    wanted_plural = "" if (chain_count == 1) else "s"
    assert len(filter_chains) == chain_count, f"Expected {chain_count} filter chain{wanted_plural}, got {len(filter_chains)}"

    for chain in filter_chains:
        # We expect one filter on this chain.
        filters = chain['filters']
        got_count = len(filters)
        got_plural = "" if (got_count == 1) else "s"
        assert got_count == 1, f"Expected just one filter, got {got_count} filter{got_plural}"

        # The http connection manager is the only filter on the chain from the one and only vhost.
        filter = filters[0]

        if need_name:
            assert filter['name'] == need_name

        typed_config = filter['typed_config']

        if need_type:
            assert typed_config['@type'] == need_type, f"bad type: {typed_config['@type']}"

        fn(typed_config)

def econf_foreach_hcm(econf, fn, envoy_version='V3', chain_count=2):
    for listener in econf['static_resources']['listeners']:
        hcm_info = EnvoyHCMInfo[envoy_version]

        econf_foreach_listener_chain(
            listener, fn, chain_count=chain_count,
            need_name=hcm_info.name, need_type=hcm_info.type)

def econf_foreach_cluster(econf, fn, name='cluster_httpbin_default'):
    for cluster in econf['static_resources']['clusters']:
        if cluster['name'] != name:
            continue

        found_cluster = True
        r = fn(cluster)
        if not r:
            break
    assert found_cluster

def assert_valid_envoy_config(config_dict, v2=False):
    with tempfile.NamedTemporaryFile() as temp:
        temp.write(bytes(json.dumps(config_dict), encoding = 'utf-8'))
        temp.flush()
        f_name = temp.name
        cmd = [ENVOY_PATH, '--config-path', f_name, '--mode', 'validate']
        if v2:
            cmd.append('--bootstrap-version 2')
        p = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        if p.returncode != 0:
            print(p.stdout)
        p.check_returncode()
