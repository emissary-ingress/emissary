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
    k8s_target: ServiceType
    k8s_ns_target: ServiceType

    def init(self):
        self.k8s_target = HTTP(name="k8s")
        self.k8s_ns_target = HTTP(name="k8s-ns", namespace="consul-test-namespace")

        # This is the datacenter we'll use.
        self.datacenter = "dc12"

        # We use Consul's local-config environment variable to set the datacenter name
        # on the actual Consul pod. That means that we need to supply the datacenter
        # name in JSON format.
        #
        # In a perfect world this would just be
        #
        # self.datacenter_dict = { "datacenter": self.datacenter }
        #
        # but the world is not perfect, so we have to supply it as JSON with LOTS of
        # escaping, since this gets passed through self.format (hence two layers of
        # doubled braces) and JSON decoding (hence backslash-escaped double quotes,
        # and of course the backslashes themselves have to be escaped...)
        self.datacenter_json = f'{{{{\\\"datacenter\\\":\\\"{self.datacenter}\\\"}}}}'

    def manifests(self) -> str:
        consul_manifest = self.format("""
---
apiVersion: v1
kind: Service
metadata:
  name: {self.path.k8s}-consul
spec:
  type: ClusterIP
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
    env:
    - name: CONSUL_LOCAL_CONFIG
      value: "{self.datacenter_json}"
  restartPolicy: Always
""")

        # Unlike usual, we have stuff both before and after super().manifests():
        # we want the namespace early, but we want the superclass before our other
        # manifests, because of some magic with ServiceAccounts?
        return self.format("""
---
apiVersion: v1
kind: Namespace
metadata:
  name: consul-test-namespace
""") + super().manifests() + consul_manifest + self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: ConsulResolver
metadata:
  name: {self.path.k8s}-resolver
spec:
  ambassador_id: [consultest]
  address: {self.path.k8s}-consul:$CONSUL_WATCHER_PORT
  datacenter: {self.datacenter}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name:  {self.path.k8s}-consul-ns-mapping
  namespace: consul-test-namespace
spec:
  ambassador_id: [consultest]
  hostname: "*"
  prefix: /{self.path.k8s}_consul_ns/
  service: {self.path.k8s}-consul-ns-service
  resolver: {self.path.k8s}-resolver
  load_balancer:
    policy: round_robin
""" + SECRETS)

    def config(self):
        yield self.k8s_target, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.path.k8s}_k8s_mapping
hostname: "*"
prefix: /{self.path.k8s}_k8s/
service: {self.k8s_target.path.k8s}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.path.k8s}_consul_mapping
hostname: "*"
prefix: /{self.path.k8s}_consul/
service: {self.path.k8s}-consul-service
# tls: {self.path.k8s}-client-context # this doesn't seem to work... ambassador complains with "no private key in secret ..."
resolver: {self.path.k8s}-resolver
load_balancer:
  policy: round_robin
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.path.k8s}_consul_node_mapping
hostname: "*"
prefix: /{self.path.k8s}_consul_node/ # this is testing that Ambassador correctly falls back to the `Address` if `Service.Address` does not exist
service: {self.path.k8s}-consul-node
# tls: {self.path.k8s}-client-context # this doesn't seem to work... ambassador complains with "no private key in secret ..."
resolver: {self.path.k8s}-resolver
load_balancer:
  policy: round_robin
---
kind:  TLSContext
name:  {self.path.k8s}-client-context
secret: {self.path.k8s}-client-cert-secret
---
apiVersion: getambassador.io/v3alpha1
kind: Host
name:  {self.path.k8s}-client-host
requestPolicy:
  insecure:
    action: Route
""")

    def requirements(self):
        yield from super().requirements()
        yield("url", Query(self.format("http://{self.path.k8s}-consul:8500/ui/")))

    def queries(self):
        # Deregister the Consul services in phase 0.
        yield Query(self.format("http://{self.path.k8s}-consul:8500/v1/catalog/deregister"),
                    method="PUT",
                    body={
                        "Datacenter": self.datacenter,
                        "Node": self.format("{self.path.k8s}-consul-service")
                    },
                    phase=0)
        yield Query(self.format("http://{self.path.k8s}-consul:8500/v1/catalog/deregister"),
                    method="PUT",
                    body={
                        "Datacenter": self.datacenter,
                        "Node": self.format("{self.path.k8s}-consul-ns-service")
                    },
                    phase=0)
        yield Query(self.format("http://{self.path.k8s}-consul:8500/v1/catalog/deregister"),
                    method="PUT",
                    body={
                        "Datacenter": self.datacenter,
                        "Node": self.format("{self.path.k8s}-consul-node")
                    },
                    phase=0)

        # The K8s service should be OK. The Consul services should 503 since they have no upstreams
        # in phase 1.
        yield Query(self.url(self.format("{self.path.k8s}_k8s/")), expected=200, phase=1)
        yield Query(self.url(self.format("{self.path.k8s}_consul/")), expected=503, phase=1)
        yield Query(self.url(self.format("{self.path.k8s}_consul_ns/")), expected=503, phase=1)
        yield Query(self.url(self.format("{self.path.k8s}_consul_node/")), expected=503, phase=1)

        # Register the Consul services in phase 2.
        yield Query(self.format("http://{self.path.k8s}-consul:8500/v1/catalog/register"),
                    method="PUT",
                    body={
                        "Datacenter": self.datacenter,
                        "Node": self.format("{self.path.k8s}-consul-service"),
                        "Address": self.k8s_target.path.k8s,
                        "Service": {"Service": self.format("{self.path.k8s}-consul-service"),
                                    "Address": self.k8s_target.path.k8s,
                                    "Port": 80}},
                    phase=2)
        yield Query(self.format("http://{self.path.k8s}-consul:8500/v1/catalog/register"),
                    method="PUT",
                    body={
                        "Datacenter": self.datacenter,
                        "Node": self.format("{self.path.k8s}-consul-ns-service"),
                        "Address": self.format("{self.k8s_ns_target.path.k8s}.consul-test-namespace"),
                        "Service": {"Service": self.format("{self.path.k8s}-consul-ns-service"),
                                    "Address": self.format("{self.k8s_ns_target.path.k8s}.consul-test-namespace"),
                                    "Port": 80}},
                    phase=2)
        yield Query(self.format("http://{self.path.k8s}-consul:8500/v1/catalog/register"),
                    method="PUT",
                    body={
                        "Datacenter": self.datacenter,
                        "Node": self.format("{self.path.k8s}-consul-node"),
                        "Address": self.k8s_target.path.k8s,
                        "Service": {"Service": self.format("{self.path.k8s}-consul-node"),
                                    "Port": 80}},
                    phase=2)

        # All services should work in phase 3.
        yield Query(self.url(self.format("{self.path.k8s}_k8s/")), expected=200, phase=3)
        yield Query(self.url(self.format("{self.path.k8s}_consul/")), expected=200, phase=3)
        yield Query(self.url(self.format("{self.path.k8s}_consul_ns/")), expected=200, phase=3)
        yield Query(self.url(self.format("{self.path.k8s}_consul_node/")), expected=200, phase=3)

    def check(self):
        pass
