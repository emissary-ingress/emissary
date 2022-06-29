from typing import Dict, Generator, Literal, Tuple, Union

from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, HTTP, Node
from tests.selfsigned import TLSCerts
from kat.harness import abstract_test
from tests.integration.manifests import namespace_manifest

# An AmbassadorTest subclass will actually create a running Ambassador.
# "self" in this class will refer to the Ambassador.


class TCPMappingTest(AmbassadorTest):
    # single_namespace = True
    namespace = "tcp-namespace"
    extra_ports = [6789, 7654, 8765, 9876]

    # This test is written assuming explicit control of which Hosts are present,
    # so don't let Edge Stack mess with that.
    edge_stack_cleartext_host = False

    # If you set debug = True here, the results of every Query will be printed
    # when the test is run.
    # debug = True

    target1: ServiceType
    target2: ServiceType
    target3: ServiceType

    # init (not __init__) is the method that initializes a KAT Node (including
    # Test, AmbassadorTest, etc.).

    def init(self):
        self.add_default_http_listener = False
        self.add_default_https_listener = False

        self.target1 = HTTP(name="target1")
        # print("TCP target1 %s" % self.target1.namespace)

        self.target2 = HTTP(name="target2", namespace="other-namespace")
        # print("TCP target2 %s" % self.target2.namespace)

        self.target3 = HTTP(name="target3")
        # print("TCP target3 %s" % self.target3.namespace)

    # manifests returns a string of Kubernetes YAML that will be applied to the
    # Kubernetes cluster before running any tests.

    def manifests(self) -> str:
        return (
            namespace_manifest("tcp-namespace")
            + namespace_manifest("other-namespace")
            + f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: supersecret
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["tls-context-host-2"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-2"].k8s_key}
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: {self.path.k8s}-listener
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ "{self.ambassador_id}" ]
  port: 8443
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL
---
# In most real-world cases, we'd just use a single wildcard Host instead
# of using three. For this test, though, we need three because we aren't
# using real domain names, and you can't do wildcards like tls-context-*
# (because the '*' has to be a domain part on its own).
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ "{self.ambassador_id}" ]
  hostname: tls-context-host-1
  tlsContext:
    name: {self.name}-tlscontext
  tlsSecret:
    name: supersecret
  requestPolicy:
    insecure:
      action: Reject
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-2
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ "{self.ambassador_id}" ]
  hostname: tls-context-host-2
  tlsContext:
    name: {self.name}-tlscontext
  tlsSecret:
    name: supersecret
  requestPolicy:
    insecure:
      action: Reject
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: {self.path.k8s}-host-3
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ "{self.ambassador_id}" ]
  hostname: tls-context-host-3
  tlsContext:
    name: {self.name}-tlscontext
  tlsSecret:
    name: supersecret
  requestPolicy:
    insecure:
      action: Reject
"""
            + super().manifests()
        )

    # config() must _yield_ tuples of Node, Ambassador-YAML where the
    # Ambassador-YAML will be annotated onto the Node.

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TLSContext
name: {self.name}-tlscontext
hosts:
- tls-context-host-1
- tls-context-host-2
- tls-context-host-3
secret: supersecret
"""
        )

        yield self.target1, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
name:  {self.name}
port: 9876
service: {self.target1.path.fqdn}:443
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
name:  {self.name}-local-only
address: 127.0.0.1
port: 8765
service: {self.target1.path.fqdn}:443
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
name:  {self.name}-clear-to-tls
port: 7654
service: https://{self.target2.path.fqdn}:443
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
name:  {self.name}-1
port: 6789
host: tls-context-host-1
service: {self.target1.path.fqdn}:80
"""
        )

        # Host-differentiated.
        yield self.target2, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
name:  {self.name}-2
port: 6789
host: tls-context-host-2
service: {self.target2.path.fqdn}
tls: {self.name}-tlscontext
"""
        )

        # Host-differentiated.
        yield self.target3, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
