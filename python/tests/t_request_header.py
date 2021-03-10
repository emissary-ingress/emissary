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


# Sanity test that Envoy headers are present if we do not suppress them
class EnvoyHeadersTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}-target
prefix: /target/
rewrite: /rewrite/
timeout_ms: 5001
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("target/"))

    def check(self):
        print("results[0]=%s" % repr(self.results[0]))
        headers = self.results[0].backend.request.headers

        # All known Envoy headers should be set. The original path header is
        # include here because we made sure to include a rewrite in the Mapping.
        assert headers['x-envoy-expected-rq-timeout-ms'] == ['5001']
        assert headers['x-envoy-original-path'] == ['/target/']

# Sanity test that we can suppress Envoy headers when configured
class SuppressEnvoyHeadersTest(AmbassadorTest):
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
  suppress_envoy_headers: true
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}-target
prefix: /target/
rewrite: /rewrite/
timeout_ms: 5001
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("target/"))

    def check(self):
        print("results[0]=%s" % repr(self.results[0]))
        headers = self.results[0].backend.request.headers

        # No Envoy headers should be set
        assert 'x-envoy-expected-rq-timeout-ms' not in headers
        assert 'x-envoy-original-path' not in headers
