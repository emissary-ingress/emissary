import os
import subprocess
import requests
import socket
import tempfile
import time
from collections import namedtuple
from retry import retry

import yaml

from kat.utils import namespace_manifest
from kat.harness import load_manifest, CLEARTEXT_HOST_YAML

httpbin_manifests ="""
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin
spec:
  type: ClusterIP
  selector:
    service: httpbin
  ports:
  - port: 80
    targetPort: http
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
spec:
  replicas: 1
  selector:
    matchLabels:
      service: httpbin
  template:
    metadata:
      labels:
        service: httpbin
    spec:
      containers:
      - name: httpbin
        image: kennethreitz/httpbin
        ports:
        - name: http
          containerPort: 80
"""

qotm_manifests = """
---
apiVersion: v1
kind: Service
metadata:
  name: qotm
spec:
  selector:
    service: qotm
  ports:
    - port: 80
      targetPort: http-api
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: qotm
spec:
  selector:
    matchLabels:
      service: qotm
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
      labels:
        service: qotm
    spec:
      serviceAccountName: ambassador
      containers:
      - name: qotm
        image: docker.io/datawire/qotm:1.3
        ports:
        - name: http-api
          containerPort: 5000
"""


def run_and_assert(command, communicate=True):
    print(f"Running command {command}")
    output = subprocess.Popen(command, stdout=subprocess.PIPE)
    if communicate:
        stdout, stderr = output.communicate()
        print('STDOUT', stdout.decode("utf-8") if stdout is not None else None)
        print('STDERR', stderr.decode("utf-8") if stderr is not None else None)
        assert output.returncode == 0
        return stdout.decode("utf-8") if stdout is not None else None
    return None


def meta_action_kube_artifacts(namespace, artifacts, action):
    temp_file = tempfile.NamedTemporaryFile()
    temp_file.write(artifacts.encode())
    temp_file.flush()

    command = ['kubectl', action, '-f', temp_file.name]
    if namespace is None:
        namespace = 'default'

    if namespace is not None:
        command.extend(['-n', namespace])

    run_and_assert(command)
    temp_file.close()


def apply_kube_artifacts(namespace, artifacts):
    meta_action_kube_artifacts(namespace=namespace, artifacts=artifacts, action='apply')


def delete_kube_artifacts(namespace, artifacts):
    meta_action_kube_artifacts(namespace=namespace, artifacts=artifacts, action='delete')


def install_ambassador(namespace, single_namespace=True, envs=None):
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

    found_single_namespace = False

    if single_namespace:
        for e in envs:
            if e['name'] == 'AMBASSADOR_SINGLE_NAMESPACE':
                e['value'] = 'true'
                found_single_namespace = True
                break
    
        if not found_single_namespace:
            envs.append({ 
                'name': 'AMBASSADOR_SINGLE_NAMESPACE',
                'value': 'true'
            })

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
        (CLEARTEXT_HOST_YAML % namespace)
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

    apply_kube_artifacts(namespace=namespace, artifacts=yaml.safe_dump_all(ambassador_yaml))


def create_namespace(namespace):
    apply_kube_artifacts(namespace=namespace, artifacts=namespace_manifest(namespace))


def create_qotm_mapping(namespace):
    qotm_mapping = f"""
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name:  qotm-mapping
  namespace: {namespace}
spec:
  prefix: /qotm/
  service: qotm
"""

    apply_kube_artifacts(namespace=namespace, artifacts=qotm_mapping)

def create_httpbin_mapping(namespace):
    httpbin_mapping = f"""
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name:  httpbin-mapping
  namespace: {namespace}
spec:
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
