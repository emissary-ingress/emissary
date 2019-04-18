from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, EGRPC

class AcceptanceGrpcTest(AmbassadorTest):

    # Yes, enable endpoints here. It needs to work with them enabled but
    # not used, after all.
    enable_endpoints = True

    target: ServiceType

    def init(self):
        self.target = EGRPC()

    def config(self):
#         yield self, self.format("""
# ---
# apiVersion: ambassador/v0
# kind:  Module
# name:  ambassador
# # """)

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
                    headers={ "content-type": "application/grpc", "requested-status": "0" },
                    expected=200,
                    grpc_type="real")

        # [1]
        yield Query(self.url("echo.EchoService/Echo"),
                    headers={ "content-type": "application/grpc", "requested-status": "7" },
                    expected=200,
                    grpc_type="real")

        # [2] -- PHASE 2
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)

    def check(self):
        # [0]
        assert self.results[0].headers["Grpc-Status"] == ["0"]

        # [1]
        assert self.results[1].headers["Grpc-Status"] == ["7"]

        # [2]
        # XXX Ew. If self.results[2].json is empty, the harness won't convert it to a response.
        errors = self.results[2].json
        assert(len(errors) == 0)


class EndpointGrpcTest(AmbassadorTest):

    enable_endpoints = True

    target: ServiceType

    def init(self):
        self.target = EGRPC()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
grpc: True
prefix: /echo.EchoService/
rewrite: /echo.EchoService/
name:  {self.target.path.k8s}
service: {self.target.path.k8s}
resolver: endpoint
""")

    def queries(self):
        # [0]
        yield Query(self.url("echo.EchoService/Echo"),
                    headers={ "content-type": "application/grpc", "requested-status": "0" },
                    expected=200,
                    grpc_type="real")

        # [1]
        yield Query(self.url("echo.EchoService/Echo"),
                    headers={ "content-type": "application/grpc", "requested-status": "7" },
                    expected=200,
                    grpc_type="real")

        # [2] -- PHASE 2
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)

    def check(self):
        # [0]
        assert self.results[0].headers["Grpc-Status"] == ["0"]

        # [1]
        assert self.results[1].headers["Grpc-Status"] == ["7"]

        # [2]
        # XXX Ew. If self.results[2].json is empty, the harness won't convert it to a response.
        errors = self.results[2].json
        assert(len(errors) == 0)
