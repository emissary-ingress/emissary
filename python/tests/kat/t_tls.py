import hashlib
from base64 import b64decode
from typing import Generator, List, Tuple, Union

from abstract_tests import HTTP, AmbassadorTest, Node, ServiceType
from kat.harness import EDGE_STACK, Query
from tests.integration.manifests import namespace_manifest
from tests.selfsigned import TLSCerts
from tests.utils import create_crl_pem_b64

bug_404_routes = (
    True  # Do we erroneously send 404 responses directly instead of redirect-to-tls first?
)


class TLSContextsTest(AmbassadorTest):
    """
    This test makes sure that TLS is not turned on when it's not intended to. For example, when an 'upstream'
    TLS configuration is passed, the port is not supposed to switch to 443
    """

    def init(self):
        self.target = HTTP()

        if EDGE_STACK:
            self.xfail = "Not yet supported in Edge Stack"

    def manifests(self) -> str:
        return (
            f"""
---
apiVersion: v1
metadata:
  name: test-tlscontexts-secret
  labels:
    kat-ambassador-id: tlscontextstest
data:
  tls.crt: {TLSCerts["master.datawire.io"].k8s_crt}
kind: Secret
type: Opaque
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  tlsSecret:
    name: test-tlscontexts-secret
"""
            + super().manifests()
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
  upstream:
    enabled: True
    secret: test-tlscontexts-secret
"""
        )

        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
prefix: /{self.name}/
service: {self.target.path.fqdn}
"""
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(
            self.url(self.name + "/"),
            error=["connection refused", "connection reset by peer", "EOF", "request canceled"],
        )

    def requirements(self):
        yield from (
            r for r in super().requirements() if r[0] == "url" and r[1].url.startswith("http://")
        )


class ClientCertificateAuthentication(AmbassadorTest):
    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            f"""
---
apiVersion: v1
metadata:
  name: test-clientcert-client-secret
  labels:
    kat-ambassador-id: clientcertificateauthentication
data:
  tls.crt: {TLSCerts["master.datawire.io"].k8s_crt}
kind: Secret
type: Opaque
---
apiVersion: v1
kind: Secret
metadata:
  name: test-clientcert-server-secret
  labels:
    kat-ambassador-id: clientcertificateauthentication
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["ambassador.example.com"].k8s_crt}
  tls.key: {TLSCerts["ambassador.example.com"].k8s_key}
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  tlsSecret:
    name: test-clientcert-server-secret
  tls:
    cert_required: True
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
config:
  forward_client_cert_details: SANITIZE_SET
  set_current_client_cert_details:
    subject: true
---
apiVersion: getambassador.io/v3alpha1
kind: Module
ambassador_id: [{self.ambassador_id}]
name: tls
config:
  server:
    enabled: True
    secret: test-clientcert-server-secret
  client:
    enabled: True
    secret: test-clientcert-client-secret
    cert_required: True
"""
        )

        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
prefix: /{self.name}/
service: {self.target.path.fqdn}
add_request_headers:
  x-cert-start:
    value: "%DOWNSTREAM_PEER_CERT_V_START%"
  x-cert-end:
    value: "%DOWNSTREAM_PEER_CERT_V_END%"
  x-cert-start-custom:
    value: "%DOWNSTREAM_PEER_CERT_V_START(%b %e %H:%M:%S %Y %Z)%"
  x-cert-end-custom:
    value: "%DOWNSTREAM_PEER_CERT_V_END(%b %e %H:%M:%S %Y %Z)%"
"""
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(
            self.url(self.name + "/"),
            insecure=True,
            client_crt=TLSCerts["presto.example.com"].pubcert,
            client_key=TLSCerts["presto.example.com"].privkey,
            client_cert_required=True,
            ca_cert=TLSCerts["master.datawire.io"].pubcert,
        )

        # In TLS < 1.3, there's not a dedicated alert code for "the client forgot to include a certificate",
        # so we get a generic alert=40 ("handshake_failure"). We also include "write: connection reset by peer"
        # because we've seen cases where Envoy and the client library don't play nicely, so the error report doesn't
        # get back before the connection closes.
        yield Query(
            self.url(self.name + "/"),
            insecure=True,
            maxTLSv="v1.2",
            error=["tls: handshake failure", "write: connection reset by peer"],
        )

        # TLS 1.3 added a dedicated alert=116 ("certificate_required") for that scenario. See above for why
        # "write: connection reset by peer " is also accepted.
        yield Query(
            self.url(self.name + "/"),
            insecure=True,
            minTLSv="v1.3",
            error=["tls: certificate required", "write: connection reset by peer"],
        )

    def check(self):
        cert = TLSCerts["presto.example.com"].pubcert
        # base64-decode the cert data after removing the "---BEGIN CERTIFICATE---" / "---END CERTIFICATE---" lines.
        certraw = b64decode("\n".join(l for l in cert.split("\n") if not l.startswith("-")))
        # take the sha256 sum aof that.
        certhash = hashlib.sha256(certraw).hexdigest()

        assert self.results[0].backend.request.headers["x-forwarded-client-cert"] == [
            f'Hash={certhash};Subject="CN=presto.example.com,OU=Engineering,O=Ambassador Labs,L=Boston,ST=MA,C=US"'
        ], (
            "unexpected x-forwarded-client-cert value: %s"
            % self.results[0].backend.request.headers["x-forwarded-client-cert"]
        )
        assert self.results[0].backend.request.headers["x-cert-start"] == [
            "2021-11-10T13:12:00.000Z"
        ], (
            "unexpected x-cert-start value: %s"
            % self.results[0].backend.request.headers["x-cert-start"]
        )
        assert self.results[0].backend.request.headers["x-cert-end"] == [
            "2099-11-10T13:12:00.000Z"
        ], (
            "unexpected x-cert-end value: %s"
            % self.results[0].backend.request.headers["x-cert-end"]
        )
        assert self.results[0].backend.request.headers["x-cert-start-custom"] == [
            "Nov 10 13:12:00 2021 UTC"
        ], (
            "unexpected x-cert-start-custom value: %s"
            % self.results[1].backend.request.headers["x-cert-start-custom"]
        )
        assert self.results[0].backend.request.headers["x-cert-end-custom"] == [
            "Nov 10 13:12:00 2099 UTC"
        ], (
            "unexpected x-cert-end-custom value: %s"
            % self.results[0].backend.request.headers["x-cert-end-custom"]
        )

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.insecure = True
            query.client_cert = TLSCerts["presto.example.com"].pubcert
            query.client_key = TLSCerts["presto.example.com"].privkey
            query.client_cert_required = True
            query.ca_cert = TLSCerts["master.datawire.io"].pubcert
            yield (r[0], query)


class ClientCertificateAuthenticationContext(AmbassadorTest):
    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            self.format(
                f"""
