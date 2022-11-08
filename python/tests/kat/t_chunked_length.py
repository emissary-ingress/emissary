from typing import Generator, Tuple, Union

from abstract_tests import HTTP, AmbassadorTest, Node, ServiceType
from kat.harness import Query


class AllowChunkedLengthTestTrue(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: ambassador
kind:  Module
name:  ambassador
config:
  allow_chunked_length: true
---
apiVersion: getambassador.io/v3alpha1
kind:  Mapping
name:  {self.target.path.k8s}-foo
prefix: /foo/
hostname: "*"
service: {self.target.path.fqdn}
"""
        )

    def queries(self):
        yield Query(self.url("foo/"))
        yield Query(self.url("ambassador/v0/diag/"))
        yield Query(self.url("foo/"), headers={"content-length": "0", "transfer-encoding": "gzip"})
        yield Query(
            self.url("ambassador/v0/diag/"),
            headers={"content-length": "0", "transfer-encoding": "gzip"},
        )

    def check(self):
        # Not getting a 400 bad request is confirmation that this setting works as long as the request can reach the upstream
        assert self.results[0].status == 200
        assert self.results[1].status == 200
        assert self.results[2].status == 200
        assert self.results[3].status == 200
