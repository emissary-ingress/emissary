from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP


class LuaTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

        self.manifest_envs = """
    - name: LUA_SCRIPTS_ENABLED
      value: Processed
"""

    def manifests(self) -> str:
        return self.format('''
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  ambassador_id: {self.ambassador_id}
  config:
    lua_scripts: |
      function envoy_on_response(response_handle)
        response_handle: headers():add("Lua-Scripts-Enabled", "$LUA_SCRIPTS_ENABLED")
      end
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: lua-target-mapping
spec:
  ambassador_id: {self.ambassador_id}
  prefix: /target/
  service: {self.target.path.fqdn}
''') + super().manifests()

    def queries(self):
        yield Query(self.url("target/"))

    def check(self):
        for r in self.results:
            assert r.headers.get('Lua-Scripts-Enabled', None) == ['Processed']
