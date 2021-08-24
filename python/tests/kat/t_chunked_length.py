from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP
import json

class AllowChunkedLengthTestTrue(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind:  Module
name:  ambassador
config:
  allow_chunked_length: true
---
apiVersion: x.getambassador.io/v3alpha1
kind:  AmbassadorMapping
name:  {self.target.path.k8s}-foo
prefix: /foo/
hostname: "*"
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("foo/"))
        yield Query(self.url("ambassador/v0/diag/"))
        yield Query(self.url("foo/"),
            headers={
                "content-length": "0",
                "transfer-encoding": "gzip"
        })
        yield Query(self.url("ambassador/v0/diag/"),
            headers={
                "content-length": "0",
                "transfer-encoding": "gzip"
        })

    def check(self):
        # Not getting a 400 bad request is confirmation that this setting works as long as the request can reach the upstream
        assert self.results[0].status == 200
        assert self.results[1].status == 200
        assert self.results[2].status == 200
        assert self.results[3].status == 200
