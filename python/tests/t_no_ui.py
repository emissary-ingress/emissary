from kat.harness import Query, EDGE_STACK

from abstract_tests import AmbassadorTest, ServiceType, HTTP


class NoUITest (AmbassadorTest):
    def manifests(self) -> str:
        return self.format("""
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  config:
    diagnostics:
      enabled: false
""") + super().manifests()

    def queries(self):
        yield(Query(self.url("ambassador/v0/diag/"), expected=404))
        yield(Query(self.url("edge_stack/admin/"), expected=404))

