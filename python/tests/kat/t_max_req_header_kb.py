from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP
import json

class MaxRequestHeaderKBTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
ambassador_id: {self.ambassador_id}
config:
  max_request_headers_kb: 30
""")
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /target/
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        h1 = 'i' * (31 * 1024)
        yield Query(self.url("target/"), expected=431,
                headers={'big':h1})
        h2 = 'i' * (29 * 1024)
        yield Query(self.url("target/"), expected=200,
                headers={'small':h2})

    def check(self):
        # We're just testing the status codes above, so nothing to check here
        assert True

class MaxRequestHeaderKBMaxTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
ambassador_id: {self.ambassador_id}
config:
  max_request_headers_kb: 96
""")
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}
host: "*"
prefix: /target/
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        # without the override the response headers will cause envoy to respond with a 503
        h1 = 'i' * (97 * 1024)
        yield Query(self.url("target/?override_extauth_header=1"), expected=431,
                headers={'big':h1})
        h2 = 'i' * (95 * 1024)
        yield Query(self.url("target/?override_extauth_header=1"), expected=200,
                headers={'small':h2})

    def check(self):
        # We're just testing the status codes above, so nothing to check here
        assert True
