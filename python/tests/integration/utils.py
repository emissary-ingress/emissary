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
from tests.manifests import cleartext_host_manifest
from tests.kubeutils import apply_kube_artifacts
from tests.runutils import run_and_assert

# Assume that both of these are on the PATH if not explicitly set
KUBESTATUS_PATH = os.environ.get('KUBESTATUS_PATH', 'kubestatus')

def install_crds() -> None:
    apply_kube_artifacts(namespace='emissary-system', artifacts=integration_manifests.crd_manifests())

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
    install_crds()

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
    apply_kube_artifacts(namespace=namespace, artifacts=integration_manifests.namespace_manifest(namespace))


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
