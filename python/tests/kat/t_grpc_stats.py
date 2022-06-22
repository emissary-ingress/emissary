import json
from typing import Any, Dict, Generator, List, Literal, Tuple, Union

from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, EGRPC, Node


class GRPCStatsTest(AmbassadorTest):
    target: ServiceType

    variant_name: str
    cfg: Dict[str, Any]
    present_metrics: List[str]
    absent_metrics: List[str]

    @classmethod
    def variants(cls) -> Generator[Node, None, None]:
        for cfgname in ['everything', 'onlyselectedservices', 'noupstreamallmethodsfalseinvalidkeys']:
            yield cls(cfgname=cfgname, name="{self.variant_name}")

    def init(self, cfgname: Literal['everything', 'onlyselectedservices', 'noupstreamallmethodsfalseinvalidkeys']):
        self.target = EGRPC()
        self.variant_name = cfgname
        if cfgname == 'everything':
            self.cfg = {
                'all_methods': True,
                'upstream_stats': True,
            }
            self.present_metrics = [
                'envoy_cluster_grpc_EchoService_0',
                'envoy_cluster_grpc_EchoService_13',
                'envoy_cluster_grpc_EchoService_request_message_count',
                'envoy_cluster_grpc_EchoService_response_message_count',
                'envoy_cluster_grpc_EchoService_success',
                'envoy_cluster_grpc_EchoService_failure',
                'envoy_cluster_grpc_EchoService_total',
                # present only when enable_upstream_stats is true
                'envoy_cluster_grpc_EchoService_upstream_rq_time_bucket',
                'envoy_cluster_grpc_EchoService_upstream_rq_time_count',
                'envoy_cluster_grpc_EchoService_upstream_rq_time_sum',
            ]
            self.absent_metrics = [
                # since all_methods is true, we should not see the generic metrics
                'envoy_cluster_grpc_0',
                'envoy_cluster_grpc_13',
                'envoy_cluster_grpc_request_message_count',
                'envoy_cluster_grpc_response_message_count',
                'envoy_cluster_grpc_success',
                'envoy_cluster_grpc_total',
            ]
        elif cfgname == 'onlyselectedservices':
            self.cfg = {
                'all_methods': True,  # this will be ignored
                'services': [
                    {
                        'name': 'echo.EchoService',
                        'method_names': ['Echo'],
                    },
                ],
            }
            self.present_metrics = [
                'envoy_cluster_grpc_EchoService_0',
                'envoy_cluster_grpc_EchoService_13',
                'envoy_cluster_grpc_EchoService_request_message_count',
                'envoy_cluster_grpc_EchoService_response_message_count',
                'envoy_cluster_grpc_EchoService_success',
                'envoy_cluster_grpc_EchoService_failure',
                'envoy_cluster_grpc_EchoService_total',
            ]
            self.absent_metrics = [
                # upstream stats is disabled
                'envoy_cluster_grpc_upstream_rq_time',
                # the generic metrics shouldn't be present since all the methods being called are on
                # the allowed list
                'envoy_cluster_grpc_0',
                'envoy_cluster_grpc_13',
                'envoy_cluster_grpc_request_message_count',
                'envoy_cluster_grpc_response_message_count',
                'envoy_cluster_grpc_success',
                'envoy_cluster_grpc_total',
            ]
        elif cfgname == 'noupstreamallmethodsfalseinvalidkeys':
            self.cfg = {
                'all_methods': False,
                'upstream_stats': False,
                'i_will_not_break_envoy': True
            }
            # stat_all_methods is disabled and the list of services is empty, so we should only see
            # generic metrics
            self.present_metrics = [
                'envoy_cluster_grpc_0',
                'envoy_cluster_grpc_13',
                'envoy_cluster_grpc_request_message_count',
                'envoy_cluster_grpc_response_message_count',
                'envoy_cluster_grpc_success',
                'envoy_cluster_grpc_failure',
                'envoy_cluster_grpc_total',
            ]
            self.absent_metrics = [
                'envoy_cluster_grpc_upstream_rq_time',
                'envoy_cluster_grpc_EchoService_0'
            ]
        else:
            assert False, f"invalid cfgname={repr(cfgname)}"

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self, f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
config:
    grpc_stats: {json.dumps(self.cfg)}
"""

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
                        headers={ "content-type": "application/grpc", "kat-req-echo-requested-status": "0" },
                        grpc_type="real",
                        phase=1)

        # [1] through [10]
        for i in range(10):
            yield Query(self.url("echo.EchoService/Echo"),
                        headers={ "content-type": "application/grpc", "kat-req-echo-requested-status": "13" },
                        grpc_type="real",
                        phase=1)

        # [-1]
        yield Query(self.url("metrics"), phase=2)


    def check(self):
        stats = {pair[0]: pair[1]
                 for pair in [line.rsplit(" ", maxsplit=1)
                              for line in self.results[-1].text.split("\n") if line.startswith("envoy_cluster_grpc")]}
        stats_shortnames = set(key.split("{")[0] for key in stats.keys())

        print(f'stats_shortnames are: {repr(stats_shortnames)}')

        for metric in self.present_metrics:
            assert metric in stats_shortnames, f'coult not find metric: {metric}'

        for metric in self.absent_metrics:
            assert not any(shortname.startswith(metric) for shortname in stats_shortnames), f'metric {metric} should not be present'

        for metric in stats_shortnames:
            assert metric in self.present_metrics, f"found metric {metric} but it isn't in self.present_metrics"
