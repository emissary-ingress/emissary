import re
from datetime import datetime
from typing import Generator, Tuple, Union

from abstract_tests import HTTP, AmbassadorTest, Node, ServiceType
from kat.harness import Query


class RetryPolicyTest(AmbassadorTest):
    target: ServiceType

    def init(self) -> None:
        self.target = HTTP()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield (
            self,
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-normal
hostname: "*"
prefix: /{self.name}-normal/
service: {self.target.path.fqdn}
timeout_ms: 3000
"""
            ),
        )

        yield (
            self,
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-target
hostname: "*"
prefix: /{self.name}-retry/
service: {self.target.path.fqdn}
timeout_ms: 3000
retry_policy:
  retry_on: "5xx"
  num_retries: 4
"""
            ),
        )

        yield (
            self,
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind:  Module
name:  ambassador
config:
  retry_policy:
    retry_on: "retriable-4xx"
    num_retries: 4
"""
            ),
        )

    def queries(self):
        yield Query(
            self.url(self.name + "-normal/"),
            headers={"Kat-Req-Http-Requested-Backend-Delay": "0"},
            expected=200,
        )
        yield Query(
            self.url(self.name + "-normal/"),
            headers={"Kat-Req-Http-Requested-Status": "500"},
            expected=500,
        )
        yield Query(
            self.url(self.name + "-retry/"),
            headers={
                "Kat-Req-Http-Requested-Status": "500",
                "Kat-Req-Http-Requested-Backend-Delay": "2000",
            },
            expected=504,
        )
        yield Query(
            self.url(self.name + "-normal/"),
            headers={
                "Kat-Req-Http-Requested-Status": "409",
                "Kat-Req-Http-Requested-Backend-Delay": "2000",
            },
            expected=504,
        )

    def get_timestamp(self, hdr):
        m = re.match(r"^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{1,6})", hdr)

        if m:
            return datetime.strptime(m.group(1), "%Y-%m-%dT%H:%M:%S.%f").timestamp()
        else:
            assert False, f'header timestamp "{hdr}" is not parseable'
            return None

    def get_duration(self, result):
        start_time = self.get_timestamp(result.headers["Client-Start-Date"][0])
        end_time = self.get_timestamp(result.headers["Client-End-Date"][0])

        return end_time - start_time

    def check(self):
        ok_result = self.results[0]
        normal_result = self.results[1]
        retry_result = self.results[2]
        conflict_result = self.results[3]

        ok_duration = self.get_duration(ok_result)
        normal_duration = self.get_duration(normal_result)
        retry_duration = self.get_duration(retry_result)
        conflict_duration = self.get_duration(conflict_result)

        assert (
            retry_duration >= 2
        ), f"retry time {retry_duration} must be at least 2 seconds"
        assert (
            conflict_duration >= 2
        ), f"conflict time {conflict_duration} must be at least 2 seconds"

        ok_vs_normal = abs(ok_duration - normal_duration)

        assert (
            ok_vs_normal <= 1
        ), f"time to 200 OK {ok_duration} is more than 1 second different from time to 500 {normal_duration}"

        retry_vs_normal = retry_duration - normal_duration

        assert (
            retry_vs_normal >= 2
        ), f"retry time {retry_duration} is not at least 2 seconds slower than normal time {normal_duration}"

        conflict_vs_ok = conflict_duration - ok_duration

        assert (
            conflict_vs_ok >= 2
        ), f"conflict time {conflict_duration} is not at least 2 seconds slower than ok time {ok_duration}"
