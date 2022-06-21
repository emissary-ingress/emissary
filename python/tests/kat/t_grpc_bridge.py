from typing import Generator, Tuple, Union

from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, EGRPC, Node

class AcceptanceGrpcBridgeTest(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = EGRPC()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind:  Module
name:  ambassador
config:
    enable_grpc_http11_bridge: True
""")

        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
grpc: True
hostname: "*"
prefix: /echo.EchoService/
rewrite: /echo.EchoService/
name:  {self.target.path.k8s}
service: {self.target.path.k8s}
""")

    def queries(self):
        # [0]
        yield Query(self.url("echo.EchoService/Echo"),
                    headers={ "content-type": "application/grpc", "kat-req-echo-requested-status": "0" },
                    expected=200,
                    grpc_type="bridge")

        # [1]
        yield Query(self.url("echo.EchoService/Echo"),
                    headers={ "content-type": "application/grpc", "kat-req-echo-requested-status": "7" },
                    expected=503,
                    grpc_type="bridge")

    def check(self):
        # [0]
        assert self.results[0].headers["Grpc-Status"] == ["0"]

        # [1]
        assert self.results[1].headers["Grpc-Status"] == ["7"]
