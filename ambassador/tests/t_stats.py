import os

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
        image: hopsoft/graphite-statsd:v0.9.15-phusion0.9.18
      restartPolicy: Always
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
    name: statsd-metrics
  - protocol: TCP
    port: 80
    name: graphite-www
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
        image: patricksanders/statsdebug:0.1.0
        ports:
        - containerPort: 8080
        - containerPort: 8125
          protocol: UDP
      restartPolicy: Always
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
    name: statsdebug-statsd
  - protocol: TCP
    port: 80
    targetPort: 8080
    name: statsdebug-http
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
""")

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name + "/"), phase=1)

        yield Query("http://dogstatsd-sink/all", phase=2)

    def check(self):
        # If we have a envoy.http.downstream_rq_total metric, we can safely
        # assume that envoy is sending dogstatsd.
        assert 0 < self.results[-1].json['envoy.http.downstream_rq_total'] <= 1000
