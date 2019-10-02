import pytest, re

from kat.utils import ShellCommand
from abstract_tests import AmbassadorTest


class DiagdLogLevelTest(AmbassadorTest):
    """Test that no DEBUG logs show up when we set the value to INFO."""
    log_level: str

    def init(self):
        self.debug_diagd = False
        self.log_level = 'INFO'

    def manifests(self) -> str:
        self.manifest_envs = f"""
    - name: AMBASSADOR_LOG_LEVEL
      value: "{self.log_level}"
"""

        return super().manifests()

    def check(self):
        cmd = ShellCommand("kubectl", "log", self.path.k8s)
        if not cmd.check("retrieve ambassador pod logs"):
            pytest.exit("Unable to retrieve logs from ambassador pod")

        # Filter out kubewatch logs since kubewatch is always launched with --debug
        allowed_debug_regex = re.compile('.*kubewatch.*DEBUG:')

        for line in cmd.stdout.splitlines():
            if allowed_debug_regex.match(line):
                continue

            assert 'DEBUG:' not in line


class InvalidLogLevel(AmbassadorTest):
    """Test when an invalid log level is provided for AMBASSADOR_LOG_LEVEL."""

    def manifests(self) -> str:
        self.manifest_envs = """
    - name: AMBASSADOR_LOG_LEVEL
      value: "invalid"
"""
        return super().manifests()

    def check(self):
        cmd = ShellCommand("kubectl", "log", self.path.k8s)
        if not cmd.check("retrieve ambassador pod logs"):
            pytest.exit("Unable to retrieve logs from ambassador pod")

        assert 'Ignoring invalid log level value "INVALID"' in cmd.stdout