name:  {self.name}-3
port: 6789
host: tls-context-host-3
service: https://{self.target3.path.fqdn}
"""
        )

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

    # scheme defaults to HTTP; if you need to use HTTPS, have it return
    # "https"...
    def scheme(self):
        return "https"

    # Any Query object yielded from queries() will be run as a test. Also,
    # you can add a keyword argument debug=True to any Query() call and the
    # complete response object will be dumped.

    def queries(self):
        # 0: should hit target1, and use TLS
        yield Query(self.url(self.name + "/wtfo/", port=9876), insecure=True)

        # 1: should hit target2, and use TLS
        yield Query(self.url(self.name + "/wtfo/", port=7654, scheme="http"), insecure=True)

        # 2: should hit target1 via SNI, and use cleartext
        yield Query(
            self.url(self.name + "/wtfo/", port=6789),
            headers={"Host": "tls-context-host-1"},
            insecure=True,
            sni=True,
        )

        # 3: should hit target2 via SNI, and use TLS
        yield Query(
            self.url(self.name + "/wtfo/", port=6789),
            headers={"Host": "tls-context-host-2"},
            insecure=True,
            sni=True,
        )

        # 4: should hit target3 via SNI, and use TLS
        yield Query(
            self.url(self.name + "/wtfo/", port=6789),
            headers={"Host": "tls-context-host-3"},
            insecure=True,
            sni=True,
        )

        # 5: should error since port 8765 is bound only to localhost
        yield Query(
            self.url(self.name + "/wtfo/", port=8765),
            error=["connection reset by peer", "EOF", "connection refused"],
            insecure=True,
        )

    # Once in check(), self.results is an ordered list of results from your
    # Queries. (You can also look at self.parent.results if you really want
    # to.)

    def check(self):
        for idx, target, tls_wanted in [
            (0, self.target1, True),
            (1, self.target2, True),
            (2, self.target1, False),
            (3, self.target2, True),
            (4, self.target3, True),
            # ( 5, self.target1 ),
        ]:
            r = self.results[idx]
            wanted_fqdn = target.path.fqdn
            backend_fqdn = target.get_fqdn(r.backend.name)
            tls_enabled = r.backend.request.tls.enabled

            assert (
                backend_fqdn == wanted_fqdn
            ), f"{idx}: backend {backend_fqdn} != expected {wanted_fqdn}"
            assert (
                tls_enabled == tls_wanted
            ), f"{idx}: TLS status {tls_enabled} != wanted {tls_wanted}"


class TCPMappingBasicTest(AmbassadorTest):
    extra_ports = [6789]
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            format(
                """
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  service: {self.target.path.fqdn}:80
"""
            )
            + super().manifests()
        )

    def queries(self):
        yield Query(self.url("", port=6789))

    def check(self):
        assert self.results[0].json["backend"] == self.target.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == False


class TCPMappingCrossNamespaceTest(AmbassadorTest):
    extra_ports = [6789]
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP(namespace="other-namespace")

    def manifests(self) -> str:
        return (
            namespace_manifest("other-namespace")
            + format(
                """
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  service: {self.target.path.fqdn}:80
"""
            )
            + super().manifests()
        )

    def queries(self):
        yield Query(self.url("", port=6789))

    def check(self):
        assert self.results[0].json["backend"] == self.target.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == False


class TCPMappingTLSOriginationBoolTest(AmbassadorTest):
    extra_ports = [6789]
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            format(
                """
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  service: {self.target.path.fqdn}:443
  tls: true
