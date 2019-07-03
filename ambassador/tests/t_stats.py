from kat.harness import Query

from abstract_tests import DEV, AmbassadorTest, HTTP


class StatsdTest(AmbassadorTest):
    skip_in_dev = True

    envs = {
        'STATSD_ENABLED': 'true',
        'STATSD_HOST': '{self.statsd.path.fqdn}'
    }

    configs = {
        'target': '''
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
service: {self.statsd.path.fqdn}
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-dump
case_sensitive: false
prefix: /dump/
rewrite: /DUMP/
service: {self.statsd.path.fqdn}
---
apiVersion: ambassador/v0
kind:  Mapping
name:  metrics
prefix: /metrics
rewrite: /metrics
service: http://127.0.0.1:8877
'''
    }

    TARGET_CLUSTER='cluster_http___statsdtest_http_er_round_robin'

    upstreams = {
        'target': {
            'servicetype': 'HTTP'
        },
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

    def requirements(self):
        yield ("url", Query(self.url("RESET/")))

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name + "/"), phase=1)

        yield Query(self.url("dump/"), phase=2)
        yield Query(self.url("metrics"), phase=2)

    def check(self):
        stats = self.results[-2].json or {}

        cluster_stats = stats.get(self.__class__.TARGET_CLUSTER, {})
        rq_total = cluster_stats.get('upstream_rq_total', -1)
        rq_200 = cluster_stats.get('upstream_rq_200', -1)

        assert rq_total == 1000, f'expected 1000 total calls, got {rq_total}'
        assert rq_200 > 990, f'expected 1000 successful calls, got {rq_200}'

        metrics = self.results[-1].text
        wanted_metric = 'envoy_cluster_internal_upstream_rq'
        wanted_status = 'envoy_response_code="200"'
        wanted_cluster_name = f'envoy_cluster_name="{self.__class__.TARGET_CLUSTER}"'

        for line in metrics.split("\n"):
            if wanted_metric in line and wanted_status in line and wanted_cluster_name in line:
                return
        assert False, 'wanted metric not found in prometheus metrics'


class DogstatsdTest(AmbassadorTest):
    skip_in_dev = True

    envs = {
        'STATSD_ENABLED': 'true',
        'STATSD_HOST': '{self.statsd.path.fqdn}',
        'DOGSTATSD': 'true'
    }

    configs = {
        'target': '''
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
service: dogstatsdtest-statsd
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

    TARGET_CLUSTER='cluster_http___dogstatsdtest_http_er_round_robin'

    upstreams = {
        'target': {
            'servicetype': 'HTTP'
        },
        'statsd': {
            'image': 'dwflynn/stats-test:0.1.0',
            'envs': {
                'STATSD_TEST_CLUSTER': TARGET_CLUSTER
            },
            'ports': [
                ( 'tcp', 80, 3000 ),
                ( 'udp', 8125, 8125 )
            ]
        }
    }

    def requirements(self):
        yield ("url", Query(self.url("RESET/")))

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name + "/"), phase=1)

        yield Query(self.url("dump/"), phase=2)

    def check(self):
        stats = self.results[-1].json or {}

        cluster_stats = stats.get(self.__class__.TARGET_CLUSTER, {})
        rq_total = cluster_stats.get('upstream_rq_total', -1)
        rq_200 = cluster_stats.get('upstream_rq_200', -1)

        assert rq_total == 1000, f'expected 1000 total calls, got {rq_total}'
        assert rq_200 > 990, f'expected 1000 successful calls, got {rq_200}'
