from typing import Generator, Tuple, Union

from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP, Node

class AllowHeadersWithUnderscoresTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  config__dump
ambassador_id: [{self.ambassador_id}]
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

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: [{self.ambassador_id}]
config:
  headers_with_underscores_action: REJECT_REQUEST
""")
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  config__dump
ambassador_id: [{self.ambassador_id}]
hostname: "*"
prefix: /target/
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("target/"), expected=400, headers={'t_underscore':'foo'})

    def check(self):
        assert self.results[0].status == 400
