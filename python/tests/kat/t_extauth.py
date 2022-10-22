import json
import os
from typing import Generator, Literal, Tuple, Union, cast

import pytest

from abstract_tests import AGRPC, AHTTP, HTTP, AmbassadorTest, Node, ServiceType, WebsocketEcho
from ambassador import Config
from kat.harness import EDGE_STACK, Query
from tests.selfsigned import TLSCerts


class AuthenticationGRPCTest(AmbassadorTest):

    target: ServiceType
    auth: ServiceType

    def init(self):
        if EDGE_STACK:
            self.xfail = "XFailing for now, custom AuthServices not supported in Edge Stack"
        self.target = HTTP()
        self.auth = AGRPC(name="auth")

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: auth-context-mapping
spec:
  ambassador_id: [{self.ambassador_id}]
  service: {self.target.path.fqdn}
  hostname: "*"
  prefix: /context-extensions-crd/
  auth_context_extensions:
    context: "auth-context-name"
    data: "auth-data"
"""
            )
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
timeout_ms: 5000
proto: grpc
protocol_version: "v3"
"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-context-extensions
hostname: "*"
prefix: /context-extensions/
service: {self.target.path.fqdn}
auth_context_extensions:
    first: "first element"
    second: "second element"
"""
        )

    def queries(self):
        # [0]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-extauth-requested-status": "401",
                "baz": "baz",
                "request-header": "baz",
            },
            expected=401,
        )
        # [1]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-extauth-requested-status": "302",
                "kat-req-extauth-requested-location": "foo",
            },
            expected=302,
        )

        # [2]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-extauth-requested-status": "401",
                "x-foo": "foo",
                "kat-req-extauth-requested-header": "x-foo",
            },
            expected=401,
        )
        # [3]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-extauth-requested-status": "200",
                "authorization": "foo-11111",
                "foo": "foo",
                "kat-req-extauth-append": "foo=bar;baz=bar",
                "kat-req-http-requested-header": "Authorization",
            },
            expected=200,
        )
        # [4]
        yield Query(
            self.url("context-extensions/"),
            headers={
                "request-status": "200",
                "authorization": "foo-22222",
                "kat-req-http-requested-header": "Authorization",
            },
            expected=200,
        )
        # [5]
        yield Query(
            self.url("context-extensions-crd/"),
            headers={
                "request-status": "200",
                "authorization": "foo-33333",
                "kat-req-http-requested-header": "Authorization",
            },
            expected=200,
        )

    def check(self):
        # [0] Verifies all request headers sent to the authorization server.
        assert self.results[0].backend
        assert self.results[0].backend.name == self.auth.path.k8s
        assert self.results[0].backend.request
        assert self.results[0].backend.request.url.path == "/target/"
        assert self.results[0].backend.request.headers["x-envoy-internal"] == ["true"]
        assert self.results[0].backend.request.headers["x-forwarded-proto"] == ["http"]
        assert "user-agent" in self.results[0].backend.request.headers
        assert "baz" in self.results[0].backend.request.headers
        assert self.results[0].status == 401
        assert self.results[0].headers["Server"] == ["envoy"]
        assert self.results[0].headers["Kat-Resp-Extauth-Protocol-Version"] == ["v3"]

        # [1] Verifies that Location header is returned from Envoy.
        assert self.results[1].backend
        assert self.results[1].backend.name == self.auth.path.k8s
        assert self.results[1].backend.request
        assert self.results[1].backend.request.headers["kat-req-extauth-requested-status"] == [
            "302"
        ]
        assert self.results[1].backend.request.headers["kat-req-extauth-requested-location"] == [
            "foo"
        ]
        assert self.results[1].status == 302
        assert self.results[1].headers["Location"] == ["foo"]
        assert self.results[1].headers["Kat-Resp-Extauth-Protocol-Version"] == ["v3"]

        # [2] Verifies Envoy returns whitelisted headers input by the user.
        assert self.results[2].backend
        assert self.results[2].backend.name == self.auth.path.k8s
        assert self.results[2].backend.request
        assert self.results[2].backend.request.headers["kat-req-extauth-requested-status"] == [
            "401"
        ]
        assert self.results[2].backend.request.headers["kat-req-extauth-requested-header"] == [
            "x-foo"
        ]
        assert self.results[2].backend.request.headers["x-foo"] == ["foo"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert self.results[2].headers["X-Foo"] == ["foo"]
        assert self.results[2].headers["Kat-Resp-Extauth-Protocol-Version"] == ["v3"]

        # [3] Verifies default whitelisted Authorization request header.
        assert self.results[3].backend
        assert self.results[3].backend.request
        assert self.results[3].backend.request.headers["kat-req-extauth-requested-status"] == [
            "200"
        ]
        assert self.results[3].backend.request.headers["kat-req-http-requested-header"] == [
            "Authorization"
        ]
        assert self.results[3].backend.request.headers["authorization"] == ["foo-11111"]
        assert self.results[3].backend.request.headers["foo"] == ["foo,bar"]
        assert self.results[3].backend.request.headers["baz"] == ["bar"]
        assert self.results[3].status == 200
        assert self.results[3].headers["Server"] == ["envoy"]
        assert self.results[3].headers["Authorization"] == ["foo-11111"]
        assert self.results[3].backend.request.headers["kat-resp-extauth-protocol-version"] == [
            "v3"
        ]

        # [4] Verifies that auth_context_extension is passed along by Envoy.
        assert self.results[4].status == 200
        assert self.results[4].headers["Server"] == ["envoy"]
        assert self.results[4].headers["Authorization"] == ["foo-22222"]
        assert self.results[4].backend
        assert self.results[4].backend.request
        context_ext = json.loads(
            self.results[4].backend.request.headers["kat-resp-extauth-context-extensions"][0]
        )
        assert context_ext["first"] == "first element"
        assert context_ext["second"] == "second element"

        # [5] Verifies that auth_context_extension is passed along by Envoy when using a crd Mapping
        assert self.results[5].status == 200
        assert self.results[5].headers["Server"] == ["envoy"]
        assert self.results[5].headers["Authorization"] == ["foo-33333"]
        assert self.results[5].backend
        assert self.results[5].backend.request
        context_ext = json.loads(
            self.results[5].backend.request.headers["kat-resp-extauth-context-extensions"][0]
        )
        assert context_ext["context"] == "auth-context-name"
        assert context_ext["data"] == "auth-data"


class AuthenticationHTTPPartialBufferTest(AmbassadorTest):

    target: ServiceType
    auth: ServiceType

    def init(self):
        if EDGE_STACK:
            self.xfail = "XFailing for now, custom AuthServices not supported in Edge Stack"
        self.target = HTTP()
        self.auth = HTTP(name="auth")

    def manifests(self) -> str:
        return (
            f"""
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["tls-context-host-1"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-1"].k8s_key}
kind: Secret
metadata:
  name: auth-partial-secret
type: kubernetes.io/tls
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-same-context-1
secret: auth-partial-secret

---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"
timeout_ms: 5000
tls: {self.name}-same-context-1

allowed_request_headers:
- Kat-Req-Http-Requested-Status
- Kat-Req-Http-Requested-Header

allowed_authorization_headers:
- Kat-Resp-Http-Request-Body

add_auth_headers:
  X-Added-Auth: auth-added

include_body:
  max_bytes: 7
  allow_partial: true
"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
"""
        )

    def queries(self):
        # [0]
        yield Query(
            self.url("target/"),
            headers={"kat-req-http-requested-status": "200"},
            body="message_body",
            expected=200,
        )

        # [1]
        yield Query(
            self.url("target/"),
            headers={"kat-req-http-requested-status": "200"},
            body="body",
            expected=200,
        )

        # [2]
        yield Query(
            self.url("target/"),
            headers={"kat-req-http-requested-status": "401"},
            body="body",
            expected=401,
        )

    def check(self):
        # [0] Verifies that the authorization server received the partial message body.
        extauth_res1 = json.loads(self.results[0].headers["Extauth"][0])
        assert self.results[0].backend
        assert self.results[0].backend.request
        assert self.results[0].backend.request.headers["kat-req-http-requested-status"] == ["200"]
        assert self.results[0].status == 200
        assert self.results[0].headers["Server"] == ["envoy"]
        assert extauth_res1["request"]["headers"]["kat-resp-http-request-body"] == ["message"]

        # [1] Verifies that the authorization server received the full message body.
        extauth_res2 = json.loads(self.results[1].headers["Extauth"][0])
        assert self.results[1].backend
        assert self.results[1].backend.request
        assert self.results[1].backend.request.headers["kat-req-http-requested-status"] == ["200"]
        assert self.results[1].status == 200
        assert self.results[1].headers["Server"] == ["envoy"]
        assert extauth_res2["request"]["headers"]["kat-resp-http-request-body"] == ["body"]

        # [2] Verifies that the authorization server received added headers
        assert self.results[2].backend
        assert self.results[2].backend.request
        assert self.results[2].backend.request.headers["kat-req-http-requested-status"] == ["401"]
        assert self.results[2].backend.request.headers["x-added-auth"] == ["auth-added"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert extauth_res2["request"]["headers"]["kat-resp-http-request-body"] == ["body"]


class AuthenticationHTTPBufferedTest(AmbassadorTest):

    target: ServiceType
    auth: ServiceType

    def init(self):
        if EDGE_STACK:
            self.xfail = "XFailing for now, custom AuthServices not supported in Edge Stack"
        self.target = HTTP()
        self.auth = HTTP(name="auth")

    def manifests(self) -> str:
        return (
            f"""
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["tls-context-host-1"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-1"].k8s_key}
kind: Secret
metadata:
  name: auth-buffered-secret
type: kubernetes.io/tls
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind:  Module
name:  ambassador
config:
  add_linkerd_headers: true
  buffer:
    max_request_bytes: 16384
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-same-context-1
secret: auth-buffered-secret
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"
timeout_ms: 5000
tls: {self.name}-same-context-1

allowed_request_headers:
- X-Foo
- X-Bar
- Kat-Req-Http-Requested-Status
- Kat-Req-Http-Requested-Header
- Kat-Req-Http-Requested-Cookie
- Location

allowed_authorization_headers:
- X-Foo
- Set-Cookie

include_body:
  max_bytes: 4096
  allow_partial: true
"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
"""
        )

    def queries(self):
        # [0]
        yield Query(
            self.url("target/"),
            headers={"kat-req-http-requested-status": "401", "Baz": "baz", "Request-Header": "Baz"},
            expected=401,
        )
        # [1]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-http-requested-status": "302",
                "location": "foo",
                "kat-req-http-requested-cookie": "foo, bar, baz",
                "kat-req-http-requested-header": "location",
            },
            expected=302,
        )
        # [2]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-http-requested-status": "401",
                "X-Foo": "foo",
                "kat-req-http-requested-header": "X-Foo",
            },
            expected=401,
        )
        # [3]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-http-requested-status": "401",
                "X-Bar": "bar",
                "kat-req-http-requested-header": "X-Bar",
            },
            expected=401,
        )
        # [4]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-http-requested-status": "200",
                "Authorization": "foo-11111",
                "kat-req-http-requested-header": "Authorization",
            },
            expected=200,
        )

    def check(self):
        # [0] Verifies all request headers sent to the authorization server.
        assert self.results[0].backend
        assert self.results[0].backend.name == self.auth.path.k8s
        assert self.results[0].backend.request
        assert self.results[0].backend.request.url.path == "/extauth/target/"
        assert self.results[0].backend.request.headers["x-forwarded-proto"] == ["http"]
        assert self.results[0].backend.request.headers["content-length"] == ["0"]
        assert "x-forwarded-for" in self.results[0].backend.request.headers
        assert "user-agent" in self.results[0].backend.request.headers
        assert "baz" not in self.results[0].backend.request.headers
        assert self.results[0].status == 401
        assert self.results[0].headers["Server"] == ["envoy"]

        # [1] Verifies that Location header is returned from Envoy.
        assert self.results[1].backend
        assert self.results[1].backend.name == self.auth.path.k8s
        assert self.results[1].backend.request
        assert self.results[1].backend.request.headers["kat-req-http-requested-status"] == ["302"]
        assert self.results[1].backend.request.headers["kat-req-http-requested-header"] == [
            "location"
        ]
        assert self.results[1].backend.request.headers["location"] == ["foo"]
        assert self.results[1].status == 302
        assert self.results[1].headers["Server"] == ["envoy"]
        assert self.results[1].headers["Location"] == ["foo"]
        assert self.results[1].headers["Set-Cookie"] == ["foo=foo", "bar=bar", "baz=baz"]

        # [2] Verifies Envoy returns whitelisted headers input by the user.
        assert self.results[2].backend
        assert self.results[2].backend.name == self.auth.path.k8s
        assert self.results[2].backend.request
        assert self.results[2].backend.request.headers["kat-req-http-requested-status"] == ["401"]
        assert self.results[2].backend.request.headers["kat-req-http-requested-header"] == ["X-Foo"]
        assert self.results[2].backend.request.headers["x-foo"] == ["foo"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert self.results[2].headers["X-Foo"] == ["foo"]

        # [3] Verifies that envoy does not return not whitelisted headers.
        assert self.results[3].backend
        assert self.results[3].backend.name == self.auth.path.k8s
        assert self.results[3].backend.request
        assert self.results[3].backend.request.headers["kat-req-http-requested-status"] == ["401"]
        assert self.results[3].backend.request.headers["kat-req-http-requested-header"] == ["X-Bar"]
        assert self.results[3].backend.request.headers["x-bar"] == ["bar"]
        assert self.results[3].status == 401
        assert self.results[3].headers["Server"] == ["envoy"]
        assert "X-Bar" not in self.results[3].headers

        # [4] Verifies default whitelisted Authorization request header.
        assert self.results[4].backend
        assert self.results[4].backend.request
        assert self.results[4].backend.request.headers["kat-req-http-requested-status"] == ["200"]
        assert self.results[4].backend.request.headers["kat-req-http-requested-header"] == [
            "Authorization"
        ]
        assert self.results[4].backend.request.headers["authorization"] == ["foo-11111"]
        assert self.results[4].backend.request.headers["l5d-dst-override"] == [
            f"{self.target.path.fqdn}:80"
        ]
        assert self.results[4].status == 200
        assert self.results[4].headers["Server"] == ["envoy"]
        assert self.results[4].headers["Authorization"] == ["foo-11111"]


class AuthenticationHTTPFailureModeAllowTest(AmbassadorTest):
    target: ServiceType
    auth: ServiceType

    def init(self):
        if EDGE_STACK:
            self.xfail = "XFailing for now, custom AuthServices not supported in Edge Stack"
        self.target = HTTP()
        self.auth = HTTP(name="auth")

    def manifests(self) -> str:
        return (
            f"""
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["tls-context-host-1"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-1"].k8s_key}
kind: Secret
metadata:
  name: auth-failure-secret
type: kubernetes.io/tls
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-failure-context
secret: auth-failure-secret

---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"
timeout_ms: 5000
tls: {self.name}-failure-context

allowed_request_headers:
- Kat-Req-Http-Requested-Status
- Kat-Req-Http-Requested-Header

failure_mode_allow: true
"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
"""
        )

    def queries(self):
        # [0]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "200"}, expected=200
        )

        # [1]
        yield Query(
            self.url("target/"), headers={"kat-req-http-requested-status": "503"}, expected=503
        )

    def check(self):
        # [0] Verifies that the authorization server received the partial message body.
        extauth_res1 = json.loads(self.results[0].headers["Extauth"][0])
        assert self.results[0].backend
        assert self.results[0].backend.request
        assert self.results[0].backend.request.headers["kat-req-http-requested-status"] == ["200"]
        assert self.results[0].status == 200
        assert self.results[0].headers["Server"] == ["envoy"]

        # [1] Verifies that the authorization server received the full message body.
        extauth_res2 = json.loads(self.results[1].headers["Extauth"][0])
        assert self.results[1].backend
        assert self.results[1].backend.request
        assert self.results[1].backend.request.headers["kat-req-http-requested-status"] == ["503"]
        assert self.results[1].headers["Server"] == ["envoy"]


