import pytest, re

from kat.utils import ShellCommand
from abstract_tests import AmbassadorTest, ServiceType, HTTP

access_log_entry_regex = re.compile('^ACCESS \\[.*?\\] \\\"GET \\/ambassador')


class EnvoyLogPathTest(AmbassadorTest):
    target: ServiceType
    log_path: str

    def init(self):
        self.target = HTTP()
        self.log_path = '/tmp/ambassador/ambassador.log'

    def config(self):
        yield self, self.format("""
---
apiVersion: ambassador/v1
kind: Module
name: ambassador
ambassador_id: {self.ambassador_id}
config:
  envoy_log_path: {self.log_path}
""")

    def check(self):
        cmd = ShellCommand("kubectl", "exec", self.path.k8s, "cat", self.log_path)
        if not cmd.check("check envoy access log"):
            pytest.exit("envoy access log does not exist")

        for line in cmd.stdout.splitlines():
            assert access_log_entry_regex.match(line)
