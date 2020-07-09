from typing import Tuple, Union

from kat.harness import Query, EDGE_STACK

from abstract_tests import AmbassadorTest, assert_default_errors, HTTP, Node, ServiceType
from kat.utils import namespace_manifest


class Empty(AmbassadorTest):
    single_namespace = True
    namespace = "empty-namespace"
    extra_ports = [8877]

    def init(self):
        if EDGE_STACK:
            self.xfail = "XFailing for now"

    @classmethod
    def variants(cls):
        yield cls()

    def manifests(self) -> str:
        return namespace_manifest("empty-namespace") + super().manifests()

    def config(self) -> Union[str, Tuple[Node, str]]:
        yield from ()

    def queries(self):
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)
        yield Query(self.url("_internal/v0/ping", scheme="http", port=8877), expected=403)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json

        # We should _not_ be seeing Ingress errors here.
        assert_default_errors(errors, include_ingress_errors=False)


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
config: 
  use_ambassador_namespace_for_service_resolution: true
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


class SafeRegexMapping(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
prefix_regex: true
host: "[a-zA-Z].*"
host_regex: true
regex_headers:
  X-Foo: "^[a-z].*"
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url(self.name + "/"), headers={"X-Foo": "hello"})
        yield Query(self.url(f'need-normalization/../{self.name}/'), headers={"X-Foo": "hello"})
        yield Query(self.url(self.name + "/"), expected=404)
        yield Query(self.url(f'need-normalization/../{self.name}/'), expected=404)

    def check(self):
        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}/'


class UnsafeRegexMapping(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
prefix_regex: true
host: "[a-zA-Z].*"
host_regex: true
regex_headers:
  X-Foo: "^[a-z].*"
service: http://{self.target.path.fqdn}
---
apiVersion: ambassador/v2
kind:  Module
name:  ambassador
config:
  regex_type: unsafe
""")

    def queries(self):
        yield Query(self.url(self.name + "/"), headers={"X-Foo": "hello"})
        yield Query(self.url(f'need-normalization/../{self.name}/'), headers={"X-Foo": "hello"})
        yield Query(self.url(self.name + "/"), expected=404)
        yield Query(self.url(f'need-normalization/../{self.name}/'), expected=404)

    def check(self):
        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)
                assert r.backend.request.headers['x-envoy-original-path'][0] == f'/{self.name}/'
