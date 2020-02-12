from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, HTTP

SECRETS="""
---
apiVersion: v1
metadata:
  name: {self.path.k8s}-client-cert-secret
data:
  tls.crt: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUR1RENDQXFDZ0F3SUJBZ0lKQUowWDU3ZXlwQk5UTUEwR0NTcUdTSWIzRFFFQkN3VUFNSEV4Q3pBSkJnTlYKQkFZVEFsVlRNUXN3Q1FZRFZRUUlEQUpOUVRFUE1BMEdBMVVFQnd3R1FtOXpkRzl1TVJFd0R3WURWUVFLREFoRQpZWFJoZDJseVpURVVNQklHQTFVRUN3d0xSVzVuYVc1bFpYSnBibWN4R3pBWkJnTlZCQU1NRW0xaGMzUmxjaTVrCllYUmhkMmx5WlM1cGJ6QWVGdzB4T1RBeE1UQXhPVEF6TXpCYUZ3MHlOREF4TURreE9UQXpNekJhTUhFeEN6QUoKQmdOVkJBWVRBbFZUTVFzd0NRWURWUVFJREFKTlFURVBNQTBHQTFVRUJ3d0dRbTl6ZEc5dU1SRXdEd1lEVlFRSwpEQWhFWVhSaGQybHlaVEVVTUJJR0ExVUVDd3dMUlc1bmFXNWxaWEpwYm1jeEd6QVpCZ05WQkFNTUVtMWhjM1JsCmNpNWtZWFJoZDJseVpTNXBiekNDQVNJd0RRWUpLb1pJaHZjTkFRRUJCUUFEZ2dFUEFEQ0NBUW9DZ2dFQkFPdlEKVjVad1NmcmQ1Vndtelo5SmNoOTdyUW40OXA2b1FiNkVIWjF5T2EyZXZBNzE2NWpkMHFqS1BPMlgyRk80MVg4QgpwQWFLZExnMmltaC9wL2NXN2JncjNHNnRHVEZVMVZHanllTE1EV0Q1MGV2TTYydnpYOFRuYVV6ZFRHTjFOdTM2CnJaM2JnK0VLcjhFYjI1b2RabEpyMm1mNktSeDdTcjZzT1N4NlE1VHhSb3NycmZ0d0tjejI5cHZlMGQ4b0NiZGkKRFJPVlZjNXpBaW0zc2Nmd3VwRUJrQzYxdlpKMzhmaXYwRENYOVpna3BMdEZKUTllTEVQSEdKUGp5ZmV3alNTeQovbk52L21Sc2J6aUNtQ3R3Z3BmbFRtODljK3EzSWhvbUE1YXhZQVFjQ0NqOXBvNUhVZHJtSUJKR0xBTVZ5OWJ5CkZnZE50aFdBeHZCNHZmQXl4OXNDQXdFQUFhTlRNRkV3SFFZRFZSME9CQllFRkdUOVAvOHBQeGI3UVJVeFcvV2gKaXpkMnNnbEtNQjhHQTFVZEl3UVlNQmFBRkdUOVAvOHBQeGI3UVJVeFcvV2hpemQyc2dsS01BOEdBMVVkRXdFQgovd1FGTUFNQkFmOHdEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBS3NWT2Fyc01aSXhLOUpLUzBHVHNnRXNjYThqCllhTDg1YmFsbndBbnBxMllSMGNIMlhvd2dLYjNyM3VmbVRCNERzWS9RMGllaENKeTMzOUJyNjVQMVBKMGgvemYKZEZOcnZKNGlvWDVMWnc5YkowQVFORCtZUTBFK010dFppbE9DbHNPOVBCdm1tUEp1dWFlYVdvS2pWZnNOL1RjMAoycUxVM1pVMHo5bmhYeDZlOWJxYUZLSU1jYnFiVk9nS2p3V0ZpbDlkRG4vQ29KbGFUUzRJWjlOaHFjUzhYMXd0ClQybWQvSUtaaEtKc3A3VlBGeDU5ZWhuZ0VPakZocGhzd20xdDhnQWVxL1A3SkhaUXlBUGZYbDNyZDFSQVJuRVIKQUpmVUxET2tzWFNFb2RTZittR0NrVWh1b2QvaDhMTUdXTFh6Q2d0SHBKMndaVHA5a1ZWVWtKdkpqSVU9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K
kind: Secret
type: Opaque
"""

class ConsulTest(AmbassadorTest):

    enable_endpoints = True

    k8s_target: ServiceType
    

    def init(self):
        self.k8s_target = HTTP(name="k8s")

    def manifests(self) -> str:
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
