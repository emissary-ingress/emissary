import logging
from urllib import request
from urllib.error import URLError, HTTPError
from retry import retry
import sys

import pytest

from kat.harness import is_knative
from kat.harness import load_manifest
from ambassador import Config, IR
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler
from utils import run_and_assert, apply_kube_artifacts, install_ambassador, qotm_manifests, create_qotm_mapping

logger = logging.getLogger('ambassador')

knative_service_example = """
---
apiVersion: serving.knative.dev/v1alpha1
kind: Service
metadata:
 name: helloworld-go
spec:
 template:
   spec:
     containers:
     - image: gcr.io/knative-samples/helloworld-go
       env:
       - name: TARGET
         value: "Ambassador is Awesome"
"""

knative_ingress_example = """
apiVersion: networking.internal.knative.dev/v1alpha1
kind: Ingress
metadata:
  name: helloworld-go
spec:
  rules:
  - hosts:
    - helloworld-go.default.svc.cluster.local
    http:
      paths:
      - retries:
          attempts: 3
          perTryTimeout: 10m0s
        splits:
        - percent: 100
          serviceName: helloworld-go-qf94m
          serviceNamespace: default
          servicePort: 80
        timeout: 10m0s
    visibility: ClusterLocal
  visibility: ExternalIP
"""


class KnativeTesting:
    @retry(URLError, tries=5, delay=2)
    def get_code_with_retry(self, req):
        try:
            conn = request.urlopen(req, timeout=5)
            conn.close()
            return 200
        except HTTPError as e:
            return e.code

    def test_knative(self):
        namespace = 'knative-testing'

        # Install Knative
        apply_kube_artifacts(namespace=None, artifacts=load_manifest("knative_serving_crds"))
        apply_kube_artifacts(namespace=None, artifacts=load_manifest("knative_serving_0.11.0"))
        run_and_assert(['kubectl', 'patch', 'configmap/config-network', '--type', 'merge', '--patch', r'{"data": {"ingress.class": "ambassador.ingress.networking.knative.dev"}}', '-n', 'knative-serving'])

        # Wait for Knative to become ready
        run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'app=activator', '-n', 'knative-serving'])
        run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'app=autoscaler-hpa', '-n', 'knative-serving'])
        run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'app=controller', '-n', 'knative-serving'])
        run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'app=webhook', '-n', 'knative-serving'])
        run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'app=autoscaler', '-n', 'knative-serving'])

        # Install Ambassador
        install_ambassador(namespace=namespace, envs=[
            {
                'name': 'AMBASSADOR_KNATIVE_SUPPORT',
                'value': 'true'
            },
            {
                'name': 'AMBASSADOR_SINGLE_NAMESPACE',
                'value': 'true'
            }
        ])

        # Install QOTM
        apply_kube_artifacts(namespace=namespace, artifacts=qotm_manifests)
        create_qotm_mapping(namespace=namespace)

        # Now let's wait for ambassador and QOTM pods to become ready
        run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'service=ambassador', '-n', namespace])
        run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'pod', '-l', 'service=qotm', '-n', namespace])

        # Create kservice
        apply_kube_artifacts(namespace=namespace, artifacts=knative_service_example)

        # Let's port-forward ambassador service to talk to QOTM
        port_forward_port = 7000
        port_forward_command = ['kubectl', 'port-forward', '--namespace', namespace, 'service/ambassador', f'{port_forward_port}:80']
        run_and_assert(port_forward_command, communicate=False)

        # Assert 200 OK at /qotm/ endpoint
        qotm_url = f'http://localhost:{port_forward_port}/qotm/'
        qotm_http_code = self.get_code_with_retry(qotm_url)
        assert qotm_http_code == 200, f"Expected 200 OK, got {qotm_http_code}"
        print(f"{qotm_url} is ready")

        # Assert 200 OK at / with Knative Host header and 404 with other/no header
        kservice_url = f'http://localhost:{port_forward_port}/'

        req_simple = request.Request(kservice_url)
        connection_simple_code = self.get_code_with_retry(req_simple)
        assert connection_simple_code == 404, f"Expected 404, got {connection_simple_code}"
        print(f"{kservice_url} returns 404 with no host")

        req_random = request.Request(kservice_url)
        req_random.add_header('Host', 'random.host.whatever')
        connection_random_code = self.get_code_with_retry(req_random)
        assert connection_random_code == 404, f"Expected 404, got {connection_random_code}"
        print(f"{kservice_url} returns 404 with a random host")

        # Wait for kservice
        run_and_assert(['kubectl', 'wait', '--timeout=90s', '--for=condition=Ready', 'ksvc', 'helloworld-go', '-n', namespace])

        req_correct = request.Request(kservice_url)
        req_correct.add_header('Host', f'helloworld-go.{namespace}.example.com')

        # kservice pod takes some time to spin up, so let's try a few times
        connection_correct_code = 000
        for _ in range(5):
            connection_correct_code = self.get_code_with_retry(req_correct)
            if connection_correct_code == 200:
                break

        assert connection_correct_code == 200, f"Expected 200, got {connection_correct_code}"
        print(f"{kservice_url} returns 200 OK with host helloworld-go.default.example.com")


def test_knative_counters():
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(knative_ingress_example, k8s=True)
    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")
    ir = IR(aconf, secret_handler=secret_handler)
    feats = ir.features()

    assert feats['knative_ingress_count'] == 1, f"Expected a Knative ingress, did not find one"
    assert feats['cluster_ingress_count'] == 0, f"Expected no Knative cluster ingresses, found at least one"


def test_knative():
    if is_knative():
        knative_test = KnativeTesting()
        knative_test.test_knative()
    else:
        pytest.xfail("Knative is not supported")


if __name__ == '__main__':
    pytest.main(sys.argv)
