from abstract_tests import AmbassadorTest, HTTP, ServiceType
from kat.harness import Query

class CircuitBreakingTest(AmbassadorTest):
    # Needs statsd, which can't work in a dev shell...
    skip_in_dev = True

    target: ServiceType

    envs = {
        'STATSD_ENABLED': 'true',
        'STATSD_HOST': '{self.statsd.path.fqdn}'
    }

    TARGET_CLUSTER='cluster_httpstat_us_er_round_robin'

    upstreams = {
        'statsd': {
            'image': 'dwflynn/stats-test:0.1.0',
            'envs': {
                'STATSD_TEST_CLUSTER': TARGET_CLUSTER,
                # 'STATSD_TEST_DEBUG': 'true'
            },
            'ports': [
                ( 'tcp', 80, 3000 ),
                ( 'udp', 8125, 8125 )
            ]
        }
    }

    def init(self):
        self.target = HTTP()

    configs = {
        'self': '''
---
apiVersion: ambassador/v1
kind:  Module
name:  ambassador
config:
  resolver: endpoint
  load_balancer:
    policy: round_robin
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.target.path.k8s}-pr
prefix: /{self.name}-pr/
service: httpstat.us
host_rewrite: httpstat.us
circuit_breakers:
- priority: default
  max_pending_requests: 1
  max_connections: 1
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-reset
case_sensitive: false
prefix: /reset/
rewrite: /RESET/
service: {self.statsd.path.fqdn}
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-dump
case_sensitive: false
prefix: /dump/
rewrite: /DUMP/
service: {self.statsd.path.fqdn}
'''
    }

    def queries(self):
        for i in range(500):
            yield Query(self.url(self.name) + '-pr/200?sleep=1000', ignore_result=True, phase=1)

        yield Query(self.url("DUMP/"), phase=2)

    def requirements(self):
        yield ("url", Query(self.url("RESET/")))

    def check(self):

        result_count = len(self.results)
        assert result_count == 501, f'wanted 501 results, got {result_count}'

        pending_results = self.results[0:500]
        stats = self.results[500].json or {}

        # pending requests tests
        pending_overloaded = 0

        printed = False

        for result in pending_results:
            if not printed:
                import json
                print(json.dumps(result.as_dict(), sort_keys=True, indent=2))
                printed = True

            if 'X-Envoy-Overloaded' in result.headers:
                pending_overloaded += 1

        assert 450 < pending_overloaded < 500, f'Expected between 450 and 500 overloaded, got {pending_overloaded}'

        cluster_stats = stats.get(self.__class__.TARGET_CLUSTER, {})
        rq_completed = cluster_stats.get('upstream_rq_completed', -1)
        rq_pending_overflow = cluster_stats.get('upstream_rq_pending_overflow', -1)

        assert rq_completed == 500, f'Expected 500 completed requests to httpstat_us, got {rq_completed}'
        assert abs(pending_overloaded - rq_pending_overflow) < 2, f'Expected {pending_overloaded} rq_pending_overflow, got {rq_pending_overflow}'


class GlobalCircuitBreakingTest(AmbassadorTest):
    target: ServiceType

    configs = {
        'self': '''
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.target.path.k8s}-pr
prefix: /{self.name}-pr/
service: httpstat.us
host_rewrite: httpstat.us
circuit_breakers:
- priority: default
  max_pending_requests: 1024
  max_connections: 1024
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.target.path.k8s}-normal
prefix: /{self.name}-normal/
service: http://httpstat.us
host_rewrite: httpstat.us
---
apiVersion: ambassador/v1
kind:  Module
name:  ambassador
config:
  circuit_breakers:
  - priority: default
    max_pending_requests: 1
    max_connections: 1
'''
    }

    def init(self):
        self.target = HTTP()

    def queries(self):
        for i in range(500):
            yield Query(self.url(self.name) + '-pr/200?sleep=1000', ignore_result=True, phase=2)
        for i in range(500):
            yield Query(self.url(self.name) + '-normal/200?sleep=1000', ignore_result=True, phase=2)

    def check(self):

        assert len(self.results) == 1000
        cb_mapping_results = self.results[0:500]
        normal_mapping_results = self.results[500:1000]

        # circuit breaker mapping tests
        cb_mapping_overloaded = 0
        for result in cb_mapping_results:
            if 'X-Envoy-Overloaded' in result.headers:
                cb_mapping_overloaded += 1
        assert cb_mapping_overloaded == 0, f'expected no -pr overloaded, got {cb_mapping_overloaded}'

        # normal mapping tests, global configuration should be in effect
        normal_overloaded = 0
        for result in normal_mapping_results:
            if 'X-Envoy-Overloaded' in result.headers:
                normal_overloaded += 1
        assert 450 < normal_overloaded < 500, f'expected between 450 and 500 -normal overloaded, got {normal_overloaded}'
