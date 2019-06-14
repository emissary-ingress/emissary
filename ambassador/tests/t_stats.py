import os
import re

from kat.harness import Query
from kat.manifests import AMBASSADOR, RBAC_CLUSTER_SCOPE

from abstract_tests import DEV, AmbassadorTest, HTTP


GRAPHITE_CONFIG = """
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: {0}
spec:
  replicas: 1
  template:
    metadata:
      labels:
        service: {0}
    spec:
      containers:
      - name: {0}
        image: dwflynn/stats-test:0.1.0
        env:
        - name: STATSD_TEST_CLUSTER
          value: cluster_http___statsdtest_http
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: {0}
  name: {0}
spec:
  ports:
  - protocol: UDP
    port: 8125
    targetPort: 8125
    name: statsd-metrics
  - protocol: TCP
    port: 80
    targetPort: 3000
    name: statsd-www
  selector:
    service: {0}
"""


DOGSTATSD_CONFIG = """
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: {0}
spec:
  replicas: 1
  template:
    metadata:
      labels:
        service: {0}
    spec:
      containers:
      - name: {0}
        image: dwflynn/stats-test:0.1.0
        env:
        - name: STATSD_TEST_CLUSTER
          value: cluster_http___dogstatsdtest_http
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: {0}
  name: {0}
spec:
  ports:
  - protocol: UDP
    port: 8125
    targetPort: 8125
    name: statsd-metrics
  - protocol: TCP
    port: 80
    targetPort: 3000
    name: statsd-www
  selector:
    service: {0}
"""


class StatsdTest(AmbassadorTest):
    def init(self):
        self.target = HTTP()
        if DEV:
            self.skip_node = True

    def manifests(self) -> str:
        envs = """
    - name: STATSD_ENABLED
      value: 'true'
"""

        return self.format(RBAC_CLUSTER_SCOPE + AMBASSADOR, image=os.environ["AMBASSADOR_DOCKER_IMAGE"],
                           envs=envs, extra_ports="") + GRAPHITE_CONFIG.format('statsd-sink')

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-reset
case_sensitive: false
prefix: /reset/
rewrite: /RESET/
service: statsd-sink
---
apiVersion: ambassador/v0
kind:  Mapping
name:  metrics
prefix: /metrics
rewrite: /metrics
service: http://127.0.0.1:8877
""")

    def requirements(self):
        yield ("url", Query(self.url("RESET/")))

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name + "/"), phase=1)

        yield Query("http://statsd-sink/DUMP/", phase=2, debug=True)
        yield Query(self.url("metrics"), phase=2)

    def check(self):
        stats = self.results[-2].json or {}

        cluster_stats = stats.get('cluster_http___statsdtest_http', {})
        rq_total = cluster_stats.get('upstream_rq_total', -1)
        rq_200 = cluster_stats.get('upstream_rq_200', -1)

        assert rq_total == 1000, f'expected 1000 total calls, got {rq_total}'
        assert rq_200 > 990, f'expected 1000 successful calls, got {rq_200}'

        metrics = self.results[-1].text
        wanted_metric = 'envoy_cluster_internal_upstream_rq'
        wanted_status = 'envoy_response_code="200"'
        wanted_cluster_name = 'envoy_cluster_name="cluster_http___statsdtest_http'

        for line in metrics.split("\n"):
            if wanted_metric in line and wanted_status in line and wanted_cluster_name in line:
                return
        assert False, 'wanted metric not found in prometheus metrics'


class DogstatsdTest(AmbassadorTest):
    def init(self):
        self.target = HTTP()
        if DEV:
            self.skip_node = True

    def manifests(self) -> str:
        envs = """
    - name: STATSD_ENABLED
      value: 'true'
    - name: STATSD_HOST
      value: 'dogstatsd-sink'
    - name: DOGSTATSD
      value: 'true'
"""

        return self.format(RBAC_CLUSTER_SCOPE + AMBASSADOR, image=os.environ["AMBASSADOR_DOCKER_IMAGE"],
                           envs=envs, extra_ports="") + DOGSTATSD_CONFIG.format('dogstatsd-sink')

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-reset
case_sensitive: false
prefix: /reset/
rewrite: /RESET/
service: dogstatsd-sink
""")

    def requirements(self):
        yield ("url", Query(self.url("RESET/")))

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name + "/"), phase=1)

        yield Query("http://dogstatsd-sink/DUMP/", phase=2, debug=True)

    def check(self):
        stats = self.results[-1].json or {}

        cluster_stats = stats.get('cluster_http___dogstatsdtest_http', {})
        rq_total = cluster_stats.get('upstream_rq_total', -1)
        rq_200 = cluster_stats.get('upstream_rq_200', -1)

        assert rq_total == 1000, f'expected 1000 total calls, got {rq_total}'
        assert rq_200 > 990, f'expected 1000 successful calls, got {rq_200}'
