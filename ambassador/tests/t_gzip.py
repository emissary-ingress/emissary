from typing import Tuple, Union

from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, Node, ServiceType

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
    content_length: 128
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.path.k8s}/backend
prefix: /backend
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("backend/"), headers={"Accept-Encoding": "gzip"}, expected=200)

    def check(self):
        assert self.results[0].headers["Content-Encoding"] == [ "gzip" ]
