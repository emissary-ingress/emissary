import time
from typing import Generator, Tuple, Union

from abstract_tests import AmbassadorTest, HealthCheckServer, Node
from kat.harness import Query

"2022-10-04T23:24:45Z"
"2022-10-04T23:26:45Z"


class ActiveHealthCheckTest(AmbassadorTest):
    def init(self):
        self.target = HealthCheckServer()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-health
hostname: "*"
prefix: /healthcheck/
service: {self.target.path.fqdn}
resolver: endpoint
load_balancer:
  policy: round_robin
health_checks:
- unhealthy_threshold: 1
  interval: 1s
  health_check:
    http:
      path: /healthcheck/actualcheck/
"""
        )  # The round robin load balancer is not necessary for the test but should help make the request distribution even across the pods

    def queries(self):
        yield Query(self.url("healthcheck/"), phase=1)  # Just making sure things are running
        yield Query(self.url("ambassador/v0/diag/"), phase=1)

        yield Query(
            self.url("healthcheck/makeUnhealthy/"), phase=1
        )  # the deployment has 5 pods. This will make one of them start returning errors

        # These two queries on their own in separate phases are just a hack way of getting the kat client
        # to wait a little bit after the previous query so that the automated health checks have time to notice
        # that one of the pods is misbehaving before we start blasting requests out.
        yield Query(self.url("healthcheck/"), expected=[200, 500], phase=2)
        yield Query(self.url("healthcheck/"), expected=[200, 500], phase=3)

        # Make 1000 requests split into two groups to reduce any flakes
        for _ in range(500):
            yield Query(self.url("healthcheck/"), expected=[200, 500], phase=4)
            time.sleep(0.06)

        for _ in range(500):
            yield Query(self.url("healthcheck/"), expected=[200, 500], phase=5)
            time.sleep(0.06)

    def check(self):
        print(self.results)
        # Check the manual requests first
        assert self.results[0].status == 200
        assert self.results[1].status == 200
        assert self.results[2].status == 200

        # Add up the number of 500 and 200 responses that we got.
        valid = 0
        errors = 0
        for i in range(5, 1005):
            assert (self.results[i].status == 200 or self.results[i].status == 500) == True
            if self.results[i].status == 200:
                valid += 1
            elif self.results[i].status == 500:
                errors += 1

        # with 1000 requests and 1/5 being an error response, we should have the following distribution +/- 10
        # assert 190 <= errors <= 210
        # assert 790 <= valid <= 810

        # But since we configure health chekcing we should actually see 0 errors because the health checks noticed
        # that one of the pods was unhealthy and didn't route any traffic to it.
        msg = "Errors: {}, Valid: {}".format(errors, valid)
        assert errors == 0, msg
        assert valid == 1000, msg


class NoHealthCheckTest(AmbassadorTest):
    def init(self):
        self.target = HealthCheckServer()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.target.path.k8s}-health
hostname: "*"
prefix: /healthcheck/
service: {self.target.path.fqdn}
resolver: endpoint
load_balancer:
  policy: round_robin
"""
        )  # The round robin load balancer is not necessary for the test but should help make the request distribution even across the pods

    def queries(self):
        yield Query(self.url("healthcheck/"))  # Just making sure things are running
        yield Query(self.url("ambassador/v0/diag/"))

        yield Query(
            self.url("healthcheck/makeUnhealthy/")
        )  # the deployment has 5 pods. This will make one of them start returning errors

        # Make 1000 requests over the course of a minute
        for _ in range(1000):
            yield Query(self.url("healthcheck/"), expected=[200, 500])

    def check(self):
        print(self.results)
        # Check the manual requests first
        assert self.results[0].status == 200
        assert self.results[1].status == 200
        assert self.results[2].status == 200

        # Since we haven't configured any health checking, we should expect to see a fair number of error responses
        valid = 0
        errors = 0
        for i in range(3, 1003):
            assert (self.results[i].status == 200 or self.results[i].status == 500) == True
            if self.results[i].status == 200:
                valid += 1
            elif self.results[i].status == 500:
                errors += 1

        # with 1000 requests and 1/5 being an error response, we should have the following distribution +/- 10
        assert 190 <= errors <= 210
        assert 790 <= valid <= 810
