#!/usr/bin/env python3

from abc import abstractmethod
from collections import OrderedDict
from typing import Any, Iterable, Optional, Sequence, Type
from parser import SequenceView

import json
import pprint
import pytest
import templates

from harness import sanitize, variant, variants, Node, Query, Test
from parser import load, dump, Tag, SequenceView

def yaml_check(gen, *tags: Tag) -> Optional[SequenceView]:
    st = gen()
    if st is None: return None
    return load(gen.__name__, st, *tags)

class QueryTest(Test):

    def config(self):
        if False: yield

    def manifests(self):
        return None

    def queries(self):
        if False: yield

    def check(self):
        pass

    @property
    def ambassador_id(self):
        return self.parent.ambassador_id

class AmbassadorTest(QueryTest):
    pass

class ConfigTest(AmbassadorTest):

    @classmethod
    def variants(cls):
        yield variant(variants(MappingTest))

    def __init__(self, mappings = ()):
        self.mappings = list(mappings)

    @property
    def ambassador_id(self):
        return self.name.k8s

    # XXX: should use format for manifests and change templates to manifests
    def manifests(self) -> str:
        return templates.ambassador(self.name.k8s)

    @abstractmethod
    def scheme(self) -> str:
        pass

    def url(self, prefix) -> str:
        return "%s://ambassador-%s/%s" % (self.scheme(), self.name.lower(), prefix)

class ServiceType(Node):

    def config(self):
        if False: yield

    def manifests(self):
        return templates.backend(self.path.k8s)

class HTTP(ServiceType):
    pass

class GRPC(ServiceType):
    pass

class MappingTest(QueryTest):

    target: ServiceType
    options: Sequence['OptionTest']

    def __init__(self, target: ServiceType, options = ()) -> None:
        self.target = target
        self.options = list(options)

class OptionTest(QueryTest):

    VALUES: Any = None

    @classmethod
    def variants(cls):
        if cls.VALUES is None:
            yield variant()
        else:
            for val in cls.VALUES:
                yield variant(val, name=sanitize(val))

    def __init__(self, value = None):
        self.value = value

##################################

class TLS(ConfigTest):

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

class Plain(ConfigTest):

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

# NEXT: docs, readiness probes, backend history queries

# Test docs:
#  test methods:
#
#     variants() -> test generator
#
#     config() -> amb config
#
#     manifests() -> k8s config
#
#     queries() -> Sequence[Query]
#
#     check() -> validates results
