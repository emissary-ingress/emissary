#!/usr/bin/env python3

from abc import ABC, abstractmethod
from parser import load, dump, Tag, SequenceView
from typing import Sequence

class Test(ABC):

    def name(self) -> str:
        return self.__class__.__name__

    @abstractmethod
    def yaml(self) -> str:
        pass

    def yaml_check(self, *tags: Tag) -> SequenceView:
        seq = load(self.name(), self.yaml())
        for o in seq:
            if o.tag not in tags:
                raise ValueError("test %s expecting %s, got %s" % (self.name(), ", ".join(t.name for t in tags),
                                                                   o.node.tag))
        return seq

class ConfigTest(Test):

    def __init__(self):
        self.mappings = []

    def add_mapping(self, m):
        m.config = self
        self.mappings.append(m)

    def assemble(self):
        result = self.yaml_check(Tag.MAPPING)
        for m in self.mappings:
            result.extend(m.assemble())
        return result

    def services(self):
        for m in self.mappings:
            yield m.target

class ServiceType:

    mapping: 'MappingTest'

class HTTP(ServiceType):

    def __str__(self):
        return "HTTP_%s" % self.mapping.name()

class GRPC(ServiceType):

    def __str__(self):
        return "GRPC_%s" % self.mapping.name()

class MappingTest(Test):

    target: ServiceType
    options: Sequence['MappingOptionTest']

    def __init__(self, target: ServiceType) -> None:
        target.mapping = self
        self.target = target
        self.options = []

    @classmethod
    def apply_options(cls):
        return False

    def add_option(self, o):
        o.mapping = self
        self.options.append(o)

    def assemble(self):
        mappings = self.yaml_check(Tag.MAPPING)
        me = mappings[0]
        for opt in self.options:
            for o in opt.yaml_check(Tag.MAPPING):
                me.update(o)
        return [me]

class MappingOptionTest(Test):
    pass


import inspect

def get_type(type):
    for k, v in globals().items():
        if inspect.isclass(v):
            if issubclass(v, type) and v != type:
                yield v

def get_configs():
    return get_type(ConfigTest)

def get_service_types():
    return get_type(ServiceType)

def get_mappings():
    return get_type(MappingTest)

def get_mapping_options():
    return get_type(MappingOptionTest)

def collect():
    permutations = []
    for cfg in get_configs():
        # we create a new permutation for each top level config we
        # want to test no idea if that is appropriate... I should
        # prolly read what we can configure
        c = cfg()
        permutations.append(c)
        for st in get_service_types():
            # we instantiate every mapping test for every service type
            for m in get_mappings():
                c.add_mapping(m(st()))
                # if the mapping tests says it's ok, we apply option
                # tests to the mapping
                if m.apply_options():
                    loaded = m(st())
                    for mo in get_mapping_options():
                        loaded.add_option(mo())
                        isolated = m(st())
                        isolated.add_option(mo())
                        c.add_mapping(isolated)
                    c.add_mapping(loaded)
    return permutations

def go():
    for p in collect():
        print("==services==")
        print(" ".join(str(s) for s in p.services()))
        print("==configuration==")
        print(dump(p.assemble()), end="")

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

class Plain(ConfigTest):

    def yaml(self):
        return "{}"

class SimpleMapping(MappingTest):

    @classmethod
    def apply_options(cls):
        return True

    def yaml(self):
        return """
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: http://%s
""" % self.target

class AddRequestHeaders(MappingOptionTest):

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

    def yaml(self):
        return """
---
apiVersion: ambassador/v0
kind:  Mapping
name:  group_mapping
prefix: /group/
service: http://%s
""" % self.target

go()
