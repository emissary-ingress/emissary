from kat.harness import Query, is_knative
from kat.manifests import KNATIVE_SERVING_071, KNATIVE_SERVING_080

from packaging import version

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


class Knative071Test(AmbassadorTest):
    target: ServiceType

    def init(self) -> None:
        if not is_knative():
            self.skip_node = True

        self.target = HTTP()

    def manifests(self) -> str:
        self.manifest_envs = """
    - name: AMBASSADOR_KNATIVE_SUPPORT
      value: "true"
"""

        return super().manifests() + KNATIVE_SERVING_071 + KNATIVE_EXAMPLE

    def queries(self):
        yield Query(self.url(""), expected=404)
        yield Query(self.url(""), headers={'Host': 'random.host.whatever'}, expected=404)
        yield Query(self.url(""), headers={'Host': 'helloworld-go.default.example.com'}, expected=200)


class Knative080Test(AmbassadorTest):
    target: ServiceType

    def init(self) -> None:
        if not is_knative():
            self.skip_node = True

        self.target = HTTP()

    def manifests(self) -> str:
        self.manifest_envs = """
    - name: AMBASSADOR_KNATIVE_SUPPORT
      value: "true"
"""

        return super().manifests() + KNATIVE_SERVING_080 + KNATIVE_EXAMPLE

    def queries(self):
        yield Query(self.url(""), expected=404)
        yield Query(self.url(""), headers={'Host': 'random.host.whatever'}, expected=404)
        yield Query(self.url(""), headers={'Host': 'helloworld-go.default.example.com'}, expected=200)
