import json
import pytest

from typing import ClassVar, Dict, List, Sequence, Tuple, Union

from kat.harness import sanitize, variants, Query, Runner
from kat import manifests

from abstract_tests import AmbassadorTest, HTTP
from abstract_tests import MappingTest, OptionTest, ServiceType, Node, Test

# XXX: should test empty ambassador config

class TLS(AmbassadorTest):

    def config(self):
        yield self, """
---
apiVersion: ambassador/v0
kind: Module
name: tls
config:
  server:
    enabled: False
  client:
    enabled: False
        """

    def scheme(self) -> str:
        return "http"


class Plain(AmbassadorTest):

    @classmethod
    def variants(cls):
        yield cls(variants(MappingTest))

    def config(self) -> Union[str, Tuple[Node, str]]:
        yield self, """
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config: {}
"""


def unique(options):
    added = set()
    result = []
    for o in options:
        if o.__class__ not in added:
            added.add(o.__class__)
            result.append(o)
    return tuple(result)

class SimpleMapping(MappingTest):

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield cls(st, name="{self.target.name}")
            for mot in variants(OptionTest):
                yield cls(st, (mot,), name="{self.target.name}-{self.options[0].name}")
            yield cls(st, unique(variants(OptionTest)), name="{self.target.name}-all")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.k8s}
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"))

    def check(self):
        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.path.k8s, (r.backend.name, self.target.path.k8s)


class AddRequestHeaders(OptionTest):

    VALUES: ClassVar[Sequence[Dict[str, str]]] = (
        { "foo": "bar" },
        { "moo": "arf" }
    )

    def config(self):
        yield "add_request_headers: %s" % json.dumps(self.value)

    def check(self):
        for r in self.parent.results:
            for k, v in self.value.items():
                actual = r.backend.request.headers.get(k.lower())
                assert actual == [v], (actual, [v])


class CaseSensitive(OptionTest):

    def config(self):
        yield "case_sensitive: false"

    def queries(self):
        for q in self.parent.queries():
            idx = q.url.find("/", q.url.find("://") + 3)
            upped = q.url[:idx] + q.url[idx:].upper()
            assert upped != q.url
            yield Query(upped)


class AutoHostRewrite(OptionTest):

    def config(self):
        yield "auto_host_rewrite: true"

    def check(self):
        for r in self.parent.results:
            host = r.backend.request.host
            assert r.backend.name == host, (r.backend.name, host)


class Rewrite(OptionTest):

    VALUES = ("/foo", "foo")

    def config(self):
        yield self.format("rewrite: {self.value}")

    def queries(self):
        if self.value[0] != "/":
            for q in self.parent.pending:
                q.xfail = "rewrite option is broken for values not beginning in slash"
        return super(OptionTest, self).queries()

    def check(self):
        if self.value[0] != "/":
            pytest.xfail("this is broken")
        for r in self.parent.results:
            assert r.backend.request.url.path == self.value


class TLSOrigination(MappingTest):

    IMPLICIT = """
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: https://{self.target.path.k8s}
    """

    EXPLICIT = """
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: {self.target.path.k8s}
tls: true
    """

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            for name, dfn in ("IMPLICIT", cls.IMPLICIT), ("EXPLICIT", cls.EXPLICIT):
                yield cls(v, dfn, name="{self.target.name}-%s" % name)

    def init(self, target, definition):
        MappingTest.init(self, target)
        self.definition = definition

    def config(self):
        yield self.target, self.format(self.definition)

    def queries(self):
        yield Query(self.parent.url(self.name + "/"))

    def check(self):
        for r in self.results:
            assert r.backend.request.tls.enabled


class CanaryMapping(MappingTest):

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            for w in (10, 50):
                yield cls(v, v.clone("canary"), w, name="{self.target.name}-{self.weight}")

    def init(self, target, canary, weight):
        MappingTest.init(self, target)
        self.canary = canary
        self.weight = weight

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.k8s}
""")
        yield self.canary, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-canary
prefix: /{self.name}/
service: http://{self.canary.path.k8s}
weight: {self.weight}
""")

    def queries(self):
        for i in range(100):
            yield Query(self.parent.url(self.name + "/"))

    def check(self):
        hist = {}
        for r in self.results:
            hist[r.backend.name] = hist.get(r.backend.name, 0) + 1
        canary = 100*hist.get(self.canary.path.k8s, 0)/len(self.results)
        main = 100*hist.get(self.target.path.k8s, 0)/len(self.results)
        assert abs(self.weight - canary) < 25, (self.weight, canary)


class AmbassadorIDTest(AmbassadorTest):

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
service: {self.target.path.k8s}
ambassador_id: {amb_id}
            """, prefix=self.format(prefix), amb_id=self.format(amb_id))

    def queries(self):
        yield Query(self.url("findme/"))
        yield Query(self.url("findme-array/"))
        yield Query(self.url("findme-array2/"))
        yield Query(self.url("missme/"), expected=404)
        yield Query(self.url("missme-array/"), expected=404)

class AuthenticationTest(AmbassadorTest):

    def init(self):
        self.target = HTTP()
        self.auth = HTTP(name="auth")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.k8s}"
path_prefix: "/extauth"
allowed_headers:
- "x-extauth-required"
- "x-authenticated-as"
- "x-qotm-session"
- "requested-status"
- "requested-header"
- "location"
""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.k8s}
""")

    def queries(self):
        yield Query(self.url("target/"), headers={"requested-status": "401"}, expected=401)
        yield Query(self.url("target/"), headers={"requested-status": "302",
                                                  "location": "foo",
                                                  "requested-header": "location"}, expected=302)
        yield Query(self.url("target/"))

    def check(self):
        assert self.results[0].backend.name == self.auth.path.k8s
        assert self.results[0].backend.request.url.path == "/extauth/target/"

        assert self.results[1].backend.name == self.auth.path.k8s
        assert self.results[1].backend.response.headers["location"] == ["foo"]
        assert self.results[1].backend.request.url.path == "/extauth/target/"

        assert self.results[2].backend.name == self.target.path.k8s
        assert self.results[2].backend.request.url.path == "/"


# pytest will find this because Runner is a toplevel callable object in a file
# that pytest is willing to look inside.
#
# Also note:
# - Runner(cls) will look for variants of _every subclass_ of cls.
# - Any class you pass to Runner needs to be standalone (it must have its
#   own manifests and be able to set up its own world).
main = Runner(AmbassadorTest)
