from typing import Tuple, Union

from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, Node, ServiceType

# class GzipTest(AmbassadorTest):

#     target: ServiceType

#     def init(self):
#         self.target = HTTP()

#     def config(self):
#         yield self, self.format("""
# ---
# apiVersion: ambassador/v0
# kind:  Module
# name:  ambassador
# config:
#   gzip:
#     min_content_length: 12
# ---
# apiVersion: ambassador/v0
# kind:  Mapping
# name:  {self.path.k8s}
# prefix: /
# service: {self.target.path.fqdn}
# """)

    # def queries(self):
    #     yield Query(self.url("/"), headers={"Accept-Encoding": "gzip"}, expected=200)

    # def check(self):
    #     assert self.results[0].headers["Content-Encoding"] == [ "gzip" ]


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
    min_content_length: 12
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
        # [0]
        yield Query(self.url("target/"), headers={"Accept-Encoding": "gzip"}, expected=200)
        
    def check(self):
        assert self.results[0].headers["Content-Encoding"] == [ "gzip" ]

    # def check(self):
    #     # [0] Verifies all request headers sent to the authorization server.
    #     assert self.results[0].backend.name == self.auth.path.k8s, f'wanted backend {self.auth.path.k8s}, got {self.results[0].backend.name}'
    #     assert self.results[0].backend.request.url.path == "/extauth/target/"
    #     assert self.results[0].backend.request.headers["content-length"]== ["0"]
    #     assert "x-forwarded-for" in self.results[0].backend.request.headers
    #     assert "user-agent" in self.results[0].backend.request.headers
    #     assert "baz" not in self.results[0].backend.request.headers
    #     assert self.results[0].status == 401
    #     assert self.results[0].headers["Server"] == ["envoy"]
    