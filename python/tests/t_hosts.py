from kat.harness import Query, EDGE_STACK
from kat.utils import namespace_manifest

from abstract_tests import AmbassadorTest, ServiceType, HTTP
from selfsigned import TLSCerts

# STILL TO ADD:
# Host referencing a Secret in another namespace?
# Mappings without host attributes (infer via Host resource)
# Host where a TLSContext with the inferred name already exists

class HostCRDSingle(AmbassadorTest):
    """
    HostCRDSingle: a single Host with a manually-configured TLS. Since the Host is handling the
    TLSContext, we expect both OSS and Edge Stack to redirect cleartext from 8080 to 8443 here.
    """
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: '''+TLSCerts["localhost"].k8s_crt+'''
  tls.key: '''+TLSCerts["localhost"].k8s_key+'''
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.name.k8s}-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.name.k8s}-secret
  selector:
    matchLabels:
      hostname: {self.path.fqdn}
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.name.k8s}-target-mapping
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') +  super().manifests()

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
        self.edge_stack_cleartext_host = False
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: '''+TLSCerts["localhost"].k8s_crt+'''
  tls.key: '''+TLSCerts["localhost"].k8s_key+'''
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.name.k8s}-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}
  acmeProvider:
    authority: none
  tlsSecret:
    name: {self.name.k8s}-secret
  selector:
    matchLabels:
      hostname: {self.path.fqdn}
  requestPolicy:
    insecure:
      additionalPort: -1
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.name.k8s}-target-mapping
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True)

        if EDGE_STACK:
            yield Query(self.url("target/", scheme="http"), expected=404)
        else:
            yield Query(self.url("target/", scheme="http"), error=[ "EOF", "connection refused" ])


class HostCRDManualContext(AmbassadorTest):
    """
    A single Host with a manually-specified TLS secret and a manually-specified TLSContext,
    too. Since the Host is _not_ handling the TLSContext, we do _not_ expect automatic redirection
    on port 8080.
    """
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False

        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}-manual-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: '''+TLSCerts["localhost"].k8s_crt+'''
  tls.key: '''+TLSCerts["localhost"].k8s_key+'''
---
apiVersion: getambassador.io/v2
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
  selector:
    matchLabels:
      hostname: {self.path.k8s}-manual-hostname
  tlsSecret:
    name: {self.path.k8s}-manual-secret
---
apiVersion: getambassador.io/v2
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
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
  labels:
    hostname: {self.path.k8s}-manual-hostname
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.2", maxTLSv="v1.3")

        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.0",  maxTLSv="v1.0",
                    error=["tls: server selected unsupported protocol version 303",
                           "tls: no supported versions satisfy MinVersion and MaxVersion",
                           "tls: protocol version not supported"])

        if EDGE_STACK:
            yield Query(self.url("target/", scheme="http"), expected=404)
        else:
            yield Query(self.url("target/", scheme="http"), error=[ "EOF", "connection refused" ])


class HostCRDSeparateTLSContext(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: '''+TLSCerts["localhost"].k8s_crt+'''
  tls.key: '''+TLSCerts["localhost"].k8s_key+'''
---
apiVersion: getambassador.io/v2
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
  selector:
    matchLabels:
      hostname: {self.path.fqdn}
  tlsSecret:
    name: {self.name.k8s}-secret
  tlsContext:
    name: {self.path.k8s}-separate-tls-context
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: {self.path.k8s}-separate-tls-context
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  secret: {self.name.k8s}-secret
  min_tls_version: v1.2
  max_tls_version: v1.3
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping-separate
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.2", maxTLSv="v1.3")

        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.0",  maxTLSv="v1.0",
                    error=["tls: server selected unsupported protocol version 303",
                           "tls: no supported versions satisfy MinVersion and MaxVersion",
                           "tls: protocol version not supported"])


class HostCRDTLSConfig(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.name.k8s}-secret
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: '''+TLSCerts["localhost"].k8s_crt+'''
  tls.key: '''+TLSCerts["localhost"].k8s_key+'''
