from typing import Generator, Tuple, Union

import json

from kat.harness import Query, EDGE_STACK

from abstract_tests import AmbassadorTest, HTTP, AHTTP, Node

from ambassador import Config

# The phase that we should wait until before performing test checks. Normally
# this would be phase 2, which is 10 seconds after the first wave of queries,
# but we increase it to phase 3 here to make sure that Zipkin and other tracers
# have _plenty_ of time to receive traces from Envoy and index them for retrieval
# through the API. We've seen this test flake when the check is performed in phase
# 2, so the hope is that phase 3 reduces the likelihood of the test flaking again.
check_phase = 3

class TracingTest(AmbassadorTest):
    def init(self):
        self.target = HTTP()

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
  type: ClusterIP
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

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.

        yield self.target, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tracing_target_mapping
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
""")

        # Configure the TracingService.
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
name: tracing
service: zipkin:9411
driver: zipkin
tag_headers:
  - "x-watsup"
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query("http://zipkin:9411/api/v2/services"))

    def queries(self):
        # Speak through each Ambassador to the traced service...

        for i in range(100):
            yield Query(self.url("target/"), headers={'x-watsup':'nothin'}, phase=1)


        # ...then ask the Zipkin for services and spans. Including debug=True in these queries
        # is particularly helpful.
        yield Query("http://zipkin:9411/api/v2/services", phase=check_phase)
        yield Query("http://zipkin:9411/api/v2/spans?serviceName=tracingtest-default", phase=check_phase)
        yield Query("http://zipkin:9411/api/v2/traces?serviceName=tracingtest-default", phase=check_phase)

        # The diagnostics page should load properly
        yield Query(self.url("ambassador/v0/diag/"), phase=check_phase)

    def check(self):
        for i in range(100):
            assert self.results[i].backend.name == self.target.path.k8s

        print(f"self.results[100] = {self.results[100]}")
        assert self.results[100].backend is not None and self.results[100].backend.name == "raw", \
                f"unexpected self.results[100] = {self.results[100]}"
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
        for t in self.results[102].json[0]:
            if t.get('tags', {}).get('node_id') == 'test-id':
                assert 'x-watsup' in t['tags']
                assert t['tags']['x-watsup'] == 'nothin'


class TracingTestLongClusterName(AmbassadorTest):
    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: zipkinservicenamewithoversixtycharacterstoforcenamecompression
spec:
  selector:
    app: zipkin-longclustername
  ports:
  - port: 9411
    name: http
    targetPort: http
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zipkin-longclustername
spec:
  selector:
    matchLabels:
      app: zipkin-longclustername
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: zipkin-longclustername
    spec:
      containers:
      - name: zipkin
        image: openzipkin/zipkin:2.17
        ports:
        - name: http
          containerPort: 9411
""" + super().manifests()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.

        yield self.target, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tracing_target_mapping_longclustername
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
""")

        # Configure the TracingService.
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
name: tracing-longclustername
service: zipkinservicenamewithoversixtycharacterstoforcenamecompression:9411
driver: zipkin
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query("http://zipkinservicenamewithoversixtycharacterstoforcenamecompression:9411/api/v2/services"))

    def queries(self):
        # Speak through each Ambassador to the traced service...

        for i in range(100):
              yield Query(self.url("target/"), phase=1)


        # ...then ask the Zipkin for services and spans. Including debug=True in these queries
        # is particularly helpful.
        yield Query("http://zipkinservicenamewithoversixtycharacterstoforcenamecompression:9411/api/v2/services", phase=check_phase)
        yield Query("http://zipkinservicenamewithoversixtycharacterstoforcenamecompression:9411/api/v2/spans?serviceName=tracingtestlongclustername-default", phase=check_phase)
        yield Query("http://zipkinservicenamewithoversixtycharacterstoforcenamecompression:9411/api/v2/traces?serviceName=tracingtestlongclustername-default", phase=check_phase)

        # The diagnostics page should load properly, even though our Tracing Service
        # has a long cluster name https://github.com/datawire/ambassador/issues/3021
        yield Query(self.url("ambassador/v0/diag/"), phase=check_phase)

    def check(self):
        for i in range(100):
            assert self.results[i].backend.name == self.target.path.k8s

        print(f"self.results[100] = {self.results[100]}")
        assert self.results[100].backend is not None and self.results[100].backend.name == "raw", \
                f"unexpected self.results[100] = {self.results[100]}"
        assert len(self.results[100].backend.response) == 1
        assert self.results[100].backend.response[0] == 'tracingtestlongclustername-default'

        assert self.results[101].backend.name == "raw"

        tracelist = { x: True for x in self.results[101].backend.response }

        assert 'router cluster_tracingtestlongclustername_http_default egress' in tracelist

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
  type: ClusterIP
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

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.

        yield self.target, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tracing_target_mapping_64
hostname: "*"
prefix: /target-64/
service: {self.target.path.fqdn}
""")

        # Configure the TracingService.
        yield self, """
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
name: tracing-64
service: zipkin-64:9411
driver: zipkin
config:
  trace_id_128bit: false
"""

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query("http://zipkin-64:9411/api/v2/services"))

    def queries(self):
        # Speak through each Ambassador to the traced service...
        yield Query(self.url("target-64/"), phase=1)

        # ...then ask the Zipkin for services and spans. Including debug=True in these queries
        # is particularly helpful.
        yield Query("http://zipkin-64:9411/api/v2/traces", phase=check_phase)

        # The diagnostics page should load properly
        yield Query(self.url("ambassador/v0/diag/"), phase=check_phase)

    def check(self):
        # Ensure we generated 64-bit traceids
        trace = self.results[1].json[0][0]
        traceId = trace['traceId']
        assert len(traceId) == 16

