from kat.harness import Query
from abstract_tests import AmbassadorTest, ServiceType, HTTP


class LuaTest(AmbassadorTest):
    target: ServiceType

    def init(self):
        self.target = HTTP()

    def config(self):
        # Use self here, not self.target, so that the Ambassador module is
        # be annotated on the Ambassador itself.
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: Module
name: ambassador
config:
  lua_scripts: |
    function envoy_on_response(response_handle)
      response_handle:headers():add("Lua-Scripts-Enabled", "Processed")
    end
""")

        # Use self.target _here_, because we want the mapping to be annotated
        # on the service, not the Ambassador.
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  lua-target-mapping
prefix: /target/
service: {self.target.path.fqdn}
""")

    def queries(self):
        yield Query(self.url("target/"))

    def check(self):
        for r in self.results:
            assert r.headers.get('Lua-Scripts-Enabled', None) == ['Processed']
