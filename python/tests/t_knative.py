import pytest

from kat.harness import Query, is_knative, load_manifest

KNATIVE_SERVING_0110 = load_manifest("knative_serving_0.11.0")

from abstract_tests import AmbassadorTest, HTTP, ServiceType

KNATIVE_EXAMPLE = """
---
apiVersion: serving.knative.dev/v1alpha1
kind: Service
metadata:
 name: helloworld-go
 namespace: default
spec:
 template:
   spec:
     containers:
     - image: gcr.io/knative-samples/helloworld-go
       env:
       - name: TARGET
         value: "Ambassador is Awesome"
"""


class Knative0110Test(AmbassadorTest):
    target: ServiceType

    no_local_mode = True

    def init(self) -> None:
        self.target = HTTP()

    def manifests(self) -> str:
        if is_knative():
            self.manifest_envs = """
    - name: AMBASSADOR_KNATIVE_SUPPORT
      value: "true"
"""

            return super().manifests() + KNATIVE_SERVING_0110 + KNATIVE_EXAMPLE
        else:
            return super().manifests()

    def queries(self):
        if is_knative():
            yield Query(self.url(""), expected=404)
            yield Query(self.url(""), headers={'Host': 'random.host.whatever'}, expected=404)
            yield Query(self.url(""), headers={'Host': 'helloworld-go.default.example.com'}, expected=200)
        else:
            yield from ()

    def check(self):
        if not is_knative():
            pytest.xfail("Knative is not supported")

        return super().check()
