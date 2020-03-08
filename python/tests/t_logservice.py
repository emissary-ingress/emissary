import json
import pytest

from typing import ClassVar, Dict, List, Sequence, Tuple, Union

from kat.harness import sanitize, variants, Query, Runner

from abstract_tests import AmbassadorTest, HTTP, AHTTP
from abstract_tests import MappingTest, OptionTest, ServiceType, Node, Test


class LogServiceTest(AmbassadorTest):
    def init(self):
        self.extra_ports = [25565]
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format("""
---
apiVersion: v1
kind: Service
metadata:
  name: stenography
spec:
  selector:
    app: stenography
  ports:
  - port: 25565
    name: http
    targetPort: http
  type: NodePort
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: stenography
spec:
  selector:
    matchLabels:
      app: stenography
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: stenography
    spec:
      containers:
      - name: stenography
        image: securityinsanity/stenography:latest
        imagePullPolicy: Always
        env:
        - name: PORT
          value: "25565"
        ports:
        - name: http
          containerPort: 25565
""") + super().manifests()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: LogService
name: custom-http-logging
service: stenography:25565
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
apiVersion: ambassador/v0
kind:  Mapping
name:  config__dump
prefix: /config_dump
rewrite: /config_dump
service: http://127.0.0.1:8001
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query(self.url("config_dump")))

    def queries(self):
      yield Query(self.url("config_dump"), phase=2)

    def check(self):
        body = json.loads(self.results[0].body)
        for config_obj in body.get('configs'):
          if config_obj.get('@type') == 'type.googleapis.com/envoy.admin.v2alpha.BootstrapConfigDump':
            clusters = config_obj.get('bootstrap').get('static_resources').get('clusters')
            found_stenography = False
            for cluster in clusters:
              if cluster.get('name') == 'cluster_logging_stenography_25565_default':
                found_stenography = True
                break
            assert found_stenography

          if config_obj.get('@type') == 'type.googleapis.com/envoy.admin.v2alpha.ClustersConfigDump':
              clusters = config_obj.get('static_clusters')
              found_stenography = False
              for cluster in clusters:
                if cluster.get('cluster').get('name') == 'cluster_logging_stenography_25565_default':
                  found_stenography = True
                  break
              assert found_stenography

          if config_obj.get('@type') == 'type.googleapis.com/envoy.admin.v2alpha.ListenersConfigDump':
            for dal in config_obj.get('dynamic_active_listeners'):
              for filter_chain in dal.get('listener').get('filter_chains'):
                for filter_obj in filter_chain.get('filters'):
                  access_logs = filter_obj.get('typed_config').get('access_log')
                  found_configured_access_log = False
                  for access_log in access_logs:
                    if access_log.get('name') == 'envoy.http_grpc_access_log' and access_log.get('config').get('common_config').get('grpc_service').get('envoy_grpc').get('cluster_name') == 'cluster_logging_stenography_25565_default':
                      found_configured_access_log = True
                      break

                  assert found_configured_access_log
