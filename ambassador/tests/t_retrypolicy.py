from email.utils import parsedate_to_datetime
from datetime import datetime
from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, MappingTest, ServiceType


class RetryPolicyTest(MappingTest):
    parent: AmbassadorTest
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP()

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-normal
prefix: /{self.name}-normal/
service: httpstat.us:80
host_rewrite: httpstat.us
""")

        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-target
prefix: /{self.name}-retry/
service: httpstat.us:80
host_rewrite: httpstat.us
retry_policy:
  retry_on: "5xx"
  num_retries: 3
  per_try_timeout: "0.5s"
""")

    def queries(self):
        for i in range(5):
            yield Query(self.parent.url(self.name + '-normal/200'), expected=200)

        for i in range(5):
            yield Query(self.parent.url(self.name + '-normal/500'), expected=500)

        for i in range(5):
            yield Query(self.parent.url(self.name + '-retry/500'), expected=500)

    @staticmethod
    def assert_time_difference(results):
        previous_time = None
        for result in results:
            current_time = parsedate_to_datetime(result.headers['Date'][0]).timestamp()
            if previous_time is None:
                previous_time = current_time
            else:
                assert abs(previous_time - current_time) <= 1, \
                    "Difference between previous time {} and current time {} is greater than 1".format(
                        previous_time, current_time)

    @staticmethod
    def get_average_time(results):
        average_time = 0.0
        for result in results:
            average_time += parsedate_to_datetime(result.headers['Date'][0]).timestamp()
        return average_time/len(results)

    def check(self):
        assert len(self.results) == 15

        # we are going to use synchronization_results to sync clocks between envoy and system
        synchronization_results = self.results[0:5]
        normal_results = self.results[5:10]
        retry_results = self.results[10:15]

        self.assert_time_difference(normal_results)
        self.assert_time_difference(retry_results)

        normal_time = self.get_average_time(normal_results)
        retry_time = self.get_average_time(retry_results)

        assert retry_time > normal_time, "retry time {} is not greater than normal time {}".format(retry_time, normal_time)

# main = Runner(AmbassadorTest)
