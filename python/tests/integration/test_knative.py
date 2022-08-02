import logging
import os
import sys
import time

import pytest
from retry import retry

import tests.integration.manifests as integration_manifests
from ambassador import IR, Config
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler, parse_bool
from kat.harness import is_knative_compatible
from tests.integration.utils import create_qotm_mapping, get_code_with_retry, install_ambassador
from tests.kubeutils import apply_kube_artifacts, delete_kube_artifacts
from tests.manifests import qotm_manifests
from tests.runutils import run_and_assert, run_with_retry

logger = logging.getLogger("ambassador")

# knative_service_example gets applied to the cluster with `kubectl --namespace=knative-testing
# apply`; we therefore DO NOT explicitly set the 'namespace:' because --namespace will imply it, and
# explicitly setting anything only adds room for something else to go wrong.
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

# knative_ingress_example is not applied to the cluster, but is instead fed directly to the
# ResourceFetcher; so we MUST explicitly set the namespace, because we can't rely on kubectl and/or
# the apiserver to auto-populate it for us.
knative_ingress_example = """
apiVersion: networking.internal.knative.dev/v1alpha1
kind: Ingress
metadata:
  name: helloworld-go
  namespace: default
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
          servicePort: 80
        timeout: 10m0s
    visibility: ClusterLocal
  visibility: ExternalIP
"""


class KnativeTesting:
    def test_knative(self):
        namespace = "knative-testing"

        # Install Knative
        apply_kube_artifacts(
            namespace=None, artifacts=integration_manifests.load("knative_serving_crds")
        )
        apply_kube_artifacts(
            namespace="knative-serving",
            artifacts=integration_manifests.load("knative_serving_0.18.0"),
        )
        run_and_assert(
            [
                "tools/bin/kubectl",
                "patch",
                "configmap/config-network",
                "--type",
                "merge",
                "--patch",
                r'{"data": {"ingress.class": "ambassador.ingress.networking.knative.dev"}}',
                "-n",
                "knative-serving",
            ]
        )

        # Wait for Knative to become ready
        run_and_assert(
            [
                "tools/bin/kubectl",
                "wait",
                "--timeout=90s",
                "--for=condition=Ready",
                "pod",
                "-l",
                "app=activator",
                "-n",
                "knative-serving",
            ]
        )
        run_and_assert(
            [
                "tools/bin/kubectl",
                "wait",
                "--timeout=90s",
                "--for=condition=Ready",
                "pod",
                "-l",
                "app=controller",
                "-n",
                "knative-serving",
            ]
        )
        run_and_assert(
            [
                "tools/bin/kubectl",
                "wait",
                "--timeout=90s",
                "--for=condition=Ready",
                "pod",
                "-l",
                "app=webhook",
                "-n",
                "knative-serving",
            ]
        )
        run_and_assert(
            [
                "tools/bin/kubectl",
                "wait",
                "--timeout=90s",
                "--for=condition=Ready",
                "pod",
                "-l",
                "app=autoscaler",
                "-n",
                "knative-serving",
            ]
        )

        # Install Ambassador
        install_ambassador(
            namespace=namespace, envs=[{"name": "AMBASSADOR_KNATIVE_SUPPORT", "value": "true"}]
        )

        # Install QOTM
        apply_kube_artifacts(namespace=namespace, artifacts=qotm_manifests)
        create_qotm_mapping(namespace=namespace)

        # Now let's wait for ambassador and QOTM pods to become ready
        run_and_assert(
            [
                "tools/bin/kubectl",
                "wait",
                "--timeout=90s",
                "--for=condition=Ready",
                "pod",
                "-l",
                "service=ambassador",
                "-n",
                namespace,
            ]
        )
        run_and_assert(
            [
                "tools/bin/kubectl",
                "wait",
                "--timeout=90s",
                "--for=condition=Ready",
                "pod",
                "-l",
                "service=qotm",
                "-n",
                namespace,
            ]
        )

        # Create kservice
        apply_kube_artifacts(namespace=namespace, artifacts=knative_service_example)

        # Assume we can reach Ambassador through telepresence
        qotm_host = "ambassador." + namespace

        # Assert 200 OK at /qotm/ endpoint
        qotm_url = f"http://{qotm_host}/qotm/"
        code = get_code_with_retry(qotm_url)
        assert code == 200, f"Expected 200 OK, got {code}"
        print(f"{qotm_url} is ready")

        # Assert 200 OK at / with Knative Host header and 404 with other/no header
        kservice_url = f"http://{qotm_host}/"

        code = get_code_with_retry(kservice_url)
        assert code == 404, f"Expected 404, got {code}"
        print(f"{kservice_url} returns 404 with no host")

        code = get_code_with_retry(kservice_url, headers={"Host": "random.host.whatever"})
        assert code == 404, f"Expected 404, got {code}"
        print(f"{kservice_url} returns 404 with a random host")

        # Wait for kservice
        run_and_assert(
            [
                "tools/bin/kubectl",
                "wait",
                "--timeout=90s",
                "--for=condition=Ready",
                "ksvc",
                "helloworld-go",
                "-n",
                namespace,
            ]
        )

        # kservice pod takes some time to spin up, so let's try a few times
        code = 000
        host = f"helloworld-go.{namespace}.example.com"
        for _ in range(5):
            code = get_code_with_retry(kservice_url, headers={"Host": host})
            if code == 200:
                break

        assert code == 200, f"Expected 200, got {code}"
        print(f"{kservice_url} returns 200 OK with host helloworld-go.{namespace}.example.com")


def test_knative_counters():
    aconf = Config()
    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(knative_ingress_example, k8s=True)
    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")
    ir = IR(aconf, secret_handler=secret_handler)
    feats = ir.features()

    assert feats["knative_ingress_count"] == 1, f"Expected a Knative ingress, did not find one"
    assert (
        feats["cluster_ingress_count"] == 0
    ), f"Expected no Knative cluster ingresses, found at least one"


@pytest.mark.flaky(reruns=1, reruns_delay=10)
def test_knative():
    if not parse_bool(os.environ.get("AMBASSADOR_PYTEST_KNATIVE_TEST", "false")):
        pytest.xfail("AMBASSADOR_PYTEST_KNATIVE_TEST is not set, xfailing...")

    if is_knative_compatible():
        knative_test = KnativeTesting()
        knative_test.test_knative()
    else:
        pytest.xfail("Knative is not supported")


if __name__ == "__main__":
    pytest.main(sys.argv)