# This test asserts that the external authorization server receives the proper tracing
# headers when Ambassador is configured with an HTTP AuthService.
class TracingExternalAuthTest(AmbassadorTest):

    def init(self):
        if EDGE_STACK:
            self.xfail = "XFailing for now, custom AuthServices not supported in Edge Stack"
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
  type: ClusterIP
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

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self.target, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tracing_target_mapping
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
""")

        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
name: tracing-auth
service: zipkin-auth:9411
driver: zipkin
""")

        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: AuthService
name:  {self.auth.path.k8s}
auth_service: "{self.auth.path.fqdn}"
path_prefix: "/extauth"
allowed_request_headers:
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


class TracingTestSampling(AmbassadorTest):
    """
    Test for the "sampling" in TracingServices
    """

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: zipkin-65
spec:
  selector:
    app: zipkin-65
  ports:
  - port: 9411
    name: http
    targetPort: http
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zipkin-65
spec:
  selector:
    matchLabels:
      app: zipkin-65
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: zipkin-65
    spec:
      containers:
      - name: zipkin
        image: openzipkin/zipkin:2.17
        ports:
        - name: http
          containerPort: 9411
""" + super().manifests()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.

        yield self.target, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tracing_target_mapping_65
hostname: "*"
prefix: /target-65/
service: {self.target.path.fqdn}
""")

        # Configure the TracingService.
        yield self, """
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
name: tracing-65
service: zipkin-65:9411
driver: zipkin
sampling:
  overall: 10
"""

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query("http://zipkin-65:9411/api/v2/services"))

    def queries(self):
        # Speak through each Ambassador to the traced service...
        for i in range(0, 100):
            yield Query(self.url("target-65/"), phase=1, ignore_result=True)

        # ...then ask the Zipkin for services and spans. Including debug=True in these queries
        # is particularly helpful.
        yield Query("http://zipkin-65:9411/api/v2/traces?limit=10000", phase=check_phase)

        # The diagnostics page should load properly
        yield Query(self.url("ambassador/v0/diag/"), phase=check_phase)

    def check(self):
        traces = self.results[100].json

        print("%d traces obtained" % len(traces))

        #import json
        #print(json.dumps(traces, indent=4, sort_keys=True))

        # We constantly find that Envoy's RNG isn't exactly predictable with small sample
        # sizes, so even though 10% of 100 is 10, we'll make this pass as long as we don't
        # go over 50 or under 1.
        assert 1 <= len(traces) <= 50

class TracingTestZipkinV2(AmbassadorTest):
    """
    Test for the "collector_endpoint_version" Zipkin config in TracingServices
    """

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: zipkin-v2
spec:
  selector:
    app: zipkin-v2
  ports:
  - port: 9411
    name: http
    targetPort: http
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zipkin-v2
spec:
  selector:
    matchLabels:
      app: zipkin-v2
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: zipkin-v2
    spec:
      containers:
      - name: zipkin
        image: openzipkin/zipkin:2.17
        ports:
        - name: http
          containerPort: 9411