"""
            )
            + super().manifests()
        )

    def queries(self):
        yield Query(self.url("", port=6789))

    def check(self):
        assert self.results[0].json["backend"] == self.target.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == True


class TCPMappingTLSOriginationV2SchemeTest(AmbassadorTest):
    """apiVersion v2 TCPMappings don't support a scheme:// on the 'service' field; if you provide
    one, then it is ignored.  Since apiVersion v3alpha1 adds support for scheme://, add a test to
    make sure we don't break anyone who is inadvertently depending on it being ignored in v2."""

    extra_ports = [6789, 6790]
    target: ServiceType

    def init(self) -> None:
        self.xfail = "bug (2.3): v2 TCPMappings don't ignore the scheme"
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            format(
                """
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}-1
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  service: https://{self.target.path.fqdn}:443
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}-2
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6790
  service: https://{self.target.path.fqdn}:80
"""
            )
            + super().manifests()
        )

    def queries(self):
        yield Query(
            self.url("", port=6789), expected=400
        )  # kat-server returns HTTP 400 "Client sent an HTTP request to an HTTPS server."
        yield Query(self.url("", port=6789, scheme="https"), insecure=True)
        yield Query(self.url("", port=6790))

    def check(self):
        assert self.results[1].json["backend"] == self.target.path.k8s
        assert self.results[1].json["request"]["tls"]["enabled"] == True
        assert self.results[2].json["backend"] == self.target.path.k8s
        assert self.results[2].json["request"]["tls"]["enabled"] == False


class TCPMappingTLSOriginationV3SchemeTest(AmbassadorTest):
    extra_ports = [6789]
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
metadata:
  name: {self.name.k8s}-1
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  service: https://{self.target.path.fqdn}:443
"""
            )
            + super().manifests()
        )

    def queries(self):
        yield Query(self.url("", port=6789))

    def check(self):
        assert self.results[0].json["backend"] == self.target.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == True


class TCPMappingTLSOriginationContextTest(AmbassadorTest):
    extra_ports = [6789]
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP()

    def manifests(self) -> str:
        # Hafta provide a client cert, see https://github.com/emissary-ingress/emissary/issues/4476
        return (
            f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-clientcert
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["presto.example.com"].k8s_crt}
  tls.key: {TLSCerts["presto.example.com"].k8s_key}
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: {self.name.k8s}-tlsclient
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: {self.name.k8s}-clientcert
  sni: my-funny-name
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  service: {self.target.path.fqdn}:443
  tls: {self.name.k8s}-tlsclient
"""
            + super().manifests()
        )

    def queries(self):
        yield Query(self.url("", port=6789))

    def check(self):
        assert self.results[0].json["backend"] == self.target.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == True
        assert self.results[0].json["request"]["tls"]["server-name"] == "my-funny-name"


class TCPMappingTLSOriginationContextWithDotTest(AmbassadorTest):
    extra_ports = [6789]
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP()

    def manifests(self) -> str:
        # Hafta provide a client cert, see https://github.com/emissary-ingress/emissary/issues/4476
        return (
            f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-clientcert
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["presto.example.com"].k8s_crt}
  tls.key: {TLSCerts["presto.example.com"].k8s_key}
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: {self.name.k8s}.tlsclient
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: {self.name.k8s}-clientcert
  sni: my-hilarious-name
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  service: {self.target.path.fqdn}:443
  tls: {self.name.k8s}.tlsclient
"""
            + super().manifests()
        )

    def queries(self):
        yield Query(self.url("", port=6789))

    def check(self):
        assert self.results[0].json["backend"] == self.target.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == True
        assert self.results[0].json["request"]["tls"]["server-name"] == "my-hilarious-name"


class TCPMappingTLSOriginationContextCrossNamespaceTest(AmbassadorTest):
    """This test is a little funny.  You can actually select a TLSContext from any namespace without
    specifying the namespace.  That's bad design, but at the same time we don't want to break anyone
    by changing it."""

    extra_ports = [6789]
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP()

    def manifests(self) -> str:
        # Hafta provide a client cert, see https://github.com/emissary-ingress/emissary/issues/4476
        return (
            namespace_manifest("other-namespace")
            + f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-clientcert
  namespace: other-namespace
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["presto.example.com"].k8s_crt}
  tls.key: {TLSCerts["presto.example.com"].k8s_key}
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: {self.name.k8s}-tlsclient
  namespace: other-namespace
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: {self.name.k8s}-clientcert
  sni: my-hysterical-name
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  service: {self.target.path.fqdn}:443
  tls: {self.name.k8s}-tlsclient
"""
            + super().manifests()
        )

    def queries(self):
        yield Query(self.url("", port=6789))

    def check(self):
        assert self.results[0].json["backend"] == self.target.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == True
        assert self.results[0].json["request"]["tls"]["server-name"] == "my-hysterical-name"


