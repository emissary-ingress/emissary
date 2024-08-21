from typing import Generator, Literal, Tuple, Union, cast

from abstract_tests import ALSGRPC, HTTP, AmbassadorTest, Node, ServiceType
from kat.harness import Query


class LogServiceTest(AmbassadorTest):
    target: ServiceType
    specified_protocol_version: Literal["v2", "v3", "default"]
    expected_protocol_version: Literal["v3", "invalid"]
    als: ServiceType

    @classmethod
    def variants(cls) -> Generator[Node, None, None]:
        for protocol_version in ["v2", "v3", "default"]:
            yield cls(protocol_version, name="{self.specified_protocol_version}")

    def init(self, protocol_version: Literal["v3", "default"]):
        self.target = HTTP()
        self.specified_protocol_version = protocol_version
        self.expected_protocol_version = cast(
            Literal["v3", "invalid"],
            protocol_version if protocol_version in ["v3"] else "invalid",
        )
        self.als = ALSGRPC()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield (
            self,
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
name: custom-http-logging
service: {self.als.path.fqdn}
grpc: true
driver: http
driver_config:
  additional_log_headers:
    - header_name: "included-on-all"
    - header_name: "not-included-on-trailer"
      during_trailer: false
    - header_name: "not-included on resp-trail"
      during_trailer: false
      during_response: false
    - header_name: "not-anywhere"
      during_trailer: false
      during_response: false
      during_request: false
flush_interval_time: 1
flush_interval_byte_size: 1
"""
            )
            + (
                ""
                if self.specified_protocol_version == "default"
                else f"protocol_version: '{self.specified_protocol_version}'"
            ),
        )
        yield (
            self,
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  accesslog_target_mapping
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
"""
            ),
        )

    def queries(self):
        yield Query(f"http://{self.als.path.fqdn}/logs", method="DELETE", phase=1)
        yield Query(self.url("target/foo"), phase=2)
        yield Query(self.url("target/bar"), phase=3)
        yield Query(f"http://{self.als.path.fqdn}/logs", phase=4)

    def check(self):
        logs = self.results[3].json
        expkey = f"als{self.expected_protocol_version}-http"
        for key in ["alsv2-http", "alsv2-tcp", "alsv3-http", "alsv3-tcp"]:
            if key == expkey:
                continue
            assert not logs[key]

        if self.expected_protocol_version == "invalid":
            assert expkey not in logs
            return

        assert logs[expkey]
        assert len(logs[expkey]) == 2
        assert logs[expkey][0]["request"]["original_path"] == "/target/foo"
        assert logs[expkey][1]["request"]["original_path"] == "/target/bar"


class LogServiceLongServiceNameTest(AmbassadorTest):
    target: ServiceType
    als: ServiceType

    def init(self):
        self.target = HTTP()
        self.als = ALSGRPC()

    def manifests(self) -> str:
        return (
            self.format(
                """
---
kind: Service
apiVersion: v1
metadata:
  name: logservicelongservicename-longnamewithnearly60characters
spec:
  selector:
    backend: {self.als.path.k8s}
  ports:
  - name: http
    protocol: TCP
    port: 80
    targetPort: 8080
  - name: https
    protocol: TCP
    port: 443
    targetPort: 8443
"""
            )
            + super().manifests()
        )

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield (
            self,
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
name: custom-http-logging
service: logservicelongservicename-longnamewithnearly60characters
grpc: true
protocol_version: "v3"
driver: http
driver_config:
  additional_log_headers:
    - header_name: "included-on-all"
    - header_name: "not-included-on-trailer"
      during_trailer: false
    - header_name: "not-included on resp-trail"
      during_trailer: false
      during_response: false
    - header_name: "not-anywhere"
      during_trailer: false
      during_response: false
      during_request: false
flush_interval_time: 1
flush_interval_byte_size: 1
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
name:  accesslog_target_mapping
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
"""
            ),
        )

    def queries(self):
        yield Query(f"http://{self.als.path.fqdn}/logs", method="DELETE", phase=1)
        yield Query(self.url("target/foo"), phase=2)
        yield Query(self.url("target/bar"), phase=3)
        yield Query(f"http://{self.als.path.fqdn}/logs", phase=4)

    def check(self):
        logs = self.results[3].json
        assert not logs["alsv2-http"]
        assert not logs["alsv2-tcp"]
        assert logs["alsv3-http"]
        assert not logs["alsv3-tcp"]

        assert len(logs["alsv3-http"]) == 2
        assert logs["alsv3-http"][0]["request"]["original_path"] == "/target/foo"
        assert logs["alsv3-http"][1]["request"]["original_path"] == "/target/bar"
