import json

from kat.harness import variants, Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP

class RegexRewriteForwardingTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="foo")

    def config(self):
        yield self.target, self.format(r"""
---
apiVersion: ambassador/v2
kind:  Mapping
name:  regex_rewrite_mapping
prefix: /foo/
service: http://{self.target.path.fqdn}
regex_rewrite:
    pattern: "foo/baz"
    substitution: "baz/foo"
""")

    def queries(self):
        yield Query(self.url("foo/bar"), expected=200)
        yield Query(self.url("foo/baz"), expected=200)
        yield Query(self.url("ffoo/"), expected=404)

    def check(self):
        assert self.results[0].backend.request.headers['x-envoy-original-path'][0] == f'/foo/bar'
        assert self.results[0].backend.request.url.path == "/foo/bar"
        assert self.results[1].backend.request.headers['x-envoy-original-path'][0] == f'/foo/baz'
        assert self.results[1].backend.request.url.path == "/baz/foo"

class RegexRewriteForwardingWithExtractAndSubstituteTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="lboards")

    def config(self):
        yield self.target, self.format(r"""
---
apiVersion: ambassador/v2
kind:  Mapping
name:  regex_rewrite_mapping
prefix: /leaderboards/
service: http://{self.target.path.fqdn}
regex_rewrite:
    pattern: "leaderboards/v1/([0-9]*)/find"
    substitution: "game/\\1"
""")

    def queries(self):
        yield Query(self.url("leaderboards/v1/123456789/find"), expected=200)
        yield Query(self.url("leaderboards/v1/987654321/find"), expected=200)
        yield Query(self.url("leaderboardddddds/v1/123456789/find"), expected=404)
        yield Query(self.url("leaderboards/v1/"), expected=200)

    def check(self):
        assert self.results[0].backend.request.headers['x-envoy-original-path'][0] == f'/leaderboards/v1/123456789/find'
        assert self.results[0].backend.request.url.path == "/game/123456789"
        assert self.results[1].backend.request.headers['x-envoy-original-path'][0] == f'/leaderboards/v1/987654321/find'
        assert self.results[1].backend.request.url.path == "/game/987654321"
