import os
from typing import Tuple, Union

from kat.harness import Query
from kat.manifests import AMBASSADOR, RBAC_CLUSTER_SCOPE

from abstract_tests import AmbassadorTest, assert_default_errors, HTTP, Node, ServiceType

class LowPortTest(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:

        eports = f"""
  - name: extra-80
    protocol: TCP
    port: 82
    targetPort: 80
"""
        capabilities_block = f"""
      capabilities:
        add: ["NET_BIND_SERVICE"]
"""
        return self.format(RBAC_CLUSTER_SCOPE + AMBASSADOR,
                           image=os.environ["AMBASSADOR_DOCKER_IMAGE"],
                           envs="",
                           extra_ports=eports,
                           capabilities_block=capabilities_block)

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  service_port: 80
""")

    def queries(self):
        yield Query(self.url("server-name/", "http", 80), expected=399)

    def check(self):
        assert self.results[0].headers["Server"] == [ "test-server" ]

