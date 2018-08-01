#!/usr/bin/env python3

from abc import abstractmethod
from typing import Any, Iterable, Optional, Sequence, Type
from parser import SequenceView

import json
import pprint
import templates

from harness import sanitize, variant, variants, Node, Test
from parser import load, dump, Tag, SequenceView

def yaml_check(gen, *tags: Tag) -> Optional[SequenceView]:
    st = gen()
    if st is None: return None
    return load(gen.__name__, st, *tags)

class AmbassadorTest(Test):

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
        return templates.AMBASSADOR % {"name": self.name.lower()}

    @abstractmethod
    def scheme(self) -> str:
        pass

    def url(self, prefix) -> str:
        return "%s://ambassador-%s/%s" % (self.scheme(), self.name.lower(), prefix)

    def urls(self):
        return (u for m in self.mappings for u in m.urls())

class ServiceType(Node):

    mapping: 'MappingTest'

    @classmethod
    def variants(cls):
        yield variant()

class HTTP(ServiceType):
    pass

class GRPC(ServiceType):
    pass

class MappingTest(Test):

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
        return templates.BACKEND % {"name": self.target.k8s_path}

class MappingOptionTest(Test):

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
            # need to figure out how to write this when there are conflicting option tests
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

    def urls(self):
        yield {"url": self.parent.url(self.name + "/")}

class AddRequestHeaders(MappingOptionTest):

    VALUES = ({"foo": "bar"},
              {"moo": "arf"})

    def yaml(self):
        return "add_request_headers: %s" % json.dumps(self.value)

class CaseSensitive(MappingOptionTest):

    def yaml(self):
        return "case_sensitive: false"

class AutoHostRewrite(MappingOptionTest):

    def yaml(self):
        return "auto_host_rewrite: true"

class Rewrite(MappingOptionTest):

    VALUES = ("/foo", "foo")

    def yaml(self):
        return self.format("rewrite: {self.value}")


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

    def urls(self):
        for i in range(25):
            yield {"url": self.parent.url(self.name + "/")}

    def k8s_yaml(self):
        return templates.BACKEND % {"name": self.target.k8s_path} + "\n" + templates.BACKEND % {"name": self.canary.k8s_path}

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
