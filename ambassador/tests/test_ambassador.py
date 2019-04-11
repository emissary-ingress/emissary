from kat.harness import Runner

from abstract_tests import AmbassadorTest

# Import all the real tests from other files, to make it easier to pick and choose during development.

import t_basics
import t_extauth
import t_grpc
import t_grpc_bridge
import t_grpc_web
import t_headerrouting
import t_loadbalancer
import t_lua_scripts
import t_mappingtests
import t_optiontests
import t_plain
import t_ratelimit
import t_redirect
import t_shadow
import t_stats
import t_tcpmapping
import t_tls
import t_tracing
import t_consul

class CircuitBreakingTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        envs = """
    - name: STATSD_ENABLED
      value: 'true'
"""

        return self.format(RBAC_CLUSTER_SCOPE + AMBASSADOR, image=os.environ["AMBASSADOR_DOCKER_IMAGE"], envs=envs, extra_ports="")

    def config(self):
        yield self, self.format("""
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
""")

    def queries(self):
        for i in range(500):
            yield Query(self.url(self.name) + '-pr/200?sleep=1000', ignore_result=True, phase=1)

        yield Query("http://statsd-sink/render?format=json&target=summarize(stats.envoy.cluster.cluster_httpstat_us.upstream_rq_pending_overflow,'1hour','sum',true)&from=-1hour", phase=2)

    def check(self):

        assert len(self.results) == 501
        pending_results = self.results[0:500]
        pending_stats = self.results[500]

        # pending requests tests
        pending_overloaded = 0
        for result in pending_results:
            if 'X-Envoy-Overloaded' in result.headers:
                pending_overloaded += 1
        assert 450 < pending_overloaded < 500

        pending_datapoints = pending_stats.json[0]['datapoints'][0][0]
        assert 450 < pending_datapoints*10 <= 500

        assert pending_overloaded == pending_datapoints*10


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
service: httpstat.us
host_rewrite: httpstat.us
circuit_breakers:
- priority: default
  max_pending_requests: 1024
  max_connections: 1024
""")

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.target.path.k8s}-normal
prefix: /{self.name}-normal/
service: http://httpstat.us
host_rewrite: httpstat.us
""")

        yield self, self.format("""
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
        assert cb_mapping_overloaded == 0

        # normal mapping tests, global configuration should be in effect
        normal_overloaded = 0
        for result in normal_mapping_results:
            if 'X-Envoy-Overloaded' in result.headers:
                normal_overloaded += 1
        assert 450 < normal_overloaded < 500


# pytest will find this because Runner is a toplevel callable object in a file
# that pytest is willing to look inside.
#
# Also note:
# - Runner(cls) will look for variants of _every subclass_ of cls.
# - Any class you pass to Runner needs to be standalone (it must have its
#   own manifests and be able to set up its own world).
main = Runner(AmbassadorTest)
