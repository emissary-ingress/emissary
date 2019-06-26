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


class AmbassadorIDTest(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self) -> Union[str, Tuple[Node, str]]:
        yield self, """
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config: {}
"""
        for prefix, amb_id in (("findme", "{self.ambassador_id}"),
                               ("findme-array", "[{self.ambassador_id}, missme]"),
                               ("findme-array2", "[missme, {self.ambassador_id}]"),
                               ("missme", "missme"),
                               ("missme-array", "[missme1, missme2]")):
            yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.path.k8s}-{prefix}
prefix: /{prefix}/
service: {self.target.path.fqdn}
ambassador_id: {amb_id}
            """, prefix=self.format(prefix), amb_id=self.format(amb_id))

    def queries(self):
        yield Query(self.url("findme/"))
        yield Query(self.url("findme-array/"))
        yield Query(self.url("findme-array2/"))
        yield Query(self.url("missme/"), expected=404)
        yield Query(self.url("missme-array/"), expected=404)


class ServerNameTest(AmbassadorTest):

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
  server_name: "test-server"
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.path.k8s}/server-name
prefix: /server-name
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("server-name/"), expected=301)

    def check(self):
        assert self.results[0].headers["Server"] == [ "test-server" ]