---
apiVersion: v1
metadata:
  name: ccauthctx-client-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
data:
  tls.crt: {TLSCerts["master.datawire.io"].k8s_crt}
kind: Secret
type: Opaque
---
apiVersion: v1
kind: Secret
metadata:
  name: ccauthctx-server-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["ambassador.example.com"].k8s_crt}
  tls.key: {TLSCerts["ambassador.example.com"].k8s_key}
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
metadata:
  name: ccauthctx-tls
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [{self.ambassador_id}]
  hosts: [ "*" ]
  secret: ccauthctx-server-secret
  ca_secret: ccauthctx-client-secret
  cert_required: True
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  tlsSecret:
    name: ccauthctx-server-secret
  tlsContext:
    name: ccauthctx-tls
"""
            )
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
prefix: /{self.name}/
service: {self.target.path.fqdn}
"""
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(
            self.url(self.name + "/"),
            insecure=True,
            client_crt=TLSCerts["presto.example.com"].pubcert,
            client_key=TLSCerts["presto.example.com"].privkey,
            client_cert_required=True,
            ca_cert=TLSCerts["master.datawire.io"].pubcert,
        )

        # In TLS < 1.3, there's not a dedicated alert code for "the client forgot to include a certificate",
        # so we get a generic alert=40 ("handshake_failure"). We also include "write: connection reset by peer"
        # because we've seen cases where Envoy and the client library don't play nicely, so the error report doesn't
        # get back before the connection closes.
        yield Query(
            self.url(self.name + "/"),
            insecure=True,
            maxTLSv="v1.2",
            error=["tls: handshake failure", "write: connection reset by peer"],
        )

        # TLS 1.3 added a dedicated alert=116 ("certificate_required") for that scenario. See above for why
        # "write: connection reset by peer" is also accepted.
        yield Query(
            self.url(self.name + "/"),
            insecure=True,
            minTLSv="v1.3",
            error=["tls: certificate required", "write: connection reset by peer"],
        )

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.insecure = True
            query.client_cert = TLSCerts["presto.example.com"].pubcert
            query.client_key = TLSCerts["presto.example.com"].privkey
            query.client_cert_required = True
            query.ca_cert = TLSCerts["master.datawire.io"].pubcert
            yield (r[0], query)


class ClientCertificateAuthenticationContextCRL(AmbassadorTest):
    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            self.format(
                f"""
---
apiVersion: v1
metadata:
  name: ccauthctxcrl-client-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
data:
  tls.crt: {TLSCerts["master.datawire.io"].k8s_crt}
kind: Secret
type: Opaque
---
apiVersion: v1
kind: Secret
metadata:
  name: ccauthctxcrl-server-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["ambassador.example.com"].k8s_crt}
  tls.key: {TLSCerts["ambassador.example.com"].k8s_key}
---
apiVersion: v1
kind: Secret
metadata:
  name: ccauthctxcrl-crl-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: Opaque
data:
  crl.pem: {create_crl_pem_b64(TLSCerts["master.datawire.io"].pubcert, TLSCerts["master.datawire.io"].privkey, [TLSCerts["presto.example.com"].pubcert])}
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
metadata:
  name: ccauthctxcrl-tls
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [{self.ambassador_id}]
  hosts: [ "*" ]
  secret: ccauthctxcrl-server-secret
  ca_secret: ccauthctxcrl-client-secret
  crl_secret: ccauthctxcrl-crl-secret
  cert_required: True
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  tlsSecret:
    name: ccauthctxcrl-server-secret
  tlsContext:
    name: ccauthctxcrl-tls
"""
            )
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
prefix: /
service: {self.target.path.fqdn}
hostname: "*"
"""
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(
            self.url(self.name + "/"),
            insecure=True,
            client_crt=TLSCerts["presto.example.com"].pubcert,
            client_key=TLSCerts["presto.example.com"].privkey,
            client_cert_required=True,
            ca_cert=TLSCerts["master.datawire.io"].pubcert,
            error=["tls: revoked certificate"],
        )

    def requirements(self):
        yield ("pod", self.path.k8s)


class TLSOriginationSecret(AmbassadorTest):
    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: test-origination-secret
  labels:
    kat-ambassador-id: tlsoriginationsecret
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["localhost"].k8s_crt}
  tls.key: {TLSCerts["localhost"].k8s_key}
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        fingerprint = (
            hashlib.sha1(
                (
                    TLSCerts["localhost"].pubcert + "\n" + TLSCerts["localhost"].privkey + "\n"
                ).encode("utf-8")
            )
            .hexdigest()
            .upper()
        )

        yield self, f"""
---
apiVersion: getambassador.io/v3alpha1
kind:  Module
ambassador_id: [{self.ambassador_id}]
name: tls
config:
  upstream:
    secret: test-origination-secret
  upstream-files:
    cert_chain_file: /tmp/ambassador/snapshots/default/secrets-decoded/test-origination-secret/{fingerprint}.crt
    private_key_file: /tmp/ambassador/snapshots/default/secrets-decoded/test-origination-secret/{fingerprint}.key
"""

        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}
prefix: /{self.name}/
service: {self.target.path.fqdn}
tls: upstream
"""
        )

        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-files
prefix: /{self.name}-files/
service: {self.target.path.fqdn}
tls: upstream-files
"""
        )

    def queries(self):
        yield Query(self.url(self.name + "/"))
        yield Query(self.url(self.name + "-files/"))

    def check(self):
        for r in self.results:
            assert r.backend.request.tls.enabled


class TLS(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: test-tls-secret
  labels:
    kat-ambassador-id: tls
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["localhost"].k8s_crt}
  tls.key: {TLSCerts["localhost"].k8s_key}
---
apiVersion: v1
kind: Secret
metadata:
  name: ambassador-certs
  labels:
    kat-ambassador-id: tls
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["localhost"].k8s_crt}
  tls.key: {TLSCerts["localhost"].k8s_key}
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: tls-host
  labels:
    kat-ambassador-id: tls
spec:
  ambassador_id: [tls]
  tlsSecret:
    name: test-tls-secret
  requestPolicy:
    insecure:
      action: Reject
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        #         # Use self here, not self.target, because we want the TLS module to
        #         # be annotated on the Ambassador itself.
        #         yield self, self.format("""
        # ---
        # apiVersion: getambassador.io/v3alpha1
        # kind: Module
        # name: tls
        # ambassador_id: [{self.ambassador_id}]
        # config:
        #   server:
        #     enabled: True
        #     secret: test-tls-secret
        # """)

        # Use self.target _here_, because we want the mapping to be annotated
        # on the service, not the Ambassador. Also, you don't need to include
        # the ambassador_id unless you need some special ambassador_id that
        # isn't something that kat already knows about.
        #
        # If the test were more complex, we'd probably need to do some sort
        # of mangling for the mapping name and prefix. For this simple test,
        # it's not necessary.
        yield self.target, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tls_target_mapping
