from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP
import json

class ListenerIdleTimeout(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
ambassador_id: {self.ambassador_id}
config:
  listener_idle_timeout_ms: 30000
""")
        yield self, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  config__dump
hostname: "*"
prefix: /config_dump
rewrite: /config_dump
service: http://127.0.0.1:8001
""")

    def queries(self):
        yield Query(self.url("config_dump"), phase=2)

    def check(self):
        expected_val = '30s'
        actual_val = ''
        body = json.loads(self.results[0].body)
        for config_obj in body.get('configs'):
          if config_obj.get('@type') == 'type.googleapis.com/envoy.admin.v3.ListenersConfigDump':
            listeners = config_obj.get('dynamic_listeners')
            found_idle_timeout = False
            for listener_obj in listeners:
              listener = listener_obj.get('active_state').get('listener')
              filter_chains = listener.get('filter_chains')
              for filters in filter_chains:
                for filter in filters.get('filters'):
                  if filter.get('name') == 'envoy.filters.network.http_connection_manager':
                    filter_config = filter.get('typed_config')
                    common_http_protocol_options = filter_config.get('common_http_protocol_options')
                    if common_http_protocol_options:
                      actual_val = common_http_protocol_options.get('idle_timeout', '')
                      if actual_val != '':
                        if actual_val == expected_val:
                          found_idle_timeout = True
                      else:
                        assert False, "Expected to find common_http_protocol_options.idle_timeout property on listener"
                    else:
                      assert False, "Expected to find common_http_protocol_options property on listener"
            assert found_idle_timeout, "Expected common_http_protocol_options.idle_timeout = {}, Got common_http_protocol_options.idle_timeout = {}".format(expected_val, actual_val)
