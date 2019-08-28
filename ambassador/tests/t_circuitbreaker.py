import os

from abstract_tests import AmbassadorTest, HTTP, ServiceType
from kat.harness import Query
from kat.manifests import AMBASSADOR, RBAC_CLUSTER_SCOPE

STATSD_MANIFEST = """
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: {name}
spec:
  replicas: 1
  template:
    metadata:
      labels:
        service: {name}
    spec:
      containers:
      - name: {name}
        image: dwflynn/stats-test:0.1.0
        env:
        - name: STATSD_TEST_CLUSTER
          value: {target}
      restartPolicy: Always
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: {name}
  name: {name}
spec:
  ports:
  - protocol: UDP
    port: 8125
    name: statsd-metrics
  - protocol: TCP
    port: 80
    targetPort: 3000
    name: statsd-http
  selector:
    service: {name}
"""

class CircuitBreakingTest(AmbassadorTest):
    target: ServiceType

    TARGET_CLUSTER='cluster_circuitbreakingtest_http_cbdc1p1'

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        envs = """
    - name: STATSD_ENABLED
      value: 'true'
    - name: STATSD_HOST
      value: 'cbstatsd-sink'
"""

        return self.format(RBAC_CLUSTER_SCOPE + AMBASSADOR, image=os.environ["AMBASSADOR_DOCKER_IMAGE"],
                           envs=envs, extra_ports="") + STATSD_MANIFEST.format(name='cbstatsd-sink',
                                                                               target=self.__class__.TARGET_CLUSTER)

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.target.path.k8s}-pr
prefix: /{self.name}-pr/
service: {self.target.path.fqdn}
circuit_breakers:
- priority: default
  max_pending_requests: 1
  max_connections: 1
---
apiVersion: ambassador/v1
kind: Mapping
name: {self.name}-reset
case_sensitive: false
prefix: /reset/
rewrite: /RESET/
service: cbstatsd-sink
---
apiVersion: ambassador/v1
kind: Mapping
name: {self.name}-dump
case_sensitive: false
prefix: /dump/
rewrite: /DUMP/
service: cbstatsd-sink
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query(self.url("RESET/")))

    def queries(self):
        for i in range(500):
            yield Query(self.url(self.name) + '-pr/', headers={ "Requested-Backend-Delay": "1000" },
                        ignore_result=True, phase=1)

        yield Query(self.url("DUMP/"), phase=2)

    def check(self):
        result_count = len(self.results)
        assert result_count == 501, f'wanted 501 results, got {result_count}'

        pending_results = self.results[0:500]
        stats = self.results[500].json or {}

        # pending requests tests
        pending_overloaded = 0

        # printed = False

        for result in pending_results:
            # if not printed:
            #     import json
            #     print(json.dumps(result.as_dict(), sort_keys=True, indent=2))
            #     printed = True

            if 'X-Envoy-Overloaded' in result.headers:
                pending_overloaded += 1

        assert 450 < pending_overloaded < 500, f'Expected between 450 and 500 overloaded, got {pending_overloaded}'

        cluster_stats = stats.get(self.__class__.TARGET_CLUSTER, {})
        rq_completed = cluster_stats.get('upstream_rq_completed', -1)
        rq_pending_overflow = cluster_stats.get('upstream_rq_pending_overflow', -1)

        assert rq_completed == 500, f'Expected 500 completed requests to {self.__class__.TARGET_CLUSTER}, got {rq_completed}'
        assert abs(pending_overloaded - rq_pending_overflow) < 2, f'Expected {pending_overloaded} rq_pending_overflow, got {rq_pending_overflow}'


class GlobalCircuitBreakingTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.target.path.k8s}-pr
prefix: /{self.name}-pr/
service: {self.target.path.fqdn}
circuit_breakers:
- priority: default
  max_pending_requests: 1024
  max_connections: 1024
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.target.path.k8s}-normal
prefix: /{self.name}-normal/
service: {self.target.path.fqdn}
---
apiVersion: ambassador/v1
kind:  Module
name:  ambassador
config:
  circuit_breakers:
  - priority: default
    max_pending_requests: 1
    max_connections: 1
""")

    def queries(self):
        for i in range(500):
            yield Query(self.url(self.name) + '-pr/', headers={ "Requested-Backend-Delay": "1000" },
                        ignore_result=True, phase=1)
        for i in range(500):
            yield Query(self.url(self.name) + '-normal/', headers={ "Requested-Backend-Delay": "1000" },
                        ignore_result=True, phase=1)

    def check(self):

        assert len(self.results) == 1000
        cb_mapping_results = self.results[0:500]
        normal_mapping_results = self.results[500:1000]

        # '-pr' mapping tests: this is a priority class of connection
        pr_mapping_overloaded = 0

        for result in cb_mapping_results:
            if 'X-Envoy-Overloaded' in result.headers:
                pr_mapping_overloaded += 1

        assert pr_mapping_overloaded == 0, f'[GCR] expected no -pr overloaded, got {pr_mapping_overloaded}'

        # '-normal' mapping tests: global configuration should be in effect
        normal_overloaded = 0
        # printed = False

        for result in normal_mapping_results:
            # if not printed:
            #     import json
            #     print(json.dumps(result.as_dict(), sort_keys=True, indent=2))
            #     printed = True

            if 'X-Envoy-Overloaded' in result.headers:
                normal_overloaded += 1

        assert 450 < normal_overloaded < 500, f'[GCF] expected 450-500 normal_overloaded, got {normal_overloaded}'
