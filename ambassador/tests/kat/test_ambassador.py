import json
import pytest

from typing import Any, Iterable, Optional, Sequence, Type

from kat.harness import abstract_test, variant, variants, Query, Runner
from kat import manifests

from abstract_tests import AmbassadorTest, MappingTest, OptionTest, ServiceType

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

#class Empty(ConfigTest):

#    def yaml(self):
#        return ""

#    def scheme(self) -> str:
#        return "http"

class Plain(AmbassadorTest):

    def config(self):
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

    VALUES = ({"foo": "bar"},
              {"moo": "arf"})

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
            yield Query(upped, xfail="this is broken")

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

t = Runner(AmbassadorTest)
