import json

from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, EGRPC

class AcceptanceGrpcBridgeTest(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = EGRPC()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
    enable_grpc_http11_bridge: True
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
grpc: True
name:  {self.target.path.k8s}
prefix: /echoservice.Echo/
prefix: /target/
service: {self.target.path.k8s}
""")

    def queries(self):
        # [0]
        yield Query(self.url("target/"), headers={  "content-type": "application/grpc", 
                                                    "requested-status": "0" }, expected=200)
