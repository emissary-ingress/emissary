#!/usr/bin/env python3

from abc import abstractmethod
from collections import OrderedDict
from typing import Any, Iterable, Optional, Sequence, Type
from parser import SequenceView

import json
import pprint
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

    def summary(self):
        statuses = OrderedDict()
        failures = 0
        for r in self.results:
            key = r.status or r.error
            statuses[key] = statuses.get(key, 0) + 1
            if r.status != r.query.expected:
                failures += 1
        result = "%s %s" % (self.path, " ".join("%s*%s" % (v, k) if v > 1 else str(k) for k, v in statuses.items()))
        if failures > 0:
            result += " \033[91mERR\033[0m"
        else:
            result += " \033[92mOK\033[0m"
        return result

    def check(self):
        if self.results:
            print(self.summary())

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
        return templates.backend(self.target.k8s_path)

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

def go():
    from harness import cli
    cli(AmbassadorTest)

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
service: http://{self.target.k8s_path}
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"))

    def check(self):
        if self.results:
            print(self.summary())
        for r in self.results:
            if r.backend:
                assert r.backend.name == self.target.k8s_path, (r.backend.name, self.target.k8s_path)

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
            yield Query(upped)

class AutoHostRewrite(MappingOptionTest):

    def yaml(self):
        return "auto_host_rewrite: true"

    def check(self):
        for r in self.parent.results:
            print(self.path, r.backend.request.url.host or None, self.parent.target.k8s_path)

class Rewrite(MappingOptionTest):

    VALUES = ("/foo", "foo")

    def yaml(self):
        return self.format("rewrite: {self.value}")

    def check(self):
        for r in self.parent.results:
            if r.backend:
                path = r.backend.request.url.path
                assert path == self.value
            else:
                path = None
            print(self.path, repr(path), repr(self.value))

class CanaryMapping(MappingTest):

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            for w in (33, 50, 75):
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
service: http://{self.target.k8s_path}
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-canary
prefix: /{self.name}/
service: http://{self.canary.k8s_path}
weight: {self.weight}
""")

    def queries(self):
        for i in range(25):
            yield Query(self.parent.url(self.name + "/"))

    def k8s_yaml(self):
        return templates.backend(self.target.k8s_path) + "\n" + templates.backend(self.canary.k8s_path)

    def check(self):
        if self.results:
            print(self.summary())
        hist = {}
        for r in self.results:
            hist[r.backend.name] = hist.get(r.backend.name, 0) + 1
        print("  " + ", ".join("%s: %s" % (k, 100*v/len(self.results)) for k, v in sorted(hist.items())))

go()

### NEXT STEPS: wire in assertions

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
