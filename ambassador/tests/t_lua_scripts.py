from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP


class LuaTest(AmbassadorTest):
    namespace = 'lua'

    configs = {
        'CRD': '''
---
apiVersion: getambassador.io/v1
kind: Module
name: ambassador
config:
  lua_scripts: |
    function envoy_on_response(response_handle)
      response_handle: headers():add("Lua-Scripts-Enabled", "Processed")
    end
---
apiVersion: getambassador.io/v1
kind: Mapping
name: lua-target-mapping
prefix: /target/
service: {self.target.path.fqdn}
'''
    }

    target: ServiceType

    upstreams = {
        'target': {
            'servicetype': 'HTTP'
        }
    }

    def queries(self):
        yield Query(self.url("target/"))

    def check(self):
        for r in self.results:
            assert r.headers.get('Lua-Scripts-Enabled', None) == ['Processed']
