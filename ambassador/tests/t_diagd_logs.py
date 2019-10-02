import pytest, re

from kat.utils import ShellCommand
from abstract_tests import AmbassadorTest


class InvalidLogLevel(AmbassadorTest):
    """Test when an invalid log level is provided for AMBASSADOR_LOG_LEVEL"""

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