@abstract_test
class TCPMappingTLSTerminationTest(AmbassadorTest):
    tls_src: Literal["tlscontext", "host"]

    @classmethod
    def variants(cls) -> Generator[Node, None, None]:
        for tls_src in ["tlscontext", "host"]:
            yield cls(tls_src, name="{self.tls_src}")

    def init(self, tls_src: Literal["tlscontext", "host"]) -> None:
        self.tls_src = tls_src

    def manifests(self) -> str:
        return (
            f"""
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  requestPolicy:
    insecure:
      action: Route
      additionalPort: 8080
"""
            + super().manifests()
        )


class TCPMappingTLSTerminationBasicTest(TCPMappingTLSTerminationTest):
    extra_ports = [6789]
    target: ServiceType

    def init(self, tls_src: Literal["tlscontext", "host"]) -> None:
        super().init(tls_src)
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-servercert
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["tls-context-host-2"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-2"].k8s_key}
"""
            + (
                f"""
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: {self.name.k8s}-tlsserver
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: {self.name.k8s}-servercert
  hosts: [ "tls-context-host-2" ]
"""
                if self.tls_src == "tlscontext"
                else f"""
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.name.k8s}-tlsserver
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "tls-context-host-2"
  tlsSecret:
    name: {self.name.k8s}-servercert
"""
            )
            + f"""
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  host: tls-context-host-2
  service: {self.target.path.fqdn}:80
"""
            + super().manifests()
        )

    def queries(self):
        yield Query(
            self.url("", scheme="https", port=6789),
            sni=True,
            headers={"Host": "tls-context-host-2"},
            ca_cert=TLSCerts["tls-context-host-2"].pubcert,
        )

    def check(self):
        assert self.results[0].json["backend"] == self.target.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == False


class TCPMappingTLSTerminationCrossNamespaceTest(TCPMappingTLSTerminationTest):
    extra_ports = [6789]
    target: ServiceType

    def init(self, tls_src: Literal["tlscontext", "host"]) -> None:
        super().init(tls_src)
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            namespace_manifest("other-namespace")
            + f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-servercert
  namespace: other-namespace
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["tls-context-host-2"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-2"].k8s_key}
"""
            + (
                f"""
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: {self.name.k8s}-tlsserver
  namespace: other-namespace
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: {self.name.k8s}-servercert
  hosts: [ "tls-context-host-2" ]
"""
                if self.tls_src == "tlscontext"
                else f"""
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.name.k8s}-tlsserver
  namespace: other-namespace
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "tls-context-host-2"
  tlsSecret:
    name: {self.name.k8s}-servercert
"""
            )
            + f"""
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  host: tls-context-host-2
  service: {self.target.path.fqdn}:80
"""
            + super().manifests()
        )

    def queries(self):
        yield Query(
            self.url("", scheme="https", port=6789),
            sni=True,
            headers={"Host": "tls-context-host-2"},
            ca_cert=TLSCerts["tls-context-host-2"].pubcert,
        )

    def check(self):
        assert self.results[0].json["backend"] == self.target.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == False


class TCPMappingSNISharedContextTest(TCPMappingTLSTerminationTest):
    extra_ports = [6789]
    target_a: ServiceType
    target_b: ServiceType

    def init(self, tls_src: Literal["tlscontext", "host"]) -> None:
        super().init(tls_src)
        self.target_a = HTTP(name="target-a")
        self.target_b = HTTP(name="target-b")

    def manifests(self) -> str:
        # Note that TCPMapping.spec.host matches with TLSContext.spec.hosts based on simple string
        # matching, not globbing.  See irbasemapping.py:match_tls_context()
        return (
            f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-servercert
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["*.domain.com"].k8s_crt}
  tls.key: {TLSCerts["*.domain.com"].k8s_key}
