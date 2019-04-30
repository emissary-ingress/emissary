from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, EGRPC

class AcceptanceGrpcWebTest(AmbassadorTest):

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
    enable_grpc_web: True
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
grpc: True
prefix: /echo.EchoService/
rewrite: /echo.EchoService/
name:  {self.target.path.k8s}
service: {self.target.path.k8s}
""")

    def queries(self):
        # [0]
        yield Query(self.url("echo.EchoService/Echo"),
                    headers={ "content-type": "application/grpc-web-text",
                              "accept": "application/grpc-web-text",
                              "requested-status": "0" },
                    expected=200,
                    grpc_type="web")

        # [1]
        yield Query(self.url("echo.EchoService/Echo"),
                    headers={ "content-type": "application/grpc-web-text",
                              "accept": "application/grpc-web-text",
                              "requested-status": "7" },
                    expected=200,
                    grpc_type="web")

    def check(self):
        # [0]
        assert self.results[0].headers["Grpc-Status"] == ["0"]

        # [1]
        assert self.results[1].headers["Grpc-Status"] == ["7"]
