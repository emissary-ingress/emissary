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
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-normal
prefix: /{self.name}-normal/
service: httpstat.us:80
host_rewrite: httpstat.us
timeout_ms: 10000
""")

        yield self, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-target
prefix: /{self.name}-retry/
service: httpstat.us:80
host_rewrite: httpstat.us
timeout_ms: 10000
retry_policy:
  retry_on: "5xx"
  num_retries: 10
""")

        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  retry_policy:
    retry_on: "retriable-4xx"
    num_retries: 10
""")

    def queries(self):
        yield Query(self.url(self.name + '-normal/200'), expected=200)
        yield Query(self.url(self.name + '-normal/500'), expected=500)
        yield Query(self.url(self.name + '-retry/500'), expected=500)
        yield Query(self.url(self.name + '-normal/409'), expected=409)

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

# main = Runner(AmbassadorTest)
