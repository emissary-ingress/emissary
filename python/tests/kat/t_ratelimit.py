from kat.harness import Query
import os

from abstract_tests import AmbassadorTest, HTTP, ServiceType, RLSGRPC
from selfsigned import TLSCerts

from ambassador import Config


class RateLimitV0Test(AmbassadorTest):
    # debug = True
    target: ServiceType
    rls: ServiceType

    def init(self):
        self.target = HTTP()
        self.rls = RLSGRPC()

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        # ambassador_id: [ {self.with_tracing.ambassador_id}, {self.no_tracing.ambassador_id} ]
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  ratelimit_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
rate_limits:
- descriptor: A test case
  headers:
  - "x-ambassador-test-allow"
  - "x-ambassador-test-headers-append"
---
apiVersion: ambassador/v1
kind:  Mapping
name:  ratelimit_label_mapping
prefix: /labels/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - host_and_user:
      - custom-label:
          header: ":authority"
          omit_if_not_present: true
      - user:
          header: "x-user"
          omit_if_not_present: true

    - omg_header:
      - custom-label:
          header: "x-omg"
          default: "OMFG!"
""")

        # For self.with_tracing, we want to configure the TracingService.
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: RateLimitService
name: {self.rls.path.k8s}
service: "{self.rls.path.fqdn}"
timeout_ms: 500
""")

    def queries(self):
        # Speak through each Ambassador to the traced service...
        # yield Query(self.with_tracing.url("target/"))
        # yield Query(self.no_tracing.url("target/"))

        # [0]
        # No matching headers, won't even go through ratelimit-service filter
        yield Query(self.url("target/"))

        # [1]
        # Header instructing dummy ratelimit-service to allow request
        yield Query(self.url("target/"), expected=200, headers={
            'x-ambassador-test-allow': 'true',
            'x-ambassador-test-headers-append': 'no header',
        })

        # [2]
        # Header instructing dummy ratelimit-service to reject request with
        # a custom response body
        yield Query(self.url("target/"), expected=429, headers={
            'x-ambassador-test-allow': 'over my dead body',
            'x-ambassador-test-headers-append': 'Hello=Foo; Hi=Baz',
        })

    def check(self):
        # [2] Verifies the 429 response and the proper content-type.
        # The kat-server gRPC ratelimit implementation explicitly overrides
        # the content-type to json, because the response is in fact json
        # and we need to verify that this override is possible/correct.
        assert self.results[2].headers["Hello"] == [ "Foo" ]
        assert self.results[2].headers["Hi"] == [ "Baz" ]
        assert self.results[2].headers["Content-Type"] == [ "application/json" ]
        assert self.results[2].headers["X-Grpc-Service-Protocol-Version"] == [ "v2" ]

class RateLimitV1Test(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        self.target = HTTP()
        self.rls = RLSGRPC()

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  ratelimit_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - request_label_group:
      - x-ambassador-test-allow:
          header: "x-ambassador-test-allow"
          omit_if_not_present: true
      - x-ambassador-test-headers-append:
          header: "x-ambassador-test-headers-append"
          omit_if_not_present: true
""")

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: RateLimitService
name: {self.rls.path.k8s}
service: "{self.rls.path.fqdn}"
timeout_ms: 500
""")

    def queries(self):
        # [0]
        # No matching headers, won't even go through ratelimit-service filter
        yield Query(self.url("target/"))

        # [1]
        # Header instructing dummy ratelimit-service to allow request
        yield Query(self.url("target/"), expected=200, headers={
            'x-ambassador-test-allow': 'true',
            'x-ambassador-test-headers-append': 'no header',
        })

        # [2]
        # Header instructing dummy ratelimit-service to reject request
        yield Query(self.url("target/"), expected=429, headers={
            'x-ambassador-test-allow': 'over my dead body',
            'x-ambassador-test-headers-append': 'Hello=Foo; Hi=Baz',
        })

    def check(self):
        # [2] Verifies the 429 response and the proper content-type.
        # The kat-server gRPC ratelimit implementation explicitly overrides
        # the content-type to json, because the response is in fact json
        # and we need to verify that this override is possible/correct.
        assert self.results[2].headers["Hello"] == [ "Foo" ]
        assert self.results[2].headers["Hi"] == [ "Baz" ]
        assert self.results[2].headers["Content-Type"] == [ "application/json" ]
        assert self.results[2].headers["X-Grpc-Service-Protocol-Version"] == [ "v2" ]

