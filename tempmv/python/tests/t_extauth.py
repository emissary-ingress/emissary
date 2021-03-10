import json

from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, HTTP, AHTTP, AGRPC
from selfsigned import TLSCerts

class AuthenticationGRPCTest(AmbassadorTest):

    target: ServiceType
    auth: ServiceType

    def init(self):
        self.target = HTTP()
        self.auth = AGRPC(name="auth")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
timeout_ms: 5000
proto: grpc
""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        # [0]
        yield Query(self.url("target/"), headers={"requested-status": "401",
                                                  "baz": "baz",
                                                  "request-header": "baz"}, expected=401)
        # [1]
        yield Query(self.url("target/"), headers={"requested-status": "302",
                                                  "requested-location": "foo"}, expected=302)

        # [2]
        yield Query(self.url("target/"), headers={"requested-status": "401",
                                                  "x-foo": "foo",
                                                  "requested-header": "x-foo"}, expected=401)
        # [3]
        yield Query(self.url("target/"), headers={"requested-status": "200",
                                                  "authorization": "foo-11111",
                                                  "foo" : "foo",
                                                  "x-grpc-auth-append": "foo=bar;baz=bar",
                                                  "requested-header": "Authorization"}, expected=200)

    def check(self):
        # [0] Verifies all request headers sent to the authorization server.
        assert self.results[0].backend.name == self.auth.path.k8s
        assert self.results[0].backend.request.url.path == "/target/"
        assert self.results[0].backend.request.headers["x-envoy-internal"]== ["true"]
        assert self.results[0].backend.request.headers["x-forwarded-proto"]== ["http"]
        assert "user-agent" in self.results[0].backend.request.headers
        assert "baz" in self.results[0].backend.request.headers
        assert self.results[0].status == 401
        assert self.results[0].headers["Server"] == ["envoy"]
        assert self.results[0].headers['X-Grpc-Service-Protocol-Version'] == ['v2']

        # [1] Verifies that Location header is returned from Envoy.
        assert self.results[1].backend.name == self.auth.path.k8s
        assert self.results[1].backend.request.headers["requested-status"] == ["302"]
        assert self.results[1].backend.request.headers["requested-location"] == ["foo"]
        assert self.results[1].status == 302
        assert self.results[1].headers["Location"] == ["foo"]
        assert self.results[1].headers['X-Grpc-Service-Protocol-Version'] == ['v2']

        # [2] Verifies Envoy returns whitelisted headers input by the user.
        assert self.results[2].backend.name == self.auth.path.k8s
        assert self.results[2].backend.request.headers["requested-status"] == ["401"]
        assert self.results[2].backend.request.headers["requested-header"] == ["x-foo"]
        assert self.results[2].backend.request.headers["x-foo"] == ["foo"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert self.results[2].headers["X-Foo"] == ["foo"]
        assert self.results[2].headers['X-Grpc-Service-Protocol-Version'] == ['v2']

        # [3] Verifies default whitelisted Authorization request header.
        assert self.results[3].backend.request.headers["requested-status"] == ["200"]
        assert self.results[3].backend.request.headers["requested-header"] == ["Authorization"]
        assert self.results[3].backend.request.headers["authorization"] == ["foo-11111"]
        assert self.results[3].backend.request.headers["foo"] == ["foo,bar"]
        assert self.results[3].backend.request.headers["baz"] == ["bar"]
        assert self.results[3].status == 200
        assert self.results[3].headers["Server"] == ["envoy"]
        assert self.results[3].headers["Authorization"] == ["foo-11111"]
        assert self.results[3].backend.request.headers['x-grpc-service-protocol-version'] == ['v2']


class AuthenticationHTTPPartialBufferTest(AmbassadorTest):

    target: ServiceType
    auth: ServiceType

    def init(self):
        self.target = HTTP()
        self.auth = HTTP(name="auth")

    def manifests(self) -> str:
        return f"""
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["tls-context-host-1"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-1"].k8s_key}
kind: Secret
metadata:
  name: auth-partial-secret
type: kubernetes.io/tls
""" + super().manifests()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: TLSContext
name: {self.name}-same-context-1
secret: auth-partial-secret

---
apiVersion: ambassador/v2
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"
timeout_ms: 5000
tls: {self.name}-same-context-1

allowed_request_headers:
- Requested-Status
- Requested-Header

allowed_authorization_headers:
- Auth-Request-Body

add_auth_headers:
  X-Added-Auth: auth-added

include_body:
  max_bytes: 7
  allow_partial: true
""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        # [0]
        yield Query(self.url("target/"), headers={"Requested-Status": "200"}, body="message_body", expected=200)

        # [1]
        yield Query(self.url("target/"), headers={"Requested-Status": "200"}, body="body", expected=200)

        # [2]
        yield Query(self.url("target/"), headers={"Requested-Status": "401"}, body="body", expected=401)

    def check(self):
        # [0] Verifies that the authorization server received the partial message body.
        extauth_res1 = json.loads(self.results[0].headers["Extauth"][0])
        assert self.results[0].backend.request.headers["requested-status"] == ["200"]
        assert self.results[0].status == 200
        assert self.results[0].headers["Server"] == ["envoy"]
        assert extauth_res1["request"]["headers"]["auth-request-body"] == ["message"]

        # [1] Verifies that the authorization server received the full message body.
        extauth_res2 = json.loads(self.results[1].headers["Extauth"][0])
        assert self.results[1].backend.request.headers["requested-status"] == ["200"]
        assert self.results[1].status == 200
        assert self.results[1].headers["Server"] == ["envoy"]
        assert extauth_res2["request"]["headers"]["auth-request-body"] == ["body"]

        # [2] Verifies that the authorization server received added headers
        assert self.results[2].backend.request.headers["requested-status"] == ["401"]
        assert self.results[2].backend.request.headers["x-added-auth"] == ["auth-added"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert extauth_res2["request"]["headers"]["auth-request-body"] == ["body"]

class AuthenticationHTTPBufferedTest(AmbassadorTest):

    target: ServiceType
    auth: ServiceType

    def init(self):
        self.target = HTTP()
        self.auth = HTTP(name="auth")

    def manifests(self) -> str:
        return f"""
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["tls-context-host-1"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-1"].k8s_key}
kind: Secret
metadata:
  name: auth-buffered-secret
type: kubernetes.io/tls
""" + super().manifests()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  add_linkerd_headers: true
  buffer:
    max_request_bytes: 16384
---
apiVersion: ambassador/v1
kind: TLSContext
name: {self.name}-same-context-1
secret: auth-buffered-secret
---
apiVersion: ambassador/v1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"
timeout_ms: 5000
tls: {self.name}-same-context-1

allowed_request_headers:
- X-Foo
- X-Bar
- Requested-Status
- Requested-Header
- Requested-Cookie
- Location

allowed_authorization_headers:
- X-Foo
- Set-Cookie

include_body:
  max_bytes: 4096
  allow_partial: true
""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        # [0]
        yield Query(self.url("target/"), headers={"Requested-Status": "401",
                                                  "Baz": "baz",
                                                  "Request-Header": "Baz"}, expected=401)
        # [1]
        yield Query(self.url("target/"), headers={"requested-status": "302",
                                                  "location": "foo",
                                                  "requested-cookie": "foo, bar, baz",
                                                  "requested-header": "location"}, expected=302)
        # [2]
        yield Query(self.url("target/"), headers={"Requested-Status": "401",
                                                  "X-Foo": "foo",
                                                  "Requested-Header": "X-Foo"}, expected=401)
        # [3]
        yield Query(self.url("target/"), headers={"Requested-Status": "401",
                                                  "X-Bar": "bar",
                                                  "Requested-Header": "X-Bar"}, expected=401)
        # [4]
        yield Query(self.url("target/"), headers={"Requested-Status": "200",
                                                  "Authorization": "foo-11111",
                                                  "Requested-Header": "Authorization"}, expected=200)

    def check(self):
        # [0] Verifies all request headers sent to the authorization server.
        assert self.results[0].backend.name == self.auth.path.k8s
        assert self.results[0].backend.request.url.path == "/extauth/target/"
        assert self.results[0].backend.request.headers["x-forwarded-proto"]== ["http"]
        assert self.results[0].backend.request.headers["content-length"]== ["0"]
        assert "x-forwarded-for" in self.results[0].backend.request.headers
        assert "user-agent" in self.results[0].backend.request.headers
        assert "baz" not in self.results[0].backend.request.headers
        assert self.results[0].status == 401
        assert self.results[0].headers["Server"] == ["envoy"]

        # [1] Verifies that Location header is returned from Envoy.
        assert self.results[1].backend.name == self.auth.path.k8s
        assert self.results[1].backend.request.headers["requested-status"] == ["302"]
        assert self.results[1].backend.request.headers["requested-header"] == ["location"]
        assert self.results[1].backend.request.headers["location"] == ["foo"]
        assert self.results[1].status == 302
        assert self.results[1].headers["Server"] == ["envoy"]
        assert self.results[1].headers["Location"] == ["foo"]
        assert self.results[1].headers["Set-Cookie"] == ["foo=foo", "bar=bar", "baz=baz"]

        # [2] Verifies Envoy returns whitelisted headers input by the user.
        assert self.results[2].backend.name == self.auth.path.k8s
        assert self.results[2].backend.request.headers["requested-status"] == ["401"]
        assert self.results[2].backend.request.headers["requested-header"] == ["X-Foo"]
        assert self.results[2].backend.request.headers["x-foo"] == ["foo"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert self.results[2].headers["X-Foo"] == ["foo"]

        # [3] Verifies that envoy does not return not whitelisted headers.
        assert self.results[3].backend.name == self.auth.path.k8s
        assert self.results[3].backend.request.headers["requested-status"] == ["401"]
        assert self.results[3].backend.request.headers["requested-header"] == ["X-Bar"]
        assert self.results[3].backend.request.headers["x-bar"] == ["bar"]
        assert self.results[3].status == 401
        assert self.results[3].headers["Server"] == ["envoy"]
        assert "X-Bar" not in self.results[3].headers

        # [4] Verifies default whitelisted Authorization request header.
        assert self.results[4].backend.request.headers["requested-status"] == ["200"]
        assert self.results[4].backend.request.headers["requested-header"] == ["Authorization"]
        assert self.results[4].backend.request.headers["authorization"] == ["foo-11111"]
        assert self.results[4].backend.request.headers["l5d-dst-override"] ==  [ 'authenticationhttpbufferedtest-http:80' ]
        assert self.results[4].status == 200
        assert self.results[4].headers["Server"] == ["envoy"]
        assert self.results[4].headers["Authorization"] == ["foo-11111"]

class AuthenticationHTTPFailureModeAllowTest(AmbassadorTest):
    target: ServiceType
    auth: ServiceType

    def init(self):
        self.target = HTTP()
        self.auth = HTTP(name="auth")

    def manifests(self) -> str:
        return f"""
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["tls-context-host-1"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-1"].k8s_key}
kind: Secret
metadata:
  name: auth-failure-secret
type: kubernetes.io/tls
""" + super().manifests()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: TLSContext
name: {self.name}-failure-context
secret: auth-failure-secret

---
apiVersion: ambassador/v1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"
timeout_ms: 5000
tls: {self.name}-failure-context

allowed_request_headers:
- Requested-Status
- Requested-Header

failure_mode_allow: true
""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        # [0]
        yield Query(self.url("target/"), headers={"Requested-Status": "200"}, expected=200)

        # [1]
        yield Query(self.url("target/"), headers={"Requested-Status": "503"}, expected=503)

    def check(self):
        # [0] Verifies that the authorization server received the partial message body.
        extauth_res1 = json.loads(self.results[0].headers["Extauth"][0])
        assert self.results[0].backend.request.headers["requested-status"] == ["200"]
        assert self.results[0].status == 200
        assert self.results[0].headers["Server"] == ["envoy"]

        # [1] Verifies that the authorization server received the full message body.
        extauth_res2 = json.loads(self.results[1].headers["Extauth"][0])
        assert self.results[1].backend.request.headers["requested-status"] == ["503"]
        assert self.results[1].headers["Server"] == ["envoy"]

class AuthenticationTestV1(AmbassadorTest):

    target: ServiceType
    auth: ServiceType

    def init(self):
        self.target = HTTP()
        self.auth1 = AHTTP(name="auth1")
        self.auth2 = AHTTP(name="auth2")
        self.backend_counts = {}

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: AuthService
name:  {self.auth1.path.k8s}
auth_service: "{self.auth1.path.fqdn}"
proto: http
path_prefix: "/extauth"
timeout_ms: 5000

allowed_request_headers:
- X-Foo
- X-Bar
- Requested-Status
- Requested-Header
- Location

allowed_authorization_headers:
- X-Foo
- Extauth

status_on_error:
  code: 503

---
apiVersion: ambassador/v1
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
- Requested-Status
- Requested-Header
- Location

allowed_authorization_headers:
- X-Foo
- Extauth

status_on_error:
  code: 503

""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.fqdn}
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.target.path.fqdn}-unauthed
prefix: /target/unauthed/
service: {self.target.path.fqdn}
bypass_auth: true
""")

    def queries(self):
        # [0]
        yield Query(self.url("target/"), headers={"Requested-Status": "401",
                                                  "Baz": "baz",
                                                  "Request-Header": "Baz"}, expected=401)
        # [1]
        yield Query(self.url("target/"), headers={"requested-status": "302",
                                                  "location": "foo",
                                                  "requested-header": "location"}, expected=302)
        # [2]
        yield Query(self.url("target/"), headers={"Requested-Status": "401",
                                                  "X-Foo": "foo",
                                                  "Requested-Header": "X-Foo"}, expected=401)
        # [3]
        yield Query(self.url("target/"), headers={"Requested-Status": "401",
                                                  "X-Bar": "bar",
                                                  "Requested-Header": "X-Bar"}, expected=401)
        # [4]
        yield Query(self.url("target/"), headers={"Requested-Status": "200",
                                                  "Authorization": "foo-11111",
                                                  "Requested-Header": "Authorization"}, expected=200)

        # [5]
        yield Query(self.url("target/"), headers={"X-Forwarded-Proto": "https"}, expected=200)

        # [6]
        yield Query(self.url("target/unauthed/"), headers={"Requested-Status": "200"}, expected=200)

        # [7]
        yield Query(self.url("target/"), headers={"Requested-Status": "500"}, expected=503)

        # Create some traffic to make it more likely that both auth services get at least one
        # request
        for i in range(20):
            yield Query(self.url("target/"), headers={"Requested-Status": "200",
                                                  "Authorization": "foo-11111",
                                                  "Requested-Header": "Authorization"})

    def check_backend_name(self, result) -> bool:
        backend_name = result.backend.name

        self.backend_counts.setdefault(backend_name, 0)
        self.backend_counts[backend_name] += 1

        return (backend_name == self.auth1.path.k8s) or (backend_name == self.auth2.path.k8s)

    def check(self):
        # [0] Verifies all request headers sent to the authorization server.
        assert self.check_backend_name(self.results[0])
        assert self.results[0].backend.request.url.path == "/extauth/target/"
        assert self.results[0].backend.request.headers["x-forwarded-proto"]== ["http"]
        assert self.results[0].backend.request.headers["content-length"]== ["0"]
        assert "x-forwarded-for" in self.results[0].backend.request.headers
        assert "user-agent" in self.results[0].backend.request.headers
        assert "baz" not in self.results[0].backend.request.headers
        assert self.results[0].status == 401
        assert self.results[0].headers["Server"] == ["envoy"]

        # [1] Verifies that Location header is returned from Envoy.
        assert self.check_backend_name(self.results[1])
        assert self.results[1].backend.request.headers["requested-status"] == ["302"]
        assert self.results[1].backend.request.headers["requested-header"] == ["location"]
        assert self.results[1].backend.request.headers["location"] == ["foo"]
        assert self.results[1].status == 302
        assert self.results[1].headers["Server"] == ["envoy"]
        assert self.results[1].headers["Location"] == ["foo"]

        # [2] Verifies Envoy returns whitelisted headers input by the user.
        assert self.check_backend_name(self.results[2])
        assert self.results[2].backend.request.headers["requested-status"] == ["401"]
        assert self.results[2].backend.request.headers["requested-header"] == ["X-Foo"]
        assert self.results[2].backend.request.headers["x-foo"] == ["foo"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert self.results[2].headers["X-Foo"] == ["foo"]

        # [3] Verifies that envoy does not return not whitelisted headers.
        assert self.check_backend_name(self.results[3])
        assert self.results[3].backend.request.headers["requested-status"] == ["401"]
        assert self.results[3].backend.request.headers["requested-header"] == ["X-Bar"]
        assert self.results[3].backend.request.headers["x-bar"] == ["bar"]
        assert self.results[3].status == 401
        assert self.results[3].headers["Server"] == ["envoy"]
        assert "X-Bar" not in self.results[3].headers

        # [4] Verifies default whitelisted Authorization request header.
        assert self.results[4].backend.name == self.target.path.k8s      # this response is from an auth success
        assert self.results[4].backend.request.headers["requested-status"] == ["200"]
        assert self.results[4].backend.request.headers["requested-header"] == ["Authorization"]
        assert self.results[4].backend.request.headers["authorization"] == ["foo-11111"]
        assert self.results[4].status == 200
        assert self.results[4].headers["Server"] == ["envoy"]
        assert self.results[4].headers["Authorization"] == ["foo-11111"]

        extauth_req = json.loads(self.results[4].backend.request.headers["extauth"][0])
        assert extauth_req["request"]["headers"]["l5d-dst-override"] ==  [ 'extauth:80' ]

        # [5] Verify that X-Forwarded-Proto makes it to the auth service.
        #
        # We use the 'extauth' header returned from the test extauth service for this, since
        # the extauth service (on success) won't actually alter other things going upstream.
        r5 = self.results[5]
        assert r5
        assert r5.backend.name == self.target.path.k8s      # this response is from an auth success

        assert r5.status == 200
        assert r5.headers["Server"] == ["envoy"]

        eahdr = r5.backend.request.headers["extauth"]
        assert eahdr, "no extauth header was returned?"
        assert eahdr[0], "an empty extauth header element was returned?"

        # [6] Verifies that Envoy bypasses external auth when disabled for a mapping.
        assert self.results[6].backend.name == self.target.path.k8s      # ensure the request made it to the backend
        assert not self.check_backend_name(self.results[6])      # ensure the request did not go to the auth service
        assert self.results[6].backend.request.headers["requested-status"] == ["200"]
        assert self.results[6].status == 200
        assert self.results[6].headers["Server"] == ["envoy"]

        try:
            eainfo = json.loads(eahdr[0])

            if eainfo:
                # Envoy should force this to HTTP, not HTTPS.
                assert eainfo['request']['headers']['x-forwarded-proto'] == [ 'http' ]
        except ValueError as e:
            assert False, "could not parse Extauth header '%s': %s" % (eahdr, e)

        assert self.backend_counts.get(self.auth1.path.k8s, 0) > 0, "auth1 got no requests"
        assert self.backend_counts.get(self.auth2.path.k8s, 0) > 0, "auth2 got no requests"

        # [7] Verifies that envoy returns customized status_on_error code.
        assert self.results[7].status == 503

        # TODO(gsagula): Write tests for all UCs which request header headers
        # are overridden, e.g. Authorization.


class AuthenticationTest(AmbassadorTest):
    target: ServiceType
    auth: ServiceType

    def init(self):
        self.target = HTTP()
        self.auth = AHTTP(name="auth")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"

allowed_headers:
- X-Foo
- X-Bar
- Requested-Location
- Requested-Status
- Requested-Header
- X-Foo
- Extauth

""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        # [0]
        yield Query(self.url("target/"), headers={"Requested-Status": "401",
                                                  "Baz": "baz",
                                                  "Request-Header": "Baz"}, expected=401)
        # [1]
        yield Query(self.url("target/"), headers={"requested-status": "302",
                                                  "requested-location": "foo",
                                                  "requested-header": "location"}, expected=302)
        # [2]
        yield Query(self.url("target/"), headers={"Requested-Status": "401",
                                                  "X-Foo": "foo",
                                                  "Requested-Header": "X-Foo"}, expected=401)
        # [3]
        yield Query(self.url("target/"), headers={"Requested-Status": "401",
                                                  "X-Bar": "bar",
                                                  "Requested-Header": "X-Bar"}, expected=401)
        # [4]
        yield Query(self.url("target/"), headers={"Requested-Status": "200",
                                                  "Authorization": "foo-11111",
                                                  "Requested-Header": "Authorization"}, expected=200)
        # [5]
        yield Query(self.url("target/"), headers={"X-Forwarded-Proto": "https"}, expected=200)

    def check(self):
        # [0] Verifies all request headers sent to the authorization server.
        assert self.results[0].backend.name == self.auth.path.k8s, f'wanted backend {self.auth.path.k8s}, got {self.results[0].backend.name}'
        assert self.results[0].backend.request.url.path == "/extauth/target/"
        assert self.results[0].backend.request.headers["content-length"]== ["0"]
        assert "x-forwarded-for" in self.results[0].backend.request.headers
        assert "user-agent" in self.results[0].backend.request.headers
        assert "baz" not in self.results[0].backend.request.headers
        assert self.results[0].status == 401
        assert self.results[0].headers["Server"] == ["envoy"]

        # [1] Verifies that Location header is returned from Envoy.
        assert self.results[1].backend.name == self.auth.path.k8s
        assert self.results[1].backend.request.headers["requested-status"] == ["302"]
        assert self.results[1].backend.request.headers["requested-header"] == ["location"]
        assert self.results[1].backend.request.headers["requested-location"] == ["foo"]
        assert self.results[1].status == 302
        assert self.results[1].headers["Server"] == ["envoy"]
        assert self.results[1].headers["Location"] == ["foo"]

        # [2] Verifies Envoy returns whitelisted headers input by the user.
        assert self.results[2].backend.name == self.auth.path.k8s
        assert self.results[2].backend.request.headers["requested-status"] == ["401"]
        assert self.results[2].backend.request.headers["requested-header"] == ["X-Foo"]
        assert self.results[2].backend.request.headers["x-foo"] == ["foo"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert self.results[2].headers["X-Foo"] == ["foo"]

        # [3] Verifies that envoy does not return not whitelisted headers.
        assert self.results[3].backend.name == self.auth.path.k8s
        assert self.results[3].backend.request.headers["requested-status"] == ["401"]
        assert self.results[3].backend.request.headers["requested-header"] == ["X-Bar"]
        assert self.results[3].backend.request.headers["x-bar"] == ["bar"]
        assert self.results[3].status == 401
        assert self.results[3].headers["Server"] == ["envoy"]
        assert "X-Bar" in self.results[3].headers

        # [4] Verifies default whitelisted Authorization request header.
        assert self.results[4].backend.request.headers["requested-status"] == ["200"]
        assert self.results[4].backend.request.headers["requested-header"] == ["Authorization"]
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

        eahdr = r5.backend.request.headers["extauth"]
        assert eahdr, "no extauth header was returned?"
        assert eahdr[0], "an empty extauth header element was returned?"

        try:
            eainfo = json.loads(eahdr[0])

            if eainfo:
                # Envoy should force this to HTTP, not HTTPS.
                assert eainfo['request']['headers']['x-forwarded-proto'] == [ 'http' ]
        except ValueError as e:
            assert False, "could not parse Extauth header '%s': %s" % (eahdr, e)

        # TODO(gsagula): Write tests for all UCs which request header headers
        # are overridden, e.g. Authorization.

class AuthenticationWebsocketTest(AmbassadorTest):

    auth: ServiceType

    def init(self):
        self.auth = HTTP(name="auth")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"
timeout_ms: 10000
allowed_request_headers:
- Requested-Status
allow_request_body: true
---
apiVersion: ambassador/v0
kind:  Mapping
name: {self.name}
prefix: /{self.name}/
service: http://echo.websocket.org
host_rewrite: echo.websocket.org
use_websocket: true
""")


    def queries(self):
        yield Query(self.url(self.name + "/"), expected=404)

        yield Query(self.url(self.name + "/"), expected=101, headers={
            "Requested-Status": "200",
            "Connection": "Upgrade",
            "Upgrade": "websocket",
            "sec-websocket-key": "DcndnpZl13bMQDh7HOcz0g==",
            "sec-websocket-version": "13"
        })

        yield Query(self.url(self.name + "/", scheme="ws"), messages=["one", "two", "three"])

    def check(self):
        assert self.results[-1].messages == ["one", "two", "three"]


