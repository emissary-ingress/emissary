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

    def queries(self):
        if False: yield

    def check(self):
        pass

class AmbassadorTest(QueryTest):

    @abstractmethod
    def yaml(self) -> str:
        pass

class ConfigTest(AmbassadorTest):

    @classmethod
    def variants(cls):
        yield variant(variants(MappingTest))

    def __init__(self, mappings = ()):
        self.mappings = list(mappings)

    def assemble(self, pattern):
        result = []
        amb_yaml = yaml_check(self.yaml, Tag.MAPPING)
        if amb_yaml is not None:
            for m in amb_yaml:
                m["ambassador_id"] = self.name.lower()
        k8s_yaml = yaml_check(self.k8s_yaml, Tag.MAPPING)
        if amb_yaml is not None:
            for item in k8s_yaml:
                if item["kind"].lower() == "service":
                    item["metadata"]["annotations"] = { "getambassador.io/config": dump(amb_yaml) }
                    break
        result.extend(k8s_yaml)

        for m in self.mappings:
            if m.matches(pattern):
                result.extend(m.assemble(pattern))
        return result

    def k8s_yaml(self) -> str:
        return templates.ambassador(self.name.lower())

    @abstractmethod
    def scheme(self) -> str:
        pass

    def url(self, prefix) -> str:
        return "%s://ambassador-%s/%s" % (self.scheme(), self.name.lower(), prefix)

class ServiceType(Node):

    mapping: 'MappingTest'

    @classmethod
    def variants(cls):
        yield variant()

class HTTP(ServiceType):
    pass

class GRPC(ServiceType):
    pass

class MappingTest(QueryTest):

    target: ServiceType
    options: Sequence['MappingOptionTest']
    config: ConfigTest

    def __init__(self, target: ServiceType, options = ()) -> None:
        self.target = target
        self.options = list(options)

    def assemble(self, pattern):
        mappings = yaml_check(self.yaml, Tag.MAPPING)
        for m in mappings:
            m["ambassador_id"] = self.parent.name.lower()
        me = mappings[0]
        for opt in self.options:
            if opt.matches(pattern):
                for o in yaml_check(opt.yaml, Tag.MAPPING):
                    me.merge(o)

        k8s_yaml = yaml_check(self.k8s_yaml, Tag.MAPPING)
        for item in k8s_yaml:
            if item["kind"].lower() == "service":
                item["metadata"]["annotations"] = { "getambassador.io/config": dump(mappings) }
                break

        return k8s_yaml

    def k8s_yaml(self):
        return templates.backend(self.target.path.k8s)

class MappingOptionTest(QueryTest):

    mapping: MappingTest

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

    def yaml(self):
        return """
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

    def yaml(self):
        return """
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
            for mot in variants(MappingOptionTest):
                yield variant(st, (mot,), name="{self.target.name}-{self.options[0].name}")
            yield variant(st, unique(variants(MappingOptionTest)), name="{self.target.name}-all")

    def yaml(self):
        return self.format("""
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

class AddRequestHeaders(MappingOptionTest):

    VALUES = ({"foo": "bar"},
              {"moo": "arf"})

    def yaml(self):
        return "add_request_headers: %s" % json.dumps(self.value)

    def check(self):
        for r in self.parent.results:
            for k, v in self.value.items():
                actual = r.backend.request.headers.get(k.capitalize())
                assert actual == [v], (actual, [v])

class CaseSensitive(MappingOptionTest):

    def yaml(self):
        return "case_sensitive: false"

    def queries(self):
        for q in self.parent.queries():
            idx = q.url.find("/", q.url.find("://") + 3)
            upped = q.url[:idx] + q.url[idx:].upper()
            assert upped != q.url
            yield Query(upped, xfail="this is broken")

class AutoHostRewrite(MappingOptionTest):

    def yaml(self):
        return "auto_host_rewrite: true"

    def check(self):
        pytest.xfail("this doesn't work for some reason")
        for r in self.parent.results:
            host = r.backend.request.url.host
            assert r.backend.name == host, (r.backend.name, host)

class Rewrite(MappingOptionTest):

    VALUES = ("/foo", "foo")

    def yaml(self):
        return self.format("rewrite: {self.value}")

    def queries(self):
        if self.value[0] != "/":
            for q in self.parent.pending:
                q.xfail = "rewrite option is broken for values not beginning in slash"
        return super(MappingOptionTest, self).queries()

    def check(self):
        if self.value[0] != "/":
            pytest.xfail("this is broken")
        for r in self.parent.results:
            assert r.backend.request.url.path == self.value

class CanaryMapping(MappingTest):

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            for w in (10, 50, 90):
                yield variant(v, v.clone("canary"), w, name="{self.target.name}-{self.weight}")

    def __init__(self, target, canary, weight):
        MappingTest.__init__(self, target)
        self.canary = canary
        self.weight = weight

    def yaml(self):
        return self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.k8s}
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

    def k8s_yaml(self):
        return templates.backend(self.target.path.k8s) + "\n" + templates.backend(self.canary.path.k8s)

    def check(self):
        hist = {}
        for r in self.results:
            hist[r.backend.name] = hist.get(r.backend.name, 0) + 1
        canary = 100*hist.get(self.canary.path.k8s, 0)/len(self.results)
        main = 100*hist.get(self.target.path.k8s, 0)/len(self.results)
        assert abs(self.weight - canary) < 10, (self.weight, canary)

### NEXT STEPS: fix assemble and friends to use better traversal/discovery technique

# Test docs:
#  test methods:
#
#     variants() -> test generator
#
#     config() -> amb config
#
#     manifests() -> k8s config
#
#     urls() -> Sequence[Tuple[ID, URL]]
#
#     validate() -> validates results
