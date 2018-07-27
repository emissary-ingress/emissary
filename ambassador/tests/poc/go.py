#!/usr/bin/env python3

from typing import Any, Iterable, Optional, Sequence, Type

import pprint
import templates

from harness import choice, variants, Test

class ConfigTest(Test):

    @classmethod
    def variants(cls):
        yield (cls, variants(MappingTest))

    def __init__(self, mappings = ()):
        self.mappings = list(mappings)
        for m in mappings:
            m.config = self

    def assemble(self):
        result = []
        amb_yaml = self.yaml_check(self.yaml, Tag.MAPPING)
        if amb_yaml is not None:
            for m in amb_yaml:
                m["ambassador_id"] = self.name().lower()
        k8s_yaml = self.yaml_check(self.k8s_yaml, Tag.MAPPING)
        if amb_yaml is not None:
            for item in k8s_yaml:
                if item["kind"].lower() == "service":
                    item["metadata"]["annotations"] = { "getambassador.io/config": dump(amb_yaml) }
                    break
        result.extend(k8s_yaml)

        for m in self.mappings:
            result.extend(m.assemble())
        return result

    def services(self):
        for m in self.mappings:
            yield m.target

    def k8s_yaml(self) -> str:
        return templates.AMBASSADOR % {"name": self.name().lower()}

class ServiceType:

    mapping: 'MappingTest'

    @classmethod
    def variants(cls):
        yield (cls,)

class HTTP(ServiceType):

    def __str__(self):
        return self.mapping.name()

class GRPC(ServiceType):

    def __str__(self):
        return self.mapping.name()

class MappingTest(Test):

    target: ServiceType
    suffix: str
    options: Sequence['MappingOptionTest']
    config: ConfigTest

    def __init__(self, target: ServiceType, suffix: str = "", options = ()) -> None:
        target.mapping = self
        self.target = target
        self.suffix = suffix
        self.options = list(options)
        for o in self.options:
            o.mapping = self

    def name(self):
        return self.target.__class__.__name__ + "-" + Test.name(self) + self.suffix

    def assemble(self):
        mappings = self.yaml_check(self.yaml, Tag.MAPPING)
        for m in mappings:
            m["ambassador_id"] = self.config.name().lower()
        me = mappings[0]
        for opt in self.options:
            for o in opt.yaml_check(opt.yaml, Tag.MAPPING):
                me.update(o)

        k8s_yaml = self.yaml_check(self.k8s_yaml, Tag.MAPPING)
        for item in k8s_yaml:
            if item["kind"].lower() == "service":
                item["metadata"]["annotations"] = { "getambassador.io/config": dump(mappings) }

        return k8s_yaml

    def k8s_yaml(self):
        return templates.BACKEND % {"name": str(self.target).lower()}

class MappingOptionTest(Test):

    mapping: MappingTest

    VALUES: Any = None

    @classmethod
    def variants(cls):
        if cls.VALUES is None:
            yield (cls,)
        else:
            yield (cls, choice(cls.VALUES))

    def __init__(self, value = None):
        self.value = value

    def name(self):
        if self.value is None:
            return Test.name(self)
        else:
            return Test.name(self) + "[" + str(self.value) + "]"

def go():
    vars = tuple(v.instantiate() for v in variants(ConfigTest))
    for v in vars:
#        print("--")
#        pprint.pprint(v, indent=2)
#        print(dump(v.assemble()), end="")
        v.list()

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
    enabled: True
  client:
    enabled: False
        """

#class Empty(ConfigTest):

#    def yaml(self):
#        return ""

class Plain(ConfigTest):

    def yaml(self):
        return """
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config: {}
"""

class SimpleMapping(MappingTest):

    @classmethod
    def variants(cls):
        yield (cls, choice(variants(ServiceType)))
        yield (cls, choice(variants(ServiceType)), "-isolated", (choice(variants(MappingOptionTest)),))
        # need to figure out how to write this when there are conflicting option tests
        yield (cls, choice(variants(ServiceType)), "-loaded", variants(MappingOptionTest))

    def yaml(self):
        return """
---
apiVersion: ambassador/v0
kind:  Mapping
name:  %s
prefix: /%s/
service: http://%s
""" % (self.name(), self.name(), self.target)

class AddRequestHeaders(MappingOptionTest):

    VALUES = ({"foo": "bar"},
              {"moo": "arf"})

    def yaml(self):
        return """
add_request_headers:
  foo: bar
        """

class CaseSensitive(MappingOptionTest):

    def yaml(self):
        return "case_sensitive: false"

class SimpleOption2(MappingOptionTest):

    def yaml(self):
        return "option2: foo"

class SimpleOption3(MappingOptionTest):

    def yaml(self):
        return "option3: 3"


class GroupMapping(MappingTest):

    @classmethod
    def variants(cls):
        yield (cls, choice(variants(ServiceType)))

    def yaml(self):
        return """
---
apiVersion: ambassador/v0
kind:  Mapping
name:  %s
prefix: /%s/
service: http://%s
""" % (self.name(), self.name(), self.target)

go()

### NEXT STEPS: write cli, wire in driver, wire in assertions

# Bugs found:
# - an empty annotation causes a crash: hard to track down what is at fault when this happens
#  + possibly need better architectural isolation so errors are more targeted to inputs at a lower level
