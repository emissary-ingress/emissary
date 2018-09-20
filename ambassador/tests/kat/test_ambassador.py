import json
import pytest

from typing import ClassVar, Dict, Sequence, Tuple, Union

from kat.harness import abstract_test, _instantiate, sanitize, variant, variants, Query, Runner
from kat import manifests

from abstract_tests import AmbassadorBaseTest, AmbassadorMixin, AmbassadorTest, MappingTest, OptionTest, ServiceType, Node, Test

class TLS(AmbassadorMixin, AmbassadorTest):
    VARIES_BY = MappingTest

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

#class Empty(ConfigTest):

#    def yaml(self):
#        return ""

#    def scheme(self) -> str:
#        return "http"


class Plain(AmbassadorMixin, AmbassadorTest):
    VARIES_BY = MappingTest

    def config(self) -> Union[str, Tuple[Node, str]]:
        yield self, """
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config: {}
"""

    def scheme(self) -> str:
        return "http"


# XXX: should put this somewhere better
def unique(variants):
    added = set()
    result = []
    for v in variants:
        if v.cls not in added:
            added.add(v.cls)
            result.append(v)
    return tuple(result)

class SimpleMapping(MappingTest):

    @classmethod
    def variants(cls):
        for st in variants(ServiceType):
            yield variant(st, name="{self.target.name}")
            for mot in variants(OptionTest):
                yield variant(st, (mot,), name="{self.target.name}-{self.options[0].name}")
            yield variant(st, unique(variants(OptionTest)), name="{self.target.name}-all")

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

class CanaryMapping(MappingTest):

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            for w in (10, 50):
                yield variant(v, v.clone("canary"), w, name="{self.target.name}-{self.weight}")

    def __init__(self, target, canary, weight):
        MappingTest.__init__(self, target)
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


class AmbassadorIDTest (Test):

    @classmethod
    def variants(cls):
        yield variant(variants(AmbassadorIDInnerTest))

    def config(self):
        if False: yield

    def manifests(self):
        m = self.format(manifests.BACKEND)
        print("AmbassadorIDTest: %s" % m)
        return m

    def requirements(self):
        yield ("pod", self.path.k8s)


class AmbassadorIDInnerTest (AmbassadorMixin, AmbassadorBaseTest):
    def config(self) -> Union[ str, Tuple[ Node, str ] ]:
        yield self, """
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config: {}
"""

    # # You MUST define scheme in order for this class not to be considered abstract.
    def scheme(self) -> str:
        return "http"

    @classmethod
    def variants(cls):
        c = 0
        for v in variants(AmbassadorIDOptions):
            yield variant(v, name=str(c))
            c += 1

    def __init__(self, idoptions: 'AmbassadorIDOptions'):
        super().__init__(mappings=(), ambassador_id=idoptions.value)

    #     ambid = idoptions.value
    #     print("AmbassadorIDInnerTest ambid %s" % ambid)
    #
    #     self.plain = Plain(mappings=(), ambassador_id=ambid)
    #     self.plain.name = "{self.name}-%s" % sanitize(ambid)
    #
    #     self.config = self.plain.config
    #     self.manifests = self.plain.manifests
    #     self.requirements = self.plain.requirements


class AmbassadorIDOptions (Test):

    @classmethod
    def variants(cls):
        for val in [
            "id_test_one",
            [ "id_test_one", "id_test_two" ]
        ]:
            yield variant(val, name=sanitize(val))

    def __init__(self, value = None):
        self.value = value

    def check(self):
        print("%s CHECK HO" % self.name)
        assert False, "kaboom"


# pytest will find this because Runner is a toplevel callable object in a file
# that pytest is willing to look inside.
#
# Also note:
# - Runner(cls) will look for variants of _every subclass_ of cls.
# - Any class you pass to Runner needs to be standalone (it must have its
#   own manifests and be able to set up its own world).
main = Runner(AmbassadorTest, AmbassadorIDTest)
