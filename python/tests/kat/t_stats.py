import os
import re

from kat.harness import Query, load_manifest

from abstract_tests import DEV, AmbassadorTest, HTTP

AMBASSADOR = load_manifest("ambassador")
RBAC_CLUSTER_SCOPE = load_manifest("rbac_cluster_scope")

STATSD_TEST_CLUSTER = "statsdtest_http"
ALT_STATSD_TEST_CLUSTER = "short-stats-name"
DOGSTATSD_TEST_CLUSTER = "dogstatsdtest_http"

GRAPHITE_CONFIG = """
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {0}
spec:
  selector:
    matchLabels:
      service: {0}
  replicas: 1
  template:
    metadata:
      labels:
        service: {0}
    spec:
      containers:
      - name: {0}
        image: {1}
        env:
        - name: STATSD_TEST_DEBUG
          value: "true"
        - name: STATSD_TEST_CLUSTER
          value: {2}
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
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {0}
spec:
  selector:
    matchLabels:
      service: {0}
  replicas: 1
  template:
    metadata:
      labels:
        service: {0}
    spec:
      containers:
      - name: {0}
        image: {1}
        env:
        - name: STATSD_TEST_CLUSTER
          value: {2}
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
        self.target2 = HTTP(name="alt-statsd")
        self.stats_name = ALT_STATSD_TEST_CLUSTER
        if DEV:
            self.skip_node = True

    def manifests(self) -> str:
        envs = """
    - name: STATSD_ENABLED
      value: 'true'
"""

        return self.format(RBAC_CLUSTER_SCOPE + AMBASSADOR, image=os.environ["AMBASSADOR_DOCKER_IMAGE"],
                           envs=envs, extra_ports="", capabilities_block="") + \
               GRAPHITE_CONFIG.format('statsd-sink', self.test_image['stats'], f"{STATSD_TEST_CLUSTER}:{ALT_STATSD_TEST_CLUSTER}")

    def config(self):
        yield self.target, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-alt
hostname: "*"
prefix: /{self.name}-alt/
stats_name: {self.stats_name}
service: http://{self.target2.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-reset
hostname: "*"
case_sensitive: false
prefix: /reset/
rewrite: /RESET/
service: statsd-sink
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  metrics
hostname: "*"
prefix: /metrics
rewrite: /metrics
service: http://127.0.0.1:8877
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query(self.url("RESET/")))

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name + "/"), phase=1)
            yield Query(self.url(self.name + "-alt/"), phase=1)

        yield Query("http://statsd-sink/DUMP/", phase=2)
        yield Query(self.url("metrics"), phase=2)

    def check(self):
        # self.results[-2] is the JSON dump from our test statsd-sink service.
        stats = self.results[-2].json or {}

        cluster_stats = stats.get(STATSD_TEST_CLUSTER, {})
        rq_total = cluster_stats.get('upstream_rq_total', -1)
        rq_200 = cluster_stats.get('upstream_rq_200', -1)

        assert rq_total == 1000, f'{STATSD_TEST_CLUSTER}: expected 1000 total calls, got {rq_total}'
        assert rq_200 > 990, f'{STATSD_TEST_CLUSTER}: expected 1000 successful calls, got {rq_200}'

        cluster_stats = stats.get(ALT_STATSD_TEST_CLUSTER, {})
        rq_total = cluster_stats.get('upstream_rq_total', -1)
        rq_200 = cluster_stats.get('upstream_rq_200', -1)

        assert rq_total == 1000, f'{ALT_STATSD_TEST_CLUSTER}: expected 1000 total calls, got {rq_total}'
        assert rq_200 > 990, f'{ALT_STATSD_TEST_CLUSTER}: expected 1000 successful calls, got {rq_200}'

        # self.results[-1] is the text dump from Envoy's '/metrics' endpoint.
        metrics = self.results[-1].text

        # Somewhere in here, we want to see a metric explicitly for both our "real"
        # cluster and our alt cluster, returning a 200. Are they there?
        wanted_metric = 'envoy_cluster_internal_upstream_rq'
        wanted_status = 'envoy_response_code="200"'
        wanted_cluster_name = f'envoy_cluster_name="{STATSD_TEST_CLUSTER}"'
        alt_wanted_cluster_name = f'envoy_cluster_name="{ALT_STATSD_TEST_CLUSTER}"'

        found_normal = False
        found_alt = False

        for line in metrics.split("\n"):
            if wanted_metric in line and wanted_status in line and wanted_cluster_name in line:
                print(f"line '{line}'")
                found_normal = True

            if wanted_metric in line and wanted_status in line and alt_wanted_cluster_name in line:
                print(f"line '{line}'")
                found_alt = True

        assert found_normal, f"wanted {STATSD_TEST_CLUSTER} in Prometheus metrics, but didn't find it"
        assert found_alt, f"wanted {ALT_STATSD_TEST_CLUSTER} in Prometheus metrics, but didn't find it"


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
                           envs=envs, extra_ports="", capabilities_block="") + \
               DOGSTATSD_CONFIG.format('dogstatsd-sink', self.test_image['stats'], DOGSTATSD_TEST_CLUSTER)

    def config(self):
        yield self.target, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}
hostname: "*"
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-reset
case_sensitive: false
hostname: "*"
prefix: /reset/
rewrite: /RESET/
service: dogstatsd-sink
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query(self.url("RESET/")))

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name + "/"), phase=1)

        yield Query("http://dogstatsd-sink/DUMP/", phase=2)

    def check(self):
        stats = self.results[-1].json or {}

        cluster_stats = stats.get(DOGSTATSD_TEST_CLUSTER, {})
        rq_total = cluster_stats.get('upstream_rq_total', -1)
        rq_200 = cluster_stats.get('upstream_rq_200', -1)

        assert rq_total == 1000, f'expected 1000 total calls, got {rq_total}'
        assert rq_200 > 990, f'expected 1000 successful calls, got {rq_200}'
