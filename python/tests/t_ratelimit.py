from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, ServiceType
from selfsigned import TLSCerts

class RateLimitV0Test(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: rate-limit-v0
spec:
  selector:
    app: rate-limit-v0
  ports:
  - port: 5000
    name: grpc
    targetPort: grpc
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rate-limit-v0
spec:
  selector:
    matchLabels:
      app: rate-limit-v0
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: rate-limit-v0
    spec:
      containers:
      - name: rate-limit
        image: {self.test_image[ratelimit]}
        ports:
        - name: grpc
          containerPort: 5000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
""" + super().manifests()

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        # ambassador_id: [ {self.with_tracing.ambassador_id}, {self.no_tracing.ambassador_id} ]
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  ratelimit_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
rate_limits:
- descriptor: A test case
  headers:
  - "x-ambassador-test-allow"
---
apiVersion: ambassador/v1
kind:  Mapping
name:  ratelimit_label_mapping
prefix: /labels/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - host_and_user:
      - custom-label:
          header: ":authority"
          omit_if_not_present: true
      - user:
          header: "x-user"
          omit_if_not_present: true

    - omg_header:
      - custom-label:
          header: "x-omg"
          default: "OMFG!"
""")

        # For self.with_tracing, we want to configure the TracingService.
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: RateLimitService
name: ratelimit-v0
service: rate-limit-v0:5000
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

class RateLimitV1Test(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: rate-limit-v1
spec:
  selector:
    app: rate-limit-v1
  ports:
  - port: 5000
    name: grpc
    targetPort: grpc
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rate-limit-v1
spec:
  selector:
    matchLabels:
      app: rate-limit-v1
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: rate-limit-v1
    spec:
      containers:
      - name: rate-limit
        image: {self.test_image[ratelimit]}
        ports:
        - name: grpc
          containerPort: 5000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
""" + super().manifests()

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  ratelimit_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - request_label_group:
      - x-ambassador-test-allow:
          header: "x-ambassador-test-allow"
          omit_if_not_present: true
""")

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: RateLimitService
name: ratelimit-v1
service: rate-limit-v1:5000
timeout_ms: 500
""")

    def queries(self):
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

class RateLimitV1WithTLSTest(AmbassadorTest):
    # debug = True
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: rate-limit-tls
spec:
  selector:
    app: rate-limit-tls
  ports:
  - port: 5000
    name: grpc
    targetPort: grpc
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rate-limit-tls
spec:
  selector:
    matchLabels:
      app: rate-limit-tls
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: rate-limit-tls
    spec:
      containers:
      - name: rate-limit
        image: {self.test_image[ratelimit]}
        env:
        - name: "USE_TLS"
          value: "true"
        ports:
        - name: grpc
          containerPort: 5000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
""" + f"""
---
apiVersion: v1
data:
  tls.crt: {TLSCerts["ratelimit.datawire.io"].k8s_crt}
  tls.key: {TLSCerts["ratelimit.datawire.io"].k8s_key}
kind: Secret
metadata:
  name: ratelimit-tls-secret
type: kubernetes.io/tls
""" + super().manifests()

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind: TLSContext
name: ratelimit-tls-context
secret: ratelimit-tls-secret
alpn_protocols: h2
---
apiVersion: ambassador/v1
kind:  Mapping
name:  ratelimit_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
labels:
  ambassador:
    - request_label_group:
      - x-ambassador-test-allow:
          header: "x-ambassador-test-allow"
          omit_if_not_present: true
""")

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: RateLimitService
name: ratelimit-tls
service: rate-limit-tls:5000
timeout_ms: 500
tls: ratelimit-tls-context
""")

    def queries(self):
        # No matching headers, won't even go through ratelimit-service filter
        yield Query(self.url("target/"))

        # Header instructing dummy ratelimit-service to allow request
        yield Query(self.url("target/"), expected=200, headers={
            'x-ambassador-test-allow': 'true'
        })

        # Header instructing dummy ratelimit-service to reject request
        yield Query(self.url("target/"), expected=429, headers={
            'x-ambassador-test-allow': 'nope'
        })
