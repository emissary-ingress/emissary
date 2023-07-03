from typing import Generator, Tuple, Union

from abstract_tests import HTTP, AmbassadorTest, Node, ServiceType
from kat.harness import EDGE_STACK, Query
from tests.integration.manifests import namespace_manifest
from tests.selfsigned import TLSCerts

#####
# XXX This file is annoying.
#
# RedirectTestsWithProxyProto and RedirectTestsInvalidSecret used to be subclasses of RedirectTests,
# which makes a certain amount of sense. Problem is that when I wanted to modify just RedirectTests
# to have secrets defined, that ended up affecting the two subclasses in bad ways. There's basically
# no way to subclass an AmbassadorTest without having your base class be run separately, which isn't
# what I wanted here. Sigh.


class RedirectTests(AmbassadorTest):
    target: ServiceType
    edge_stack_cleartext_host = False

    def init(self):
        if EDGE_STACK:
            self.xfail = "Not yet supported in Edge Stack"

        self.xfail = "FIXME: IHA"

        self.target = HTTP()

    def requirements(self):
        # only check https urls since test readiness will only end up barfing on redirect
        yield from (
            r for r in super().requirements() if r[0] == "url" and r[1].url.startswith("https")
        )

    def manifests(self):
        return (
            namespace_manifest("redirect-namespace")
            + f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: redirect-cert
  namespace: redirect-namespace
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["localhost"].k8s_crt}
  tls.key: {TLSCerts["localhost"].k8s_key}
---
apiVersion: v1
kind: Secret
metadata:
  name: redirect-cert
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["localhost"].k8s_crt}
  tls.key: {TLSCerts["localhost"].k8s_key}
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        # Use self here, not self.target, because we want the TLS module to
        # be annotated on the Ambassador itself.
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: tls
ambassador_id: [{self.ambassador_id}]
config:
  server:
    enabled: True
    secret: redirect-cert
    redirect_cleartext_from: 8080
"""
        )

        yield self.target, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tls_target_mapping
hostname: "*"
prefix: /tls-target/
service: {self.target.path.fqdn}
"""
        )

    def queries(self):
        # [0]
        yield Query(self.url("tls-target/", scheme="http"), expected=301)

        # [1] -- PHASE 2
        yield Query(
            self.url("ambassador/v0/diag/?json=true&filter=errors", scheme="https"),
            insecure=True,
            phase=2,
        )

    def check(self):
        # For query 0, check the redirection target.
        assert len(self.results[0].headers["Location"]) > 0
        assert self.results[0].headers["Location"][0].find("/tls-target/") > 0

        # For query 1, we require no errors.
        # XXX Ew. If self.results[1].json is empty, the harness won't convert it to a response.
        errors = self.results[1].json
        assert len(errors) == 0


