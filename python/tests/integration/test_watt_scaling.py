import time
import sys

import pytest

from tests.integration.utils import install_ambassador, get_code_with_retry
from tests.kubeutils import apply_kube_artifacts, delete_kube_artifacts
from tests.runutils import run_with_retry, run_and_assert
from tests.manifests import qotm_manifests


class WattTesting:
    def manifests(self):
        pass

    def apply_manifests(self):
        pass

    def create_listeners(self, namespace):
        manifest = f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: listener-8080
spec:
  port: 8080
  protocol: HTTP
  securityModel: INSECURE
  hostBinding:
    namespace:
      from: SELF
"""

        apply_kube_artifacts(namespace=namespace, artifacts=manifest)

    def apply_qotm_endpoint_manifests(self, namespace):
        qotm_resolver = f"""
apiVersion: getambassador.io/v3alpha1
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
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name:  qotm-mapping
  namespace: {namespace}
spec:
  hostname: "*"
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
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name:  qotm-mapping
  namespace: {namespace}
spec:
  hostname: "*"
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

        # Set up our listener.
        self.create_listeners(namespace)

        # Install QOTM
        apply_kube_artifacts(namespace=namespace, artifacts=qotm_manifests)

        # Install QOTM Ambassador manifests
        self.apply_qotm_endpoint_manifests(namespace=namespace)

        # Now let's wait for ambassador and QOTM pods to become ready
        run_and_assert(['tools/bin/kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'service=ambassador', '-n', namespace])
        run_and_assert(['tools/bin/kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'service=qotm', '-n', namespace])

        # Assume we can reach Ambassador through telepresence
        qotm_host = "ambassador." + namespace
        qotm_url = f"http://{qotm_host}/qotm/"

        # Assert 200 OK at /qotm/ endpoint
        qotm_ready = False

        loop_limit = 60
        while not qotm_ready:
            assert loop_limit > 0, "QOTM is not ready yet, aborting..."
            try:
                qotm_http_code = get_code_with_retry(qotm_url)
                assert qotm_http_code == 200, f"Expected 200 OK, got {qotm_http_code}"
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

        # Let's give Ambassador a few seconds to register the changes...
        time.sleep(5)

        # Assert 200 OK at /qotm/ endpoint
        qotm_http_code = get_code_with_retry(qotm_url)
        assert qotm_http_code == 200, f"Expected 200 OK, got {qotm_http_code}"


@pytest.mark.flaky(reruns=1, reruns_delay=10)
def test_watt():
    watt_test = WattTesting()
    watt_test.test_rapid_additions_and_deletions()


if __name__ == '__main__':
    pytest.main(sys.argv)
