from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP
from abstract_tests import ServiceType, TLSRedirect


class RedirectTests(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def requirements(self):
        # only check https urls since test readiness will only end up barfing on redirect
        yield from (r for r in super().requirements() if r[0] == "url" and r[1].url.startswith("https"))

    def config(self):
        # Use self here, not self.target, because we want the TLS module to
        # be annotated on the Ambassador itself.
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: Module
name: tls
ambassador_id: {self.ambassador_id}
config:
  server:
    enabled: True
    redirect_cleartext_from: 8080
""")

        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  tls_target_mapping
prefix: /tls-target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("tls-target/"), expected=301)


class RedirectTestsWithProxyProto(RedirectTests):

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  use_proxy_proto: true
  enable_ipv6: true
""")

    def queries(self):
        # TODO (concaf): FWIW, this query only covers one side of the story. This tests that this is the correct
        #  deviation from the normal behavior (301 response), but does not test a 301 when proxy proto is actually sent.
        #  This is because net/http does not yet support adding proxy proto to HTTP requests, and hence it's difficult
        #  to test with kat. We will need to open a raw TCP connection (e.g. telnet/nc) and send the entire HTTP Request
        #  in plaintext to test this behavior (or use curl with --haproxy-protocol).
        yield Query(self.url("tls-target/"), error="EOF")


class RedirectTestsInvalidSecret(RedirectTests):
    """
    This test tests that even if the specified secret is invalid, the rest of TLS Context should
    go through. In this case, even though the secret does not exist, redirect_cleartext_from
    should still take effect.
    """
    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: Module
name: tls
ambassador_id: {self.ambassador_id}
config:
  server:
    enabled: True
    secret: does-not-exist-secret
    redirect_cleartext_from: 8080
""")

        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  tls_target_mapping
prefix: /tls-target/
service: {self.target.path.fqdn}
""")

class XFPRedirect(TLSRedirect):
    parent: AmbassadorTest
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind: Module
name: ambassador
config:
  x_forwarded_proto_redirect: true
  use_remote_address: false
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: foobar.com
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/target/"), headers={ "X-Forwarded-Proto": "http" }, expected=301)
        yield Query(self.parent.url(self.name + "/target/"), headers={ "X-Forwarded-Proto": "https" }, expected=200)

    def check(self):
        assert self.results[0].headers['Location'] == [
            self.format("https://foobar.com/target/")
        ]
