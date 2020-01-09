from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP

class IdleTimeout(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: Module
name: ambassador
ambassador_id: {self.ambassador_id}
config:
  idle_timeout: '30s'
""")

    def queries(self):
        yield Query(self.url("ambassador/v0/diag/?json=true"), phase=2)

    def check(self):
        expected_val = '30s'
        assert self.results[0].json['envoy_elements']['idletimeout.default.1']['listener'][0]['filter_chains'][0]['filters'][0]['config'].get('common_http_protocol_options', False), "expected common_http_protocol_options to be present"
        got_val = self.results[0].json['envoy_elements']['idletimeout.default.1']['listener'][0]['filter_chains'][0]['filters'][0]['config']['common_http_protocol_options'].get('idle_timeout')
        assert expected_val == got_val, "expected idle_timeout to be {}, got {}".format(expected_val, got_val)