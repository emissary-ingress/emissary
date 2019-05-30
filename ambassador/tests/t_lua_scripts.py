from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP


class LuaTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:
        return super().manifests() + self.format('''
---
apiVersion: getambassador.io/v1
kind: Module
metadata:
  name: ambassador
spec:
  ambassador_id: {self.ambassador_id}
  config:
    lua_scripts: |
      function envoy_on_response(response_handle)
        response_handle: headers():add("Lua-Scripts-Enabled", "Processed")
      end
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: lua-target-mapping
spec:
  ambassador_id: {self.ambassador_id}
  prefix: /target/
  service: {self.target.path.fqdn}
''')

    def queries(self):
        yield Query(self.url("target/"))

    def check(self):
        for r in self.results:
            assert r.headers.get('Lua-Scripts-Enabled', None) == ['Processed']