class RateLimitV1WithTLSTest(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        self.target = HTTP()
        self.rls = RLSGRPC()

    def manifests(self) -> str:
        return f"""
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["ratelimit.datawire.io"].k8s_crt}
  tls.key: {TLSCerts["ratelimit.datawire.io"].k8s_key}
kind: Secret
metadata:
  name: ratelimit-tls-secret
type: kubernetes.io/tls
""" + super().manifests()

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind: TLSContext
name: ratelimit-tls-context
secret: ratelimit-tls-secret
alpn_protocols: h2
---
apiVersion: ambassador/v1
kind:  Mapping
name:  ratelimit_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - request_label_group:
      - x-ambassador-test-allow:
          header: "x-ambassador-test-allow"
          omit_if_not_present: true
      - x-ambassador-test-headers-append:
          header: "x-ambassador-test-headers-append"
          omit_if_not_present: true
""")

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: RateLimitService
name: {self.rls.path.k8s}
service: "{self.rls.path.fqdn}"
timeout_ms: 500
tls: ratelimit-tls-context
""")

    def queries(self):
        # No matching headers, won't even go through ratelimit-service filter
        yield Query(self.url("target/"))

        # Header instructing dummy ratelimit-service to allow request
        yield Query(self.url("target/"), expected=200, headers={
            'x-ambassador-test-allow': 'true'
        })

        # Header instructing dummy ratelimit-service to reject request
        yield Query(self.url("target/"), expected=429, headers={
            'x-ambassador-test-allow': 'nope',
            'x-ambassador-test-headers-append': 'Hello=Foo; Hi=Baz'
        })

    def check(self):
        # [2] Verifies the 429 response and the proper content-type.
        # The kat-server gRPC ratelimit implementation explicitly overrides
        # the content-type to json, because the response is in fact json
        # and we need to verify that this override is possible/correct.
        assert self.results[2].headers["Hello"] == [ "Foo" ]
        assert self.results[2].headers["Hi"] == [ "Baz" ]
        assert self.results[2].headers["Content-Type"] == [ "application/json" ]
        assert self.results[2].headers["X-Grpc-Service-Protocol-Version"] == [ "v2" ]


class RateLimitV2Test(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        if Config.envoy_api_version == "V3":
            self.skip_node = True
        self.target = HTTP()
        self.rls = RLSGRPC(protocol_version="v2")

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        yield self.target, self.format("""
---
apiVersion: ambassador/v2
kind:  Mapping
name:  ratelimit_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - request_label_group:
      - x-ambassador-test-allow:
          header: "x-ambassador-test-allow"
          omit_if_not_present: true
      - x-ambassador-test-headers-append:
          header: "x-ambassador-test-headers-append"
          omit_if_not_present: true
""")

        yield self, self.format("""
---
apiVersion: ambassador/v2
kind: RateLimitService
name: {self.rls.path.k8s}
service: "{self.rls.path.fqdn}"
timeout_ms: 500
protocol_version: "v2"
""")

    def queries(self):
        # [0]
        # No matching headers, won't even go through ratelimit-service filter
        yield Query(self.url("target/"))

        # [1]
        # Header instructing dummy ratelimit-service to allow request
        yield Query(self.url("target/"), expected=200, headers={
            'x-ambassador-test-allow': 'true',
            'x-ambassador-test-headers-append': 'no header',
        })

        # [2]
        # Header instructing dummy ratelimit-service to reject request
        yield Query(self.url("target/"), expected=429, headers={
            'x-ambassador-test-allow': 'over my dead body',
            'x-ambassador-test-headers-append': 'Hello=Foo; Hi=Baz',
        })

    def check(self):
        # [2] Verifies the 429 response and the proper content-type.
        # The kat-server gRPC ratelimit implementation explicitly overrides
        # the content-type to json, because the response is in fact json
        # and we need to verify that this override is possible/correct.
        assert self.results[2].headers["Hello"] == [ "Foo" ]
        assert self.results[2].headers["Hi"] == [ "Baz" ]
        assert self.results[2].headers["Content-Type"] == [ "application/json" ]
        assert self.results[2].headers["X-Grpc-Service-Protocol-Version"] == [ "v2" ]


class RateLimitV3Test(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        if Config.envoy_api_version != "V3":
            self.skip_node = True
        self.target = HTTP()
        self.rls = RLSGRPC(protocol_version="v3")

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        yield self.target, self.format("""
---
apiVersion: ambassador/v2
kind:  Mapping
name:  ratelimit_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - request_label_group:
      - x-ambassador-test-allow:
          header: "x-ambassador-test-allow"
          omit_if_not_present: true
      - x-ambassador-test-headers-append:
          header: "x-ambassador-test-headers-append"
          omit_if_not_present: true
""")

        yield self, self.format("""
---
apiVersion: ambassador/v2
kind: RateLimitService
name: {self.rls.path.k8s}
service: "{self.rls.path.fqdn}"
timeout_ms: 500
protocol_version: "v3"
""")

    def queries(self):
        # [0]
        # No matching headers, won't even go through ratelimit-service filter
        yield Query(self.url("target/"))

        # [1]
        # Header instructing dummy ratelimit-service to allow request
        yield Query(self.url("target/"), expected=200, headers={
            'x-ambassador-test-allow': 'true',
            'x-ambassador-test-headers-append': 'no header',
        })

        # [2]
        # Header instructing dummy ratelimit-service to reject request
        yield Query(self.url("target/"), expected=429, headers={
            'x-ambassador-test-allow': 'over my dead body',
            'x-ambassador-test-headers-append': 'Hello=Foo; Hi=Baz',
        })

    def check(self):
        # [2] Verifies the 429 response and the proper content-type.
        # The kat-server gRPC ratelimit implementation explicitly overrides
        # the content-type to json, because the response is in fact json
        # and we need to verify that this override is possible/correct.
        assert self.results[2].headers["Hello"] == [ "Foo" ]
        assert self.results[2].headers["Hi"] == [ "Baz" ]
        assert self.results[2].headers["Content-Type"] == [ "application/json" ]
        assert self.results[2].headers["X-Grpc-Service-Protocol-Version"] == [ "v3" ]