prefix: /tls-target/
service: {self.target.path.fqdn}
"""
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("tls-target/"), insecure=True)


class TLSInvalidSecret(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

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
    secret: test-certs-secret-invalid
  missing-secret-key:
    cert_chain_file: /nonesuch
  bad-path-info:
    cert_chain_file: /nonesuch
    private_key_file: /nonesuch
  validation-without-termination:
    enabled: True
    secret: test-certs-secret-invalid
    ca_secret: ambassador-certs
"""
        )

        yield self.target, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tls_target_mapping
prefix: /tls-target/
service: {self.target.path.fqdn}
"""
        )

    def scheme(self) -> str:
        return "http"

    def queries(self):
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)

    def check(self):
        errors = self.results[0].backend.response

        expected = set(
            {
                "TLSContext server found no certificate in secret test-certs-secret-invalid in namespace default, ignoring...",
                "TLSContext bad-path-info found no cert_chain_file '/nonesuch'",
                "TLSContext bad-path-info found no private_key_file '/nonesuch'",
                "TLSContext validation-without-termination found no certificate in secret test-certs-secret-invalid in namespace default, ignoring...",
                "TLSContext missing-secret-key: 'cert_chain_file' requires 'private_key_file' as well",
            }
        )

        current = set({})
        for errsvc, errtext in errors:
            current.add(errtext)

        diff = expected - current

        assert len(diff) == 0, f"expected {len(expected)} errors, got {len(errors)}: Missing {diff}"


class TLSContextTest(AmbassadorTest):
    # debug = True

    def init(self):
        self.target = HTTP()

        if EDGE_STACK:
            self.xfail = "XFailing for now"

    def manifests(self) -> str:
        return (
            namespace_manifest("secret-namespace")
            + f"""
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["localhost"].k8s_crt}
  tls.key: {TLSCerts["localhost"].k8s_key}
kind: Secret
metadata:
  name: test-tlscontext-secret-0
  labels:
    kat-ambassador-id: tlscontexttest
type: kubernetes.io/tls
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["tls-context-host-1"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-1"].k8s_key}
kind: Secret
metadata:
  name: test-tlscontext-secret-1
  namespace: secret-namespace
  labels:
    kat-ambassador-id: tlscontexttest
type: kubernetes.io/tls
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["tls-context-host-2"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-2"].k8s_key}
kind: Secret
metadata:
  name: test-tlscontext-secret-2
  labels:
    kat-ambassador-id: tlscontexttest
type: kubernetes.io/tls
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-1
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  tlsSecret:
    name: test-tlscontext-secret-1
    namespace: secret-namespace
  tlsContext:
    name: {self.name}-same-context-1
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-2
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-2
  tlsSecret:
    name: test-tlscontext-secret-2
  tlsContext:
    name: {self.name}-same-context-2
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-no-secret
spec:
  ambassador_id: [ {self.ambassador_id} ]
  tlsContext:
    name: {self.name}-no-secret
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-same-context-error
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  tlsContext:
    name: {self.name}-same-context-error
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-rcf-error
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  tlsContext:
    name: {self.name}-rcf-error
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-same-prefix-1
prefix: /tls-context-same/
service: http://{self.target.path.fqdn}
host: tls-context-host-1
"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-same-context-1
hosts:
- tls-context-host-1
secret: test-tlscontext-secret-1.secret-namespace
min_tls_version: v1.0
max_tls_version: v1.3
redirect_cleartext_from: 8080
"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-same-prefix-2
prefix: /tls-context-same/
service: http://{self.target.path.fqdn}
host: tls-context-host-2
"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-same-context-2
hosts:
- tls-context-host-2
secret: test-tlscontext-secret-2
alpn_protocols: h2,http/1.1
redirect_cleartext_from: 8080
"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: tls
config:
  server:
    enabled: True
    secret: test-tlscontext-secret-0
"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-other-mapping
prefix: /{self.name}/
service: https://{self.target.path.fqdn}
"""
        )
        # Ambassador should not return an error when hostname is not present.
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-no-secret
min_tls_version: v1.0
max_tls_version: v1.3
redirect_cleartext_from: 8080
"""
        )
        # Ambassador should return an error for this configuration.
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-same-context-error
hosts:
- tls-context-host-1
redirect_cleartext_from: 8080
"""
        )
        # Ambassador should return an error for this configuration.
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-rcf-error
hosts:
- tls-context-host-1
redirect_cleartext_from: 8081
"""
        )

    def scheme(self) -> str:
        return "https"

    @staticmethod
    def _go_close_connection_error(url):
        """
        :param url: url passed to the query
        :return: error message string that Go's net/http package throws when server closes connection
        """
        return "Get {}: EOF".format(url)

    def queries(self):
        # 0
        yield Query(
            self.url("ambassador/v0/diag/?json=true&filter=errors"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
        )

        # 1 - Correct host #1
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
        )
        # 2 - Correct host #2
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-2"},
            expected=200,
            insecure=True,
            sni=True,
        )

        # 3 - Incorrect host
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-3"},
            # error=self._go_close_connection_error(self.url("tls-context-same/")),
            expected=404,
            insecure=True,
        )

        # 4 - Incorrect path, correct host
        yield Query(
            self.url("tls-context-different/"),
            headers={"Host": "tls-context-host-1"},
            expected=404,
            insecure=True,
            sni=True,
        )

        # Other mappings with no host will respond with the fallbock cert.
        # 5 - no Host header, fallback cert from the TLS module
        yield Query(
            self.url(self.name + "/"),
            # error=self._go_close_connection_error(self.url(self.name + "/")),
            insecure=True,
        )

        # 6 - explicit Host header, fallback cert
        yield Query(
            self.url(self.name + "/"),
            # error=self._go_close_connection_error(self.url(self.name + "/")),
            # sni=True,
            headers={"Host": "tls-context-host-3"},
            insecure=True,
        )

        # 7 - explicit Host header 1 wins, we'll get the SNI cert for this overlapping path
        yield Query(
            self.url(self.name + "/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
        )

        # 8 - explicit Host header 2 wins, we'll get the SNI cert for this overlapping path
        yield Query(
            self.url(self.name + "/"),
            headers={"Host": "tls-context-host-2"},
            expected=200,
            insecure=True,
            sni=True,
        )

        # 9 - Redirect cleartext from actually redirects.
        yield Query(
            self.url("tls-context-same/", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=301,
            insecure=True,
            sni=True,
        )

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json
        num_errors = len(errors)
        assert num_errors == 5, "expected 5 errors, got {} -\n{}".format(num_errors, errors)

        errors_that_should_be_found = {
            "TLSContext TLSContextTest-no-secret has no certificate information at all?": False,
            "TLSContext TLSContextTest-same-context-error has no certificate information at all?": False,
            "TLSContext TLSContextTest-same-context-error is missing cert_chain_file": False,
            "TLSContext TLSContextTest-same-context-error is missing private_key_file": False,
            "TLSContext: TLSContextTest-rcf-error; configured conflicting redirect_from port: 8081": False,
        }

        unknown_errors: List[str] = []
        for err in errors:
            text = err[1]

            if text in errors_that_should_be_found:
                errors_that_should_be_found[text] = True
            else:
                unknown_errors.append(f"Unexpected error {text}")

        for err, found in errors_that_should_be_found.items():
            if not found:
                unknown_errors.append(f"Missing error {err}")

        assert not unknown_errors, f"Problems with errors: {unknown_errors}"

        idx = 0

        for result in self.results:
            if result.status == 200 and result.query.headers:
                host_header = result.query.headers["Host"]
                tls_common_name = result.tls[0]["Issuer"]["CommonName"]

                # XXX Weirdness with the fallback cert here! You see, if we use host
                # tls-context-host-3 (or, really, anything except -1 or -2), then the
                # fallback cert actually has CN 'localhost'. We should replace this with
                # a real fallback cert, but for now, just hack the host_header.
                #
                # Ew.

                if host_header == "tls-context-host-3":
                    host_header = "localhost"

                assert host_header == tls_common_name, "test %d wanted CN %s, but got %s" % (
                    idx,
                    host_header,
                    tls_common_name,
                )

            idx += 1

    def requirements(self):
        # We're replacing super()'s requirements deliberately here. Without a Host header they can't work.
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_ready"),
                headers={"Host": "tls-context-host-1"},
                insecure=True,
                sni=True,
            ),
        )
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_alive"),
                headers={"Host": "tls-context-host-1"},
                insecure=True,
                sni=True,
            ),
        )
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_ready"),
                headers={"Host": "tls-context-host-2"},
                insecure=True,
                sni=True,
            ),
        )
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_alive"),
                headers={"Host": "tls-context-host-2"},
                insecure=True,
                sni=True,
            ),
        )


