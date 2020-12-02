import os
import re

import pytest

from abstract_tests import AmbassadorTest, HTTP, ServiceType
from kat.harness import Query, load_manifest

AMBASSADOR = load_manifest("ambassador")
RBAC_CLUSTER_SCOPE = load_manifest("rbac_cluster_scope")

class DroppedCapabilitiesStillWorks(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:

        capabilities_block = f"""
      capabilities:
        drop: ["NET_BIND_SERVICE"]
"""

        return self.format(RBAC_CLUSTER_SCOPE + AMBASSADOR,
                           image=os.environ["AMBASSADOR_DOCKER_IMAGE"],
                           envs="",
                           extra_ports="",
                           capabilities_block=capabilities_block)

    def config(self):
        yield self, self.format("""
""")

    def queries(self):
        yield Query(self.url("server-name/"), expected=404)

    def check(self):
        assert self.results[0].headers["Server"] == [ "envoy" ]


class CanBindToLowPort(AmbassadorTest):

    target: ServiceType

    def init(self):
        self.target = HTTP()

    def manifests(self) -> str:

        port_block = f"""
    ports:
      - containerPort: 81
"""
        capabilities_block = f"""
      capabilities:
        add: ["NET_BIND_SERVICE"]
"""

        ambassador_new_ports = re.sub(r'targetPort: 8080\b', r'targetPort: 81', AMBASSADOR)
        ambassador_new_ports = re.sub(r'(image: .*)', r'\1' + port_block, ambassador_new_ports)
        ambassador_new_ports = re.sub(r'allowPrivilegeEscalation: false', r'allowPrivilegeEscalation: true', ambassador_new_ports)
        return self.format(RBAC_CLUSTER_SCOPE + ambassador_new_ports,
                           image=os.environ["AMBASSADOR_DOCKER_IMAGE"],
                           envs="",
                           extra_ports="",
                           capabilities_block=capabilities_block)

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  service_port: 81
""")

    def queries(self):
        # sean~/play/ambassador (use_capabilities_wrapper)$ kubectl get pod canbindtolowport -o go-template --template '{{range .status.containerStatuses}}{{.containerID}}{{end}}'
        # docker://91e59d6864eee6ad97d48119aca37f829ee1c7e00dc7a8d15f672cddbceda9b1
        # sean~/play/ambassador (use_capabilities_wrapper)$ docker inspect --format='{{.HostConfig.CapAdd}}'  91e59d6864eee6ad97d48119aca37f829ee1c7e00dc7a8d15f672cddbceda9b1
        # [NET_BIND_SERVICE]
        if sys.platform != 'darwin':
            pytest.xfail('This only works on Darwin')
        yield Query(self.url("server-name/", "http", 80), expected=404)

    def check(self):
        assert self.results[0].headers["Server"] == [ "envoy" ]