"""
            + (
                f"""
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: {self.name.k8s}-tlsserver
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: {self.name.k8s}-servercert
  hosts:
    - "a.domain.com"
    - "b.domain.com"
"""
                if self.tls_src == "tlscontext"
                else f"""
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.name.k8s}-tlsserver
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "*.domain.com"
  tlsSecret:
    name: {self.name.k8s}-servercert
  requestPolicy:
    insecure:
      action: Route
      additionalPort: 8080
"""
            )
            + f"""
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}-a
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  host: a.domain.com
  service: {self.target_a.path.fqdn}:80
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}-b
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  host: b.domain.com
  service: {self.target_b.path.fqdn}:80
"""
            + super().manifests()
        )

    def queries(self):
        yield Query(
            self.url("", scheme="https", port=6789),
            sni=True,
            headers={"Host": "a.domain.com"},
            ca_cert=TLSCerts["*.domain.com"].pubcert,
        )
        yield Query(
            self.url("", scheme="https", port=6789),
            sni=True,
            headers={"Host": "b.domain.com"},
            ca_cert=TLSCerts["*.domain.com"].pubcert,
        )

    def check(self):
        assert self.results[0].json["backend"] == self.target_a.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == False
        assert self.results[1].json["backend"] == self.target_b.path.k8s
        assert self.results[1].json["request"]["tls"]["enabled"] == False


class TCPMappingSNISeparateContextsTest(TCPMappingTLSTerminationTest):
    extra_ports = [6789]
    target_a: ServiceType
    target_b: ServiceType

    def init(self, tls_src: Literal["tlscontext", "host"]) -> None:
        super().init(tls_src)
        self.target_a = HTTP(name="target-a")
        self.target_b = HTTP(name="target-b")

    def manifests(self) -> str:
        return (
            f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-servercert-a
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["tls-context-host-1"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-1"].k8s_key}
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-servercert-b
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["tls-context-host-2"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-2"].k8s_key}
"""
            + (
                f"""
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: {self.name.k8s}-tlsserver-a
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: {self.name.k8s}-servercert-a
  hosts: [tls-context-host-1]
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: {self.name.k8s}-tlsserver-b
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: {self.name.k8s}-servercert-b
  hosts: [tls-context-host-2]
"""
                if self.tls_src == "tlscontext"
                else f"""
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.name.k8s}-tlsserver-a
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "tls-context-host-1"
  tlsSecret:
    name: {self.name.k8s}-servercert-a
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.name.k8s}-tlsserver-b
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "tls-context-host-2"
  tlsSecret:
    name: {self.name.k8s}-servercert-b
"""
            )
            + f"""
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}-a
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  host: tls-context-host-1
  service: {self.target_a.path.fqdn}:80
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}-b
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  host: tls-context-host-2
  service: {self.target_b.path.fqdn}:80
"""
            + super().manifests()
        )

    def queries(self):
        yield Query(
            self.url("", scheme="https", port=6789),
            sni=True,
            headers={"Host": "tls-context-host-1"},
            ca_cert=TLSCerts["tls-context-host-1"].pubcert,
        )
        yield Query(
            self.url("", scheme="https", port=6789),
            sni=True,
            headers={"Host": "tls-context-host-2"},
            ca_cert=TLSCerts["tls-context-host-2"].pubcert,
        )

    def check(self):
        assert self.results[0].json["backend"] == self.target_a.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == False
        assert self.results[1].json["backend"] == self.target_b.path.k8s
        assert self.results[1].json["request"]["tls"]["enabled"] == False