class RedirectTestsWithProxyProto(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.xfail = "FIXME: IHA"
        self.target = HTTP()

    def requirements(self):
        # only check https urls since test readiness will only end up barfing on redirect
        yield from (
            r for r in super().requirements() if r[0] == "url" and r[1].url.startswith("https")
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind:  Module
name:  ambassador
config:
  use_proxy_proto: true
  enable_ipv6: true
"""
        )

        yield self.target, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tls_target_mapping
hostname: "*"
prefix: /tls-target/
service: {self.target.path.fqdn}
"""
        )

    def queries(self):
        # TODO (concaf): FWIW, this query only covers one side of the story. This tests that this is the correct
        #  deviation from the normal behavior (301 response), but does not test a 301 when proxy proto is actually sent.
        #  This is because net/http does not yet support adding proxy proto to HTTP requests, and hence it's difficult
        #  to test with kat. We will need to open a raw TCP connection (e.g. telnet/nc) and send the entire HTTP Request
        #  in plaintext to test this behavior (or use curl with --haproxy-protocol).
        yield Query(self.url("tls-target/"), error=["EOF", "connection reset by peer"])

    # We can't do the error check until we have the PROXY client mentioned above.
    #     # [1] -- PHASE 2
    #     yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)
    #
    # def check(self):
    #     # We don't have to check anything about query 0, the "expected" clause is enough.
    #
    #     # For query 1, we require no errors.
    #     # XXX Ew. If self.results[1].json is empty, the harness won't convert it to a response.
    #     errors = self.results[1].json
    #     assert(len(errors) == 0)


class RedirectTestsInvalidSecret(AmbassadorTest):
    """
    This test tests that even if the specified secret is invalid, the rest of TLS Context should
    go through. In this case, even though the secret does not exist, redirect_cleartext_from
    should still take effect.
    """

    target: ServiceType

    def init(self):
        if EDGE_STACK:
            self.xfail = "Not yet supported in Edge Stack"

        self.xfail = "FIXME: IHA"
        self.target = HTTP()

    def requirements(self):
        # only check https urls since test readiness will only end up barfing on redirect
        yield from (
            r for r in super().requirements() if r[0] == "url" and r[1].url.startswith("https")
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: tls
ambassador_id: [{self.ambassador_id}]
config:
  server:
    enabled: True
    secret: does-not-exist-secret
    redirect_cleartext_from: 8080
"""
        )

        yield self.target, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tls_target_mapping
hostname: "*"
prefix: /tls-target/
service: {self.target.path.fqdn}
"""
        )

    def queries(self):
        # [0]
        yield Query(self.url("tls-target/"), expected=301)

    # There's kind of no way to do this. Looks like we need to speak HTTP to the port on which we
    # think the server is listening for HTTPS? This is a bad config all the way around, really.
    #     # [1] -- PHASE 2
    #     yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors", scheme="https"), phase=2)
    #
    # def check(self):
    #     # We don't have to check anything about query 0, the "expected" clause is enough.
    #
    #     # For query 1, we require no errors.
    #     # XXX Ew. If self.results[1].json is empty, the harness won't convert it to a response.
    #     errors = self.results[1].json
    #     assert(len(errors) == 0)


class XFPRedirect(AmbassadorTest):
    parent: AmbassadorTest
    target: ServiceType
    edge_stack_cleartext_host = False

    def init(self):
        if EDGE_STACK:
            self.xfail = "Not yet supported in Edge Stack"

        self.target = HTTP()
        self.add_default_http_listener = False
        self.add_default_https_listener = False

    def manifests(self):
        return (
            self.format(
                """
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
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [{self.ambassador_id}]
  requestPolicy:
    insecure:
      action: Redirect
"""
            )
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self.target, self.format(
            """
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
config:
  use_remote_address: false
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/
service: {self.target.path.fqdn}
"""
        )

    def queries(self):
        # [0]
        yield Query(
            self.url(self.name + "/target/"), headers={"X-Forwarded-Proto": "http"}, expected=301
        )

        # [1]
        yield Query(
            self.url(self.name + "/target/"), headers={"X-Forwarded-Proto": "https"}, expected=200
        )

        # [2] -- PHASE 2
        yield Query(
            self.url("ambassador/v0/diag/?json=true&filter=errors"),
            headers={"X-Forwarded-Proto": "https"},
            phase=2,
        )

    def check(self):
        # For query 0, check the redirection target.
        expected_location = ["https://" + self.path.fqdn + "/" + self.name + "/target/"]
        actual_location = self.results[0].headers["Location"]
        assert (
            actual_location == expected_location
        ), "Expected redirect location to be {}, got {} instead".format(
            expected_location, actual_location
        )

        # For query 1, we don't have to check anything, the "expected" clause is enough.

        # For query 2, we require no errors.
        # XXX Ew. If self.results[2].json is empty, the harness won't convert it to a response.
        errors = self.results[2].json
        assert len(errors) == 0

    def requirements(self):
        # We're replacing super()'s requirements deliberately here: we need the XFP header or they can't work.
        yield (
            "url",
            Query(self.url("ambassador/v0/check_ready"), headers={"X-Forwarded-Proto": "https"}),
        )
        yield (
            "url",
            Query(self.url("ambassador/v0/check_alive"), headers={"X-Forwarded-Proto": "https"}),
        )
