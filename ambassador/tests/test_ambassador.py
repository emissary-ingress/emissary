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

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.target.path.k8s}
prefix: /{self.name}/
service: httpstat.us
host_rewrite: httpstat.us
circuit_breakers:
- priority: default
  max_connections: 1
  max_pending_requests: 1
  max_requests: 1
  max_retries: 1
- priority: high
  max_connections: 1
  max_pending_requests: 1
  max_requests: 1
  max_retries: 1
""")

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name) + '/200?sleep=1000', ignore_result=True)

    def check(self):
        overloaded = 0
        for result in self.results:
            if 'X-Envoy-Overloaded' in result.headers:
                overloaded += 1
        assert 500 < overloaded < 1000


# pytest will find this because Runner is a toplevel callable object in a file
# that pytest is willing to look inside.
#
# Also note:
# - Runner(cls) will look for variants of _every subclass_ of cls.
# - Any class you pass to Runner needs to be standalone (it must have its
#   own manifests and be able to set up its own world).
main = Runner(AmbassadorTest)
