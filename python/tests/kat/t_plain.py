from typing import Generator, Tuple, Union

from kat.harness import variants, Query, EDGE_STACK

from abstract_tests import AmbassadorTest, MappingTest, Node
from tests.integration.manifests import namespace_manifest

import t_mappingtests_plain
import t_optiontests

# Plain is the place that all the MappingTests get pulled in.


class Plain(AmbassadorTest):
    single_namespace = True
    namespace = "plain-namespace"

    @classmethod
    def variants(cls) -> Generator[Node, None, None]:
        yield cls(variants(MappingTest))

    def manifests(self) -> str:
        m = (
            namespace_manifest("plain-namespace")
            + namespace_manifest("evil-namespace")
            + """
---
kind: Service
apiVersion: v1
metadata:
  name: plain-simplemapping-http-all-http
  namespace: evil-namespace
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: getambassador.io/v3alpha1
      kind: Mapping
      name: SimpleMapping-HTTP-all
      hostname: "*"
      prefix: /SimpleMapping-HTTP-all/
      service: http://plain-simplemapping-http-all-http.plain
      ambassador_id: [plain]
      ---
      apiVersion: getambassador.io/v3alpha1
      kind: Host
      name: cleartext-host-{self.path.k8s}
      ambassador_id: [ "plain" ]
      hostname: "*"
      mappingSelector:
        matchLabels:
          hostname: {self.path.k8s}
      acmeProvider:
        authority: none
      requestPolicy:
        insecure:
          action: Route
          # additionalPort: 8080
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
"""
        )

        if EDGE_STACK:
            m += """
---
kind: Service
apiVersion: v1
metadata:
  name: cleartext-host-{self.path.k8s}
  namespace: plain-namespace
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: getambassador.io/v3alpha1
      kind: Host
      name: cleartext-host-{self.path.k8s}
      ambassador_id: [ "plain" ]
      hostname: "*"
      mappingSelector:
        matchLabels:
          hostname: {self.path.k8s}
      acmeProvider:
        authority: none
      requestPolicy:
        insecure:
          action: Route
          # Since this is cleartext already, additionalPort: 8080 is technically
          # an error. Leave it in to make sure it's a harmless no-op error.
          additionalPort: 8080
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
"""

        return m + super().manifests()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, """
---
apiVersion: getambassador.io/v3alpha1
kind:  Module
name:  ambassador
config: {}
"""

    def queries(self):
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json

        # We shouldn't have any missing-CRD-types errors any more.
        for source, error in errors:
            if ("could not find" in error) and ("CRD definitions" in error):
                assert False, f"Missing CRDs: {error}"

            if "Ingress resources" in error:
                assert False, f"Ingress resource error: {error}"

        # The default errors assume that we have missing CRDs, and that's not correct any more,
        # so don't try to use assert_default_errors here.
