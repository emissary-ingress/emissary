from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, EGRPC

class AcceptanceGrpcTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = EGRPC()

    def config(self):
#         yield self, self.format("""
# ---
# apiVersion: getambassador.io/v2
# kind:  Module
# name:  ambassador
# # """)

        yield self, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
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
        assert self.results[0].headers["Grpc-Status"] == ["0"], f'0 expected ["0"], got {self.results[0].headers["Grpc-Status"]}'

        # [1]
        assert self.results[1].headers["Grpc-Status"] == ["7"], f'0 expected ["0"], got {self.results[0].headers["Grpc-Status"]}'

        # [2]
        # XXX Ew. If self.results[2].json is empty, the harness won't convert it to a response.
        errors = self.results[2].json
        assert(len(errors) == 0)


class EndpointGrpcTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = EGRPC()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: getambassador.io/v2
kind: KubernetesEndpointResolver
metadata:
    name: my-endpoint
spec:    
    ambassador_id: ["endpointgrpctest"]
''') + super().manifests()

    def config(self):
        yield self, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
grpc: True
hostname: "*"
prefix: /echo.EchoService/
rewrite: /echo.EchoService/
name:  {self.target.path.k8s}
service: {self.target.path.k8s}
resolver: my-endpoint
load_balancer:
  policy: round_robin
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
        assert self.results[0].headers["Grpc-Status"] == ["0"], f'results[0]: expected ["0"], got {self.results[0].headers["Grpc-Status"]}'

        # [1]
        assert self.results[1].headers["Grpc-Status"] == ["7"], f'results[1]: expected ["7"], got {self.results[0].headers["Grpc-Status"]}'

        # [2]
        # XXX Ew. If self.results[2].json is empty, the harness won't convert it to a response.
        errors = self.results[2].json
        assert(len(errors) == 0)
