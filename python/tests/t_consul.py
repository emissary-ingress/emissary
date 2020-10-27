from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, HTTP
from selfsigned import TLSCerts

SECRETS="""
---
apiVersion: v1
metadata:
  name: {self.path.k8s}-client-cert-secret
data:
  tls.crt: """+TLSCerts["master.datawire.io"].k8s_crt+"""
kind: Secret
type: Opaque
"""

class ConsulTest(AmbassadorTest):

    enable_endpoints = True

    k8s_target: ServiceType
    

    def init(self):
        self.k8s_target = HTTP(name="k8s")

    def manifests(self) -> str:
        # Unlike usual, super().manifests() must come before our added
        # manifests, because of some magic with ServiceAccounts?
        return super().manifests() + self.format("""
---
apiVersion: v1
kind: Service
metadata:
  name: {self.path.k8s}-consul
spec:
  type: NodePort
  ports:
  - name: consul
    protocol: TCP
    port: 8500
    targetPort: 8500
  selector:
    service: {self.path.k8s}-consul
---
apiVersion: v1
kind: Pod
metadata:
  name: {self.path.k8s}-consul
  annotations:
    sidecar.istio.io/inject: "false"
  labels:
    service: {self.path.k8s}-consul
spec:
  serviceAccountName: {self.path.k8s}
  containers:
  - name: consul
    image: consul:1.4.3
  restartPolicy: Always
---
apiVersion: getambassador.io/v2
kind: ConsulResolver
metadata:
  name: {self.path.k8s}-resolver
spec:
  ambassador_id: consultest
  address: {self.path.k8s}-consul:8500
  datacenter: dc1
""" + SECRETS)

    def config(self):
        yield self.k8s_target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.path.k8s}_k8s_mapping
prefix: /{self.path.k8s}_k8s/
service: {self.k8s_target.path.k8s}
---
apiVersion: getambassador.io/v1
kind: KubernetesServiceResolver
name: kubernetes-service
---
apiVersion: getambassador.io/v1
kind: KubernetesEndpointResolver
name: endpoint
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.path.k8s}_consul_mapping
prefix: /{self.path.k8s}_consul/
service: {self.path.k8s}-consul-service
# tls: {self.path.k8s}-client-context # this doesn't seem to work... ambassador complains with "no private key in secret ..."
resolver: {self.path.k8s}-resolver
load_balancer:
  policy: round_robin
---
apiVersion: ambassador/v1
kind:  TLSContext
name:  {self.path.k8s}-client-context
secret: {self.path.k8s}-client-cert-secret
""")

    def requirements(self):
        yield from super().requirements()
        yield("url", Query(self.format("http://{self.path.k8s}-consul:8500/ui/")))

    def queries(self):
        # The K8s service should be OK. The Consul service should 503 because it has no upstreams
        # in phase 1.
        yield Query(self.url(self.format("{self.path.k8s}_k8s/")), expected=200, phase=1)
        yield Query(self.url(self.format("{self.path.k8s}_consul/")), expected=503, phase=1)

        # Register the Consul service in phase 2.
        yield Query(self.format("http://{self.path.k8s}-consul:8500/v1/catalog/register"),
                    method="PUT",
                    body={
                        "Datacenter": "dc1",
                        "Node": self.format("{self.path.k8s}-consul-service"),
                        "Address": self.k8s_target.path.k8s,
                        "Service": {"Service": self.format("{self.path.k8s}-consul-service"),
                                    "Address": self.k8s_target.path.k8s,
                                    "Port": 80}},
                    phase=2)

        # Both services should work in phase 3.
        yield Query(self.url(self.format("{self.path.k8s}_k8s/")), expected=200, phase=3)
        yield Query(self.url(self.format("{self.path.k8s}_consul/")), expected=200, phase=3)

    def check(self):
        pass
