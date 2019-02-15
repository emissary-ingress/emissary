# import json

from kat.harness import variants, Query
from abstract_tests import AmbassadorTest, MappingTest, ServiceType


class HeaderRoutingTest(MappingTest):
    debug = True
    parent: AmbassadorTest
    target: ServiceType
    target2: ServiceType
    weight: int

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            yield cls(v, v.clone("target2"), name="{self.target.name}")

    def init(self, target: ServiceType, target2: ServiceType):
        MappingTest.init(self, target)
        self.target2 = target2

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.k8s}
""")
        yield self.target2, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-canary
prefix: /{self.name}/
service: http://{self.target2.path.k8s}
headers:
    X-Route: target2
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"))
        yield Query(self.parent.url(self.name + "/"), headers={"X-Route": "target2"})

    def check(self):
        assert self.results[0].backend.name == self.target.path.k8s, f"r0 wanted {self.target.name} got {self.results[0].backend.name}"
        assert self.results[1].backend.name == self.target2.path.k8s, f"r1 wanted {self.target2.name} got {self.results[1].backend.name}"


