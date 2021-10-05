from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP
import json

# tests that using logical_dns does not impact requests
# strict_dns is already the default setting so we only need to validate it's config pytest
class LogicalDnsType(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-foo
prefix: /foo/
service: {self.target.path.fqdn}
hostname: "*"
dns_type: logical_dns
""")

    def queries(self):
        yield Query(self.url("foo/"))

    def check(self):
        # Not getting a 400 bad request is confirmation that this setting works as long as the request can reach the upstream
        assert self.results[0].status == 200

# this test is just to confirm that when both dns_type and the endpoint resolver are in use, the endpoint resolver wins
class LogicalDnsTypeEndpoint(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-foo
prefix: /foo/
service: {self.target.path.fqdn}
hostname: "*"
dns_type: logical_dns
resolver: endpoint
""")

    def queries(self):
        yield Query(self.url("foo/"))

    def check(self):
        # Not getting a 400 bad request is confirmation that this setting works as long as the request can reach the upstream
        assert self.results[0].status == 200

# Tests Respecting DNS TTL alone
class DnsTtl(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind:  Mapping
name:  {self.target.path.k8s}-foo
prefix: /foo/
service: {self.target.path.fqdn}
respect_dns_ttl: true
""")

    def queries(self):
        yield Query(self.url("foo/"))

    def check(self):
        # Not getting a 400 bad request is confirmation that this setting works as long as the request can reach the upstream
        assert self.results[0].status == 200

# Tests the DNS TTL with the endpoint resolver
class DnsTtlEndpoint(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind:  Mapping
name:  {self.target.path.k8s}-foo
prefix: /foo/
service: {self.target.path.fqdn}
respect_dns_ttl: true
resolver: endpoint
""")

    def queries(self):
        yield Query(self.url("foo/"))

    def check(self):
        # Not getting a 400 bad request is confirmation that this setting works as long as the request can reach the upstream
        assert self.results[0].status == 200

# Tests the DNS TTL with Logical DNS type
class DnsTtlDnsType(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind:  Mapping
name:  {self.target.path.k8s}-foo
prefix: /foo/
service: {self.target.path.fqdn}
respect_dns_ttl: true
dns_type: logical_dns
""")

    def queries(self):
        yield Query(self.url("foo/"))

    def check(self):
        # Not getting a 400 bad request is confirmation that this setting works as long as the request can reach the upstream
        assert self.results[0].status == 200
