from typing import Generator, Tuple, Union

import json

from kat.harness import Query

from abstract_tests import AmbassadorTest, HTTP, AHTTP, Node


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
  type: ClusterIP
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
        image: securityinsanity/stenography:latest  # https://github.com/Mythra/stenography
        env:
        - name: PORT
          value: "25565"
        ports:
        - name: http
          containerPort: 25565
""") + super().manifests()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
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
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  config__dump
hostname: "*"
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
        found_bootstrap_dump = False
        found_clusters_dump = False
        found_listeners_dump = False
        body = json.loads(self.results[0].body)
        for config_obj in body.get('configs'):
            if config_obj.get('@type') == 'type.googleapis.com/envoy.admin.v3.BootstrapConfigDump':
                found_bootstrap_dump = True
                clusters = config_obj.get('bootstrap').get('static_resources').get('clusters')
                found_stenography = False
                assert len(clusters) > 0, "No clusters found"
                for cluster in clusters:
                    if cluster.get('name') == 'cluster_logging_stenography_25565_default':
                        found_stenography = True
                        break
                assert found_stenography

            if config_obj.get('@type') == 'type.googleapis.com/envoy.admin.v3.ClustersConfigDump':
                found_clusters_dump = True
                clusters = config_obj.get('static_clusters')
                found_stenography = False
                assert len(clusters) > 0, "No clusters found"
                for cluster in clusters:
                    if cluster.get('cluster').get('name') == 'cluster_logging_stenography_25565_default':
                        found_stenography = True
                        break
                assert found_stenography

            if config_obj.get('@type') == 'type.googleapis.com/envoy.admin.v3.ListenersConfigDump':
                found_listeners_dump = True
                for listener in config_obj.get('dynamic_listeners'):
                    for filter_chain in listener.get('active_state').get('listener').get('filter_chains'):
                        for filter_obj in filter_chain.get('filters'):
                            access_logs = filter_obj.get('typed_config').get('access_log')
                            found_configured_access_log = False
                            assert len(
                                access_logs) > 0, "No access log configurations found in any listeners filter chains"
                            for access_log in access_logs:
                                if access_log.get('name') == 'envoy.access_loggers.http_grpc' and access_log.get(
                                    'typed_config').get('common_config').get('grpc_service').get('envoy_grpc').get(
                                    'cluster_name') == 'cluster_logging_stenography_25565_default':
                                    found_configured_access_log = True
                                    break
                            assert found_configured_access_log

        assert found_listeners_dump, "Could not find listeners config dump. Did the config dump endpoint work? Did we change Envoy API versions?"
        assert found_clusters_dump, "Could not find clusters config dump. Did the config dump endpoint work? Did we change Envoy API versions?"
        assert found_bootstrap_dump, "Could not find bootstrap config dump. Did the config dump endpoint work? Did we change Envoy API versions?"


class LogServiceTestLongServiceName(AmbassadorTest):
    def init(self):
        self.extra_ports = [25565]
        self.target = HTTP()

    def manifests(self) -> str:
        return self.format("""
---
apiVersion: v1
kind: Service
metadata:
  name: stenographylongservicenamewithnearly60characterss
spec:
  selector:
    app: stenography-longservicename
  ports:
  - port: 25565
    name: http
    targetPort: http
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: stenography-longservicename
spec:
  selector:
    matchLabels:
      app: stenography-longservicename
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: stenography-longservicename
    spec:
      containers:
      - name: stenography
        image: securityinsanity/stenography:latest  # https://github.com/Mythra/stenography
        env:
        - name: PORT
          value: "25565"
        ports:
        - name: http
          containerPort: 25565
""") + super().manifests()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: LogService
name: custom-http-logging
service: stenographylongservicenamewithnearly60characterss:25565
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
name:  config__dump-longservicename
hostname: "*"
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
        found_bootstrap_dump = False
        found_clusters_dump = False
        found_listeners_dump = False
        body = json.loads(self.results[0].body)
        for config_obj in body.get('configs'):
            if config_obj.get('@type') == 'type.googleapis.com/envoy.admin.v3.BootstrapConfigDump':
                found_bootstrap_dump = True
                clusters = config_obj.get('bootstrap').get('static_resources').get('clusters')
                found_stenography = False
                assert len(clusters) > 0, "No clusters found"
                for cluster in clusters:
                    if cluster.get('name') == 'cluster_logging_stenographylongservicena-0':
                        found_stenography = True
                        break
                assert found_stenography

            if config_obj.get('@type') == 'type.googleapis.com/envoy.admin.v3.ClustersConfigDump':
                found_clusters_dump = True
                clusters = config_obj.get('static_clusters')
                found_stenography = False
                assert len(clusters) > 0, "No clusters found"
                for cluster in clusters:
                    if cluster.get('cluster').get('name') == 'cluster_logging_stenographylongservicena-0':
                        found_stenography = True
                        break
                assert found_stenography

            if config_obj.get('@type') == 'type.googleapis.com/envoy.admin.v3.ListenersConfigDump':
                found_listeners_dump = True
                for listener in config_obj.get('dynamic_listeners'):
                    for filter_chain in listener.get('active_state').get('listener').get('filter_chains'):
                        for filter_obj in filter_chain.get('filters'):
                            access_logs = filter_obj.get('typed_config').get('access_log')
                            found_configured_access_log = False
                            assert len(
                                access_logs) > 0, "No access log configurations found in any listeners filter chains"
                            for access_log in access_logs:
                                if access_log.get('name') == 'envoy.access_loggers.http_grpc' and access_log.get(
                                    'typed_config').get('common_config').get('grpc_service').get('envoy_grpc').get(
                                    'cluster_name') == 'cluster_logging_stenographylongservicena-0':
                                    found_configured_access_log = True
                                    break
                            assert found_configured_access_log

        assert found_listeners_dump, "Could not find listeners config dump. Did the config dump endpoint work? Did we change Envoy API versions?"
        assert found_clusters_dump, "Could not find clusters config dump. Did the config dump endpoint work? Did we change Envoy API versions?"
        assert found_bootstrap_dump, "Could not find bootstrap config dump. Did the config dump endpoint work? Did we change Envoy API versions?"
