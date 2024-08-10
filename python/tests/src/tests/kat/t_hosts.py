from abstract_tests import HTTP, AmbassadorTest, ServiceType
from kat.harness import Query
from tests.integration.manifests import namespace_manifest
from tests.selfsigned import TLSCerts
from tests.utils import create_crl_pem_b64

# STILL TO ADD:
# Host referencing a Secret in another namespace?
# Mappings without host attributes (infer via Host resource)
# Host where a TLSContext with the inferred name already exists

bug_single_insecure_action = False  # Do all Hosts have to have the same insecure.action?
bug_forced_star = (
    True  # Do we erroneously send replies in cleartext instead of TLS for unknown hosts?
)
bug_404_routes = (
    True  # Do we erroneously send 404 responses directly instead of redirect-to-tls first?
)
bug_clientcert_reset = True  # Do we sometimes just close the connection instead of sending back tls certificate_required?


class HostCRDSingle(AmbassadorTest):
    """
    HostCRDSingle: a single Host with a manually-configured TLS. Since the Host is handling the
    TLSContext, we expect to redirect cleartext from 8080 to 8443 here.
    """

    target: ServiceType

    def init(self):
        self.target = HTTP()

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
kind: Host
metadata:
  name: {self.path.k8s}-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}-secret
  mappingSelector:
    matchLabels:
      hostname: {self.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True)
        yield Query(self.url("target/", scheme="http"), expected=301)


class HostCRDNo8080(AmbassadorTest):
    """
    HostCRDNo8080: a single Host with manually-configured TLS that explicitly turns off redirection
    from 8080.
    """

    target: ServiceType

    def init(self):
        self.add_default_http_listener = False
        self.add_default_https_listener = False
        self.target = HTTP()

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
  name: {self.path.k8s}-listener
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 8443
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}-secret
  mappingSelector:
    matchLabels:
      hostname: {self.path.fqdn}
  requestPolicy:
    insecure:
      action: Reject
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True)
        yield Query(self.url("target/", scheme="http"), error=["EOF", "connection refused"])


class HostCRDManualContext(AmbassadorTest):
    """
    A single Host with a manually-specified TLS secret and a manually-specified TLSContext,
    too.
    """

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-manual-secret
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
kind: Host
metadata:
  name: {self.path.k8s}-manual-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      hostname: {self.path.k8s}-manual-hostname
  tlsSecret:
    name: {self.path.k8s}-manual-secret
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
metadata:
  name: {self.path.k8s}-manual-host-context
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hosts:
  - {self.path.fqdn}
  secret: {self.path.k8s}-manual-secret
  min_tls_version: v1.2
  max_tls_version: v1.3
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
  labels:
    hostname: {self.path.k8s}-manual-hostname
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/tls-1.2-1.3"), insecure=True, minTLSv="v1.2", maxTLSv="v1.3")

        yield Query(
            self.url("target/tls-1.0-1.0"),
            insecure=True,
            minTLSv="v1.0",
            maxTLSv="v1.0",
            error=[
                "tls: server selected unsupported protocol version 303",
                "tls: no supported versions satisfy MinVersion and MaxVersion",
                "tls: protocol version not supported",
            ],
        )

        yield Query(self.url("target/cleartext", scheme="http"), expected=301)


