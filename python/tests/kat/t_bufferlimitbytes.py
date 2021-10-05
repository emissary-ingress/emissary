from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP
import json

class BufferLimitBytesTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    # Test generating config with an increased buffer and that the lua body() funciton runs to buffer the request body
    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind:  Module
name:  ambassador
config:
  buffer_limit_bytes: 5242880
  lua_scripts: |
    function envoy_on_request(request_handle)
      request_handle:headers():add("request_body_size", request_handle:body():length())
    end
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.target.path.k8s}-foo
hostname: "*"
prefix: /foo/
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("foo/"))
        yield Query(self.url("ambassador/v0/diag/"))   

    def check(self):
        assert self.results[0].status == 200
        assert self.results[1].status == 200