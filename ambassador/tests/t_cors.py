# Note that there's also a CORS OptionTest in t_optiontests.py.

from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, Node, ServiceType


class GlobalCORSTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Module
name:  ambassador
config:
  cors:
    origins: http://foo.example.com
    methods: POST, GET, OPTIONS
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.target.path.k8s}-foo
prefix: /foo/
service: {self.target.path.fqdn}
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.target.path.k8s}-bar
prefix: /bar/
service: {self.target.path.fqdn}
cors:
  origins: http://bar.example.com
  methods: POST, GET, OPTIONS
""")

    def queries(self):
        yield Query(self.url("foo/"))
        yield Query(self.url("foo/"), headers={ "Origin": "http://foo.example.com" })
        yield Query(self.url("bar/"))
        yield Query(self.url("bar/"), headers={ "Origin": "http://bar.example.com" })

    def check(self):
        assert self.results[0].backend.name == self.target.path.k8s
        assert "Access-Control-Allow-Origin" not in self.results[0].headers

        assert self.results[1].backend.name == self.target.path.k8s
        assert self.results[1].headers["Access-Control-Allow-Origin"] == [ "http://foo.example.com" ]

        assert self.results[2].backend.name == self.target.path.k8s
        assert "Access-Control-Allow-Origin" not in self.results[2].headers

        assert self.results[3].backend.name == self.target.path.k8s
        assert self.results[3].headers["Access-Control-Allow-Origin"] == [ "http://bar.example.com" ]