class TCPMappingSNIWithHTTPTest(AmbassadorTest):
    # Note: TCPMappingSNIWithHTTPTest does *not* inherit from TCPMappingTLSTerminationTest because
    # TCPMappingSNIWithHTTPTest wants to take more ownership of the HTTP Host.

    target: ServiceType

    tls_src: Literal["tlscontext", "host"]

    @classmethod
    def variants(cls) -> Generator[Node, None, None]:
        for tls_src in ["tlscontext", "host"]:
            yield cls(tls_src, name="{self.tls_src}")

    def init(self, tls_src: Literal["tlscontext", "host"]) -> None:
        self.tls_src = tls_src
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            f"""
# HTTP Host ##########################################################
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["tls-context-host-1"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-1"].k8s_key}
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.path.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.name.k8s}
# TCPMapping #########################################################
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-servercert
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["tls-context-host-2"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-2"].k8s_key}
"""
            + (
                f"""
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: {self.name.k8s}-tlsserver
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: {self.name.k8s}-servercert
  hosts: [ "tls-context-host-2" ]
"""
                if self.tls_src == "tlscontext"
                else f"""
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.name.k8s}-tlsserver
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: "tls-context-host-2"
  tlsSecret:
    name: {self.name.k8s}-servercert
"""
            )
            + f"""
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 8443
  host: tls-context-host-2
  service: {self.target.path.fqdn}:80
"""
            + super().manifests()
        )

    def scheme(self):
        return "https"

    def queries(self):
        yield Query(
            self.url(""),
            sni=True,
            headers={"Host": "tls-context-host-2"},
            ca_cert=TLSCerts["tls-context-host-2"].pubcert,
        )

    def check(self):
        assert self.results[0].json["backend"] == self.target.path.k8s
        assert self.results[0].json["request"]["tls"]["enabled"] == False


class TCPMappingAddressTest(AmbassadorTest):
    extra_ports = [6789, 6790]
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP()

    def manifests(self) -> str:
        return (
            format(
                """
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}-local-only
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  address: 127.0.0.1
  service: {self.target.path.fqdn}:80
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}-proxy
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6790
  service: localhost:6789
"""
            )
            + super().manifests()
        )

    def queries(self):
        # Check that it only bound to localhost and doesn't allow external connections.
        yield Query(
            self.url("", port=6789), error=["connection reset by peer", "EOF", "connection refused"]
        )
        # Use a second mapping that proxies to the first to check that it was even created.
        yield Query(self.url("", port=6790))

    def check(self):
        assert self.results[1].json["backend"] == self.target.path.k8s
        assert self.results[1].json["request"]["tls"]["enabled"] == False


class TCPMappingWeightTest(AmbassadorTest):
    extra_ports = [6789]
    target70: ServiceType
    target30: ServiceType

    def init(self) -> None:
        self.target70 = HTTP(name="tgt70")
        self.target30 = HTTP(name="tgt30")

    def manifests(self) -> str:
        return (
            format(
                """
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}-70
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  service: {self.target70.path.fqdn}:80
  weight: 70
---
apiVersion: getambassador.io/v2
kind: TCPMapping
metadata:
  name: {self.name.k8s}-30
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 6789
  service: {self.target30.path.fqdn}:80
  weight: 30
"""
            )
            + super().manifests()
        )

    def queries(self):
        for i in range(1000):
            yield Query(self.url("", port=6789))

    def check(self):
        counts: Dict[str, int] = {}
        for result in self.results:
            backend = result.json["backend"]
            counts[backend] = counts.get(backend, 0) + 1
        assert counts[self.target70.path.k8s] + counts[self.target30.path.k8s] == 1000
        # Probabalistic, margin might need tuned
        margin = 150
        assert abs(counts[self.target70.path.k8s] - 700) < margin
        assert abs(counts[self.target30.path.k8s] - 300) < margin


# TODO: Add tests for all of the config knobs for the upstream connection:
#  - enable_ipv4: false
#  - enable_ipv6: false
#  - circuit_breakers
#  - idle_timeout_ms
#  - resolver
#
# TODO: Add tests for the config knobs for stats:
#  - cluster_tag
#  - stats_name (v3alpha1 only)