class TLSIngressTest(AmbassadorTest):
    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        self.manifest_envs = """
    - name: AMBASSADOR_DEBUG
      value: "diagd"
"""

        return (
            namespace_manifest("secret-namespace-ingress")
            + f"""
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["localhost"].k8s_crt}
  tls.key: {TLSCerts["localhost"].k8s_key}
kind: Secret
metadata:
  name: test-tlscontext-secret-ingress-0
  labels:
    kat-ambassador-id: tlsingresstest
type: kubernetes.io/tls
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["tls-context-host-1"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-1"].k8s_key}
kind: Secret
metadata:
  name: test-tlscontext-secret-ingress-1
  namespace: secret-namespace-ingress
  labels:
    kat-ambassador-id: tlsingresstest
type: kubernetes.io/tls
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["tls-context-host-2"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-2"].k8s_key}
kind: Secret
metadata:
  name: test-tlscontext-secret-ingress-2
  labels:
    kat-ambassador-id: tlsingresstest
type: kubernetes.io/tls
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: tlsingresstest
  name: {self.name.lower()}-1
spec:
  tls:
  - secretName: test-tlscontext-secret-ingress-1.secret-namespace-ingress
    hosts:
    - tls-context-host-1
  rules:
  - host: tls-context-host-1
    http:
      paths:
      - backend:
          service:
            name: {self.target.path.k8s}
            port:
              number: 80
        path: /tls-context-same/
        pathType: Prefix
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: tlsingresstest
  name: {self.name.lower()}-2
spec:
  tls:
  - secretName: test-tlscontext-secret-ingress-2
    hosts:
    - tls-context-host-2
  rules:
  - host: tls-context-host-2
    http:
      paths:
      - backend:
          service:
            name: {self.target.path.k8s}
            port:
              number: 80
        path: /tls-context-same/
        pathType: Prefix
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  tlsSecret:
    name: test-tlscontext-secret-ingress-0
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: tls
config:
  server:
    enabled: True
    secret: test-tlscontext-secret-ingress-0
"""
        )

        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
