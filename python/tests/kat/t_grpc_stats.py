from kat.harness import Query

from abstract_tests import AmbassadorTest, EGRPC

class AcceptanceGrpcStatsTest(AmbassadorTest):
    def init(self):
        self.target = EGRPC()

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
config:
    grpc_stats:
        all_methods: true
        upstream_stats: true
""")

        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
grpc: True
hostname: "*"
prefix: /echo.EchoService/
rewrite: /echo.EchoService/
name:  {self.target.path.k8s}
service: {self.target.path.k8s}
""")

        yield self, self.format("""
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  metrics
hostname: "*"
prefix: /metrics
rewrite: /metrics
service: http://127.0.0.1:8877
""")


    def queries(self):
        # [0]
        for i in range(10):
            yield Query(self.url("echo.EchoService/Echo"),
                        headers={ "content-type": "application/grpc", "requested-status": "0" },
                        grpc_type="real",
                        phase=1)

        for i in range(10):
            yield Query(self.url("echo.EchoService/Echo"),
                        headers={ "content-type": "application/grpc", "requested-status": "13" },
                        grpc_type="real",
                        phase=1)

        # [1]
        yield Query(self.url("metrics"), phase=2)


    def check(self):
        # [0]
        stats = self.results[-1].text

        metrics = [
            'envoy_cluster_grpc_EchoService_0',
            'envoy_cluster_grpc_EchoService_13',
            'envoy_cluster_grpc_EchoService_request_message_count',
            'envoy_cluster_grpc_EchoService_response_message_count',
            'envoy_cluster_grpc_EchoService_success',
            'envoy_cluster_grpc_EchoService_total',
            # present only when enable_upstream_stats is true
            'envoy_cluster_grpc_EchoService_upstream_rq_time'
        ]

        # these metrics SHOULD NOT be there based on the filter config
        absent_metrics = [
            # since all_methods is true, we should not see the generic metrics
            'envoy_cluster_grpc_0'
            'envoy_cluster_grpc_13',
            'envoy_cluster_grpc_request_message_count',
            'envoy_cluster_grpc_response_message_count',
            'envoy_cluster_grpc_success',
            'envoy_cluster_grpc_total'
        ]

        # check if the metrics are there
        for metric in metrics:
            assert metric in stats, f'coult not find metric: {metric}'

        for absent_metric in absent_metrics:
            assert absent_metric not in stats, f'metric {metric} should not be present'

        # check if the metrics are there
        for metric in metrics:
            assert metric in stats, f'coult not find metric: {metric}'


class GrpcStatsTestOnlySelectedServices(AmbassadorTest):
    def init(self):
        self.target = EGRPC()

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
config:
    grpc_stats:
        # this will be ignored
        all_methods: true
        services:
            - name: echo.EchoService
              method_names: [Echo]
""")

        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
grpc: True
hostname: "*"
prefix: /echo.EchoService/
rewrite: /echo.EchoService/
name:  {self.target.path.k8s}
service: {self.target.path.k8s}
""")

        yield self, self.format("""
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  metrics
hostname: "*"
prefix: /metrics
rewrite: /metrics
service: http://127.0.0.1:8877
""")


    def queries(self):
        # [0]
        for i in range(10):
            yield Query(self.url("echo.EchoService/Echo"),
                        headers={ "content-type": "application/grpc", "requested-status": "0" },
                        grpc_type="real",
                        phase=1)

        for i in range(10):
            yield Query(self.url("echo.EchoService/Echo"),
                        headers={ "content-type": "application/grpc", "requested-status": "13" },
                        grpc_type="real",
                        phase=1)

        # [1]
        yield Query(self.url("metrics"), phase=2)


    def check(self):
        # [0]
        stats = self.results[-1].text

        metrics = [
            'envoy_cluster_grpc_EchoService_0',
            'envoy_cluster_grpc_EchoService_13',
            'envoy_cluster_grpc_EchoService_request_message_count',
            'envoy_cluster_grpc_EchoService_response_message_count',
            'envoy_cluster_grpc_EchoService_success',
            'envoy_cluster_grpc_EchoService_total',
        ]

        # these metrics SHOULD NOT be there based on the filter config
        absent_metrics = [
            # upstream stats is disabled
            'envoy_cluster_grpc_upstream_rq_time',
            # the generic metrics shouldn't be present since all the methods being called are on the allowed list
            'envoy_cluster_grpc_0'
            'envoy_cluster_grpc_13',
            'envoy_cluster_grpc_request_message_count',
            'envoy_cluster_grpc_response_message_count',
            'envoy_cluster_grpc_success',
            'envoy_cluster_grpc_total'
        ]

        # check if the metrics are there
        for metric in metrics:
            assert metric in stats, f'coult not find metric: {metric}'

        for absent_metric in absent_metrics:
            assert absent_metric not in stats, f'metric {metric} should not be present'

class GrpcStatsTestNoUpstreamAllMethodsFalseInvalidKeys(AmbassadorTest):
    def init(self):
        self.target = EGRPC()

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
config:
    grpc_stats:
        all_methods: false
        upstream_stats: false
        i_will_not_break_envoy: true
""")

        yield self, self.format("""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
grpc: True
hostname: "*"
prefix: /echo.EchoService/
rewrite: /echo.EchoService/
name:  {self.target.path.k8s}
service: {self.target.path.k8s}
""")

        yield self, self.format("""
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  metrics
hostname: "*"
prefix: /metrics
rewrite: /metrics
service: http://127.0.0.1:8877
""")


    def queries(self):
        # [0]
        for i in range(10):
            yield Query(self.url("echo.EchoService/Echo"),
                        headers={ "content-type": "application/grpc", "requested-status": "0" },
                        grpc_type="real",
                        phase=1)

        for i in range(10):
            yield Query(self.url("echo.EchoService/Echo"),
                        headers={ "content-type": "application/grpc", "requested-status": "13" },
                        grpc_type="real",
                        phase=1)

        # [1]
        yield Query(self.url("metrics"), phase=2)


    def check(self):
        # [0]
        stats = self.results[-1].text

        # stat_all_methods is disabled and the list of services is empty, so we should only see generic metrics
        metrics = [
            'envoy_cluster_grpc_0',
            'envoy_cluster_grpc_13',
            'envoy_cluster_grpc_request_message_count',
            'envoy_cluster_grpc_response_message_count',
            'envoy_cluster_grpc_success',
            'envoy_cluster_grpc_total',
        ]

        # these metrics SHOULD NOT be there based on the filter config
        absent_metrics = [
            'envoy_cluster_grpc_upstream_rq_time',
            'envoy_cluster_grpc_EchoService_0'
        ]

        # check if the metrics are there
        for metric in metrics:
            assert metric in stats, f'coult not find metric: {metric}'

        for absent_metric in absent_metrics:
            assert absent_metric not in stats, f'metric {metric} should not be present'
