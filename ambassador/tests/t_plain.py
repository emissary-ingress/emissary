from typing import Tuple, Union

from kat.harness import variants, Query

from abstract_tests import AmbassadorTest, assert_default_errors
from abstract_tests import MappingTest, Node

# Plain is the place that all the MappingTests get pulled in.


class Plain(AmbassadorTest):
    single_namespace = True
    namespace = "plain-namespace"

    @classmethod
    def variants(cls):
        yield cls(variants(MappingTest))

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Namespace
metadata:
  name: plain-namespace
---
apiVersion: v1
kind: Namespace
metadata:
  name: evil-namespace
---
kind: Service
apiVersion: v1
metadata:
  name: plain-simplemapping-http-all-http
  namespace: evil-namespace
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind: Mapping
      name: SimpleMapping-HTTP-all
      prefix: /SimpleMapping-HTTP-all/
      service: http://plain-simplemapping-http-all-http.plain
      ambassador_id: plain
  labels:
    scope: AmbassadorTest
spec:
  selector:
    backend: plain-simplemapping-http-all-http
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 8080
  - name: https
    protocol: TCP
    port: 443
    targetPort: 8443
""" + super().manifests()

    def config(self) -> Union[str, Tuple[Node, str]]:
        yield self, """
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config: {}
"""

    def queries(self):
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json
        assert_default_errors(errors)
