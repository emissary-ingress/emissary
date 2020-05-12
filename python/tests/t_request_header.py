from kat.harness import variants, Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP

class XRequestIdHeaderPreserveTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  preserve_external_request_id: true
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}-target
prefix: /target/
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("target/"), headers={"x-request-id": "hello"})

    def check(self):
        assert self.results[0].backend.request.headers['x-request-id'] == ['hello']

class XRequestIdHeaderDefaultTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.xfail = "Need to figure out passing header through external connections from KAT"
        self.target = HTTP(name="target")

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador

---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}-target
prefix: /target/
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("target/"), headers={"X-Request-Id": "hello"})

    def check(self):
        assert self.results[0].backend.request.headers['x-request-id'] != ['hello']
