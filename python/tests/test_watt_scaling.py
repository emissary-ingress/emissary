import os
import subprocess
import tempfile
import time
import yaml
from urllib import request

class WattTesting:
    def __init__(self):
        self.qotm_manifests = """
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

    def create_namespace(self, namespace):
        namespace_manifest = f"""
---
apiVersion: v1
kind: Namespace
metadata:
  name: {namespace}
"""

        self.apply_kube_artifacts(namespace=namespace, artifacts=namespace_manifest)

    def manifests(self):
        pass

    def apply_manifests(self):
        pass

    @staticmethod
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

    def install_ambassador(self, namespace):
        if namespace is None:
            namespace = 'default'

        self.create_namespace(namespace)

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

                final_yaml.append(manifest)

        self.apply_kube_artifacts(namespace=namespace, artifacts=yaml.safe_dump_all(final_yaml))

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

        self.apply_kube_artifacts(namespace=namespace, artifacts=namespace_crb)

        ambassador_service_path = "/buildroot/ambassador/docs/yaml/ambassador/ambassador-service.yaml"
        install_ambassador_service_cmd = ['kubectl', 'apply', '-n', namespace, '-f', ambassador_service_path]
        self.run_and_assert(install_ambassador_service_cmd)

    def meta_action_kube_artifacts(self, namespace, artifacts, action):
        temp_file = tempfile.NamedTemporaryFile()
        temp_file.write(artifacts.encode())
        temp_file.flush()
        self.run_and_assert(['kubectl', action, '-n', namespace, '-f', temp_file.name])
        temp_file.close()

    def apply_kube_artifacts(self, namespace, artifacts):
        self.meta_action_kube_artifacts(namespace=namespace, artifacts=artifacts, action='apply')

    def delete_kube_artifacts(self, namespace, artifacts):
        self.meta_action_kube_artifacts(namespace=namespace, artifacts=artifacts, action='delete')

    def apply_qotm_endpoint_manifests(self, namespace):
        qotm_resolver = f"""
apiVersion: getambassador.io/v1
kind: KubernetesEndpointResolver
metadata:
  name: qotm-resolver
  namespace: {namespace}
"""

        self.apply_kube_artifacts(namespace=namespace, artifacts=qotm_resolver)
        self.create_qotm_mapping(namespace=namespace)

    def create_qotm_mapping(self, namespace):
        qotm_mapping = f"""
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name:  qotm-mapping
  namespace: {namespace}
spec:
  prefix: /qotm/
  service: qotm.{namespace}
  resolver: qotm-resolver
  load_balancer:
    policy: round_robin
        """

        self.apply_kube_artifacts(namespace=namespace, artifacts=qotm_mapping)

    def delete_qotm_mapping(self, namespace):
        qotm_mapping = f"""
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name:  qotm-mapping
  namespace: {namespace}
spec:
  prefix: /qotm/
  service: qotm.{namespace}
  resolver: qotm-resolver
  load_balancer:
    policy: round_robin
        """

        self.delete_kube_artifacts(namespace=namespace, artifacts=qotm_mapping)

    def test_rapid_additions_and_deletions(self):
        namespace = 'watt-rapid'

        # Install Ambassador
        self.install_ambassador(namespace=namespace)

        # Install QOTM
        self.apply_kube_artifacts(namespace=namespace, artifacts=self.qotm_manifests)

        # Install QOTM Ambassador manifests
        self.apply_qotm_endpoint_manifests(namespace=namespace)

        # Now let's wait for ambassador and QOTM pods to become ready
        self.run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'service=ambassador', '-n', namespace])
        self.run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'service=qotm', '-n', namespace])

        # Let's port-forward ambassador service to talk to QOTM
        port_forward_port = 6000
        port_forward_command = ['kubectl', 'port-forward', '--namespace', namespace, 'service/ambassador', f'{port_forward_port}:80']
        self.run_and_assert(port_forward_command, communicate=False)
        qotm_url = f'http://localhost:{port_forward_port}/qotm/'

        # Assert 200 OK at /qotm/ endpoint
        qotm_ready = False

        loop_limit = 60
        while not qotm_ready:
            assert loop_limit > 0, "QOTM is not ready yet, aborting..."
            try:
                connection = request.urlopen(qotm_url, timeout=5)
                qotm_http_code = connection.getcode()
                assert qotm_http_code == 200, f"Expected 200 OK, got {qotm_http_code}"
                connection.close()
                print(f"{qotm_url} is ready")
                qotm_ready = True

            except Exception as e:
                print(f"Error: {e}")
                print(f"{qotm_url} not ready yet, trying again...")
                time.sleep(1)
                loop_limit -= 1

        # Try to mess up Ambassador by applying and deleting QOTM mapping over and over
        for i in range(10):
            self.delete_qotm_mapping(namespace=namespace)
            self.create_qotm_mapping(namespace=namespace)

        # Let's give Ambassador some time to register the changes
        time.sleep(60)

        # Assert 200 OK at /qotm/ endpoint
        connection = request.urlopen(qotm_url, timeout=5)
        qotm_http_code = connection.getcode()
        assert qotm_http_code == 200, f"Expected 200 OK, got {qotm_http_code}"
        connection.close()

def test_watt():
    watt_test = WattTesting()
    watt_test.test_rapid_additions_and_deletions()


if __name__ == '__main__':
    test_watt()
