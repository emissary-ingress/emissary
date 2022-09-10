import os
from typing import Generator, Tuple, Union

from abstract_tests import DEV, HTTP, AmbassadorTest, Node, ServiceType, StatsDSink
from kat.harness import Query

STATSD_TEST_CLUSTER = "statsdtest_http"
ALT_STATSD_TEST_CLUSTER = "short-stats-name"
DOGSTATSD_TEST_CLUSTER = "dogstatsdtest_http"


class StatsdTest(AmbassadorTest):
    sink: ServiceType

    def init(self):
        self.target = HTTP()
        self.target2 = HTTP(name="alt-statsd")
        self.sink = StatsDSink(target_cluster=f"{STATSD_TEST_CLUSTER}:{ALT_STATSD_TEST_CLUSTER}")
        self.stats_name = ALT_STATSD_TEST_CLUSTER
        if DEV:
            self.skip_node = True

    def manifests(self) -> str:
        self.manifest_envs += f"""
    - name: STATSD_ENABLED
      value: 'true'
    - name: STATSD_HOST
      value: {self.sink.path.fqdn}
"""
        return super().manifests()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self.target, self.format(
            """
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
service: {self.sink.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  metrics
hostname: "*"
prefix: /metrics
rewrite: /metrics
service: http://127.0.0.1:8877
"""
        )

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query(self.url("RESET/")))

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name + "/"), phase=1)
            yield Query(self.url(self.name + "-alt/"), phase=1)

        yield Query(f"http://{self.sink.path.fqdn}/DUMP/", phase=2)
        yield Query(self.url("metrics"), phase=2)

    def check(self):
        # self.results[-2] is the JSON dump from our test self.sink service.
        stats = self.results[-2].json or {}
        print(f"stats = {repr(stats)}")

        cluster_stats = stats.get(STATSD_TEST_CLUSTER, {})
        rq_total = cluster_stats.get("upstream_rq_total", -1)
        rq_200 = cluster_stats.get("upstream_rq_200", -1)

        assert rq_total == 1000, f"{STATSD_TEST_CLUSTER}: expected 1000 total calls, got {rq_total}"
        assert rq_200 > 900, f"{STATSD_TEST_CLUSTER}: expected ~1000 successful calls, got {rq_200}"

        cluster_stats = stats.get(ALT_STATSD_TEST_CLUSTER, {})
        rq_total = cluster_stats.get("upstream_rq_total", -1)
        rq_200 = cluster_stats.get("upstream_rq_200", -1)

        assert (
            rq_total == 1000
        ), f"{ALT_STATSD_TEST_CLUSTER}: expected 1000 total calls, got {rq_total}"
        assert (
            rq_200 > 900
        ), f"{ALT_STATSD_TEST_CLUSTER}: expected ~1000 successful calls, got {rq_200}"

        # self.results[-1] is the text dump from Envoy's '/metrics' endpoint.
        metrics = self.results[-1].text

        # Somewhere in here, we want to see a metric explicitly for both our "real"
        # cluster and our alt cluster, returning a 200. Are they there?
        wanted_metric = "envoy_cluster_internal_upstream_rq"
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

        assert (
            found_normal
        ), f"wanted {STATSD_TEST_CLUSTER} in Prometheus metrics, but didn't find it"
        assert (
            found_alt
        ), f"wanted {ALT_STATSD_TEST_CLUSTER} in Prometheus metrics, but didn't find it"


class DogstatsdTest(AmbassadorTest):
    dogstatsd: ServiceType

    def init(self):
        self.target = HTTP()
        self.sink = StatsDSink(target_cluster=DOGSTATSD_TEST_CLUSTER)
        if DEV:
            self.skip_node = True

    def manifests(self) -> str:
        self.manifest_envs += f"""
    - name: STATSD_ENABLED
      value: 'true'
    - name: STATSD_HOST
      value: {self.sink.path.fqdn}
    - name: DOGSTATSD
      value: 'true'
"""
        return super().manifests()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self.target, self.format(
            """
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
service: {self.sink.path.fqdn}
"""
        )

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query(self.url("RESET/")))

    def queries(self):
        for i in range(1000):
            yield Query(self.url(self.name + "/"), phase=1)

        yield Query(f"http://{self.sink.path.fqdn}/DUMP/", phase=2)

    def check(self):
        stats = self.results[-1].json or {}
        print(f"stats = {repr(stats)}")

        cluster_stats = stats.get(DOGSTATSD_TEST_CLUSTER, {})
        rq_total = cluster_stats.get("upstream_rq_total", -1)
        rq_200 = cluster_stats.get("upstream_rq_200", -1)

        assert rq_total == 1000, f"expected 1000 total calls, got {rq_total}"
        assert rq_200 > 900, f"expected ~1000 successful calls, got {rq_200}"
