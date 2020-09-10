from kat.harness import Query, EDGE_STACK

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


class HostCRDMultipleInsecure(AmbassadorTest):
    """
    Multiple hosts specifying insecure behavior.
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
  name: {self.path.k8s}-insecure-host-a
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}-a
  acmeProvider:
    authority: none
  selector:
    matchLabels:
      hostname: {self.path.fqdn}-a
  requestPolicy:
    insecure:
      action: Route
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.path.k8s}-insecure-host-b
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}-b
  acmeProvider:
    authority: none
  selector:
    matchLabels:
      hostname: {self.path.fqdn}-b
  requestPolicy:
    insecure:
      action: Redirect
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.path.k8s}-insecure-host-c
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}-c
  acmeProvider:
    authority: none
  selector:
    matchLabels:
      hostname: {self.path.fqdn}-c
  requestPolicy:
    insecure:
      action: Route
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: {self.path.k8s}-insecure-host-d
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  hostname: {self.path.fqdn}-d
  acmeProvider:
    authority: none
  selector:
    matchLabels:
      hostname: {self.path.fqdn}-d
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
        # 404 for a request that does not match any host. No forced-star behavior here.
        yield Query(self.url("target/"),
                    insecure=True)

        yield Query(self.url("target/", scheme="https"),
                    error=[ "EOF", "connection refused" ])

        yield Query(self.url("target/", scheme="http"),
                    headers={"Host": self.path.k8s + "-a"},
                    expected=200,
                    insecure=True)
        yield Query(self.url("target/", scheme="http"),
                    headers={"Host": self.path.k8s + "-b"},
                    expected=301,
                    insecure=True)
        yield Query(self.url("target/", scheme="http"),
                    headers={"Host": self.path.k8s + "-c"},
                    expected=200,
                    insecure=True)
        yield Query(self.url("target/", scheme="http"),
                    headers={"Host": self.path.k8s + "-d"},
                    expected=301,
                    insecure=True)


class HostCRDDouble(AmbassadorTest):
    """
    HostCRDDouble: two Hosts with manually-configured TLS secrets, and Mappings specifying host matches.
    Since the Hosts are handling TLSContexts, we expect both OSS and Edge Stack to redirect cleartext
    from 8080 to 8443 here.

    XXX In the future, the hostname matches should be unnecessary.
    """
    target1: ServiceType
    target2: ServiceType

    def init(self):
        self.edge_stack_cleartext_host = False
        self.target1 = HTTP(name="target1")
        self.target2 = HTTP(name="target2")

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: v1
data:
  tls.crt: '''+TLSCerts["localhost"].k8s_crt+'''
  tls.key: '''+TLSCerts["localhost"].k8s_key+'''
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-0
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
---
apiVersion: v1
data:
  tls.crt: '''+TLSCerts["tls-context-host-1"].k8s_crt+'''
  tls.key: '''+TLSCerts["tls-context-host-1"].k8s_key+'''
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-1
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
---
apiVersion: v1
data:
  tls.crt: '''+TLSCerts["tls-context-host-2"].k8s_crt+'''
  tls.key: '''+TLSCerts["tls-context-host-2"].k8s_key+'''
kind: Secret
metadata:
  name: {self.path.k8s}-test-tlscontext-secret-2
  labels:
    kat-ambassador-id: {self.ambassador_id}
type: kubernetes.io/tls
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
  prefix: /target/
  service: {self.target1.path.fqdn}
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
  prefix: /target/
  service: {self.target2.path.fqdn}
''') + super().manifests()

    def scheme(self) -> str:
        return "https"

    def queries(self):
        # 0
        yield Query(self.url("ambassador/v0/diag/?json=true&filter=errors"),
                    headers={"Host": "tls-context-host-1" },
                    insecure=True,
                    sni=True)

        # 1 - Correct host #1
        yield Query(self.url("target/"),
                    headers={"Host": "tls-context-host-1" },
                    expected=200,
                    insecure=True,
                    sni=True)
        # 2 - Correct host #2
        yield Query(self.url("target/"),
                    headers={"Host": "tls-context-host-2" },
                    expected=200,
                    insecure=True,
                    sni=True)

        # Setting the Host header really shouldn't be necessary here.
        yield Query(self.url("target/", scheme="http"),
                    headers={ "Host": "tls-context-host-1" },
                    expected=301)
        yield Query(self.url("target/", scheme="http"),
                    headers={ "Host": "tls-context-host-2" },
                    expected=301)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or []
        num_errors = len(errors)
        assert num_errors == 0, "expected 0 errors, got {} -\n{}".format(num_errors, errors)

        idx = 0

        for result in self.results:
            if result.status == 200 and result.query.headers:
                host_header = result.query.headers['Host']
                tls_common_name = result.tls[0]['Issuer']['CommonName']

                assert host_header == tls_common_name, "test %d wanted CN %s, but got %s" % (idx, host_header, tls_common_name)

            idx += 1

    def requirements(self):
        # We're replacing super()'s requirements deliberately here. Without a Host header they can't work.
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-1"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_ready"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))
        yield ("url", Query(self.url("ambassador/v0/check_alive"), headers={"Host": "tls-context-host-2"}, insecure=True, sni=True))