---
apiVersion: getambassador.io/v2
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
  selector:
    matchLabels:
      hostname: {self.path.fqdn}
  tlsSecret:
    name: {self.name.k8s}-secret
  tls:
    min_tls_version: v1.2
    max_tls_version: v1.3
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.path.k8s}-target-mapping
  labels:
    hostname: {self.path.fqdn}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.2", maxTLSv="v1.3")

        yield Query(self.url("target/"), insecure=True,
                    minTLSv="v1.0",  maxTLSv="v1.0",
                    error=["tls: server selected unsupported protocol version 303",
                           "tls: no supported versions satisfy MinVersion and MaxVersion",
                           "tls: protocol version not supported"])


class HostCRDClearText(AmbassadorTest):
    """
    A single Host specifying cleartext only. Since it's just cleartext, no redirection comes
    into play.
    """
    target: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: getambassador.io/v2
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
  selector:
    matchLabels:
      hostname: {self.path.k8s}-host-cleartext
  requestPolicy:
    insecure: 
      action: Route
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.path.k8s}-cleartext-target-mapping
  labels:
    hostname: {self.path.k8s}-host-cleartext
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target/
  service: {self.target.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "http"

    def queries(self):
        yield Query(self.url("target/"), insecure=True)
        yield Query(self.url("target/", scheme="https"),
                    error=[ "EOF", "connection refused" ])


class HostCRDDouble(AmbassadorTest):
    """
    HostCRDDouble: two Hosts with manually-configured TLS secrets, and Mappings specifying host matches.
    Since the Hosts are handling TLSContexts, we expect both OSS and Edge Stack to redirect cleartext
    from 8080 to 8443 here.

    XXX In the future, the hostname matches should be unnecessary, as it should use
    metadata.labels.hostname.
    """
    target1: ServiceType
    target2: ServiceType
    target3: ServiceType
    targetshared: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.target1 = HTTP(name="target1")
        self.target2 = HTTP(name="target2")
        self.target3 = HTTP(name="target3")
        self.targetshared = HTTP(name="targetshared")

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: getambassador.io/v2
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
  selector:
    matchLabels:
      hostname: tls-context-host-1
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
  tls.crt: '''+TLSCerts["tls-context-host-1"].k8s_crt+'''
  tls.key: '''+TLSCerts["tls-context-host-1"].k8s_key+'''
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.path.k8s}-host-1-mapping
  labels:
    hostname: tls-context-host-1
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  host: "tls-context-host-1"
  prefix: /target-1/
  service: {self.target1.path.fqdn}

---
apiVersion: getambassador.io/v2
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
  selector:
    matchLabels:
      hostname: tls-context-host-2
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
  tls.crt: '''+TLSCerts["tls-context-host-2"].k8s_crt+'''
  tls.key: '''+TLSCerts["tls-context-host-2"].k8s_key+'''
---
apiVersion: getambassador.io/v2
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
apiVersion: getambassador.io/v2
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
  selector:
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
  tls.crt: '''+TLSCerts["ambassador.example.com"].k8s_crt+'''
  tls.key: '''+TLSCerts["ambassador.example.com"].k8s_key+'''
