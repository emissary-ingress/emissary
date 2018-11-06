import json
import pytest

from typing import ClassVar, Dict, List, Sequence, Tuple, Union

from kat.harness import sanitize, variants, Query, Runner
from kat import manifests

from abstract_tests import AmbassadorTest, HTTP
from abstract_tests import MappingTest, OptionTest, ServiceType, Node, Test


class AuthenticationTestV1(AmbassadorTest):
    def init(self):
        self.target = HTTP()
        self.auth = HTTP(name="auth")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.k8s}"
path_prefix: "/extauth"
timeout_ms: 5s
allowed_authorization_headers:
- X-foo
allowed_request_headers:
- X-bar
- location
- requested-status
- requested-header

""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.k8s}
""")

    def queries(self):
        yield Query(self.url("target/"), headers={"requested-status": "401"}, expected=401)
        
        yield Query(self.url("target/"), headers={"requested-status": "302",
                                                  "location": "foo",
                                                  "requested-header": "location"}, expected=302)
        
        yield Query(self.url("target/"), headers={"requested-status": "200", 
                                                  "requested-header": "x-forwarded-proto"}, expected=200)

    def check(self):
        assert self.results[0].backend.name == self.auth.path.k8s
        assert self.results[0].backend.request.url.path == "/extauth/target/"
        assert self.results[0].backend.request.headers["x-forwarded-proto"]== ["http"]

        assert self.results[1].backend.response.headers["location"] == ["foo"]


# assert self.results[0].backend.name == self.auth.path.k8s
#         assert self.results[0].backend.request.url.path == "/extauth/target/"

#         assert self.results[1].backend.name == self.auth.path.k8s
#         assert self.results[1].backend.response.headers["location"] == ["foo"]
#         assert self.results[1].backend.request.url.path == "/extauth/target/"

#         assert self.results[2].backend.name == self.target.path.k8s
#         assert self.results[2].backend.request.url.path == "/"