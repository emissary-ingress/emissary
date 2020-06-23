import os
import subprocess
import tempfile
from collections import namedtuple

import yaml

from kat.utils import namespace_manifest
from kat.harness import load_manifest, CLEARTEXT_HOST_YAML

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

    if namespace is not None:
        command.extend(['-n', namespace])

    run_and_assert(command)
    temp_file.close()


def apply_kube_artifacts(namespace, artifacts):
    meta_action_kube_artifacts(namespace=namespace, artifacts=artifacts, action='apply')


def delete_kube_artifacts(namespace, artifacts):
    meta_action_kube_artifacts(namespace=namespace, artifacts=artifacts, action='delete')


def install_ambassador(namespace, envs=None):
    """
    :param namespace: namespace to install Ambassador in
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

    if namespace is None:
        namespace = 'default'

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

    ambassador_yaml = list(yaml.safe_load_all((
        load_manifest('rbac_cluster_scope')+
        load_manifest('ambassador')+
        (CLEARTEXT_HOST_YAML % namespace)
    ).format(
        capabilities_block="",
        envs="",
        extra_ports="",
        serviceAccountExtra=serviceAccountExtra,
        image=os.environ["AMBASSADOR_DOCKER_IMAGE"],
        self=namedtuple('self', 'namespace path ambassador_id')(
            namespace,
            namedtuple('path', 'k8s')(
                'ambassador'),
            'default',
        ),
    )))

    for manifest in ambassador_yaml:
        if manifest.get('kind', '') == 'Pod' and manifest.get('metadata', {}).get('name', '') == 'ambassador':

            # Don't set AMBASSADOR_ID={self.k8s.path}
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
