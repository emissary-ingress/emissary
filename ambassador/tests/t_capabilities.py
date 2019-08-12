from typing import Tuple, Union

from kat.harness import Query

from abstract_tests import AmbassadorTest, assert_default_errors, HTTP, Node, ServiceType


class Empty(AmbassadorTest):
    single_namespace = True
    namespace = "empty-namespace"

    @classmethod
    def variants(cls):
        yield cls()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Namespace
metadata:
  name: empty-namespace
""" + super().manifests()

    def config(self) -> Union[str, Tuple[Node, str]]:
        yield from ()

    def queries(self):
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json
        assert_default_errors(errors)


class LowPortTest(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
       capabilities_block = f"""
  capabilities:
    add: ["NET_BIND_SERVICE"]
"""
       return self.format(RBAC_CLUSTER_SCOPE + AMBASSADOR,
                           image=os.environ["AMBASSADOR_DOCKER_IMAGE"],
                           envs=envs,
                           extra_ports="",
                           capabilities_block=capabilities_block)

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  service_port: 82
""")

    def queries(self):
        yield Query(self.url("server-name/", "http", 8099), expected=399)

    def check(self):
        assert self.results[0].headers["Server"] == [ "test-server" ]

