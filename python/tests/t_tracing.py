import json
import pytest

from typing import ClassVar, Dict, List, Sequence, Tuple, Union

from kat.harness import sanitize, variants, Query, Runner

from abstract_tests import AmbassadorTest, HTTP, AHTTP
from abstract_tests import MappingTest, OptionTest, ServiceType, Node, Test


class TracingTest(AmbassadorTest):
    def init(self):
        self.target = HTTP()
        # self.with_tracing = AmbassadorTest(name="ambassador-with-tracing")
        # self.no_tracing = AmbassadorTest(name="ambassador-no-tracing")

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: zipkin
spec:
  selector:
    app: zipkin
  ports:
  - port: 9411
    name: http
    targetPort: http
  type: NodePort
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zipkin
spec:
  selector:
    matchLabels:
      app: zipkin
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: zipkin
    spec:
      containers:
      - name: zipkin
        image: openzipkin/zipkin:2.17
        ports:
        - name: http
          containerPort: 9411
""" + super().manifests()

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
service: {self.target.path.fqdn}
""")

        # For self.with_tracing, we want to configure the TracingService.
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: TracingService
name: tracing
service: zipkin:9411
driver: zipkin
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query("http://zipkin:9411/api/v2/services"))

    def queries(self):
        # Speak through each Ambassador to the traced service...
        # yield Query(self.with_tracing.url("target/"))
        # yield Query(self.no_tracing.url("target/"))

        for i in range(100):
              yield Query(self.url("target/"), phase=1)


        # ...then ask the Zipkin for services and spans. Including debug=True in these queries
        # is particularly helpful.
        yield Query("http://zipkin:9411/api/v2/services", phase=2)
        yield Query("http://zipkin:9411/api/v2/spans?serviceName=tracingtest-default", phase=2)
        yield Query("http://zipkin:9411/api/v2/traces?serviceName=tracingtest-default", phase=2)

    def check(self):
        for i in range(100):
            assert self.results[i].backend.name == self.target.path.k8s

        assert self.results[100].backend.name == "raw"
        assert len(self.results[100].backend.response) == 1
        assert self.results[100].backend.response[0] == 'tracingtest-default'

        assert self.results[101].backend.name == "raw"

        tracelist = { x: True for x in self.results[101].backend.response }

        assert 'router cluster_tracingtest_http_default egress' in tracelist

        # Look for the host that we actually queried, since that's what appears in the spans.
        assert self.results[0].backend.request.host in tracelist

        # Ensure we generate 128-bit traceids by default
        trace = self.results[102].json[0][0]
        traceId = trace['traceId']
        assert len(traceId) == 32

class TracingTestShortTraceId(AmbassadorTest):
    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: zipkin-64
spec:
  selector:
    app: zipkin-64
  ports:
  - port: 9411
    name: http
    targetPort: http
  type: NodePort
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zipkin-64
spec:
  selector:
    matchLabels:
      app: zipkin-64
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: zipkin-64
    spec:
      containers:
      - name: zipkin
        image: openzipkin/zipkin:2.17
        ports:
        - name: http
          containerPort: 9411
""" + super().manifests()

    def config(self):
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        # ambassador_id: [ {self.with_tracing.ambassador_id}, {self.no_tracing.ambassador_id} ]
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  tracing_target_mapping_64
prefix: /target-64/
service: {self.target.path.fqdn}
""")

        # For self.with_tracing, we want to configure the TracingService.
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: TracingService
name: tracing-64
service: zipkin-64:9411
driver: zipkin
config:
  trace_id_128bit: false
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query("http://zipkin-64:9411/api/v2/services"))

    def queries(self):
        # Speak through each Ambassador to the traced service...
        yield Query(self.url("target-64/"), phase=1)
        # ...then ask the Zipkin for services and spans. Including debug=True in these queries
        # is particularly helpful.
        yield Query("http://zipkin-64:9411/api/v2/traces", phase=2)

    def check(self):
        # Ensure we generated 64-bit traceids
        trace = self.results[1].json[0][0]
        traceId = trace['traceId']
        assert len(traceId) == 16

# This test asserts that the external authorization server receives the proper tracing
# headers when Ambassador is configured with an HTTP AuthService.
class TracingExternalAuthTest(AmbassadorTest):
    
    def init(self):
        self.target = HTTP()
        self.auth = AHTTP(name="auth")
        
    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: zipkin-auth
spec:
  selector:
    app: zipkin-auth
  ports:
  - port: 9411
    name: http
    targetPort: http
  type: NodePort
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zipkin-auth
spec:
  selector:
    matchLabels:
      app: zipkin-auth
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: zipkin-auth
    spec:
      containers:
      - name: zipkin-auth
        image: openzipkin/zipkin:2.17
        ports:
        - name: http
          containerPort: 9411
""" + super().manifests()

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  tracing_target_mapping
prefix: /target/
service: {self.target.path.fqdn}
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: TracingService
name: tracing-auth
service: zipkin-auth:9411
driver: zipkin
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"
allowed_headers:
- Requested-Status
- Requested-Header
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query("http://zipkin-auth:9411/api/v2/services"))

    def queries(self):
        yield Query(self.url("target/"), headers={"Requested-Status": "200"}, expected=200)

    def check(self):
        extauth_res = json.loads(self.results[0].headers["Extauth"][0])
        request_headers = self.results[0].backend.request.headers

        assert self.results[0].status == 200
        assert self.results[0].headers["Server"] == ["envoy"]
        assert extauth_res["request"]["headers"]["x-b3-parentspanid"] == request_headers["x-b3-parentspanid"]
        assert extauth_res["request"]["headers"]["x-b3-sampled"] == request_headers["x-b3-sampled"]
        assert extauth_res["request"]["headers"]["x-b3-spanid"] == request_headers["x-b3-spanid"]
        assert extauth_res["request"]["headers"]["x-b3-traceid"] == request_headers["x-b3-traceid"]
        assert extauth_res["request"]["headers"]["x-request-id"] == request_headers["x-request-id"]