class HostCRDManualContextCRL(AmbassadorTest):
    """
    A single Host with a manually-specified TLS secret, a manually-specified TLSContext and
    a manually specified mTLS config with CRL list too.
    """

    target: ServiceType

    def init(self):
        self.add_default_http_listener = False
        self.add_default_https_listener = False

        self.target = HTTP()

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: {self.path.k8s}-listener
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 8443
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: SELF
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-server-manual-crl-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["ambassador.example.com"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["ambassador.example.com"].k8s_key
                + """
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-ca-manual-crl-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["master.datawire.io"].k8s_crt
                + """
  tls.key: ""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-crl-manual-crl-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: Opaque
data:
  crl.pem: """
                + create_crl_pem_b64(
                    TLSCerts["master.datawire.io"].pubcert,
                    TLSCerts["master.datawire.io"].privkey,
                    [TLSCerts["presto.example.com"].pubcert],
                )
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-manual-crl-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: ambassador.example.com
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      hostname: {self.path.k8s}-manual-crl-hostname
  tlsSecret:
    name: {self.path.k8s}-server-manual-crl-secret
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
metadata:
  name: {self.path.k8s}-manual-crl-host-context
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hosts:
  - ambassador.example.com
  ca_secret: {self.path.k8s}-ca-manual-crl-secret
  secret: {self.path.k8s}-server-manual-crl-secret
  cert_required: true
  crl_secret: {self.path.k8s}-crl-manual-crl-secret
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
  labels:
    hostname: {self.path.k8s}-manual-crl-hostname
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: ambassador.example.com
  prefix: /
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        base = {
            "url": self.url(""),
            "ca_cert": TLSCerts["master.datawire.io"].pubcert,
            "headers": {"Host": "ambassador.example.com"},
            "sni": True,  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
        }

        yield Query(**base, error="tls: certificate required")

        yield Query(
            **base,
            client_crt=TLSCerts["presto.example.com"].pubcert,
            client_key=TLSCerts["presto.example.com"].privkey,
            error="tls: revoked certificate",
        )

    def requirements(self):
        yield ("pod", self.path.k8s)


class HostCRDSeparateTLSContext(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

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
kind: Host
metadata:
  name: {self.path.k8s}-manual-host-separate
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      hostname: {self.path.fqdn}
  tlsSecret:
    name: {self.path.k8s}-secret
  tlsContext:
    name: {self.path.k8s}-separate-tls-context
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
metadata:
  name: {self.path.k8s}-separate-tls-context
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: {self.path.k8s}-secret
  min_tls_version: v1.2
  max_tls_version: v1.3
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping-separate
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True, minTLSv="v1.2", maxTLSv="v1.3")

        yield Query(
            self.url("target/"),
            insecure=True,
            minTLSv="v1.0",
            maxTLSv="v1.0",
            error=[
                "tls: server selected unsupported protocol version 303",
                "tls: no supported versions satisfy MinVersion and MaxVersion",
                "tls: protocol version not supported",
            ],
        )


class HostCRDTLSConfig(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

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
kind: Host
metadata:
  name: {self.path.k8s}-manual-host-tls
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      hostname: {self.path.fqdn}
  tlsSecret:
    name: {self.path.k8s}-secret
  tls:
    min_tls_version: v1.2
    max_tls_version: v1.3
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True, minTLSv="v1.2", maxTLSv="v1.3")

        yield Query(
            self.url("target/"),
            insecure=True,
            minTLSv="v1.0",
            maxTLSv="v1.0",
            error=[
                "tls: server selected unsupported protocol version 303",
                "tls: no supported versions satisfy MinVersion and MaxVersion",
                "tls: protocol version not supported",
            ],
        )


class HostCRDClearText(AmbassadorTest):
    """
    A single Host specifying cleartext only. Since it's just cleartext, no redirection comes
    into play.
    """

    target: ServiceType

    def init(self):
        # Only add the default HTTP listener (we're mimicking the no-TLS case here.)
        self.add_default_http_listener = True
        self.add_default_https_listener = False

        self.target = HTTP()

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-cleartext-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      hostname: {self.path.k8s}-host-cleartext
  requestPolicy:
    insecure:
      action: Route
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-cleartext-target-mapping
  labels:
    hostname: {self.path.k8s}-host-cleartext
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "http"

    def queries(self):
        yield Query(self.url("target/"), insecure=True)
        yield Query(self.url("target/", scheme="https"), error=["EOF", "connection refused"])


class HostCRDDouble(AmbassadorTest):
    """
    HostCRDDouble: "double" is actually a misnomer. We have multiple Hosts, each with a
    manually-configured TLS secrets, and varying insecure actions:
    - tls-context-host-1: Route
    - tls-context-host-2: Redirect
    - tls-context-host-3: Reject

    We also have Mappings that specify Host matches, and we test the various combinations.

    XXX In the future, the hostname matches should be unnecessary, as it should use
    metadata.labels.hostname.
    """

    target1: ServiceType
    target2: ServiceType
    target3: ServiceType
    targetshared: ServiceType

    def init(self):
        self.target1 = HTTP(name="target1")
        self.target2 = HTTP(name="target2")
        self.target3 = HTTP(name="target3")
        self.targetshared = HTTP(name="targetshared")

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      host-one: tls-context-host-1
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret-1
  requestPolicy:
    insecure:
      action: Route
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-1"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-1"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-1-mapping
  labels:
    host-one: tls-context-host-1
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  host: "tls-context-host-1"
  prefix: /target-1/
  service: {self.target1.path.fqdn}

---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-2
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-2
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret-2
  requestPolicy:
    insecure:
      action: Redirect
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-2
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-2"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-2"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-2-mapping
  labels:
    hostname: tls-context-host-2
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  host: "tls-context-host-2"
  prefix: /target-2/
  service: {self.target2.path.fqdn}

---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-3
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: ambassador.example.com
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      hostname: ambassador.example.com
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret-3
  requestPolicy:
    insecure:
      action: Reject
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-3
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["ambassador.example.com"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["ambassador.example.com"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-3-mapping
  labels:
    hostname: ambassador.example.com
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  host: "ambassador.example.com"
  prefix: /target-3/
  service: {self.target3.path.fqdn}
---
# Add a bogus ACME mapping so that we can distinguish "invalid
# challenge" from "rejected".
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-3-acme
  labels:
    hostname: ambassador.example.com
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  host: "ambassador.example.com"
  prefix: /.well-known/acme-challenge/
  service: {self.target3.path.fqdn}

---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-shared-mapping
  labels:
    kat-ambassador-id: {self.ambassador_id}
    host-one: tls-context-host-1
    hostname: ambassador.example.com
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  prefix: /target-shared/
  service: {self.targetshared.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        # 0: Get some info from diagd for self.check() to inspect
        yield Query(
            self.url("ambassador/v0/diag/?json=true&filter=errors"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
        )

        # 1-5: Host #1 - TLS
        yield Query(
            self.url("target-1/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-2/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-3/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=404,
        )
        # host-shared-mapping must have the labels required by each Host even though it specifies `hostname: "*"`
        yield Query(
            self.url("target-shared/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url(".well-known/acme-challenge/foo", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=404,
        )
        # 6-10: Host #1 - cleartext (action: Route)
        yield Query(
            self.url("target-1/", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
        )
        yield Query(
            self.url("target-2/", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=404,
        )
        yield Query(
            self.url("target-3/", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=404,
        )
        yield Query(
            self.url("target-shared/", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
        )
        yield Query(
            self.url(".well-known/acme-challenge/foo", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=404,
        )

        # 11-15: Host #2 - TLS
        yield Query(
            self.url("target-1/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-2/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-3/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-shared/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url(".well-known/acme-challenge/foo", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=404,
        )
        # 16-20: Host #2 - cleartext (action: Redirect)
        yield Query(
            self.url("target-1/", scheme="http"),
            headers={"Host": "tls-context-host-2"},
            expected=404,
        )
        yield Query(
            self.url("target-2/", scheme="http"),
            headers={"Host": "tls-context-host-2"},
            expected=301,
        )
        yield Query(
            self.url("target-3/", scheme="http"),
            headers={"Host": "tls-context-host-2"},
            expected=404,
        )
        yield Query(
            self.url("target-shared/", scheme="http"),
            headers={"Host": "tls-context-host-2"},
            expected=301,
        )
        yield Query(
            self.url(".well-known/acme-challenge/foo", scheme="http"),
            headers={"Host": "tls-context-host-2"},
            expected=404,
        )

        # 21-25: Host #3 - TLS
        yield Query(
            self.url("target-1/", scheme="https"),
            headers={"Host": "ambassador.example.com"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-2/", scheme="https"),
            headers={"Host": "ambassador.example.com"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-3/", scheme="https"),
            headers={"Host": "ambassador.example.com"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-shared/", scheme="https"),
            headers={"Host": "ambassador.example.com"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url(".well-known/acme-challenge/foo", scheme="https"),
            headers={"Host": "ambassador.example.com"},
            insecure=True,
            sni=True,
            expected=200,
        )
        # 26-30: Host #3 - cleartext (action: Reject)
        yield Query(
            self.url("target-1/", scheme="http"),
            headers={"Host": "ambassador.example.com"},
            expected=404,
        )
        yield Query(
            self.url("target-2/", scheme="http"),
            headers={"Host": "ambassador.example.com"},
            expected=404,
        )
        yield Query(
            self.url("target-3/", scheme="http"),
            headers={"Host": "ambassador.example.com"},
            expected=404,
        )
        yield Query(
            self.url("target-shared/", scheme="http"),
            headers={"Host": "ambassador.example.com"},
            expected=404,
        )
        yield Query(
            self.url(".well-known/acme-challenge/foo", scheme="http"),
            headers={"Host": "ambassador.example.com"},
            expected=200,
        )

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or []
        num_errors = len(errors)
        assert num_errors == 0, "expected 0 errors, got {} -\n{}".format(num_errors, errors)

        idx = 0

        for result in self.results:
            if result.status == 200 and result.query.headers and result.tls:
                host_header = result.query.headers["Host"]
                tls_common_name = result.tls[0]["Subject"]["CommonName"]

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


class HostCRDLooseLabelSelector(AmbassadorTest):
    """
    Ambassador (2.0-2.3) & (3.0-3.1) consider a match on a single label as a "good enough" match.
    In versions 2.4+ and 3.2+ _ALL_ labels in a selector must be present for it to be considered a match.
    DISABLE_STRICT_LABEL_SELECTORS provides a way to restore the old unintended loose matching behaviour
    in the event that it is desired. The ability to disable strict label matching will be removed in a future version.

    When DISABLE_STRICT_LABEL_SELECTORS is "true", a Host should be associated with a Mapping if any of the labels in the
    mappingSelector matches rather than requiring them all to match. Aditionally, even if the mappingSelector fails to match,
    the Mapping should be associated with the Host if the hostname of the Mapping matches the Hostname of the Host
    """

    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")
        self.manifest_envs += """
    - name: DISABLE_STRICT_LABEL_SELECTORS
      value: "true"
"""

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      host-one: tls-context-host-1
      h: foo
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret-1
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-1"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-1"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-1-mapping
  labels:
    host-one: tls-context-host-1
    h: bar # This does not match the mappingSelector of the host but for loose matching the above label satisfies the requirement
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "tls-context-host-1"
  prefix: /target-1/
  service: {self.target.path.fqdn}

---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-2
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-2
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      host-two: tls-context-host-2
      h: foo # Hosts 1 and 2 share a requirement for this label/val combo
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret-2
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-2
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-2"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-2"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-2-mapping
  labels:
    host-two: tls-context-host-2
    h: foo # Both of these labels match the requirement by the host's selector
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  host: "tls-context-host-2"
  prefix: /target-2/
  service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-shared-label-mapping-all
  labels:
    kat-ambassador-id: {self.ambassador_id}
    host-one: tls-context-host-1
    host-two: tls-context-host-2
    h: foo
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target-shared-labels-all/
  service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-shared-label-mapping-one
  labels:
    kat-ambassador-id: {self.ambassador_id}
    h: foo # This mapping only has this one label that is required by both hosts.
    # With strict matching, neither should be associated, but with loose matching both should.
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target-shared-labels-one/
  service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-shared-hostname
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*" # This Mapping does not have any of the required lables.
  # With strict matching it should not work with either host, with loose matching it should work with both.
  prefix: /target-shared-hostname/
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        # 0: Get some info from diagd for self.check() to inspect
        yield Query(
            self.url("ambassador/v0/diag/?json=true&filter=errors"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
        )

        # 1-5: Host #1
        yield Query(
            self.url("target-1/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-2/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-shared-labels-all/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-shared-labels-one/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-shared-hostname/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=200,
        )

        # 11-15: Host #2
        yield Query(
            self.url("target-1/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-2/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-shared-labels-all/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-shared-labels-one/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-shared-hostname/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=200,
        )

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or []
        num_errors = len(errors)
        assert num_errors == 0, "expected 0 errors, got {} -\n{}".format(num_errors, errors)

        idx = 0

        for result in self.results:
            if result.status == 200 and result.query.headers and result.tls:
                host_header = result.query.headers["Host"]
                tls_common_name = result.tls[0]["Subject"]["CommonName"]

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


class HostCRDStrictLabelSelector(AmbassadorTest):
    """
    Ambassador (2.0-2.3) & (3.0-3.1) consider a match on a single label as a "good enough" match.
    In versions 2.4+ and 3.2+ _ALL_ labels in a selector must be present for it to be considered a match.
    DISABLE_STRICT_LABEL_SELECTORS provides a way to restore the old unintended loose matching behaviour
    in the event that it is desired. The ability to disable strict label matching will be removed in a future version.

    When DISABLE_STRICT_LABEL_SELECTORS is "true", a Host should be associated with a Mapping if any of the labels in the
    mappingSelector matches rather than requiring them all to match. Aditionally, even if the mappingSelector fails to match,
    the Mapping should be associated with the Host if the hostname of the Mapping matches the Hostname of the Host
    """

    target: ServiceType

    def init(self):
        self.target = HTTP(name="target")

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      host-one: tls-context-host-1
      h: foo
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret-1
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-1"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-1"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-1-mapping
  labels:
    host-one: tls-context-host-1
    h: bar # This does not match the mappingSelector of the host but for loose matching the above label satisfies the requirement
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "tls-context-host-1"
  prefix: /target-1/
  service: {self.target.path.fqdn}

---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-2
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-2
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      host-two: tls-context-host-2
      h: foo # Hosts 1 and 2 share a requirement for this label/val combo
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret-2
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-2
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-2"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-2"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-2-mapping
  labels:
    host-two: tls-context-host-2
    h: foo # Both of these labels match the requirement by the host's selector
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  host: "tls-context-host-2"
  prefix: /target-2/
  service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-shared-label-mapping-all
  labels:
    kat-ambassador-id: {self.ambassador_id}
    host-one: tls-context-host-1
    host-two: tls-context-host-2
    h: foo
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target-shared-labels-all/
  service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-shared-label-mapping-one
  labels:
    kat-ambassador-id: {self.ambassador_id}
    h: foo # This mapping only has this one label that is required by both hosts.
    # With strict matching, neither should be associated, but with loose matching both should.
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target-shared-labels-one/
  service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-shared-hostname
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*" # This Mapping does not have any of the required lables.
  # With strict matching it should not work with either host, with loose matching it should work with both.
  prefix: /target-shared-hostname/
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        # 0: Get some info from diagd for self.check() to inspect
        yield Query(
            self.url("ambassador/v0/diag/?json=true&filter=errors"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
        )

        # 1-5: Host #1
        yield Query(
            self.url("target-1/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-2/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-shared-labels-all/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-shared-labels-one/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-shared-hostname/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
            expected=404,
        )

        # 11-15: Host #2
        yield Query(
            self.url("target-1/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-2/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-shared-labels-all/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("target-shared-labels-one/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("target-shared-hostname/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
            expected=404,
        )

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or []
        num_errors = len(errors)
        assert num_errors == 0, "expected 0 errors, got {} -\n{}".format(num_errors, errors)

        idx = 0

        for result in self.results:
            if result.status == 200 and result.query.headers and result.tls:
                host_header = result.query.headers["Host"]
                tls_common_name = result.tls[0]["Subject"]["CommonName"]

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


class HostCRDWildcards(AmbassadorTest):
    """This test could be expanded to include more scenarios, like testing
    handling of precedence between suffix-match host globs and
    prefix-match host-globs.  But this is a solid start.

    """

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-wc
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}-tls
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-wc.domain.com
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*.domain.com"
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}-tls
  requestPolicy:
    insecure:
      action: Route
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-a.domain.com
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "a.domain.com"
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}-tls
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-tls
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
  name: {self.path.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  prefix: /foo/
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def insecure(self, suffix):
        return {
            "url": self.url("foo/%s" % suffix, scheme="http"),
        }

    def secure(self, suffix):
        return {
            "url": self.url("foo/%s" % suffix, scheme="https"),
            "ca_cert": TLSCerts["*.domain.com"].pubcert,
            "sni": True,
        }

    def queries(self):
        yield Query(
            **self.secure("0-200"), headers={"Host": "a.domain.com"}, expected=200
        )  # Host=a.domain.com
        yield Query(
            **self.secure("1-200"), headers={"Host": "wc.domain.com"}, expected=200
        )  # Host=*.domain.com
        yield Query(**self.secure("2-200"), headers={"Host": "127.0.0.1"}, expected=200)  # Host=*

        yield Query(
            **self.insecure("3-301"), headers={"Host": "a.domain.com"}, expected=301
        )  # Host=a.domain.com
        yield Query(
            **self.insecure("4-200"), headers={"Host": "wc.domain.com"}, expected=200
        )  # Host=*.domain.com
        yield Query(**self.insecure("5-301"), headers={"Host": "127.0.0.1"}, expected=301)  # Host=*

    def scheme(self) -> str:
        return "https"

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.headers = {"Host": "127.0.0.1"}
            query.sni = (
                True  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
            )
            query.ca_cert = TLSCerts["*.domain.com"].pubcert
            yield (r[0], query)


class HostCRDClientCertCrossNamespace(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        # All of the things referenced from a Host have a '.' in their
        # name, to make sure that Ambassador is correctly interpreting
        # the '.' as a namespace-separator (or not).  Because most of
        # the references are core.v1.LocalObjectReferences, the '.' is
        # not taken as a namespace-separator, but it is for the
        # tls.ca_secret.  And for ca_secret we still put the '.' in
        # the name so that we check that it's choosing the correct '.'
        # as the separator.
        return (
            namespace_manifest("alt-namespace")
            + self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: ambassador.example.com
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}.server
  tls:
    # ca_secret supports cross-namespace references, so test it
    ca_secret: {self.path.k8s}.ca.alt-namespace
    cert_required: true
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}.ca
  namespace: alt-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["master.datawire.io"].k8s_crt
                + """
  tls.key: ""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}.server
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["ambassador.example.com"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["ambassador.example.com"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  prefix: /
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        base = {
            "url": self.url(""),
            "ca_cert": TLSCerts["master.datawire.io"].pubcert,
            "headers": {"Host": "ambassador.example.com"},
            "sni": True,  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
        }

        yield Query(
            **base,
            client_crt=TLSCerts["presto.example.com"].pubcert,
            client_key=TLSCerts["presto.example.com"].privkey,
        )

        # Check that it requires the client cert.
        #
        # In TLS < 1.3, there's not a dedicated alert code for "the client forgot to include a certificate",
        # so we get a generic alert=40 ("handshake_failure").
        yield Query(**base, maxTLSv="v1.2", error="tls: handshake failure")
        # TLS 1.3 added a dedicated alert=116 ("certificate_required") for that scenario.
        yield Query(
            **base,
            minTLSv="v1.3",
            error=(
                ["tls: certificate required"]
                + (
                    ["write: connection reset by peer", "write: broken pipe"]
                    if bug_clientcert_reset
                    else []
                )
            ),
        )

        # Check that it's validating the client cert against the CA cert.
        yield Query(
            **base,
            client_crt=TLSCerts["localhost"].pubcert,
            client_key=TLSCerts["localhost"].privkey,
            maxTLSv="v1.2",
            error="tls: handshake failure",
        )

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.headers = {"Host": "ambassador.example.com"}
            query.sni = (
                True  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
            )
            query.ca_cert = TLSCerts["master.datawire.io"].pubcert
            query.client_cert = TLSCerts["presto.example.com"].pubcert
            query.client_key = TLSCerts["presto.example.com"].privkey
            yield (r[0], query)


class HostCRDClientCertSameNamespace(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()
        self.add_default_http_listener = False
        self.add_default_https_listener = False

    def manifests(self) -> str:
        # Same as HostCRDClientCertCrossNamespace, all of the things
        # referenced by a Host have a '.' in their name; except
        # (unlike HostCRDClientCertCrossNamespace) the ca_secret
        # doesn't, so that we can check that it chooses the correct
        # namespace when a ".{namespace}" suffix isn't specified.
        return (
            namespace_manifest("alt2-namespace")
            + self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-listener-8443    # This name is to match existing test stuff
  namespace: alt2-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 8443
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: SELF
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
  namespace: alt2-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: ambassador.example.com
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}.server
  tls:
    # ca_secret supports cross-namespace references, so test it
    ca_secret: {self.path.k8s}-ca
    cert_required: true
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-ca
  namespace: alt2-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["master.datawire.io"].k8s_crt
                + """
  tls.key: ""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}.server
  namespace: alt2-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["ambassador.example.com"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["ambassador.example.com"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  prefix: /
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        base = {
            "url": self.url(""),
            "ca_cert": TLSCerts["master.datawire.io"].pubcert,
            "headers": {"Host": "ambassador.example.com"},
            "sni": True,  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
        }

        yield Query(
            **base,
            client_crt=TLSCerts["presto.example.com"].pubcert,
            client_key=TLSCerts["presto.example.com"].privkey,
        )

        # Check that it requires the client cert.
        #
        # In TLS < 1.3, there's not a dedicated alert code for "the client forgot to include a certificate",
        # so we get a generic alert=40 ("handshake_failure").
        yield Query(**base, maxTLSv="v1.2", error="tls: handshake failure")
        # TLS 1.3 added a dedicated alert=116 ("certificate_required") for that scenario.
        yield Query(
            **base,
            minTLSv="v1.3",
            error=(
                ["tls: certificate required"]
                + (
                    ["write: connection reset by peer", "write: broken pipe"]
                    if bug_clientcert_reset
                    else []
                )
            ),
        )

        # Check that it's validating the client cert against the CA cert.
        yield Query(
            **base,
            client_crt=TLSCerts["localhost"].pubcert,
            client_key=TLSCerts["localhost"].privkey,
            maxTLSv="v1.2",
            error="tls: handshake failure",
        )

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.headers = {"Host": "ambassador.example.com"}
            query.sni = (
                True  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
            )
            query.ca_cert = TLSCerts["master.datawire.io"].pubcert
            query.client_cert = TLSCerts["presto.example.com"].pubcert
            query.client_key = TLSCerts["presto.example.com"].privkey
            yield (r[0], query)


class HostCRDClientCertCRLEmptyList(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()
        self.add_default_http_listener = False
        self.add_default_https_listener = False

    def manifests(self) -> str:
        # Similar to HostCRDClientCertSameNamespace, except we also
        # include a Certificate Revocation List in the TLS config
        return (
            namespace_manifest("alt3-namespace")
            + self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-listener-8443    # This name is to match existing test stuff
  namespace: alt3-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 8443
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: SELF
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
  namespace: alt3-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: ambassador.example.com
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}.server
  tls:
    ca_secret: {self.path.k8s}-ca
    cert_required: true
    crl_secret: {self.path.k8s}-crl
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-ca
  namespace: alt3-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["master.datawire.io"].k8s_crt
                + """
  tls.key: ""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-crl
  namespace: alt3-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: Opaque
data:
  crl.pem: """
                + create_crl_pem_b64(
                    TLSCerts["master.datawire.io"].pubcert,
                    TLSCerts["master.datawire.io"].privkey,
                    [],
                )
                + """
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}.server
  namespace: alt3-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["ambassador.example.com"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["ambassador.example.com"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  prefix: /
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        base = {
            "url": self.url(""),
            "ca_cert": TLSCerts["master.datawire.io"].pubcert,
            "headers": {"Host": "ambassador.example.com"},
            "sni": True,  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
        }

        yield Query(**base, error="tls: certificate required")

        yield Query(
            **base,
            client_crt=TLSCerts["presto.example.com"].pubcert,
            client_key=TLSCerts["presto.example.com"].privkey,
        )

    def requirements(self):
        yield ("pod", self.path.k8s)


class HostCRDClientCertCRLRevokeList(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()
        self.add_default_http_listener = False
        self.add_default_https_listener = False

    def manifests(self) -> str:
        # Similar to HostCRDClientCertSameNamespace, except we also
        # include a Certificate Revocation List in the TLS config
        return (
            namespace_manifest("alt4-namespace")
            + self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-listener-8443    # This name is to match existing test stuff
  namespace: alt4-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 8443
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: SELF
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
  namespace: alt4-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: ambassador.example.com
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}.server
  tls:
    ca_secret: {self.path.k8s}-ca
    cert_required: true
    crl_secret: {self.path.k8s}-crl
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-ca
  namespace: alt4-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["master.datawire.io"].k8s_crt
                + """
  tls.key: ""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-crl
  namespace: alt4-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: Opaque
data:
  crl.pem: """
                + create_crl_pem_b64(
                    TLSCerts["master.datawire.io"].pubcert,
                    TLSCerts["master.datawire.io"].privkey,
                    [TLSCerts["presto.example.com"].pubcert],
                )
                + """
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}.server
  namespace: alt4-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["ambassador.example.com"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["ambassador.example.com"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  prefix: /
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        base = {
            "url": self.url(""),
            "ca_cert": TLSCerts["master.datawire.io"].pubcert,
            "headers": {"Host": "ambassador.example.com"},
            "sni": True,  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
        }

        yield Query(**base, error="tls: certificate required")

        yield Query(
            **base,
            client_crt=TLSCerts["presto.example.com"].pubcert,
            client_key=TLSCerts["presto.example.com"].privkey,
            error="tls: revoked certificate",
        )

    def requirements(self):
        yield ("pod", self.path.k8s)


class HostCRDRootRedirectCongratulations(AmbassadorTest):
    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      hostname: tls-context-host-1
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret-1
  requestPolicy:
    insecure:
      action: Redirect
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-1"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-1"].k8s_key
                + """
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(
            self.url("", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=(404 if bug_404_routes else 301),
        )
        yield Query(
            self.url("other", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=(404 if bug_404_routes else 301),
        )

        yield Query(
            self.url("", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("other", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
            sni=True,
            expected=404,
        )

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.headers = {"Host": "tls-context-host-1"}
            query.sni = (
                True  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
            )
            query.ca_cert = TLSCerts["tls-context-host-1"].pubcert
            yield (r[0], query)


class HostCRDRootRedirectSlashMapping(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      hostname: {self.path.fqdn}
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret-1
  requestPolicy:
    insecure:
      action: Redirect
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-1"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-1"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(
            self.url("", scheme="http"), headers={"Host": "tls-context-host-1"}, expected=301
        )
        yield Query(
            self.url("other", scheme="http"), headers={"Host": "tls-context-host-1"}, expected=301
        )

        yield Query(
            self.url("", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("other", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
            sni=True,
            expected=200,
        )

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.headers = {"Host": "tls-context-host-1"}
            query.sni = (
                True  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
            )
            query.ca_cert = TLSCerts["tls-context-host-1"].pubcert
            yield (r[0], query)


class HostCRDRootRedirectRE2Mapping(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      hostname: {self.path.fqdn}
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret-1
  requestPolicy:
    insecure:
      action: Redirect
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-1"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-1"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: "/[[:word:]]*" # :word: is in RE2 but not ECMAScript RegExp or Python 're'
  prefix_regex: true
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(
            self.url("", scheme="http"), headers={"Host": "tls-context-host-1"}, expected=301
        )
        yield Query(
            self.url("other", scheme="http"), headers={"Host": "tls-context-host-1"}, expected=301
        )
        yield Query(
            self.url("-other", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=(404 if bug_404_routes else 301),
        )

        yield Query(
            self.url("", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("other", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("-other", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
            sni=True,
            expected=404,
        )

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.headers = {"Host": "tls-context-host-1"}
            query.sni = (
                True  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
            )
            query.ca_cert = TLSCerts["tls-context-host-1"].pubcert
            yield (r[0], query)


class HostCRDForcedStar(AmbassadorTest):
    """This test verifies that Ambassador responds properly if we try
    talking to it on a hostname that it doesn't recognize.
    """

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        # These manifests are set up to test that we don't accidentally
        # accidentally consider a cleartext hostname="*" to be a TLS
        # hostname="*".
        return (
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-cleartext-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ "{self.ambassador_id}" ]
  hostname: "*"
  acmeProvider:
    authority: none
  requestPolicy:
    insecure:
      action: Redirect
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-tls-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret-1
  requestPolicy:
    insecure:
      action: Route
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-1"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-1"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*"
  prefix: /foo/
  service: {self.target.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        # For each of these, we'll first try it on a recognized hostname ("tls-context-host-1") as a
        # sanity check, and then we'll try the same query for an unrecongized hostname
        # ("nonmatching-host") to make sure that it is handled the same way.

        # 0-1: cleartext 200/301
        yield Query(
            self.url("foo/0-200", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=200,
        )
        yield Query(
            self.url("foo/1-301", scheme="http"),
            headers={"Host": "nonmatching-hostname"},
            expected=301,
        )

        # 2-3: cleartext 404
        yield Query(
            self.url("bar/2-404", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=404,
        )
        yield Query(
            self.url("bar/3-301-or-404", scheme="http"),
            headers={"Host": "nonmatching-hostname"},
            expected=404 if bug_404_routes else 301,
        )

        # 4-5: TLS 200
        yield Query(
            self.url("foo/4-200", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
            sni=True,
            expected=200,
        )
        yield Query(
            self.url("foo/5-200", scheme="https"),
            headers={"Host": "nonmatching-hostname"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
            sni=True,
            insecure=True,
            expected=200,
            error=("http: server gave HTTP response to HTTPS client" if bug_forced_star else None),
        )

        # 6-7: TLS 404
        yield Query(
            self.url("bar/6-404", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
            sni=True,
            expected=404,
        )
        yield Query(
            self.url("bar/7-404", scheme="https"),
            headers={"Host": "nonmatching-hostname"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
            sni=True,
            insecure=True,
            expected=404,
            error=("http: server gave HTTP response to HTTPS client" if bug_forced_star else None),
        )

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.headers = {"Host": "tls-context-host-1"}
            query.sni = (
                True  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
            )
            query.ca_cert = TLSCerts["tls-context-host-1"].pubcert
            yield (r[0], query)


class HostCRDCrossNamespaceNoNamespacing(AmbassadorTest):
    """
    HostCRDCrossNamespaceNoNamespacing tests that the value of tls_secret_namespacing does not interfere
    with the function of being able to specify atlsSecret in another namespace due to the way that the
    implicit TLSContext handles secret names.
    """

    target: ServiceType

    def init(self):
        self.target1 = HTTP(name="target")

    def manifests(self) -> str:
        return (
            namespace_manifest("foobar")
            + self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      hostname: tls-context-host-1
  tlsSecret:
    name: {self.path.k8s}.test.tlscontext.secret
    namespace: foobar
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}.test.tlscontext.secret
  namespace: foobar
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-1"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-1"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-1-mapping
  labels:
    hostname: tls-context-host-1
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  prefix: /target/
  service: {self.target1.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        # Get some info from diagd for self.check() to inspect
        yield Query(
            self.url("ambassador/v0/diag/?json=true&filter=errors"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
        )

        yield Query(
            self.url("target/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            sni=True,
            insecure=True,
            expected=200,
        )
        yield Query(
            self.url("target/", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=301,
        )

    def check(self):
        # XXX If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or []
        num_errors = len(errors)
        assert num_errors == 0, "expected 0 errors, got {} -\n{}".format(num_errors, errors)

        idx = 0

        for result in self.results:
            if result.status == 200 and result.query.headers and result.tls:
                host_header = result.query.headers["Host"]
                tls_common_name = result.tls[0]["Subject"]["CommonName"]

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


class HostCRDDoubleCrossNamespace(AmbassadorTest):
    """
    HostCRDDouble: We have two Hosts, each with a
    manually-configured TLS secret, the secrets have
    the same name but are in different namespaces.
    """

    target1: ServiceType
    target2: ServiceType

    def init(self):
        self.target1 = HTTP(name="target1")
        self.target2 = HTTP(name="target2")

    def manifests(self) -> str:
        return (
            namespace_manifest("bar")
            + namespace_manifest("foo")
            + self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  acmeProvider:
    authority: none
  mappingSelector:
    matchLabels:
      hostname: tls-context-host-1
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret
    namespace: foo
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret
  namespace: foo
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-1"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-1"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-1-mapping
  labels:
    hostname: tls-context-host-1
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-1
  prefix: /target-1/
  service: {self.target1.path.fqdn}

---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-2
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-2
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.path.k8s}-test-tlscontext-secret
    namespace: bar
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret
  namespace: bar
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: """
                + TLSCerts["tls-context-host-2"].k8s_crt
                + """
  tls.key: """
                + TLSCerts["tls-context-host-2"].k8s_key
                + """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: {self.path.k8s}-host-2-mapping
  labels:
    hostname: tls-context-host-2
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: tls-context-host-2
  prefix: /target-2/
  service: {self.target2.path.fqdn}
"""
            )
            + super().manifests()
        )

    def scheme(self) -> str:
        return "https"

    def queries(self):
        # Get some info from diagd for self.check() to inspect
        yield Query(
            self.url("ambassador/v0/diag/?json=true&filter=errors"),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
        )

        # Host #1 - TLS
        yield Query(
            self.url("target-1/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            sni=True,
            insecure=True,
            expected=200,
        )
        yield Query(
            self.url("target-2/", scheme="https"),
            headers={"Host": "tls-context-host-1"},
            sni=True,
            insecure=True,
            expected=404,
        )
        yield Query(
            self.url("target-1/", scheme="http"),
            headers={"Host": "tls-context-host-1"},
            expected=301,
        )

        # Host #2 - TLS
        yield Query(
            self.url("target-1/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            sni=True,
            insecure=True,
            expected=404,
        )
        yield Query(
            self.url("target-2/", scheme="https"),
            headers={"Host": "tls-context-host-2"},
            sni=True,
            insecure=True,
            expected=200,
        )
        yield Query(
            self.url("target-2/", scheme="http"),
            headers={"Host": "tls-context-host-2"},
            expected=301,
        )

    def check(self):
        # XXX If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or []
        num_errors = len(errors)
        assert num_errors == 0, "expected 0 errors, got {} -\n{}".format(num_errors, errors)

        idx = 0

        for result in self.results:
            if result.status == 200 and result.query.headers and result.tls:
                host_header = result.query.headers["Host"]
                tls_common_name = result.tls[0]["Subject"]["CommonName"]

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
