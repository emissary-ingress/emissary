# import json

from typing import ClassVar

from kat.harness import variants, Query
from abstract_tests import AmbassadorTest, MappingTest, ServiceType, HTTP


class HeaderRoutingTest(MappingTest):
    parent: AmbassadorTest
    target: ServiceType
    target2: ServiceType
    weight: int

    @classmethod
    def variants(cls):
        for v in variants(ServiceType):
            yield cls(v, v.clone("target2"), name="{self.target.name}")

    def init(self, target: ServiceType, target2: ServiceType):
        MappingTest.init(self, target)
        self.target2 = target2

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-target1
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
""")
        yield self.target2, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-target2
prefix: /{self.name}/
service: http://{self.target2.path.fqdn}
headers:
    X-Route: target2
""")

    def queries(self):
        yield Query(self.parent.url(self.name + "/"))
        yield Query(self.parent.url(self.name + "/"), headers={"X-Route": "target2"})

    def check(self):
        assert self.results[0].backend.name == self.target.path.k8s, f"r0 wanted {self.target.path.k8s} got {self.results[0].backend.name}"
        assert self.results[1].backend.name == self.target2.path.k8s, f"r1 wanted {self.target2.path.k8s} got {self.results[1].backend.name}"

class HeaderRoutingAuth(ServiceType):
    skip_variant: ClassVar[bool] = True

    def __init__(self, *args, **kwargs) -> None:
        manifests = """
---
kind: Service
apiVersion: v1
metadata:
  name: {self.path.k8s}
spec:
  selector:
    backend: {self.path.k8s}
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 80
  - name: https
    protocol: TCP
    port: 443
    targetPort: 443
---
apiVersion: v1
kind: Pod
metadata:
  name: {self.path.k8s}
  labels:
    backend: {self.path.k8s}
spec:
  containers:
  - name: backend
    image: {self.test_image[auth]}
    ports:
    - containerPort: 80
    env:
    - name: BACKEND
      value: {self.path.k8s}
"""

        super().__init__(*args, service_manifests=manifests, **kwargs)

    def requirements(self):
        yield ("url", Query("http://%s/ambassador/check/" % self.path.fqdn))

class AuthenticationHeaderRouting(AmbassadorTest):
    target1: ServiceType
    target2: ServiceType
    auth: ServiceType

    def init(self):
        self.target1 = HTTP(name="target1")
        self.target2 = HTTP(name="target2")
        self.auth = HeaderRoutingAuth()

    def config(self):
        # The auth service we're using works like this:
        #
        # prefix ENDS WITH /good/ -> 200, include X-Auth-Route -> we should hit target2
        # prefix ENDS WITH /nohdr/ -> 200, no X-Auth-Route -> we should hit target1
        # anything else -> 403 -> we should see the 403

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
proto: http
path_prefix: ""
timeout_ms: 5000

allowed_authorization_headers:
- X-Auth-Route
- Extauth
""")
        yield self.target1, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-target1
prefix: /target/
service: http://{self.target1.path.fqdn}
""")
        yield self.target2, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-target2
prefix: /target/
service: http://{self.target2.path.fqdn}
headers:
    X-Auth-Route: Route
""")

    def queries(self):
        # [0]
        yield Query(self.url("target/"), expected=403)

        # [1]
        yield Query(self.url("target/good/"), expected=200)

        # [2]
        yield Query(self.url("target/nohdr/"), expected=200)

        # [3]
        yield Query(self.url("target/crap/"), expected=403)

    def check(self):
        # [0] should be a 403 from auth
        assert self.results[0].backend.name == self.auth.path.k8s, f"r0 wanted {self.auth.path.k8s} got {self.results[0].backend.name}"

        # [1] should go to target2
        assert self.results[1].backend.name == self.target2.path.k8s, f"r1 wanted {self.target2.path.k8s} got {self.results[1].backend.name}"

        # [2] should go to target1
        assert self.results[2].backend.name == self.target1.path.k8s, f"r2 wanted {self.target1.path.k8s} got {self.results[2].backend.name}"

        # [3] should be a 403 from auth
        assert self.results[3].backend.name == self.auth.path.k8s, f"r3 wanted {self.auth.path.k8s} got {self.results[3].backend.name}"
