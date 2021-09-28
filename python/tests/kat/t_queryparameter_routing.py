from kat.harness import variants, Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP

class QueryParameterRoutingTest(AmbassadorTest):
    target1: ServiceType
    target2: ServiceType

    def init(self):
        self.target1 = HTTP(name="target1")
        self.target2 = HTTP(name="target2")

    def config(self):
        yield self.target1, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-target1
host: "*"
prefix: /target/
service: http://{self.target1.path.fqdn}
""")
        yield self.target2, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-target2
host: "*"
prefix: /target/
service: http://{self.target2.path.fqdn}
query_parameters:
    test_param: target2
""")

    def queries(self):
        yield Query(self.url("target/"), expected=200)
        yield Query(self.url("target/?test_param=target2"), expected=200)

    def check(self):
        assert self.results[0].backend.name == self.target1.path.k8s, f"r0 wanted {self.target1.path.k8s} got {self.results[0].backend.name}"
        assert self.results[1].backend.name == self.target2.path.k8s, f"r1 wanted {self.target2.path.k8s} got {self.results[1].backend.name}"

class QueryParameterRoutingWithRegexTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self.target, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-target
host: "*"
prefix: /target/
service: http://{self.target.path.fqdn}
regex_query_parameters:
    test_param: "^[a-z].*"
""")

    def queries(self):
        yield Query(self.url("target/?test_param=hello"), expected=200)

        # These should not match the regex and therefore not be found
        yield Query(self.url("target/"), expected=404)
        yield Query(self.url("target/?test_param=HeLlO"), expected=404)

    def check(self):
        assert self.results[0].backend.name == self.target.path.k8s, f"r0 wanted {self.target.path.k8s} got {self.results[0].backend.name}"

class QueryParameterPresentRoutingTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self.target, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-target
host: "*"
prefix: /target/
service: http://{self.target.path.fqdn}
query_parameters:
    test_param: true
""")

    def queries(self):
        yield Query(self.url("target/?test_param=true"), expected=200)
        yield Query(self.url("target/"), expected=404)

    def check(self):
        assert self.results[0].backend.name == self.target.path.k8s, f"r0 wanted {self.target.path.k8s} got {self.results[0].backend.name}"