hostname: "*"
name:  {self.name}-other-mapping
prefix: /{self.name}/
service: https://{self.target.path.fqdn}
"""
        )

    def scheme(self) -> str:
        return "https"

    @staticmethod
    def _go_close_connection_error(url):
        """
        :param url: url passed to the query
        :return: error message string that Go's net/http package throws when server closes connection
        """
        return "Get {}: EOF".format(url)

    def queries(self):
        # 0
        yield Query(
            self.url("ambassador/v0/diag/?json=true&filter=errors"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
        )

        # 1 - Correct host #1
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
        )
        # 2 - Correct host #2
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-2"},
            expected=200,
            insecure=True,
            sni=True,
        )

        # 3 - Incorrect host
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-3"},
            # error=self._go_close_connection_error(self.url("tls-context-same/")),
            expected=404,
            insecure=True,
        )

        # 4 - Incorrect path, correct host
        yield Query(
            self.url("tls-context-different/"),
            headers={"Host": "tls-context-host-1"},
            expected=404,
            insecure=True,
            sni=True,
        )

        # Other mappings with no host will respond with the fallbock cert.
        # 5 - no Host header, fallback cert from the TLS module
        yield Query(
            self.url(self.name + "/"),
            # error=self._go_close_connection_error(self.url(self.name + "/")),
            insecure=True,
        )

        # 6 - explicit Host header, fallback cert
        yield Query(
            self.url(self.name + "/"),
            # error=self._go_close_connection_error(self.url(self.name + "/")),
            # sni=True,
            headers={"Host": "tls-context-host-3"},
            insecure=True,
        )

        # 7 - explicit Host header 1 wins, we'll get the SNI cert for this overlapping path
        yield Query(
            self.url(self.name + "/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
        )

        # 7 - explicit Host header 2 wins, we'll get the SNI cert for this overlapping path
        yield Query(
            self.url(self.name + "/"),
            headers={"Host": "tls-context-host-2"},
            expected=200,
            insecure=True,
            sni=True,
        )

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json
        num_errors = len(errors)
        assert num_errors == 0, "expected 0 errors, got {} -\n{}".format(num_errors, errors)

        idx = 0

        for result in self.results:
            if result.status == 200 and result.query.headers:
                host_header = result.query.headers["Host"]
                tls_common_name = result.tls[0]["Issuer"]["CommonName"]

                # XXX Weirdness with the fallback cert here! You see, if we use host
                # tls-context-host-3 (or, really, anything except -1 or -2), then the
                # fallback cert actually has CN 'localhost'. We should replace this with
                # a real fallback cert, but for now, just hack the host_header.
                #
                # Ew.

                if host_header == "tls-context-host-3":
                    host_header = "localhost"

                # Yep, that's expected. Since the TLS secret for 'tls-context-host-1' is
                # not namespaced it should only resolve to the Ingress' own
                # namespace, and can't use the 'secret.namespace' Ambassador syntax
                if host_header == "tls-context-host-1":
                    host_header = "localhost"

                assert host_header == tls_common_name, "test %d wanted CN %s, but got %s" % (
                    idx,
                    host_header,
                    tls_common_name,
                )

            idx += 1

    def requirements(self):
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_ready"),
                headers={"Host": "tls-context-host-1"},
                insecure=True,
                sni=True,
            ),
        )
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_alive"),
                headers={"Host": "tls-context-host-1"},
                insecure=True,
                sni=True,
            ),
        )
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_ready"),
                headers={"Host": "tls-context-host-2"},
                insecure=True,
                sni=True,
            ),
        )
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_alive"),
                headers={"Host": "tls-context-host-2"},
                insecure=True,
                sni=True,
            ),
        )


class TLSContextProtocolMaxVersion(AmbassadorTest):
    # Here we're testing that the client can't exceed the maximum TLS version
    # configured.

    def init(self):
        self.target = HTTP()

        if EDGE_STACK:
            self.xfail = "Not yet supported in Edge Stack"

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
  name: secret.max-version
  labels:
    kat-ambassador-id: tlscontextprotocolmaxversion
type: kubernetes.io/tls
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  tlsSecret:
    name: secret.max-version
  tlsContext:
    name: {self.name}-same-context-1
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
  defaults:
    tls_secret_namespacing: False
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-same-prefix-1
prefix: /tls-context-same/
service: http://{self.target.path.fqdn}
host: tls-context-host-1
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-same-context-1
hosts:
- tls-context-host-1
secret: secret.max-version
min_tls_version: v1.1
max_tls_version: v1.2
"""
        )

    def scheme(self) -> str:
        return "https"

    @staticmethod
    def _go_close_connection_error(url):
        """
        :param url: url passed to the query
        :return: error message string that Go's net/http package throws when server closes connection
        """
        return "Get {}: EOF".format(url)

    def queries(self):
        # ----
        # XXX 2019-09-11
        # These aren't actually reporting the negotiated version, alhough correct
        # behavior can be verified with a custom log format. What, does the silly thing just not
        # report the negotiated version if it's the max you've requested??
        #
        # For now, we're checking for the None result, but, ew.
        # ----

        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
            minTLSv="v1.2",
            maxTLSv="v1.2",
        )

        # This should give us TLS v1.1
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
            minTLSv="v1.0",
            maxTLSv="v1.1",
        )

        # This should be an error.
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
            minTLSv="v1.3",
            maxTLSv="v1.3",
            error=[
                "tls: server selected unsupported protocol version 303",
                "tls: no supported versions satisfy MinVersion and MaxVersion",
                "tls: protocol version not supported",
                "read: connection reset by peer",
            ],
        )  # The TLS inspector just closes the connection. Wow.

    def check(self):
        tls_0_version = self.results[0].backend.request.tls.negotiated_protocol_version
        tls_1_version = self.results[1].backend.request.tls.negotiated_protocol_version

        # See comment in queries for why these are None. They should be v1.2 and v1.1 respectively.
        assert tls_0_version == None, f"requesting TLS v1.2 got TLS {tls_0_version}"
        assert tls_1_version == None, f"requesting TLS v1.0-v1.1 got TLS {tls_1_version}"

    def requirements(self):
        # We're replacing super()'s requirements deliberately here. Without a Host header they can't work.
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_ready"),
                headers={"Host": "tls-context-host-1"},
                insecure=True,
                sni=True,
                minTLSv="v1.2",
            ),
        )
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_alive"),
                headers={"Host": "tls-context-host-1"},
                insecure=True,
                sni=True,
                minTLSv="v1.2",
            ),
        )


class TLSContextProtocolMinVersion(AmbassadorTest):
    # Here we're testing that the client can't drop below the minimum TLS version
    # configured.
    #
    # XXX 2019-09-11: vet that the test client's support for TLS v1.3 is up-to-date.
    # It appears not to be.

    # debug = True

    def init(self):
        self.target = HTTP()

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
  name: secret.min-version
  labels:
    kat-ambassador-id: tlscontextprotocolminversion
