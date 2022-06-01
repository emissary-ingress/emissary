from typing import Generator, Tuple, Union

import json

from kat.harness import Query

from abstract_tests import AmbassadorTest, ServiceType, HTTP, ALSGRPC, Node


class LogServiceTest(AmbassadorTest):
    target: ServiceType
    als: ServiceType

    def init(self):
        self.target = HTTP()
        self.als = ALSGRPC()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
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
      """)
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  accesslog_target_mapping
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(f"http://{self.als.path.fqdn}/logs", method='DELETE', phase=1)
        yield Query(self.url("target/foo"), phase=2)
        yield Query(self.url("target/bar"), phase=3)
        yield Query(f"http://{self.als.path.fqdn}/logs", phase=4)

    def check(self):
        logs = self.results[3].json
        assert logs['alsv2-http']
        assert not logs['alsv2-tcp']
        assert not logs['alsv3-http']
        assert not logs['alsv3-tcp']

        assert len(logs['alsv2-http']) == 2
        assert logs['alsv2-http'][0]['request']['original_path'] == '/target/foo'
        assert logs['alsv2-http'][1]['request']['original_path'] == '/target/bar'


class LogServiceLongServiceNameTest(AmbassadorTest):
    target: ServiceType
    als: ServiceType

    def init(self):
        self.target = HTTP()
        self.als = ALSGRPC()

    def manifests(self) -> str:
        return self.format("""
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
""") + super().manifests()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
name: custom-http-logging
service: logservicelongservicename-longnamewithnearly60characters
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
      """)
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  accesslog_target_mapping
hostname: "*"
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(f"http://{self.als.path.fqdn}/logs", method='DELETE', phase=1)
        yield Query(self.url("target/foo"), phase=2)
        yield Query(self.url("target/bar"), phase=3)
        yield Query(f"http://{self.als.path.fqdn}/logs", phase=4)

    def check(self):
        logs = self.results[3].json
        assert logs['alsv2-http']
        assert not logs['alsv2-tcp']
        assert not logs['alsv3-http']
        assert not logs['alsv3-tcp']

        assert len(logs['alsv2-http']) == 2
        assert logs['alsv2-http'][0]['request']['original_path'] == '/target/foo'
        assert logs['alsv2-http'][1]['request']['original_path'] == '/target/bar'
