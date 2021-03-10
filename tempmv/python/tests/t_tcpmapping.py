import json

from kat.harness import Query, Test, variants

from abstract_tests import AmbassadorTest, ServiceType, HTTP
from selfsigned import TLSCerts
from kat.utils import namespace_manifest

# An AmbassadorTest subclass will actually create a running Ambassador.
# "self" in this class will refer to the Ambassador.

class TCPMappingTest(AmbassadorTest):
    # single_namespace = True
    namespace = "tcp-namespace"
    extra_ports = [ 6789, 7654, 8765, 9876 ]

    # If you set debug = True here, the results of every Query will be printed
    # when the test is run.
    # debug = True

    target1: ServiceType
    target2: ServiceType
    target3: ServiceType

    # init (not __init__) is the method that initializes a KAT Node (including
    # Test, AmbassadorTest, etc.).

    def init(self):
        self.target1 = HTTP(name="target1")
        # print("TCP target1 %s" % self.target1.namespace)

        self.target2 = HTTP(name="target2", namespace="other-namespace")
        # print("TCP target2 %s" % self.target2.namespace)

        self.target3 = HTTP(name="target3")
        # print("TCP target3 %s" % self.target3.namespace)

    # manifests returns a string of Kubernetes YAML that will be applied to the
    # Kubernetes cluster before running any tests.

    def manifests(self) -> str:
        return namespace_manifest("tcp-namespace") + namespace_manifest("other-namespace") + f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: supersecret
type: kubernetes.io/tls
data:
  tls.crt: {TLSCerts["tls-context-host-2"].k8s_crt}
  tls.key: {TLSCerts["tls-context-host-2"].k8s_key}
""" + super().manifests()

    # config() must _yield_ tuples of Node, Ambassador-YAML where the
    # Ambassador-YAML will be annotated onto the Node.

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: TLSContext
name: {self.name}-tlscontext
hosts:
- tls-context-host-1
- tls-context-host-2
- tls-context-host-3
secret: supersecret
""")

        yield self.target1, self.format("""
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}
port: 9876
service: {self.target1.path.fqdn}:443
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}-local-only
address: 127.0.0.1
port: 8765
service: {self.target1.path.fqdn}:443
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}-clear-to-tls
port: 7654
tls: true
service: {self.target2.path.fqdn}:443
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}-1
port: 6789
host: tls-context-host-1
service: {self.target1.path.fqdn}:80
""")

        # Host-differentiated.
        yield self.target2, self.format("""
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}-2
port: 6789
host: tls-context-host-2
service: {self.target2.path.fqdn}
tls: {self.name}-tlscontext
""")

        # Host-differentiated.
        yield self.target3, self.format("""
---
apiVersion: ambassador/v1
kind:  TCPMapping
name:  {self.name}-3
port: 6789
host: tls-context-host-3
service: {self.target3.path.fqdn}
tls: true
""")

    def requirements(self):
        # We're replacing super()'s requirements deliberately here. Without a Host header they can't work.
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))

    # scheme defaults to HTTP; if you need to use HTTPS, have it return
    # "https"...
    def scheme(self):
        return "https"

    # Any Query object yielded from queries() will be run as a test. Also,
    # you can add a keyword argument debug=True to any Query() call and the
    # complete response object will be dumped.

    def queries(self):
        # 0: should hit target1, and use TLS
        yield Query(self.url(self.name + "/wtfo/", port=9876),
                    insecure=True)

        # 1: should hit target2, and use TLS
        yield Query(self.url(self.name + "/wtfo/", port=7654, scheme='http'),
                    insecure=True)

        # 2: should hit target1 via SNI, and use cleartext
        yield Query(self.url(self.name + "/wtfo/", port=6789),
                    headers={"Host": "tls-context-host-1"},
                    insecure=True,
                    sni=True)

        # 3: should hit target2 via SNI, and use TLS
        yield Query(self.url(self.name + "/wtfo/", port=6789),
                    headers={"Host": "tls-context-host-2"},
                    insecure=True,
                    sni=True)

        # 4: should hit target3 via SNI, and use TLS
        yield Query(self.url(self.name + "/wtfo/", port=6789),
                    headers={"Host": "tls-context-host-3"},
                    insecure=True,
                    sni=True)

        # 5: should error since port 8765 is bound only to localhost
        yield Query(self.url(self.name + "/wtfo/", port=8765),
                    error=[ 'connection reset by peer', 'EOF', 'connection refused' ],
                    insecure=True)

    # Once in check(), self.results is an ordered list of results from your
    # Queries. (You can also look at self.parent.results if you really want
    # to.)

    def check(self):
        for idx, target, tls_wanted in [
            ( 0, self.target1, True ),
            ( 1, self.target2, True ),
            ( 2, self.target1, False ),
            ( 3, self.target2, True ),
            ( 4, self.target3, True ),
            # ( 5, self.target1 ),
        ]:
            r = self.results[idx]
            wanted_fqdn = target.path.fqdn
            backend_fqdn = target.get_fqdn(r.backend.name)
            tls_enabled = r.backend.request.tls.enabled

            assert backend_fqdn == wanted_fqdn, f'{idx}: backend {backend_fqdn} != expected {wanted_fqdn}'
            assert tls_enabled == tls_wanted, f'{idx}: TLS status {tls_enabled} != wanted {tls_wanted}'
