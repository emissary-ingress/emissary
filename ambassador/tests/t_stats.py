from kat.harness import Query

from abstract_tests import DEV, AmbassadorTest, HTTP


class StatsdTest(AmbassadorTest):
    skip_in_dev = True

    envs = {
        'STATSD_ENABLED': 'true',
        'STATSD_HOST': 'statsdtest-statsd'
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
service: statsdtest-statsd
---
apiVersion: ambassador/v0
kind:  Mapping
name:  metrics
prefix: /metrics
rewrite: /metrics
service: http://127.0.0.1:8877
'''
    }

    upstreams = {
        'target': {
            'servicetype': 'HTTP'
        },
        'statsdtest-statsd': {
            'image': 'dwflynn/stats-test:0.1.0',
            'envs': {
                'STATSD_TEST_CLUSTER': "cluster_http___statsdtest_http",
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

        yield Query("http://statsdtest-statsd/DUMP/", phase=2)
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
    skip_in_dev = True

    envs = {
        'STATSD_ENABLED': 'true',
        'STATSD_HOST': 'dogstatsdtest-statsd',
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
'''
    }

    upstreams = {
        'target': {
            'servicetype': 'HTTP'
        },
        'dogstatsdtest-statsd': {
            'image': 'dwflynn/stats-test:0.1.0',
            'envs': {
                'STATSD_TEST_CLUSTER': "cluster_http___dogstatsdtest_http"
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

        yield Query("http://dogstatsdtest-statsd/DUMP/", phase=2, debug=True)

    def check(self):
        stats = self.results[-1].json or {}

        cluster_stats = stats.get('cluster_http___dogstatsdtest_http', {})
        rq_total = cluster_stats.get('upstream_rq_total', -1)
        rq_200 = cluster_stats.get('upstream_rq_200', -1)

        assert rq_total == 1000, f'expected 1000 total calls, got {rq_total}'
        assert rq_200 > 990, f'expected 1000 successful calls, got {rq_200}'