type: kubernetes.io/tls
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  tlsSecret:
    name: secret.min-version
  tlsContext:
    name: {self.name}-same-context-1
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-same-prefix-1
prefix: /tls-context-same/
service: https://{self.target.path.fqdn}
host: tls-context-host-1
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-same-context-1
hosts:
- tls-context-host-1
secret: secret.min-version
secret_namespacing: False
min_tls_version: v1.2
max_tls_version: v1.3
"""
        )

    def scheme(self) -> str:
        return "https"

    @staticmethod
    def _go_close_connection_error(url):
        """
        :param url: url passed to the query
        :return: error message string that Go's net/http package throws when server closes connection
        """
        return "Get {}: EOF".format(url)

    def queries(self):
        # This should give v1.3, but it currently seems to give 1.2.
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
            minTLSv="v1.2",
            maxTLSv="v1.3",
        )

        # This should give v1.2
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
            minTLSv="v1.1",
            maxTLSv="v1.2",
        )

        # This should be an error.
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
            minTLSv="v1.0",
            maxTLSv="v1.0",
            error=[
                "tls: server selected unsupported protocol version 303",
                "tls: no supported versions satisfy MinVersion and MaxVersion",
                "tls: protocol version not supported",
            ],
        )

    def check(self):
        tls_0_version = self.results[0].backend.request.tls.negotiated_protocol_version
        tls_1_version = self.results[1].backend.request.tls.negotiated_protocol_version

        # Hmmm. Why does Envoy prefer 1.2 to 1.3 here?? This may be a client thing -- have to
        # rebuild with Go 1.13.
        assert tls_0_version == "v1.2", f"requesting TLS v1.2-v1.3 got TLS {tls_0_version}"
        assert tls_1_version == "v1.2", f"requesting TLS v1.1-v1.2 got TLS {tls_1_version}"

    def requirements(self):
        # We're replacing super()'s requirements deliberately here. Without a Host header they can't work.
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_ready"),
                headers={"Host": "tls-context-host-1"},
                insecure=True,
                sni=True,
            ),
        )
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_alive"),
                headers={"Host": "tls-context-host-1"},
                insecure=True,
                sni=True,
            ),
        )


class TLSContextCipherSuites(AmbassadorTest):
    # debug = True

    def init(self):
        self.target = HTTP()

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
  name: secret.cipher-suites
  labels:
    kat-ambassador-id: tlscontextciphersuites
type: kubernetes.io/tls
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  tlsSecret:
    name: secret.cipher-suites
  tlsContext:
    name: {self.name}-same-context-1
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-same-prefix-1
prefix: /tls-context-same/
service: https://{self.target.path.fqdn}
host: tls-context-host-1
"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-same-context-1
hosts:
- tls-context-host-1
secret: secret.cipher-suites
secret_namespacing: False
max_tls_version: v1.2
cipher_suites:
- ECDHE-RSA-AES128-GCM-SHA256
ecdh_curves:
- P-256
"""
        )

    def scheme(self) -> str:
        return "https"

    @staticmethod
    def _go_close_connection_error(url):
        """
        :param url: url passed to the query
        :return: error message string that Go's net/http package throws when server closes connection
        """
        return "Get {}: EOF".format(url)

    def queries(self):
        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
            cipherSuites=["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"],
            maxTLSv="v1.2",
        )

        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
            cipherSuites=["TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"],
            maxTLSv="v1.2",
            error="tls: handshake failure",
        )

        yield Query(
            self.url("tls-context-same/"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
            insecure=True,
            sni=True,
            cipherSuites=["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"],
            ecdhCurves=["X25519"],
            maxTLSv="v1.2",
            error="tls: handshake failure",
        )

    def check(self):
        tls_0_version = self.results[0].backend.request.tls.negotiated_protocol_version

        assert tls_0_version == "v1.2", f"requesting TLS v1.2 got TLS {tls_0_version}"

    def requirements(self):
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_ready"),
                headers={"Host": "tls-context-host-1"},
                insecure=True,
                sni=True,
            ),
        )
        yield (
            "url",
            Query(
                self.url("ambassador/v0/check_alive"),
                headers={"Host": "tls-context-host-1"},
                insecure=True,
                sni=True,
            ),
        )


