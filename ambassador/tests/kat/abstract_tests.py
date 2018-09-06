from abc import abstractmethod
from typing import Any, Iterable, Optional, Sequence, Type

from kat.harness import abstract_test, sanitize, variant, variants, Node, Test
from kat import manifests

@abstract_test
class AmbassadorTest(Test):

    @classmethod
    def variants(cls):
        yield variant(variants(MappingTest))

    def __init__(self, mappings = ()):
        self.mappings = list(mappings)

    # XXX: should use format for manifests and change templates to manifests
    def manifests(self) -> str:
        return self.format(manifests.AMBASSADOR, image="quay.io/datawire/ambassador:0.35.3")

    @abstractmethod
    def scheme(self) -> str:
        pass

    def url(self, prefix) -> str:
        return "%s://%s/%s" % (self.scheme(), self.name.lower(), prefix)

    def requirements(self):
        yield ("pod", "%s" % self.name.k8s)

@abstract_test
class ServiceType(Node):

    def config(self):
        if False: yield

    def manifests(self):
        return self.format(manifests.BACKEND)

    def requirements(self):
        yield ("pod", self.path.k8s)

class HTTP(ServiceType):
    pass

class GRPC(ServiceType):
    pass

@abstract_test
class MappingTest(Test):

    target: ServiceType
    options: Sequence['OptionTest']

    def __init__(self, target: ServiceType, options = ()) -> None:
        self.target = target
        self.options = list(options)

@abstract_test
class OptionTest(Test):

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
