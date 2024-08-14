import re
from typing import Generator, Tuple, Union

import pytest

from abstract_tests import HTTP, AmbassadorTest, Node, ServiceType
from kat.utils import ShellCommand


class EnvoyLogTest(AmbassadorTest):
    target: ServiceType
    log_path: str

    def init(self):
        self.target = HTTP()
        self.log_path = "/tmp/ambassador/ambassador.log"
        self.log_format = 'MY_REQUEST %RESPONSE_CODE% "%REQ(:AUTHORITY)%" "%REQ(USER-AGENT)%" "%REQ(X-REQUEST-ID)%" "%UPSTREAM_HOST%"'

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield (
            self,
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: [{self.ambassador_id}]
config:
  envoy_log_path: {self.log_path}
  envoy_log_format: {self.log_format}
"""
            ),
        )

    def check(self):
        access_log_entry_regex = re.compile("^MY_REQUEST 200 .*")

        cmd = ShellCommand(
            "tools/bin/kubectl", "exec", self.path.k8s, "cat", self.log_path
        )
        if not cmd.check("check envoy access log"):
            pytest.exit("envoy access log does not exist")

        for line in cmd.stdout.splitlines():
            assert access_log_entry_regex.match(
                line
            ), f"{line} does not match {access_log_entry_regex}"


class EnvoyLogJSONTest(AmbassadorTest):
    target: ServiceType
    log_path: str

    def init(self):
        self.target = HTTP()
        self.log_path = "/tmp/ambassador/ambassador.log"

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield (
            self,
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: [{self.ambassador_id}]
config:
  envoy_log_path: {self.log_path}
  envoy_log_format:
    protocol: "%PROTOCOL%"
    duration: "%DURATION%"
  envoy_log_type: json
"""
            ),
        )

    def check(self):
        access_log_entry_regex = re.compile('^({"duration":|{"protocol":)')

        cmd = ShellCommand(
            "tools/bin/kubectl", "exec", self.path.k8s, "cat", self.log_path
        )
        if not cmd.check("check envoy access log"):
            pytest.exit("envoy access log does not exist")

        for line in cmd.stdout.splitlines():
            assert access_log_entry_regex.match(
                line
            ), f"{line} does not match {access_log_entry_regex}"


class EnvoyLogTypeJSONTest(AmbassadorTest):
    target: ServiceType
    log_path: str

    def init(self):
        self.target = HTTP()
        self.log_path = "/tmp/ambassador/ambassador.log"

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield (
            self,
            self.format(
                """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
name: ambassador
ambassador_id: [{self.ambassador_id}]
config:
  envoy_log_path: {self.log_path}
  envoy_log_format:
    protocol: "%PROTOCOL%"
    duration: "%DURATION%"
  envoy_log_type: typed_json
"""
            ),
        )

    def check(self):
        access_log_entry_regex = re.compile('^({"duration":|{"protocol":)')

        cmd = ShellCommand(
            "tools/bin/kubectl", "exec", self.path.k8s, "cat", self.log_path
        )
        if not cmd.check("check envoy access log"):
            pytest.exit("envoy access log does not exist")

        for line in cmd.stdout.splitlines():
            assert access_log_entry_regex.match(
                line
            ), f"{line} does not match {access_log_entry_regex}"
