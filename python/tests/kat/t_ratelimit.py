from typing import Generator, Tuple, Union

from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, ServiceType, RLSGRPC, Node
from tests.selfsigned import TLSCerts

from ambassador import Config


class RateLimitV3Test(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        if Config.envoy_api_version != "V3":
            self.skip_node = True
        self.target = HTTP()
        self.rls = RLSGRPC(protocol_version="v3")

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        yield self.target, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  ratelimit_target_mapping
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - request_label_group:
      - request_headers:
          key: x-ambassador-test-allow
          header_name: "x-ambassador-test-allow"
          omit_if_not_present: true
      - request_headers:
          key: x-ambassador-test-headers-append
          header_name: "x-ambassador-test-headers-append"
          omit_if_not_present: true
""")

        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
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
