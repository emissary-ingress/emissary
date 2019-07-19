from email.utils import parsedate_to_datetime
from datetime import datetime
from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, ServiceType


class RetryPolicyTest(AmbassadorTest):
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-normal
prefix: /{self.name}-normal/
service: {self.target.path.fqdn}
timeout_ms: 3000
""")

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-target
prefix: /{self.name}-retry/
service: {self.target.path.fqdn}
timeout_ms: 3000
retry_policy:
  retry_on: "5xx"
  num_retries: 4
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  retry_policy:
    retry_on: "retriable-4xx"
    num_retries: 4
""")

    def queries(self):
        yield Query(self.url(self.name + '-normal/'), headers={"Requested-Backend-Delay": "0"}, expected=200)
        yield Query(self.url(self.name + '-normal/'), headers={"Requested-Status": "500"}, expected=500)
        yield Query(self.url(self.name + '-retry/'), headers={"Requested-Status": "500", "Requested-Backend-Delay": "2000"}, expected=504)
        yield Query(self.url(self.name + '-normal/'), headers={"Requested-Status": "409", "Requested-Backend-Delay": "2000"}, expected=504)

    def check(self):
        ok_result = self.results[0]
        normal_result = self.results[1]
        retry_result = self.results[2]
        conflict_result = self.results[3]

        ok_time = parsedate_to_datetime(ok_result.headers['Date'][0]).timestamp()
        normal_time = parsedate_to_datetime(normal_result.headers['Date'][0]).timestamp()
        retry_time = parsedate_to_datetime(retry_result.headers['Date'][0]).timestamp()
        conflict_time = parsedate_to_datetime(conflict_result.headers['Date'][0]).timestamp()

        assert abs(ok_time - normal_time) <= 1, "time to get 200 OK {} deviates more than a second from time to get a 500 {}".format(ok_time, normal_time)
        assert retry_time - normal_time >= 2, "retry time {} should be at least 2 seconds slower than normal time {}".format(retry_time, normal_time)
        assert conflict_time - ok_time >= 2, "conflict time {} should be at least 2 seconds slower than 200 OK time {}".format(conflict_time, ok_time)
