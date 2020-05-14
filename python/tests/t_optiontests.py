import json
import pytest

from typing import ClassVar, Dict, Sequence, Any

from kat.harness import Query, Test

from abstract_tests import MappingTest, OptionTest

# This is the place to add new OptionTests.


class AddRequestHeaders(OptionTest):

    parent: Test

    VALUES: ClassVar[Sequence[Dict[str, Any]]] = (
        { "foo": "bar" },
        { "moo": "arf" },
        { "zoo": {
            "append": True,
            "value": "bar"
        }},
        { "xoo": {
            "append": False,
            "value": "dwe"
        }},
        { "aoo": {
            "value": "tyu"
        }}
    )

    def config(self):
        yield "add_request_headers: %s" % json.dumps(self.value)

    def check(self):
        for r in self.parent.results:
            for k, v in self.value.items():
                actual = r.backend.request.headers.get(k.lower())
                if isinstance(v,dict):
                    assert actual == [v["value"]], (actual, [v["value"]])
                else:
                    assert actual == [v], (actual, [v])


class AddResponseHeaders(OptionTest):

    parent: Test

    VALUES: ClassVar[Sequence[Dict[str, str]]] = (
        { "foo": "bar" },
        { "moo": "arf" },
        { "zoo": {
            "append": True,
            "value": "bar"
        }},
        { "xoo": {
            "append": False,
            "value": "dwe"
        }},
        { "aoo": {
            "value": "tyu"
        }}
    )

    def config(self):
        yield "add_response_headers: %s" % json.dumps(self.value)

    def check(self):
        for r in self.parent.results:
            # Why do we end up with capitalized headers anyway??
            lowercased_headers = { k.lower(): v for k, v in r.headers.items() }

            for k, v in self.value.items():
                actual = lowercased_headers.get(k.lower())
                if isinstance(v,dict):
                    assert actual == [v["value"]], "expected %s: %s but got %s" % (k, v["value"], lowercased_headers)
                else:
                    assert actual == [v], "expected %s: %s but got %s" % (k, v, lowercased_headers)


class UseWebsocket(OptionTest):
    # TODO: add a check with a websocket client as soon as we have backend support for it

    def config(self):
        yield 'use_websocket: true'


class CORS(OptionTest):
    # isolated = True
    # debug = True

    # Note that there's also a GlobalCORSTest in t_cors.py.

    parent: MappingTest

    def config(self):
        yield 'cors: { origins: "*" }'

    def queries(self):
        for q in self.parent.queries():
            yield Query(q.url)  # redundant with parent
            yield Query(q.url, headers={ "Origin": "https://www.test-cors.org" })

    def check(self):
        # can assert about self.parent.results too
        assert self.results[0].backend.name == self.parent.target.path.k8s
        # Uh. Is it OK that this is case-sensitive?
        assert "Access-Control-Allow-Origin" not in self.results[0].headers

        assert self.results[1].backend.name == self.parent.target.path.k8s
        # Uh. Is it OK that this is case-sensitive?
        assert self.results[1].headers["Access-Control-Allow-Origin"] == [ "https://www.test-cors.org" ]


class CaseSensitive(OptionTest):

    parent: MappingTest

    def config(self):
        yield "case_sensitive: false"

    def queries(self):
        for q in self.parent.queries():
            idx = q.url.find("/", q.url.find("://") + 3)
            upped = q.url[:idx] + q.url[idx:].upper()
            assert upped != q.url
            yield Query(upped)


class AutoHostRewrite(OptionTest):

    parent: MappingTest

    def config(self):
        yield "auto_host_rewrite: true"

    def check(self):
        for r in self.parent.results:
            request_host = r.backend.request.host
            response_host = self.parent.get_fqdn(r.backend.name)

            assert response_host == request_host, f'backend {response_host} != request host {request_host}'


class Rewrite(OptionTest):

    parent: MappingTest

    VALUES = ("/foo", "foo")

    def config(self):
        yield self.format("rewrite: {self.value}")

    def queries(self):
        if self.value[0] != "/":
            for q in self.parent.pending:
                q.xfail = "rewrite option is broken for values not beginning in slash"

        return super(OptionTest, self).queries()

    def check(self):
        if self.value[0] != "/":
            pytest.xfail("this is broken")

        for r in self.parent.results:
            assert r.backend.request.url.path == self.value


class RemoveResponseHeaders(OptionTest):

    parent: Test

    def config(self):
        yield "remove_response_headers: x-envoy-upstream-service-time"

    def check(self):
        for r in self.parent.results:
            assert r.headers.get("x-envoy-upstream-service-time", None) == None, "x-envoy-upstream-service-time header was meant to be dropped but wasn't"
