import time
from urllib import request
from utils import run_and_assert, apply_kube_artifacts, delete_kube_artifacts, install_ambassador, qotm_manifests


class WattTesting:
    def manifests(self):
        pass

    def apply_manifests(self):
        pass

    def apply_qotm_endpoint_manifests(self, namespace):
        qotm_resolver = f"""
apiVersion: getambassador.io/v1
kind: KubernetesEndpointResolver
metadata:
  name: qotm-resolver
  namespace: {namespace}
"""

        apply_kube_artifacts(namespace=namespace, artifacts=qotm_resolver)
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

        apply_kube_artifacts(namespace=namespace, artifacts=qotm_mapping)

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

        delete_kube_artifacts(namespace=namespace, artifacts=qotm_mapping)

    def test_rapid_additions_and_deletions(self):
        namespace = 'watt-rapid'

        # Install Ambassador
        install_ambassador(namespace=namespace)

        # Install QOTM
        apply_kube_artifacts(namespace=namespace, artifacts=qotm_manifests)

        # Install QOTM Ambassador manifests
        self.apply_qotm_endpoint_manifests(namespace=namespace)

        # Now let's wait for ambassador and QOTM pods to become ready
        run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'service=ambassador', '-n', namespace])
        run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'service=qotm', '-n', namespace])

        # Let's port-forward ambassador service to talk to QOTM
        port_forward_port = 6000
        port_forward_command = ['kubectl', 'port-forward', '--namespace', namespace, 'service/ambassador', f'{port_forward_port}:80']
        run_and_assert(port_forward_command, communicate=False)
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