class TLSContextIstioSecretTest(AmbassadorTest):
    # debug = True

    def init(self):
        self.target = HTTP()

        if EDGE_STACK:
            self.xfail = "XFailing for now"

    def manifests(self) -> str:
        return (
            namespace_manifest("secret-namespace")
            + """
---
apiVersion: v1
data:
  cert-chain.pem: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURJVENDQWdtZ0F3SUJBZ0lSQU8wbFh1OVhOYkNrejJJTEhiYlVBbDh3RFFZSktvWklodmNOQVFFTEJRQXcKR0RFV01CUUdBMVVFQ2hNTlkyeDFjM1JsY2k1c2IyTmhiREFlRncweU1EQXhNakF4TmpReE5EbGFGdzB5TURBMApNVGt4TmpReE5EbGFNQUF3Z2dFaU1BMEdDU3FHU0liM0RRRUJBUVVBQTRJQkR3QXdnZ0VLQW9JQkFRQ3h2RWxuCmd6SldTejR6RGM5TE5od0xCZm1nTStlY3k0T096UEFtSGhnZER2RFhLVE40Qll0bS8veTFRT2tGNG9JeHVMVnAKYW5ULzdHdUJHNzlrbUg1TkpkcWhzV0c1b1h0TWpiZnZnZFJ6dW50UVg1OFI5d0pWT2YwNlo4dHFUYmE4VVI3YQpYZFY1c2VSbGtINU1VWmhVNXkxNzA1ZVNycVBROGVBd1hiazdOejNlTUd4Ujc1NjZOK3g2UDIrcEZmTDF1dEJ3CnRhSVVpYlVNR0liODcwYmtxVmlzSHQ1aC95blkrV3FlclJLREhTLzVRQlZiMytZSXd4N3o1b3FPbDBvZ05YODkKVnlzNFM0NzdXNDBPWGRZaStHeGwwKzFVT2F3NEw2a0tTaWhjVTZJUm1YbWhiUXpRb0VvazN6TDNaR2hWS3FhbwpUaFdqTVhrMkZxS1pNSnBCQWdNQkFBR2pmakI4TUE0R0ExVWREd0VCL3dRRUF3SUZvREFkQmdOVkhTVUVGakFVCkJnZ3JCZ0VGQlFjREFRWUlLd1lCQlFVSEF3SXdEQVlEVlIwVEFRSC9CQUl3QURBOUJnTlZIUkVCQWY4RU16QXgKaGk5emNHbG1abVU2THk5amJIVnpkR1Z5TG14dlkyRnNMMjV6TDJGdFltRnpjMkZrYjNJdmMyRXZaR1ZtWVhWcwpkREFOQmdrcWhraUc5dzBCQVFzRkFBT0NBUUVBaHQ3c1dSOHEzeFNaM1BsTGFnS0REc1c2UlYyRUJCRkNhR08rCjlJb2lrQXZTdTV2b3VKS3EzVHE0WU9LRzJnbEpvVSs1c2lmL25DYzFva1ZTakNJSnh1UVFhdzd5QkV0WWJaZkYKSXI2WEkzbUtCVC9kWHpOM00yL1g4Q3RBNHI5SFQ4VmxmMitJMHNqb01hVE80WHdPNVQ5eXdoREJXdzdrdThVRApnMjdzTFlHVy9UNzIvT0JGUEcxa2VlRUpva3BhSXZQOVliWS9qSlRWZVVIYk1FODVOckJFMWNndUVnSlVod1VKCkhiam4xcEFKMHZsUWZrVW9mT3VRZkFtZGpHWjc2N2phOE5ldHZBdk9tRExPV2dzQWM4KzRsRXBKVURwcmhlVEoKazBrSFh6cUMyTzN4a250U0QxM2FFa2VUMXJocjM3MXc1OTVJUjgvR1llSis3a3JqRkE9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
  key.pem: LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBc2J4Slo0TXlWa3MrTXczUFN6WWNDd1g1b0RQbm5NdURqc3p3Smg0WUhRN3cxeWt6CmVBV0xadi84dFVEcEJlS0NNYmkxYVdwMC8reHJnUnUvWkpoK1RTWGFvYkZodWFGN1RJMjM3NEhVYzdwN1VGK2YKRWZjQ1ZUbjlPbWZMYWsyMnZGRWUybDNWZWJIa1paQitURkdZVk9jdGU5T1hrcTZqMFBIZ01GMjVPemM5M2pCcwpVZStldWpmc2VqOXZxUlh5OWJyUWNMV2lGSW0xREJpRy9POUc1S2xZckI3ZVlmOHAyUGxxbnEwU2d4MHYrVUFWClc5L21DTU1lOCthS2pwZEtJRFYvUFZjck9FdU8rMXVORGwzV0l2aHNaZFB0VkRtc09DK3BDa29vWEZPaUVabDUKb1cwTTBLQktKTjh5OTJSb1ZTcW1xRTRWb3pGNU5oYWltVENhUVFJREFRQUJBb0lCQUI1bXdIK09OMnYvVHRKWQp5RjVyRVB6cHRyc3FaYkd5TmZ5VkhYYkhxd1E5YkFEQnNXWVVQTFlQajJCSmpCSlBua2wyK01EaFRzWC80SnVpCjdXZjlsWTBJcm83OTBtTjROYWp3ak1mUkExQVFVOHQ1cjdIWStITXZpaHNWYWZ2eTh4RGZKMUhldndjajRKZG0KMGRPb0dWQmNnckV0amoydTFhS0YzUDBvNnVndno2SmtSWld2SjZ4SGlya0NETk5MWlpzbHB5UzFHRjZmYm9aTwp1SmFTLzc2S25JS1FQT3hCaE83ME80WHF6am5wMVk1UzduTjRoM1Z2RmVPREcvQ2pWaGhOcE4xV0NadFNvSXBwCk9XOVdONVRvUnZhVDhnelljcG9TOEMzYXVqSzVvV1FiVzdRZys2NXRoWGNqcFpRM0VFSnNaLzNsTWRsbGE3TFcKT2k3Vkhpa0NnWUVBeHBUQjZodnBRcnNXUUhjcDhRdG94RitNUThVL2l5WjZ6dU5BNHZyWFdwUlFDVVg4d1ZiRwowTFNZN1lSVGhuOGtUZ09vWlNWMU9VcThUTjlnOG91UUh6bS9ta1FpV0p0bnNXWGJtNjF3SFozaWNlQ1FxWDU4CmoyUjM2eXBONGpuUENPREVwcDVKWExZLzNFTnZnYTBxSm9ZVWp4UUpHZDgyWUxKRmJrMHZmTzhDZ1lFQTVTQ0MKcHJTR0NBL0dUVkY4MjRmaW1YTkNMcllOVmV1TStqZFdqQUFBZkQzWHpUK1JWeFZsTENTVUluQUdtYjh2djZlcApreHYrdWlBZTg2TDBhUVVDTENSRFF2SjR3MnNPRWkwWWMwTGlKUGdBN1JLeFhwVGUrQ09vS1VmcTZyVi96TTdNCmhCbWtDT2ZoUnRDT3NENGNBcWQ0MzluQjZBVm01K21VV0FqNHU4OENnWUJFTXBSQi9TSG5xKzZoWndzOVgraTAKQUFoZ3dkM253T2hPSXRlRzNCU1hZL1gwcVZkN1luelc4aDdPK3pIZ0w4dmRDdjZLOWdsRENycU9QK3pBZjFPWQpsYkdLbmptWmFvMTY2L3MyaEtMTFdReUtoVS9KRmNwYlNHcXlsWTIzMHBpYWVPNndOZzRGekFVMGRPaFhoWXZEClBTclVWRkluMDNPT1U4cnFiWkdRZXdLQmdFMVJPaVZNOTRtUzRTVElJYXptM3NWUFNuNyt1ZU5MZUNnYk1sNU4KeGR3bTlrSnhkL2I5NWtVT0Z0ckVHTVlhNk43d2tkMXRiZmlhekRjRXZ4c05NSjE2b3lQZE5Ia2xEL3Q4TWlyNgozOXIvd1RnK3ZaR2dCTm1SRnJiUGFPdEkwZFpuMWtXaGJXUC84MW4xR0tGS1pDTlZKZ25Mcm80Ly9HaTN2bkl5Cm5OU3JBb0dBVGVidmRLamtENk5XQmJMWSsrVUVQN1ZLd0pOWlR1VWUvT0FBdlJIKzZaWEV4SkhtM3pjV280TVkKMG8vL2dyNzhBdDM4NEk5QVBwMnQwV3lmTmlaTStWUFh4a1lKTU5IU01mcXdGcVRVSmE3NGttNVUrYnB4Mm1ueAovUlR6aElHMDE4SXN3NHBGeUZ4ekpTSVdCK2VpVEF6NFZsMEw2ZU0yNUp5R3lyU2x0Q2M9Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg==
  root-cert.pem: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUMzVENDQWNXZ0F3SUJBZ0lRVGRHUmJPampxWjBEZDUxOVJqdXlzREFOQmdrcWhraUc5dzBCQVFzRkFEQVkKTVJZd0ZBWURWUVFLRXcxamJIVnpkR1Z5TG14dlkyRnNNQjRYRFRJd01ERXlNREUyTkRBek5Gb1hEVE13TURFeApOekUyTkRBek5Gb3dHREVXTUJRR0ExVUVDaE1OWTJ4MWMzUmxjaTVzYjJOaGJEQ0NBU0l3RFFZSktvWklodmNOCkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFMNzhadlRtQ2hxYUM5Z0lFUFlWSWYrVkFsU0tJR2JsdktvUUJNNmwKWlNBTmxNQXg3elJQTjFQdVMrV2I5M1hxMXNzN1hEUEY4UmlIL2dCWE05aGZsNUpGTDErbmlLYWR3RHh5UUdXQQpPMUFBQXNmZlpud3NkWDhDOGdCcE5zUkVZYVo5SzExdDI5NmV5WUc1d3ozMW9rZVFYSTVrSU0vdWgxL2wwN3pKClU3eG8zSmVZbHpMZnJSVWhNRnc1Vk5ETkNCY3JldEoyOWgvZzRpS1plM2JDS3laVmJRUkN3VjR5ck12YTA4Z3kKYzRhSGJud1VtRThKT0JvcE5abW1uOHc0bFcwQjFsS1Q3aFhBRldJdW55WVhIOWFabUJJd1pPVk9kV0N4SmZnTQpKSWY1UVJSY0s5MVZGMjYvcUp2RHlwaVpxcHFJcEdQWHJHbHF2dGtTSmwxdHhYMENBd0VBQWFNak1DRXdEZ1lEClZSMFBBUUgvQkFRREFnSUVNQThHQTFVZEV3RUIvd1FGTUFNQkFmOHdEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUIKQUpjWXl3WkoxeUZpQzRpT0xNbXY4MTZYZEhUSWdRTGlLaXNBdGRqb21TdXhLc0o1eXZ3M2lGdkROSklseEQ4SgoyVVROR2JJTFN2d29qQ1JzQVcyMlJtelpjZG95SXkvcFVIR25EVUpiMk14T0svaEVWU0x4cnN6RHlEK2YwR1liCjdhL1Q2ZmJFbUdYK0JHTnBKZ2lTKytwUm5JMzE3THN6aldtTUlmbVF3T1NtZXNvKzhMSXAxZS9STGVKcThoM0cKREZzcVA4c1BLaHNEM1M1RWNGYU5vSVg4OThVK3UvUWlKd3BoS2lDK3RRRzExeGJZanMxaURNcFJpUGsvSi9NRwpiaTZnQm8zZGdjZ1RWWFdOY2YzeHRiQWErMmkzK3k1V25ydHoyK1d4ZG96cEhpN3FLL1BEbGpwVG5JdkY2Nm0wCjBFYVA0T3ZOY29hNk12MUpoYkFVK0w0PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
kind: Secret
metadata:
  name: istio.test-tlscontext-istio-secret-1
  namespace: secret-namespace
  labels:
    kat-ambassador-id: tlscontextistiosecret
type: istio.io/key-and-cert
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-istio-prefix-1
prefix: /tls-context-istio/
service: https://{self.target.path.fqdn}
tls: {self.name}-istio-context-1
"""
        )
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-istio-context-1
secret: istio.test-tlscontext-istio-secret-1
namespace: secret-namespace
secret_namespacing: False
"""
        )

    def queries(self):
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"), phase=2)

    def check(self):
        assert (
            self.results[0].backend is None
        ), f"expected 0 errors, got {len(self.results[0].backend.response)}: received {self.results[0].backend.response}"


class TLSCoalescing(AmbassadorTest):
    def init(self):
        self.target = HTTP()

        if EDGE_STACK:
            self.xfail = "Not yet supported in Edge Stack"

    def manifests(self) -> str:
        return (
            f"""