class AuthenticationTestV1(AmbassadorTest):

    target: ServiceType
    auth: ServiceType

    def init(self):
        if EDGE_STACK:
            self.xfail = "XFailing for now, custom AuthServices not supported in Edge Stack"
        self.target = HTTP()
        self.auth1 = AHTTP(name="auth1")
        self.auth2 = AHTTP(name="auth2")
        self.backend_counts = {}

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
name:  {self.auth1.path.k8s}
auth_service: "{self.auth1.path.fqdn}"
proto: http
path_prefix: "/extauth"
timeout_ms: 5000

allowed_request_headers:
- X-Foo
- X-Bar
- Kat-Req-Http-Requested-Status
- Kat-Req-Http-Requested-Header
- Location

allowed_authorization_headers:
- X-Foo
- Extauth

status_on_error:
  code: 503

---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
name:  {self.auth2.path.k8s}
auth_service: "{self.auth2.path.fqdn}"
proto: http
path_prefix: "/extauth"
timeout_ms: 5000
add_linkerd_headers: true

allowed_request_headers:
- X-Foo
- X-Bar
- Kat-Req-Http-Requested-Status
- Kat-Req-Http-Requested-Header
- Location

allowed_authorization_headers:
- X-Foo
- Extauth

status_on_error:
  code: 503

