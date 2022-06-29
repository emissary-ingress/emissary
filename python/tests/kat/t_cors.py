from typing import Generator, Tuple, Union

from abstract_tests import HTTP, AmbassadorTest, Node, ServiceType
from kat.harness import Query

# Note that there's also a CORS OptionTest in t_optiontests.py.




class GlobalCORSTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind:  Module
name:  ambassador
config:
  cors:
    origins: [http://foo.example.com]
    methods: [POST, GET, OPTIONS]
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-foo
hostname: "*"
prefix: /foo/
service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-bar
hostname: "*"
prefix: /bar/
service: {self.target.path.fqdn}
cors:
  origins: [http://bar.example.com]
  methods: [POST, GET, OPTIONS]
"""
        )

    def queries(self):
        # 0. No Access-Control-Allow-Origin because no Origin was provided.
        yield Query(self.url("foo/"))

        # 1. Access-Control-Allow-Origin because a matching Origin was provided.
        yield Query(self.url("foo/"), headers={"Origin": "http://foo.example.com"})

        # 2. No Access-Control-Allow-Origin because the provided Origin does not match.
        yield Query(self.url("foo/"), headers={"Origin": "http://wrong.example.com"})

        # 3. No Access-Control-Allow-Origin because no Origin was provided.
        yield Query(self.url("bar/"))

        # 4. Access-Control-Allow-Origin because a matching Origin was provided.
        yield Query(self.url("bar/"), headers={"Origin": "http://bar.example.com"})

        # 5. No Access-Control-Allow-Origin because no Origin was provided.
        yield Query(self.url("bar/"), headers={"Origin": "http://wrong.example.com"})

    def check(self):
        assert self.results[0].backend.name == self.target.path.k8s
        assert "Access-Control-Allow-Origin" not in self.results[0].headers

        assert self.results[1].backend.name == self.target.path.k8s
        assert self.results[1].headers["Access-Control-Allow-Origin"] == ["http://foo.example.com"]

        assert self.results[2].backend.name == self.target.path.k8s
        assert "Access-Control-Allow-Origin" not in self.results[2].headers

        assert self.results[3].backend.name == self.target.path.k8s
        assert "Access-Control-Allow-Origin" not in self.results[3].headers

        assert self.results[4].backend.name == self.target.path.k8s
        assert self.results[4].headers["Access-Control-Allow-Origin"] == ["http://bar.example.com"]

        assert self.results[5].backend.name == self.target.path.k8s
        assert "Access-Control-Allow-Origin" not in self.results[5].headers