class AuthenticationGRPCAlphaV2Test(AmbassadorTest):

    target: ServiceType
    auth: ServiceType

    def init(self):
        self.target = HTTP()
        self.auth = AGRPC(name="auth", protocol_version="v2alpha")

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v2
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
timeout_ms: 5000
protocol_version: "v2alpha"
proto: grpc
""")
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        # TODO add more
        # [0]
        yield Query(self.url("target/"), headers={"requested-status": "401",
                                                  "baz": "baz",
                                                  "request-header": "baz"}, expected=401)

        # [1]
        yield Query(self.url("target/"), headers={"requested-status": "302",
                                                  "requested-location": "foo"}, expected=302)

        # [2]
        yield Query(self.url("target/"), headers={"requested-status": "401",
                                                  "x-foo": "foo",
                                                  "requested-header": "x-foo"}, expected=401)
        # [3]
        yield Query(self.url("target/"), headers={"requested-status": "200",
                                                  "authorization": "foo-11111",
                                                  "foo" : "foo",
                                                  "x-grpc-auth-append": "foo=bar;baz=bar",
                                                  "requested-header": "Authorization"}, expected=200)

    def check(self):
        # [0] Verifies all request headers sent to the authorization server.
        assert self.results[0].backend.name == self.auth.path.k8s
        assert self.results[0].backend.request.url.path == "/target/"
        assert self.results[0].backend.request.headers["x-forwarded-proto"]== ["http"]
        assert "user-agent" in self.results[0].backend.request.headers
        assert "baz" in self.results[0].backend.request.headers
        assert self.results[0].status == 401
        assert self.results[0].headers["Server"] == ["envoy"]
        assert self.results[0].headers['X-Grpc-Service-Protocol-Version'] == ['v2alpha']

        # [1] Verifies that Location header is returned from Envoy.
        assert self.results[1].backend.name == self.auth.path.k8s
        assert self.results[1].backend.request.headers["requested-status"] == ["302"]
        assert self.results[1].backend.request.headers["requested-location"] == ["foo"]
        assert self.results[1].status == 302
        assert self.results[1].headers["Location"] == ["foo"]
        assert self.results[1].headers['X-Grpc-Service-Protocol-Version'] == ['v2alpha']

        # [2] Verifies Envoy returns whitelisted headers input by the user.
        assert self.results[2].backend.name == self.auth.path.k8s
        assert self.results[2].backend.request.headers["requested-status"] == ["401"]
        assert self.results[2].backend.request.headers["requested-header"] == ["x-foo"]
        assert self.results[2].backend.request.headers["x-foo"] == ["foo"]
        assert self.results[2].status == 401
        assert self.results[2].headers["Server"] == ["envoy"]
        assert self.results[2].headers["X-Foo"] == ["foo"]
        assert self.results[2].headers['X-Grpc-Service-Protocol-Version'] == ['v2alpha']

        # [3] Verifies default whitelisted Authorization request header.
        assert self.results[3].backend.request.headers["requested-status"] == ["200"]
        assert self.results[3].backend.request.headers["requested-header"] == ["Authorization"]
        assert self.results[3].backend.request.headers["authorization"] == ["foo-11111"]
        assert self.results[3].backend.request.headers["foo"] == ["foo,bar"]
        assert self.results[3].backend.request.headers["baz"] == ["bar"]
        assert self.results[3].status == 200
        assert self.results[3].headers["Server"] == ["envoy"]
        assert self.results[3].headers["Authorization"] == ["foo-11111"]
        assert self.results[3].backend.request.headers['x-grpc-service-protocol-version'] == ['v2alpha']