"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.fqdn}-unauthed
hostname: "*"
prefix: /target/unauthed/
service: {self.target.path.fqdn}
bypass_auth: true
"""
        )

    def queries(self):
        # [0]
        yield Query(
            self.url("target/0"),
            headers={"kat-req-http-requested-status": "401", "Baz": "baz", "Request-Header": "Baz"},
            expected=401,
        )
        # [1]
        yield Query(
            self.url("target/1"),
            headers={
                "kat-req-http-requested-status": "302",
                "location": "foo",
                "kat-req-http-requested-header": "location",
            },
            expected=302,
        )
        # [2]
        yield Query(
            self.url("target/2"),
            headers={
                "kat-req-http-requested-status": "401",
                "X-Foo": "foo",
                "kat-req-http-requested-header": "X-Foo",
            },
            expected=401,
        )
        # [3]
        yield Query(
            self.url("target/3"),
            headers={
                "kat-req-http-requested-status": "401",
                "X-Bar": "bar",
                "kat-req-http-requested-header": "X-Bar",
            },
            expected=401,
        )
        # [4]
        yield Query(
            self.url("target/4"),
            headers={
                "kat-req-http-requested-status": "200",
                "Authorization": "foo-11111",
                "kat-req-http-requested-header": "Authorization",
            },
            expected=200,
        )

        # [5]
        yield Query(self.url("target/5"), headers={"X-Forwarded-Proto": "https"}, expected=200)

        # [6]
        yield Query(
            self.url("target/unauthed/6"),
            headers={"kat-req-http-requested-status": "200"},
            expected=200,
        )

        # [7]
        yield Query(
            self.url("target/7"), headers={"kat-req-http-requested-status": "500"}, expected=503
        )

        # Create some traffic to make it more likely that both auth services get at least one
        # request
        for i in range(20):
            yield Query(
                self.url("target/" + str(8 + i)),
                headers={"kat-req-http-requested-status": "403"},
                expected=403,
            )

    def check_backend_name(self, result) -> bool:
        backend_name = result.backend.name

        self.backend_counts.setdefault(backend_name, 0)
        self.backend_counts[backend_name] += 1

        return (backend_name == self.auth1.path.k8s) or (backend_name == self.auth2.path.k8s)

    def check(self):

        # [0] Verifies all request headers sent to the authorization server.
        assert self.check_backend_name(self.results[0])
        assert self.results[0].backend
        assert self.results[0].backend.request
        assert self.results[0].backend.request.url.path == "/extauth/target/0"
        assert self.results[0].backend.request.headers["x-forwarded-proto"] == ["http"]
        assert self.results[0].backend.request.headers["content-length"] == ["0"]
        assert "x-forwarded-for" in self.results[0].backend.request.headers
        assert "user-agent" in self.results[0].backend.request.headers
        assert "baz" not in self.results[0].backend.request.headers
        assert self.results[0].status == 401
        assert self.results[0].headers["Server"] == ["envoy"]

        # [1] Verifies that Location header is returned from Envoy.
        assert self.check_backend_name(self.results[1])
        assert self.results[1].backend
        assert self.results[1].backend.request
        assert self.results[1].backend.request.headers["kat-req-http-requested-status"] == ["302"]
        assert self.results[1].backend.request.headers["kat-req-http-requested-header"] == [
            "location"
        ]
        assert self.results[1].backend.request.headers["location"] == ["foo"]
        assert self.results[1].status == 302
        assert self.results[1].headers["Server"] == ["envoy"]
        assert self.results[1].headers["Location"] == ["foo"]

        # [2] Verifies Envoy returns whitelisted headers input by the user.
        assert self.check_backend_name(self.results[2])
        assert self.results[2].backend
        assert self.results[2].backend.request
        assert self.results[2].backend.request.headers["kat-req-http-requested-status"] == ["401"]
        assert self.results[2].backend.request.headers["kat-req-http-requested-header"] == ["X-Foo"]
        assert self.results[2].backend.request.headers["x-foo"] == ["foo"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert self.results[2].headers["X-Foo"] == ["foo"]

        # [3] Verifies that envoy does not return not whitelisted headers.
        assert self.check_backend_name(self.results[3])
        assert self.results[3].backend
        assert self.results[3].backend.request
        assert self.results[3].backend.request.headers["kat-req-http-requested-status"] == ["401"]
        assert self.results[3].backend.request.headers["kat-req-http-requested-header"] == ["X-Bar"]
        assert self.results[3].backend.request.headers["x-bar"] == ["bar"]
        assert self.results[3].status == 401
        assert self.results[3].headers["Server"] == ["envoy"]
        assert "X-Bar" not in self.results[3].headers

        # [4] Verifies default whitelisted Authorization request header.
        assert self.results[4].backend
        assert (
            self.results[4].backend.name == self.target.path.k8s
        )  # this response is from an auth success
        assert self.results[4].backend.request
        assert self.results[4].backend.request.headers["kat-req-http-requested-status"] == ["200"]
        assert self.results[4].backend.request.headers["kat-req-http-requested-header"] == [
            "Authorization"
        ]
        assert self.results[4].backend.request.headers["authorization"] == ["foo-11111"]
        assert self.results[4].status == 200
        assert self.results[4].headers["Server"] == ["envoy"]
        assert self.results[4].headers["Authorization"] == ["foo-11111"]

        extauth_req = json.loads(self.results[4].backend.request.headers["extauth"][0])
        assert extauth_req["request"]["headers"]["l5d-dst-override"] == ["extauth:80"]

        # [5] Verify that X-Forwarded-Proto makes it to the auth service.
        #
        # We use the 'extauth' header returned from the test extauth service for this, since
        # the extauth service (on success) won't actually alter other things going upstream.
        r5 = self.results[5]
        assert r5
        assert r5.backend
        assert r5.backend.name == self.target.path.k8s  # this response is from an auth success

        assert r5.status == 200
        assert r5.headers["Server"] == ["envoy"]

        assert r5.backend.request
        eahdr = r5.backend.request.headers["extauth"]
        assert eahdr, "no extauth header was returned?"
        assert eahdr[0], "an empty extauth header element was returned?"

        # [6] Verifies that Envoy bypasses external auth when disabled for a mapping.
        assert self.results[6].backend
        assert (
            self.results[6].backend.name == self.target.path.k8s
        )  # ensure the request made it to the backend
        assert not self.check_backend_name(
            self.results[6]
        )  # ensure the request did not go to the auth service
        assert self.results[6].backend.request
        assert self.results[6].backend.request.headers["kat-req-http-requested-status"] == ["200"]
        assert self.results[6].status == 200
        assert self.results[6].headers["Server"] == ["envoy"]

        try:
            eainfo = json.loads(eahdr[0])

            if eainfo:
                # Envoy should force this to HTTP, not HTTPS.
                assert eainfo["request"]["headers"]["x-forwarded-proto"] == ["http"]
        except ValueError as e:
            assert False, "could not parse Extauth header '%s': %s" % (eahdr, e)

        # [7] Verifies that envoy returns customized status_on_error code.
        assert self.results[7].status == 503

        # TODO(gsagula): Write tests for all UCs which request header headers
        # are overridden, e.g. Authorization.

        for i in range(20):
            assert self.check_backend_name(self.results[8 + i])

        print("auth1 service got %d requests" % self.backend_counts.get(self.auth1.path.k8s, -1))
        print("auth2 service got %d requests" % self.backend_counts.get(self.auth2.path.k8s, -1))
        assert self.backend_counts.get(self.auth1.path.k8s, 0) > 0, "auth1 got no requests"
        assert self.backend_counts.get(self.auth2.path.k8s, 0) > 0, "auth2 got no requests"


class AuthenticationTest(AmbassadorTest):
    target: ServiceType
    auth: ServiceType

    def init(self):
        if EDGE_STACK:
            self.xfail = "XFailing for now, custom AuthServices not supported in Edge Stack"
        self.target = HTTP()
        self.auth = AHTTP(name="auth")

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"

allowed_request_headers:
- X-Foo
- X-Bar
- Kat-Req-Http-Requested-Location
- Kat-Req-Http-Requested-Status
- Kat-Req-Http-Requested-Header

allowed_authorization_headers:
- X-Foo
- X-Bar
- Extauth

"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
"""
        )

    def queries(self):
        # [0]
        yield Query(
            self.url("target/"),
            headers={"kat-req-http-requested-status": "401", "Baz": "baz", "Request-Header": "Baz"},
            expected=401,
        )
        # [1]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-http-requested-status": "302",
                "kat-req-http-requested-location": "foo",
                "kat-req-http-requested-header": "location",
            },
            expected=302,
        )
        # [2]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-http-requested-status": "401",
                "X-Foo": "foo",
                "kat-req-http-requested-header": "X-Foo",
            },
            expected=401,
        )
        # [3]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-http-requested-status": "401",
                "X-Bar": "bar",
                "kat-req-http-requested-header": "X-Bar",
            },
            expected=401,
        )
        # [4]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-http-requested-status": "200",
                "Authorization": "foo-11111",
                "kat-req-http-requested-header": "Authorization",
            },
            expected=200,
        )
        # [5]
        yield Query(self.url("target/"), headers={"X-Forwarded-Proto": "https"}, expected=200)

    def check(self):
        # [0] Verifies all request headers sent to the authorization server.
        assert self.results[0].backend
        assert (
            self.results[0].backend.name == self.auth.path.k8s
        ), f"wanted backend {self.auth.path.k8s}, got {self.results[0].backend.name}"
        assert self.results[0].backend.request
        assert self.results[0].backend.request.url.path == "/extauth/target/"
        assert self.results[0].backend.request.headers["content-length"] == ["0"]
        assert "x-forwarded-for" in self.results[0].backend.request.headers
        assert "user-agent" in self.results[0].backend.request.headers
        assert "baz" not in self.results[0].backend.request.headers
        assert self.results[0].status == 401
        assert self.results[0].headers["Server"] == ["envoy"]

        # [1] Verifies that Location header is returned from Envoy.
        assert self.results[1].backend
        assert self.results[1].backend.name == self.auth.path.k8s
        assert self.results[1].backend.request
        assert self.results[1].backend.request.headers["kat-req-http-requested-status"] == ["302"]
        assert self.results[1].backend.request.headers["kat-req-http-requested-header"] == [
            "location"
        ]
        assert self.results[1].backend.request.headers["kat-req-http-requested-location"] == ["foo"]
        assert self.results[1].status == 302
        assert self.results[1].headers["Server"] == ["envoy"]
        assert self.results[1].headers["Location"] == ["foo"]

        # [2] Verifies Envoy returns whitelisted headers input by the user.
        assert self.results[2].backend
        assert self.results[2].backend.name == self.auth.path.k8s
        assert self.results[2].backend.request
        assert self.results[2].backend.request.headers["kat-req-http-requested-status"] == ["401"]
        assert self.results[2].backend.request.headers["kat-req-http-requested-header"] == ["X-Foo"]
        assert self.results[2].backend.request.headers["x-foo"] == ["foo"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert self.results[2].headers["X-Foo"] == ["foo"]

        # [3] Verifies that envoy does not return not whitelisted headers.
        assert self.results[3].backend
        assert self.results[3].backend.name == self.auth.path.k8s
        assert self.results[3].backend.request
        assert self.results[3].backend.request.headers["kat-req-http-requested-status"] == ["401"]
        assert self.results[3].backend.request.headers["kat-req-http-requested-header"] == ["X-Bar"]
        assert self.results[3].backend.request.headers["x-bar"] == ["bar"]
        assert self.results[3].status == 401
        assert self.results[3].headers["Server"] == ["envoy"]
        assert "X-Bar" in self.results[3].headers

        # [4] Verifies default whitelisted Authorization request header.
        assert self.results[4].backend
        assert self.results[4].backend.request
        assert self.results[4].backend.request.headers["kat-req-http-requested-status"] == ["200"]
        assert self.results[4].backend.request.headers["kat-req-http-requested-header"] == [
            "Authorization"
        ]
        assert self.results[4].backend.request.headers["authorization"] == ["foo-11111"]
        assert self.results[4].status == 200
        assert self.results[4].headers["Server"] == ["envoy"]
        assert self.results[4].headers["Authorization"] == ["foo-11111"]

        # [5] Verify that X-Forwarded-Proto makes it to the auth service.
        #
        # We use the 'extauth' header returned from the test extauth service for this, since
        # the extauth service (on success) won't actually alter other things going upstream.
        r5 = self.results[5]
        assert r5

        assert r5.status == 200
        assert r5.headers["Server"] == ["envoy"]

        assert r5.backend
        assert r5.backend.request
        eahdr = r5.backend.request.headers["extauth"]
        assert eahdr, "no extauth header was returned?"
        assert eahdr[0], "an empty extauth header element was returned?"

        try:
            eainfo = json.loads(eahdr[0])

            if eainfo:
                # Envoy should force this to HTTP, not HTTPS.
                assert eainfo["request"]["headers"]["x-forwarded-proto"] == ["http"]
        except ValueError as e:
            assert False, "could not parse Extauth header '%s': %s" % (eahdr, e)

        # TODO(gsagula): Write tests for all UCs which request header headers
        # are overridden, e.g. Authorization.


class AuthenticationWebsocketTest(AmbassadorTest):

    auth: ServiceType
    backend: ServiceType

    def init(self):
        if EDGE_STACK:
            self.xfail = "XFailing for now, custom AuthServices not supported in Edge Stack"
        self.auth = HTTP(name="auth")
        self.backend = WebsocketEcho()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"
timeout_ms: 10000
allowed_request_headers:
- Kat-Req-Http-Requested-Status
allow_request_body: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: {self.name}
hostname: "*"
prefix: /{self.name}/
service: {self.backend.path.fqdn}
use_websocket: true
"""
        )

    def queries(self):
        yield Query(self.url(self.name + "/"), expected=404)

        yield Query(self.url(self.name + "/", scheme="ws"), messages=["one", "two", "three"])

    def check(self):
        assert self.results[-1].messages == ["one", "two", "three"]


class AuthenticationGRPCVerTest(AmbassadorTest):

    target: ServiceType
    specified_protocol_version: Literal["v2", "v3", "default"]
    expected_protocol_version: Literal["v3", "invalid"]
    auth: ServiceType

    @classmethod
    def variants(cls) -> Generator[Node, None, None]:
        for protocol_version in ["v2", "v3", "default"]:
            yield cls(protocol_version, name="{self.specified_protocol_version}")

    def init(self, protocol_version: Literal["v2", "v3", "default"]):
        self.target = HTTP()
        self.specified_protocol_version = protocol_version
        self.expected_protocol_version = cast(
            Literal["v3", "invalid"], protocol_version if protocol_version in ["v3"] else "invalid"
        )
        self.auth = AGRPC(
            name="auth",
            protocol_version=(
                self.expected_protocol_version
                if self.expected_protocol_version != "invalid"
                else "v3"
            ),
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
timeout_ms: 5000
proto: grpc
"""
        ) + (
            ""
            if self.specified_protocol_version == "default"
            else f"protocol_version: '{self.specified_protocol_version}'"
        )

        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
