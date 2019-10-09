import subprocess
import tempfile
import json
import time
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
  replicas: 3
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
    def run_and_assert(command):
        print(f"Running command {command}")
        output = subprocess.Popen(command, stdout=subprocess.PIPE)
        stdout, stderr = output.communicate()
        print('STDOUT', stdout.decode("utf-8") if stdout is not None else None)
        print('STDERR', stderr.decode("utf-8") if stderr is not None else None)
        assert output.returncode == 0
        return stdout.decode("utf-8") if stdout is not None else None

    def install_latest_ambassador(self, namespace):
        if namespace is None:
            namespace = 'default'

        self.create_namespace(namespace)

        install_ambassador_cmd = ['kubectl', 'apply', '-n', namespace, '-f', 'https://getambassador.io/yaml/ambassador/ambassador-rbac.yaml']
        self.run_and_assert(install_ambassador_cmd)

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

        self.run_and_assert(['kubectl', 'scale', 'deployment', 'ambassador', '--replicas', '1', '-n', namespace])

        self.apply_kube_artifacts(namespace=namespace, artifacts=namespace_crb)

        install_ambassador_service_cmd = ['kubectl', 'apply', '-n', namespace, '-f', 'https://getambassador.io/yaml/ambassador/ambassador-service.yaml']
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
        self.install_latest_ambassador(namespace=namespace)

        # Install QOTM
        self.apply_kube_artifacts(namespace=namespace, artifacts=self.qotm_manifests)

        # Install QOTM Ambassador manifests
        self.apply_qotm_endpoint_manifests(namespace=namespace)

        # Get ambassador service's ClusterIP
        ambassador_service_json_cmd = ['kubectl', 'get', 'service', '-n', namespace, 'ambassador', '-o', 'json']
        ambassador_service_json = self.run_and_assert(ambassador_service_json_cmd)
        ambassador_service = json.loads(ambassador_service_json)
        cluster_ip = ambassador_service['spec']['clusterIP']
        qotm_url = f'http://{cluster_ip}/qotm/'

        # Assert 200 OK at /qotm/ endpoint
        qotm_ready = False

        loop_limit = 20
        while not qotm_ready:
            assert loop_limit > 0, "QOTM is not ready yet, aborting..."
            try:
                connection = request.urlopen(qotm_url, timeout=5)
                qotm_http_code = connection.getcode()
                assert qotm_http_code == 200, f"Expected 200 OK, got {qotm_http_code}"
                connection.close()
                qotm_ready = True

            except Exception as e:
                print(f"Error: {e}")
                print(f"{qotm_url} not ready yet, trying again...")
                time.sleep(1)
                loop_limit -= 1

        # Try to mess up Ambassador by applying and deleting QOTM mapping over and over
        self.delete_qotm_mapping(namespace=namespace)
        self.create_qotm_mapping(namespace=namespace)
        self.delete_qotm_mapping(namespace=namespace)
        self.create_qotm_mapping(namespace=namespace)
        self.delete_qotm_mapping(namespace=namespace)
        self.create_qotm_mapping(namespace=namespace)
        self.delete_qotm_mapping(namespace=namespace)
        self.create_qotm_mapping(namespace=namespace)
        self.delete_qotm_mapping(namespace=namespace)
        self.create_qotm_mapping(namespace=namespace)
        self.delete_qotm_mapping(namespace=namespace)
        self.create_qotm_mapping(namespace=namespace)
        self.delete_qotm_mapping(namespace=namespace)
        self.create_qotm_mapping(namespace=namespace)
        self.delete_qotm_mapping(namespace=namespace)
        self.create_qotm_mapping(namespace=namespace)
        self.delete_qotm_mapping(namespace=namespace)
        self.create_qotm_mapping(namespace=namespace)

        # Let's give Ambassador some time to register the changes
        time.sleep(5)

        # Assert 200 OK at /qotm/ endpoint
        connection = request.urlopen(qotm_url)
        qotm_http_code = connection.getcode()
        assert qotm_http_code == 200, f"Expected 200 OK, got {qotm_http_code}"
        connection.close()

def test_watt():
    watt_test = WattTesting()
    watt_test.test_rapid_additions_and_deletions()


if __name__ == '__main__':
    test_watt()