---
apiVersion: v1
metadata:
  name: tlscoalescing-certs
  labels:
    kat-ambassador-id: tlscoalescing
data:
  tls.crt: {TLSCerts["*.domain.com"].k8s_crt}
  tls.key: {TLSCerts["*.domain.com"].k8s_key}
kind: Secret
type: kubernetes.io/tls
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-apex
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: domain.com
  tlsSecret:
    name: tlscoalescing-certs
  tlsContext:
    name: tlscoalescing-context
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-a
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: a.domain.com
  tlsSecret:
    name: tlscoalescing-certs
  tlsContext:
    name: tlscoalescing-context
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-b
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: b.domain.com
  tlsSecret:
    name: tlscoalescing-certs
  tlsContext:
    name: tlscoalescing-context
"""
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: tlscoalescing-context
secret: tlscoalescing-certs
alpn_protocols: h2, http/1.1
hosts:
- domain.com
- a.domain.com
- b.domain.com
"""
        )

    def scheme(self) -> str:
        return "https"

    @staticmethod
    def _go_close_connection_error(url):
        """
        :param url: url passed to the query
        :return: error message string that Go's net/http package throws when server closes connection
        """
        return "Get {}: EOF".format(url)

    def queries(self):
        yield Query(
            self.url("ambassador/v0/diag/"),
            headers={"Host": "a.domain.com"},
            insecure=True,
            sni=True,
        )
        yield Query(
            self.url("ambassador/v0/diag/"),
            headers={"Host": "b.domain.com"},
            insecure=True,
            sni=True,
        )

    def requirements(self):
        yield ("url", Query(self.url("ambassador/v0/check_ready"), insecure=True, sni=True))


class TLSInheritFromModule(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        # These are annotations instead of resources because the name matters.
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
    redirect_cleartext_from: 8080
"""
        )

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: a.domain.com
  tlsSecret:
    name: {self.name.k8s}
  tlsContext:
    name: {self.name.k8s}
  requestPolicy:
    insecure:
      action: Redirect
      additionalPort: 8080
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
metadata:
  name: {self.name.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  alpn_protocols: "h2,http/1.1"
  hosts:
  - a.domain.com
  secret: {self.name.k8s}
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["a.domain.com"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["a.domain.com"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.name.k8s}-target-mapping
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /foo
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("foo", scheme="http"), headers={"Host": "a.domain.com"}, expected=301)
        yield Query(
            self.url("bar", scheme="http"),
            headers={"Host": "a.domain.com"},
            expected=(404 if bug_404_routes else 301),
        )
        yield Query(
            self.url("foo", scheme="https"),
            headers={"Host": "a.domain.com"},
            ca_cert=TLSCerts["a.domain.com"].pubcert,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("bar", scheme="https"),
            headers={"Host": "a.domain.com"},
            ca_cert=TLSCerts["a.domain.com"].pubcert,
            sni=True,
            expected=404,
        )

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.headers = {"Host": "a.domain.com"}
            query.sni = (
                True  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
            )
            query.ca_cert = TLSCerts["a.domain.com"].pubcert
            yield (r[0], query)
