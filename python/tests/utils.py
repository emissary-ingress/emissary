import os
import subprocess
import tempfile

import yaml

from kat.utils import namespace_manifest

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
        image: datawire/qotm:1.3
        imagePullPolicy: Always
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
    ambassador_crds_path = "/buildroot/ambassador/docs/yaml/ambassador/ambassador-crds.yaml"
    install_ambassador_crds_cmd = ['kubectl', 'apply', '-n', namespace, '-f', ambassador_crds_path]
    run_and_assert(install_ambassador_crds_cmd)

    # Proceed to install Ambassador now
    final_yaml = []
    ambassador_yaml_path = "/buildroot/ambassador/docs/yaml/ambassador/ambassador-rbac.yaml"
    with open(ambassador_yaml_path, 'r') as f:
        ambassador_yaml = list(yaml.safe_load_all(f))

        for manifest in ambassador_yaml:
            if manifest.get('kind', '') == 'Deployment' and manifest.get('metadata', {}).get('name', '') == 'ambassador':
                # we want only one replica of Ambassador to run
                manifest['spec']['replicas'] = 1

                # let's fix the image
                manifest['spec']['template']['spec']['containers'][0]['image'] = os.environ['AMBASSADOR_DOCKER_IMAGE']

                # we don't want to do everything in /ambassador/
                manifest['spec']['template']['spec']['containers'][0]['env'].append({
                    'name': 'AMBASSADOR_CONFIG_BASE_DIR',
                    'value': '/tmp/ambassador'
                })

                # add new envs, if any
                manifest['spec']['template']['spec']['containers'][0]['env'].extend(envs)

            final_yaml.append(manifest)

    apply_kube_artifacts(namespace=namespace, artifacts=yaml.safe_dump_all(final_yaml))

    namespace_crb = f"""
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: ambassador
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ambassador
subjects:
- kind: ServiceAccount
  name: ambassador
  namespace: {namespace}
"""

    apply_kube_artifacts(namespace=namespace, artifacts=namespace_crb)

    ambassador_service_path = "/buildroot/ambassador/docs/yaml/ambassador/ambassador-service.yaml"
    install_ambassador_service_cmd = ['kubectl', 'apply', '-n', namespace, '-f', ambassador_service_path]
    run_and_assert(install_ambassador_service_cmd)


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