---
apiVersion: getambassador.io/v2
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
apiVersion: getambassador.io/v2
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
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.path.k8s}-host-shared-mapping
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /target-shared/
  service: {self.targetshared.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        # 0: Get some info from diagd for self.check() to inspect
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"),
                    headers={"Host": "tls-context-host-1" },
                    insecure=True,
                    sni=True)

        # 1-5: Host #1 - TLS
        yield Query(self.url("target-1/", scheme="https"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True,
                    expected=200)
        yield Query(self.url("target-2/", scheme="https"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True,
                    expected=404)
        yield Query(self.url("target-3/", scheme="https"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True,
                    expected=404)
        yield Query(self.url("target-shared/", scheme="https"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True,
                    expected=200)
        yield Query(self.url(".well-known/acme-challenge/foo", scheme="https"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True,
                    expected=404)
        # 6-10: Host #1 - cleartext (action: Route)
        yield Query(self.url("target-1/", scheme="http"), headers={"Host": "tls-context-host-1"},
                    expected=200)
        yield Query(self.url("target-2/", scheme="http"), headers={"Host": "tls-context-host-1"},
                    expected=404)
        yield Query(self.url("target-3/", scheme="http"), headers={"Host": "tls-context-host-1"},
                    expected=404)
        yield Query(self.url("target-shared/", scheme="http"), headers={"Host": "tls-context-host-1"},
                    expected=200)
        yield Query(self.url(".well-known/acme-challenge/foo", scheme="http"), headers={"Host": "tls-context-host-1"},
                    expected=404)

        # 11-15: Host #2 - TLS
        yield Query(self.url("target-1/", scheme="https"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True,
                    expected=404)
        yield Query(self.url("target-2/", scheme="https"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True,
                    expected=200)
        yield Query(self.url("target-3/", scheme="https"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True,
                    expected=404)
        yield Query(self.url("target-shared/", scheme="https"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True,
                    expected=200)
        yield Query(self.url(".well-known/acme-challenge/foo", scheme="https"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True,
                    expected=404)
        # 16-20: Host #2 - cleartext (action: Redirect)
        yield Query(self.url("target-1/", scheme="http"), headers={"Host": "tls-context-host-2"},
                    expected=301)
        yield Query(self.url("target-2/", scheme="http"), headers={"Host": "tls-context-host-2"},
                    expected=301)
        yield Query(self.url("target-3/", scheme="http"), headers={"Host": "tls-context-host-2"},
                    expected=301)
        yield Query(self.url("target-shared/", scheme="http"), headers={"Host": "tls-context-host-2"},
                    expected=301)
        yield Query(self.url(".well-known/acme-challenge/foo", scheme="http"), headers={"Host": "tls-context-host-2"},
                    expected=404)

        # 21-25: Host #3 - TLS
        yield Query(self.url("target-1/", scheme="https"), headers={"Host": "ambassador.example.com"}, insecure=True, sni=True,
                    expected=404)
        yield Query(self.url("target-2/", scheme="https"), headers={"Host": "ambassador.example.com"}, insecure=True, sni=True,
                    expected=404)
        yield Query(self.url("target-3/", scheme="https"), headers={"Host": "ambassador.example.com"}, insecure=True, sni=True,
                    expected=200)
        yield Query(self.url("target-shared/", scheme="https"), headers={"Host": "ambassador.example.com"}, insecure=True, sni=True,
                    expected=200)
        yield Query(self.url(".well-known/acme-challenge/foo", scheme="https"), headers={"Host": "ambassador.example.com"}, insecure=True, sni=True,
                    expected=200)
        # 26-30: Host #3 - cleartext (action: Reject)
        yield Query(self.url("target-1/", scheme="http"), headers={"Host": "ambassador.example.com"},
                    expected=404)
        yield Query(self.url("target-2/", scheme="http"), headers={"Host": "ambassador.example.com"},
                    expected=404)
        yield Query(self.url("target-3/", scheme="http"), headers={"Host": "ambassador.example.com"},
                    expected=404)
        yield Query(self.url("target-shared/", scheme="http"), headers={"Host": "ambassador.example.com"},
                    expected=404)
        yield Query(self.url(".well-known/acme-challenge/foo", scheme="http"), headers={"Host": "ambassador.example.com"},
                    expected=200)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or []
        num_errors = len(errors)
        assert num_errors == 0, "expected 0 errors, got {} -\n{}".format(num_errors, errors)

        idx = 0

        for result in self.results:
            if result.status == 200 and result.query.headers and result.tls:
                host_header = result.query.headers['Host']
                tls_common_name = result.tls[0]['Subject']['CommonName']

                assert host_header == tls_common_name, "test %d wanted CN %s, but got %s" % (idx, host_header, tls_common_name)

            idx += 1

    def requirements(self):
        # We're replacing super()'s requirements deliberately here. Without a Host header they can't work.
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))


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
        return namespace_manifest("alt-namespace") + self.format('''
---
apiVersion: getambassador.io/v2
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
  tls.crt: '''+TLSCerts["master.datawire.io"].k8s_crt+'''
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
  tls.crt: '''+TLSCerts["ambassador.example.com"].k8s_crt+'''
  tls.key: '''+TLSCerts["ambassador.example.com"].k8s_key+'''
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.path.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /
  service: {self.target.path.fqdn}
''') +  super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        base = {
            'url': self.url(""),
            'ca_cert': TLSCerts["master.datawire.io"].pubcert,
            'headers': {"Host": "ambassador.example.com"},
            'sni': True,  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
        }

        yield Query(**base,
                    client_crt=TLSCerts["presto.example.com"].pubcert,
                    client_key=TLSCerts["presto.example.com"].privkey)

        # Check that it requires the client cert.
        #
        # In TLS < 1.3, there's not a dedicated alert code for "the client forgot to include a certificate",
        # so we get a generic alert=40 ("handshake_failure").
        yield Query(**base, maxTLSv="v1.2", error="tls: handshake failure")
        # TLS 1.3 added a dedicated alert=116 ("certificate_required") for that scenario.
        yield Query(**base, minTLSv="v1.3", error="tls: certificate required")

        # Check that it's validating the client cert against the CA cert.
        yield Query(**base,
                    client_crt=TLSCerts["localhost"].pubcert,
                    client_key=TLSCerts["localhost"].privkey,
                    maxTLSv="v1.2", error="tls: handshake failure")

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.headers={"Host": "ambassador.example.com"}
            query.sni = True  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
            query.ca_cert = TLSCerts["master.datawire.io"].pubcert
            query.client_cert = TLSCerts["presto.example.com"].pubcert
            query.client_key = TLSCerts["presto.example.com"].privkey
            yield (r[0], query)

class HostCRDClientCertSameNamespace(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        # Same as HostCRDClientCertCrossNamespace, all of the things
        # referenced by a Host have a '.' in their name; except
        # (unlike HostCRDClientCertCrossNamespace) the ca_secret
        # doesn't, so that we can check that it chooses the correct
        # namespace when a ".{namespace}" suffix isn't specified.
        return namespace_manifest("alt-namespace") + self.format('''
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.path.k8s}
  namespace: alt-namespace
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
  namespace: alt-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: '''+TLSCerts["master.datawire.io"].k8s_crt+'''
  tls.key: ""
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}.server
  namespace: alt-namespace
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
data:
  tls.crt: '''+TLSCerts["ambassador.example.com"].k8s_crt+'''
  tls.key: '''+TLSCerts["ambassador.example.com"].k8s_key+'''
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: {self.path.k8s}
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  prefix: /
  service: {self.target.path.fqdn}
''') +  super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        base = {
            'url': self.url(""),
            'ca_cert': TLSCerts["master.datawire.io"].pubcert,
            'headers': {"Host": "ambassador.example.com"},
            'sni': True,  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
        }

        yield Query(**base,
                    client_crt=TLSCerts["presto.example.com"].pubcert,
                    client_key=TLSCerts["presto.example.com"].privkey)

        # Check that it requires the client cert.
        #
        # In TLS < 1.3, there's not a dedicated alert code for "the client forgot to include a certificate",
        # so we get a generic alert=40 ("handshake_failure").
        yield Query(**base, maxTLSv="v1.2", error="tls: handshake failure")
        # TLS 1.3 added a dedicated alert=116 ("certificate_required") for that scenario.
        yield Query(**base, minTLSv="v1.3", error="tls: certificate required")

        # Check that it's validating the client cert against the CA cert.
        yield Query(**base,
                    client_crt=TLSCerts["localhost"].pubcert,
                    client_key=TLSCerts["localhost"].privkey,
                    maxTLSv="v1.2", error="tls: handshake failure")

    def requirements(self):
        for r in super().requirements():
            query = r[1]
            query.headers={"Host": "ambassador.example.com"}
            query.sni = True  # Use query.headers["Host"] instead of urlparse(query.url).hostname for SNI
            query.ca_cert = TLSCerts["master.datawire.io"].pubcert
            query.client_cert = TLSCerts["presto.example.com"].pubcert
            query.client_key = TLSCerts["presto.example.com"].privkey
            yield (r[0], query)
