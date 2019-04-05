import os

from abstract_tests import AmbassadorTest, HTTP, DEV
from harness import Query
from manifests import RBAC_CLUSTER_SCOPE, AMBASSADOR
from test_ambassador import GRAPHITE_CONFIG


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
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}
prefix: /{self.name}/
service: http://{self.target.path.fqdn}
""")

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name + "/"), phase=1)

        yield Query("http://statsd-sink/render?format=json&target=summarize(stats_counts.envoy.cluster.cluster_http___statsdtest_http.upstream_rq_200,'1hour','sum',true)&from=-1hour", phase=2)

    def check(self):
        assert 0 < self.results[-1].json[0]['datapoints'][0][0] <= 1000