from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP
import json

class AllowHeadersWithUnderscoresTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  config__dump
ambassador_id: {self.ambassador_id}
hostname: "*"
prefix: /target/
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("target/"), expected=200, headers={'t_underscore':'foo'})

    def check(self):
        assert self.results[0].status == 200

class RejectHeadersWithUnderscoresTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind: Module
name: ambassador
ambassador_id: {self.ambassador_id}
config:
  headers_with_underscores_action: REJECT_REQUEST
""")
        yield self, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  config__dump
ambassador_id: {self.ambassador_id}
hostname: "*"
prefix: /target/
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("target/"), expected=400, headers={'t_underscore':'foo'})

    def check(self):
        assert self.results[0].status == 400
