from typing import Generator, Tuple, Union

from abstract_tests import HTTP, AmbassadorTest, Node, ServiceType
from kat.harness import Query, variants


class RegexRewriteForwardingTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="foo")

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self.target, self.format(
            r"""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  regex_rewrite_mapping
hostname: "*"
prefix: /foo/
service: http://{self.target.path.fqdn}
regex_rewrite:
    pattern: "/foo/baz"
    substitution: "/baz/foo"
"""
        )

    def queries(self):
        yield Query(self.url("foo/bar"), expected=200)
        yield Query(self.url("foo/baz"), expected=200)
        yield Query(self.url("ffoo/"), expected=404)

    def check(self):
        assert self.results[0].backend
        assert self.results[0].backend.request
        assert self.results[0].backend.request.headers["x-envoy-original-path"][0] == f"/foo/bar"
        assert self.results[0].backend.request.url.path == "/foo/bar"
        assert self.results[1].backend
        assert self.results[1].backend.request
        assert self.results[1].backend.request.headers["x-envoy-original-path"][0] == f"/foo/baz"
        assert self.results[1].backend.request.url.path == "/baz/foo"


class RegexRewriteForwardingWithExtractAndSubstituteTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP(name="foo")

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self.target, self.format(
            r"""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  regex_rewrite_mapping
hostname: "*"
prefix: /foo/
service: http://{self.target.path.fqdn}
regex_rewrite:
    pattern: "/foo/([0-9]*)/list"
    substitution: "/bar/\\1"
"""
        )

    def queries(self):
        yield Query(self.url("foo/123456789/list"), expected=200)
        yield Query(self.url("foo/987654321/list"), expected=200)
        yield Query(self.url("fooooo/123456789/list"), expected=404)
        yield Query(self.url("foo/"), expected=200)

    def check(self):
        assert self.results[0].backend
        assert self.results[0].backend.request
        assert (
            self.results[0].backend.request.headers["x-envoy-original-path"][0]
            == f"/foo/123456789/list"
        )
        assert self.results[0].backend.request.url.path == "/bar/123456789"
        assert self.results[1].backend
        assert self.results[1].backend.request
        assert (
            self.results[1].backend.request.headers["x-envoy-original-path"][0]
            == f"/foo/987654321/list"
        )
        assert self.results[1].backend.request.url.path == "/bar/987654321"
