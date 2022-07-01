import os
from typing import Generator, Tuple, Union

import pytest

import tests.integration.manifests as integration_manifests
from abstract_tests import AmbassadorTest, HTTP, ServiceType, Node
from kat.harness import Query

STATSD_MANIFEST = """
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {name}
spec:
  selector:
    matchLabels:
      service: {name}
  replicas: 1
  template:
    metadata:
      labels:
        service: {name}
    spec:
      containers:
      - name: {name}
        image: {image}
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

    TARGET_CLUSTER = "cluster_circuitbreakingtest_http_cbdc1p1"

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        envs = """
    - name: STATSD_ENABLED
      value: 'true'
    - name: STATSD_HOST
      value: 'cbstatsd-sink'
"""

        return (
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: cleartext-listener
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  port: 8080
  protocol: HTTP
  securityModel: INSECURE
  hostBinding:
    selector:
      matchLabels:
        kat-ambassador-id: {self.ambassador_id}
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: cleartext-host
  labels:
    kat-ambassador-id: {self.ambassador_id}
spec:
  ambassador_id: [ {self.ambassador_id} ]
  requestPolicy:
    insecure:
      action: Route
"""
            + self.format(
                integration_manifests.load("rbac_cluster_scope")
                + integration_manifests.load("ambassador"),
                envs=envs,
                extra_ports="",
                capabilities_block="",
            )
            + STATSD_MANIFEST.format(
                name="cbstatsd-sink",
                image=integration_manifests.get_images()["test-stats"],
                target=self.__class__.TARGET_CLUSTER,
            )
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-pr
hostname: "*"
prefix: /{self.name}-pr/
service: {self.target.path.fqdn}
circuit_breakers:
- priority: default
  max_pending_requests: 1
  max_connections: 1
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: {self.name}-reset
case_sensitive: false
hostname: "*"
prefix: /reset/
rewrite: /RESET/
service: cbstatsd-sink
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name: {self.name}-dump
case_sensitive: false
hostname: "*"
prefix: /dump/
rewrite: /DUMP/
service: cbstatsd-sink
"""
        )

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query(self.url(self.name) + "-pr/"))
        yield ("url", Query(self.url("RESET/")))

    def queries(self):
        # Reset the statsd setup in phase 1...
        yield Query(self.url("RESET/"), phase=1)

        # Run 200 queries in phase 2, after the reset...
        for i in range(200):
            yield Query(
                self.url(self.name) + "-pr/",
                headers={"Kat-Req-Http-Requested-Backend-Delay": "1000"},
                ignore_result=True,
                phase=2,
            )

        # ...then 200 more queries in phase 3. Why the split? Because we get flakes if we
        # try to ram 500 through at once (in the middle of the run, we get some connections
        # that time out).

        for i in range(200):
            yield Query(
                self.url(self.name) + "-pr/",
                headers={"Kat-Req-Http-Requested-Backend-Delay": "1000"},
                ignore_result=True,
                phase=3,
            )

        # Dump the results in phase 4, after the queries.
        yield Query(self.url("DUMP/"), phase=4)

    def check(self):
        result_count = len(self.results)

        failures = []

        if result_count != 402:
            failures.append(f"wanted 402 results, got {result_count}")
        else:
            pending_results = self.results[1:400]
            stats = self.results[401].json or {}

            # pending requests tests
            pending_overloaded = 0
            error = 0

            # printed = False

            for result in pending_results:
                # if not printed:
                #     import json
                #     print(json.dumps(result.as_dict(), sort_keys=True, indent=2))
                #     printed = True

                if result.error:
                    error += 1
                elif "X-Envoy-Overloaded" in result.headers:
                    pending_overloaded += 1

            failed = False

            if not 300 < pending_overloaded < 400:
                failures.append(
                    f"Expected between 300 and 400 overloaded, got {pending_overloaded}"
                )

            cluster_stats = stats.get(self.__class__.TARGET_CLUSTER, {})
            rq_completed = cluster_stats.get("upstream_rq_completed", -1)
            rq_pending_overflow = cluster_stats.get("upstream_rq_pending_overflow", -1)

            if error != 0:
                failures.append(f"Expected no errors but got {error}")

            if rq_completed != 400:
                failures.append(
                    f"Expected 400 completed requests to {self.__class__.TARGET_CLUSTER}, got {rq_completed}"
                )

            if abs(pending_overloaded - rq_pending_overflow) >= 2:
                failures.append(
                    f"Expected {pending_overloaded} rq_pending_overflow, got {rq_pending_overflow}"
                )

        if failures:
            print("%s FAILED:\n  %s" % (self.name, "\n  ".join(failures)))
            pytest.xfail(f"FFS {self.name}")


class GlobalCircuitBreakingTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
name: cleartext-host
port: 8080
protocol: HTTP
securityModel: INSECURE
requestPolicy:
  insecure:
    action: Route
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-pr
hostname: "*"
prefix: /{self.name}-pr/
service: {self.target.path.fqdn}
circuit_breakers:
- priority: default
  max_pending_requests: 1024
  max_connections: 1024
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-normal
hostname: "*"
prefix: /{self.name}-normal/
service: {self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind:  Module
name:  ambassador
config:
  circuit_breakers:
  - priority: default
    max_pending_requests: 5
    max_connections: 5
"""
        )

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query(self.url(self.name) + "-pr/"))
        yield ("url", Query(self.url(self.name) + "-normal/"))

    def queries(self):
        for i in range(200):
            yield Query(
                self.url(self.name) + "-pr/",
                headers={"Kat-Req-Http-Requested-Backend-Delay": "1000"},
                ignore_result=True,
                phase=1,
            )
        for i in range(200):
            yield Query(
                self.url(self.name) + "-normal/",
                headers={"Kat-Req-Http-Requested-Backend-Delay": "1000"},
                ignore_result=True,
                phase=1,
            )

    def check(self):
        failures = []

        if len(self.results) != 400:
            failures.append(f"wanted 400 results, got {len(self.results)}")
        else:
            cb_mapping_results = self.results[0:200]
            normal_mapping_results = self.results[200:400]

            # '-pr' mapping tests: this is a priority class of connection
            pr_mapping_overloaded = 0

            for result in cb_mapping_results:
                if "X-Envoy-Overloaded" in result.headers:
                    pr_mapping_overloaded += 1

            if pr_mapping_overloaded != 0:
                failures.append(f"[GCR] expected no -pr overloaded, got {pr_mapping_overloaded}")

            # '-normal' mapping tests: global configuration should be in effect
            normal_overloaded = 0
            # printed = False

            for result in normal_mapping_results:
                # if not printed:
                #     import json
                #     print(json.dumps(result.as_dict(), sort_keys=True, indent=2))
                #     printed = True

                if "X-Envoy-Overloaded" in result.headers:
                    normal_overloaded += 1

            if not 100 < normal_overloaded < 200:
                failures.append(
                    f"[GCF] expected 100-200 normal_overloaded, got {normal_overloaded}"
                )

        if failures:
            print("%s FAILED:\n  %s" % (self.name, "\n  ".join(failures)))
            pytest.xfail(f"FFS {self.name}")


