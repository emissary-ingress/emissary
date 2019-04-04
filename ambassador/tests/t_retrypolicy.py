from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, MappingTest, ServiceType


class RetryPolicyTest(MappingTest):
    parent: AmbassadorTest
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP(name="target")

    def manifests(self) -> str:
        s = super().manifests() or ""

        return s + """
---
apiVersion: v1
kind: Service
metadata:
  name: retry
spec:
  selector:
    app: retry
  ports:
  - port: 3000
    name: http
    targetPort: http
  type: NodePort
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: retry
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: retry
    spec:
      containers:
      - name: retry
        image: live/ambassador-retry-service:1.0.0
        imagePullPolicy: Always
        ports:
        - name: http
          containerPort: 3000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
"""

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-target
prefix: /{self.name}/retry/
rewrite: /retry/
service: https://{self.target.path.fqdn}
retry_policy:
  retry_on: "5xx"
  num_retries: 3
  per_try_timeout: "500ms"
""")

    def queries(self):
        yield Query(self.parent.url("%s/retry/" % self.name), expected=200)

# main = Runner(AmbassadorTest)
