from typing import Tuple, Union

from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, Node, ServiceType


class GzipMinimumConfigTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  gzip:
    enabled: true
""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("target/"), headers={"Accept-Encoding": "gzip"}, expected=200)

    def check(self):
        assert self.results[0].headers["Content-Encoding"] == [ "gzip" ]


class GzipTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  gzip:
    min_content_length: 32
    window_bits: 15
    content_type:
    - text/plain
""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("target/"), headers={"Accept-Encoding": "gzip"}, expected=200)

    def check(self):
        assert self.results[0].headers["Content-Encoding"] == [ "gzip" ]


class GzipNotSupportedContentTypeTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  gzip:
    min_content_length: 32
    content_type:
    - application/json
""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("target/"), headers={"Accept-Encoding": "gzip"}, expected=200)

    def check(self):
        assert "Content-Encoding" not in self.results[0].headers
