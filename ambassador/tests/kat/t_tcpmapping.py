import json

from kat.harness import Query, Test

from abstract_tests import AmbassadorTest, ServiceType, HTTP

class TCPMappingTest(Test):

    target: ServiceType
    # options: Sequence['OptionTest']
    parent: AmbassadorTest

    @classmethod
    def variants(cls):
        yield cls(HTTP(), name="{self.target.name}")

    def init(self, target: ServiceType, options=()) -> None:
        self.target = target
        self.options = list(options)

class TCPPortMapping(TCPMappingTest):

    parent: AmbassadorTest
    #
    # @classmethod
    # def variants(cls):
    #     yield cls(HTTP(), name="{self.target.name}")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}
port: 9876
service: {self.target.path.fqdn}:80
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"), expected=404)
