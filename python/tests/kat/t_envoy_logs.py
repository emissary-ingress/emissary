import pytest, re

from kat.harness import EDGE_STACK

from kat.utils import ShellCommand
from abstract_tests import AmbassadorTest, ServiceType, HTTP


class EnvoyLogTest(AmbassadorTest):
    target: ServiceType
    log_path: str

    def init(self):
        if EDGE_STACK:
            self.xfail = "Not yet supported in Edge Stack"

        self.target = HTTP()
        self.log_path = '/tmp/ambassador/ambassador.log'
        self.log_format = 'MY_REQUEST %RESPONSE_CODE% \"%REQ(:AUTHORITY)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%UPSTREAM_HOST%\"'

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
ambassador_id: {self.ambassador_id}
config:
  envoy_log_path: {self.log_path}
  envoy_log_format: {self.log_format}
""")

    def check(self):
        access_log_entry_regex = re.compile('^MY_REQUEST 200 .*')

        cmd = ShellCommand("kubectl", "exec", self.path.k8s, "cat", self.log_path)
        if not cmd.check("check envoy access log"):
            pytest.exit("envoy access log does not exist")

        for line in cmd.stdout.splitlines():
            assert access_log_entry_regex.match(line), f"{line} does not match {access_log_entry_regex}"


class EnvoyLogJSONTest(AmbassadorTest):
    target: ServiceType
    log_path: str

    def init(self):
        self.target = HTTP()
        self.log_path = '/tmp/ambassador/ambassador.log'

    def config(self):
        yield self, self.format("""
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
ambassador_id: {self.ambassador_id}
config:
  envoy_log_path: {self.log_path}
  envoy_log_format:
    protocol: "%PROTOCOL%"
    duration: "%DURATION%"
  envoy_log_type: json
""")

    def check(self):
        access_log_entry_regex = re.compile('^({"duration":|{"protocol":)')

        cmd = ShellCommand("kubectl", "exec", self.path.k8s, "cat", self.log_path)
        if not cmd.check("check envoy access log"):
            pytest.exit("envoy access log does not exist")

        for line in cmd.stdout.splitlines():
            assert access_log_entry_regex.match(line), f"{line} does not match {access_log_entry_regex}"
