from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, ServiceType


class RateLimitTest(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return super().manifests() + """
---
apiVersion: v1
kind: Service
metadata:
  name: rate-limit
spec:
  selector:
    app: rate-limit
  ports:
  - port: 5000
    name: grpc
    targetPort: grpc
  type: ClusterIP
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: rate-limit
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: rate-limit
    spec:
      containers:
      - name: rate-limit
        image: agervais/ambassador-ratelimit-service:1.0.0
        imagePullPolicy: Always
        ports:
        - name: grpc
          containerPort: 5000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
"""

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        # ambassador_id: [ {self.with_tracing.ambassador_id}, {self.no_tracing.ambassador_id} ]
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  tracing_target_mapping
prefix: /target/
service: {self.target.path.k8s}
rate_limits:
- descriptor: A test case
  headers:
  - "x-ambassador-test-allow"
""")

        # For self.with_tracing, we want to configure the TracingService.
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: RateLimitService
name: ratelimit
service: rate-limit:5000
timeout_ms: 500
""")

    def queries(self):
        # Speak through each Ambassador to the traced service...
        # yield Query(self.with_tracing.url("target/"))
        # yield Query(self.no_tracing.url("target/"))

        # No matching headers, won't even go through ratelimit-service filter
        yield Query(self.url("target/"))

        # Header instructing dummy ratelimit-service to allow request
        yield Query(self.url("target/"), expected=200, headers={
            'x-ambassador-test-allow': 'true'
        })

        # Header instructing dummy ratelimit-service to reject request
        yield Query(self.url("target/"), expected=429, headers={
            'x-ambassador-test-allow': 'over my dead body'
        })

# main = Runner(AmbassadorTest)