"""
        )

    def queries(self):
        # TODO add more
        # [0]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-extauth-requested-status": "401",
                "baz": "baz",
                "kat-req-extauth-request-header": "baz",
            },
            expected=(500 if self.expected_protocol_version == "invalid" else 401),
        )

        # [1]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-extauth-requested-status": "302",
                "kat-req-extauth-requested-location": "foo",
            },
            expected=(500 if self.expected_protocol_version == "invalid" else 302),
        )

        # [2]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-extauth-requested-status": "401",
                "x-foo": "foo",
                "kat-req-extauth-requested-header": "x-foo",
            },
            expected=(500 if self.expected_protocol_version == "invalid" else 401),
        )
        # [3]
        yield Query(
            self.url("target/"),
            headers={
                "kat-req-extauth-requested-status": "200",
                "authorization": "foo-11111",
                "foo": "foo",
                "kat-req-extauth-append": "foo=bar;baz=bar",
                "kat-req-http-requested-header": "Authorization",
            },
            expected=(500 if self.expected_protocol_version == "invalid" else 200),
        )

    def check(self):
        if self.expected_protocol_version == "invalid":
            for i, result in enumerate(self.results):
                # Verify the basic structure of the HTTP 500's JSON body.
                assert result.json, f"self.results[{i}] does not have a JSON body"
                assert (
                    result.json["status_code"] == 500
                ), f"self.results[{i}] JSON body={repr(result.json)} does not have status_code=500"
                assert result.json[
                    "request_id"
                ], f"self.results[{i}] JSON body={repr(result.json)} does not have request_id"
                assert (
                    self.path.k8s in result.json["message"]
                ), f"self.results[{i}] JSON body={repr(result.json)} does not have thing-containing-the-annotation-containing-the-AuthService name {repr(self.path.k8s)} in message"
                assert (
                    "AuthService" in result.json["message"]
                ), f"self.results[{i}] JSON body={repr(result.json)} does not have type 'AuthService' in message"
            return

        # [0] Verifies all request headers sent to the authorization server.
        assert self.results[0].backend
        assert self.results[0].backend.name == self.auth.path.k8s
        assert self.results[0].backend.request
        assert self.results[0].backend.request.url.path == "/target/"
        assert self.results[0].backend.request.headers["x-forwarded-proto"] == ["http"]
        assert "user-agent" in self.results[0].backend.request.headers
        assert "baz" in self.results[0].backend.request.headers
        assert self.results[0].status == 401
        assert self.results[0].headers["Server"] == ["envoy"]
        assert self.results[0].headers["Kat-Resp-Extauth-Protocol-Version"] == [
            self.expected_protocol_version
        ]

        # [1] Verifies that Location header is returned from Envoy.
        assert self.results[1].backend
        assert self.results[1].backend.name == self.auth.path.k8s
        assert self.results[1].backend.request
        assert self.results[1].backend.request.headers["kat-req-extauth-requested-status"] == [
            "302"
        ]
        assert self.results[1].backend.request.headers["kat-req-extauth-requested-location"] == [
            "foo"
        ]
        assert self.results[1].status == 302
        assert self.results[1].headers["Location"] == ["foo"]
        assert self.results[1].headers["Kat-Resp-Extauth-Protocol-Version"] == [
            self.expected_protocol_version
        ]

        # [2] Verifies Envoy returns whitelisted headers input by the user.
        assert self.results[2].backend
        assert self.results[2].backend.name == self.auth.path.k8s
        assert self.results[2].backend.request
        assert self.results[2].backend.request.headers["kat-req-extauth-requested-status"] == [
            "401"
        ]
        assert self.results[2].backend.request.headers["kat-req-extauth-requested-header"] == [
            "x-foo"
        ]
        assert self.results[2].backend.request.headers["x-foo"] == ["foo"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert self.results[2].headers["X-Foo"] == ["foo"]
        assert self.results[2].headers["Kat-Resp-Extauth-Protocol-Version"] == [
            self.expected_protocol_version
        ]

        # [3] Verifies default whitelisted Authorization request header.
        assert self.results[3].backend
        assert self.results[3].backend.request
        assert self.results[3].backend.request.headers["kat-req-extauth-requested-status"] == [
            "200"
        ]
        assert self.results[3].backend.request.headers["kat-req-http-requested-header"] == [
            "Authorization"
        ]
        assert self.results[3].backend.request.headers["authorization"] == ["foo-11111"]
        assert self.results[3].backend.request.headers["foo"] == ["foo,bar"]
        assert self.results[3].backend.request.headers["baz"] == ["bar"]
        assert self.results[3].status == 200
        assert self.results[3].headers["Server"] == ["envoy"]
        assert self.results[3].headers["Authorization"] == ["foo-11111"]
        assert self.results[3].backend.request.headers["kat-resp-extauth-protocol-version"] == [
            self.expected_protocol_version
        ]


class AuthenticationDisabledOnRedirectTest(AmbassadorTest):
    """
    AuthenticationDisableOnRedirectTest: ensures that when a route is configured
    for https_redirect or host_redirect that it will perform the redirect
    without calling the AuthService (ext_authz).
    """

    target: ServiceType
    auth: ServiceType

    def init(self):
        if EDGE_STACK:
            self.xfail = "custom AuthServices not supported in Edge Stack"
        self.target = HTTP()
        self.auth = AHTTP(name="auth")
        self.add_default_http_listener = False
        self.add_default_https_listener = True

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["localhost"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["localhost"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [{self.ambassador_id}]
  port: 8080
  protocol: HTTP
  securityModel: XFP
  l7Depth: 1
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
metadata:
  name:  {self.auth.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  auth_service: "{self.auth.path.fqdn}"
  proto: http
  protocol_version: "v3"
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}-secret
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name:  {self.target.path.k8s}
spec:
  ambassador_id: [{self.ambassador_id}]
  hostname: "*"
  prefix: /target/
  service: {self.target.path.fqdn}
  host_redirect: true
"""
            )
            + super().manifests()
        )

    def requirements(self):
        # The client doesn't follow redirects so we must force checks to
        # match the XFP https route. The Listener is configured with
        # l7depth: 1 so that Envoy trusts the header XFP header forwarded
        # by the client.
        yield (
            "url",
            Query(self.url("ambassador/v0/check_ready"), headers={"X-Forwarded-Proto": "https"}),
        )
        yield (
            "url",
            Query(self.url("ambassador/v0/check_alive"), headers={"X-Forwarded-Proto": "https"}),
        )

    def queries(self):
        # send http request
        yield Query(
            self.url("target/", scheme="http"), headers={"X-Forwarded-Proto": "http"}, expected=301
        )

        # send https request
        yield Query(
            self.url("target/", scheme="https"),
            insecure=True,
            headers={"X-Forwarded-Proto": "https"},
            expected=301,
        )

    def check(self):
        # we should NOT make a call to the backend service,
        # rather envoy should have redirected to https
        assert self.results[0].backend is None
        assert self.results[0].headers["Location"] == [f"https://{self.path.fqdn}/target/"]

        assert self.results[1].backend is None
        assert self.results[1].headers["Location"] == [f"https://{self.target.path.fqdn}/target/"]