""" + super().manifests()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.
        yield self.target, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tracing_target_mapping
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
""")

        # Configure the TracingService.
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
name: tracing
service: zipkin-v2:9411
driver: zipkin
config:
  collector_endpoint: /api/v2/spans
  collector_endpoint_version: HTTP_JSON
  collector_hostname: zipkin-v2
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query("http://zipkin-v2:9411/api/v2/services"))

    def queries(self):
        # Speak through each Ambassador to the traced service...

        for i in range(100):
              yield Query(self.url("target/"), phase=1)


        # ...then ask the Zipkin for services and spans. Including debug=True in these queries
        # is particularly helpful.
        yield Query("http://zipkin-v2:9411/api/v2/services", phase=check_phase)
        yield Query("http://zipkin-v2:9411/api/v2/spans?serviceName=tracingtestzipkinv2-default", phase=check_phase)
        yield Query("http://zipkin-v2:9411/api/v2/traces?serviceName=tracingtestzipkinv2-default", phase=check_phase)

        # The diagnostics page should load properly
        yield Query(self.url("ambassador/v0/diag/"), phase=check_phase)

    def check(self):
        for i in range(100):
            assert self.results[i].backend.name == self.target.path.k8s

        print(f"self.results[100] = {self.results[100]}")
        assert self.results[100].backend is not None and self.results[100].backend.name == "raw", \
                f"unexpected self.results[100] = {self.results[100]}"
        assert len(self.results[100].backend.response) == 1
        assert self.results[100].backend.response[0] == 'tracingtestzipkinv2-default'

        assert self.results[101].backend.name == "raw"

        tracelist = { x: True for x in self.results[101].backend.response }

        assert 'router cluster_tracingtestzipkinv2_http_default egress' in tracelist

        # Look for the host that we actually queried, since that's what appears in the spans.
        assert self.results[0].backend.request.host in tracelist

        # Ensure we generate 128-bit traceids by default
        trace = self.results[102].json[0][0]
        traceId = trace['traceId']
        assert len(traceId) == 32

class TracingTestZipkinV1(AmbassadorTest):
    """
    Test for the "collector_endpoint_version" Zipkin config in TracingServices
    """

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: zipkin-v1
spec:
  selector:
    app: zipkin-v1
  ports:
  - port: 9411
    name: http
    targetPort: http
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zipkin-v1
spec:
  selector:
    matchLabels:
      app: zipkin-v1
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: zipkin-v1
    spec:
      containers:
      - name: zipkin
        image: openzipkin/zipkin:2.17
        ports:
        - name: http
          containerPort: 9411
""" + super().manifests()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        # Use self.target here, because we want this mapping to be annotated
        # on the service, not the Ambassador.

        yield self.target, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  tracing_target_mapping
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
""")

        # Configure the TracingService.
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: TracingService
name: tracing
service: zipkin-v1:9411
driver: zipkin
config:
  collector_endpoint: /api/v1/spans
  collector_endpoint_version: HTTP_JSON_V1
  collector_hostname: zipkin-v1
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query("http://zipkin-v1:9411/api/v2/services"))

    def queries(self):
        # Speak through each Ambassador to the traced service...

        for i in range(100):
              yield Query(self.url("target/"), phase=1)


        # ...then ask the Zipkin for services and spans. Including debug=True in these queries
        # is particularly helpful.
        yield Query("http://zipkin-v1:9411/api/v2/services", phase=check_phase)
        yield Query("http://zipkin-v1:9411/api/v2/spans?serviceName=tracingtestzipkinv1-default", phase=check_phase)
        yield Query("http://zipkin-v1:9411/api/v2/traces?serviceName=tracingtestzipkinv1-default", phase=check_phase)

        # The diagnostics page should load properly
        yield Query(self.url("ambassador/v0/diag/"), phase=check_phase)

    def check(self):
        for i in range(100):
            assert self.results[i].backend.name == self.target.path.k8s

        print(f"self.results[100] = {self.results[100]}")
        assert self.results[100].backend is not None and self.results[100].backend.name == "raw", \
                f"unexpected self.results[100] = {self.results[100]}"
        assert len(self.results[100].backend.response) == 1
        assert self.results[100].backend.response[0] == 'tracingtestzipkinv1-default'

        assert self.results[101].backend.name == "raw"

        tracelist = { x: True for x in self.results[101].backend.response }

        assert 'router cluster_tracingtestzipkinv1_http_default egress' in tracelist

        # Look for the host that we actually queried, since that's what appears in the spans.
        assert self.results[0].backend.request.host in tracelist

        # Ensure we generate 128-bit traceids by default
        trace = self.results[102].json[0][0]
        traceId = trace['traceId']
        assert len(traceId) == 32
