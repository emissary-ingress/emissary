import json

from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, EGRPC

class AcceptanceGrpcWebTest(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = EGRPC()

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind:  Module
name:  ambassador
config:
    enable_grpc_web: True
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
        # print('AcceptanceGrpcWebTest results:')
        #
        # i = 0
        #
        # for result in self.results:
        #     print(f'{i}: {json.dumps(result.as_dict(), sort_keys=True, indent=4)}')
        #     print(f'    headers {json.dumps(result.headers, sort_keys=True)}')
        #     print(f'    grpc-status {json.dumps(result.headers.get("Grpc-Status", "-none-"), sort_keys=True)}')
        #     i += 1

        # [0]
        gstat = self.results[0].headers.get("Grpc-Status", "-none-")
        # print(f'    grpc-status {gstat}')

        if gstat == [ '0' ]:
            assert True
        else:
            assert False, f'0: got {gstat} instead of ["0"]'

        # # [0]
        # assert self.results[0].headers["Grpc-Status"] == ["0"]

        # [1]
        assert self.results[1].headers["Grpc-Status"] == ["7"]