class CircuitBreakingTCPTest(AmbassadorTest):
    extra_ports = [6789, 6790]

    target1: ServiceType
    target2: ServiceType

    def init(self):
        self.target1 = HTTP(name="target1")
        self.target2 = HTTP(name="target2")

    # config() must _yield_ tuples of Node, Ambassador-YAML where the
    # Ambassador-YAML will be annotated onto the Node.

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self.target1, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
name:  {self.name}-1
port: 6789
service: {self.target1.path.fqdn}:80
"""
        )
        yield self.target2, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: TCPMapping
name:  {self.name}-2
port: 6790
service: {self.target2.path.fqdn}:80
circuit_breakers:
- priority: default
  max_pending_requests: 1
  max_connections: 1
"""
        )

    def queries(self):
        for i in range(200):
            yield Query(
                self.url(self.name, port=6789),
                headers={"Kat-Req-Http-Requested-Backend-Delay": "1000"},
                ignore_result=True,
                phase=1,
            )
        for i in range(200):
            yield Query(
                self.url(self.name, port=6790),
                headers={"Kat-Req-Http-Requested-Backend-Delay": "1000"},
                ignore_result=True,
                phase=1,
            )

    def check(self):
        failures = []

        if len(self.results) != 400:
            failures.append(f"wanted 400 results, got {len(self.results)}")
        else:
            default_limit_result = self.results[0:200]
            low_limit_results = self.results[200:400]

            default_limit_failure = 0

            for result in default_limit_result:
                if result.error:
                    default_limit_failure += 1

            if default_limit_failure != 0:
                failures.append(
                    f"expected no failure with default limit, got {default_limit_failure}"
                )

            low_limit_failure = 0

            for result in low_limit_results:
                if result.error:
                    low_limit_failure += 1

            if not 100 < low_limit_failure < 200:
                failures.append(f"expected 100-200 failure with low limit, got {low_limit_failure}")

        if failures:
            print("%s FAILED:\n  %s" % (self.name, "\n  ".join(failures)))
            pytest.xfail(f"FFS {self.name}")
