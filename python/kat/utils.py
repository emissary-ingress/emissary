from base64 import b64encode
import json
import os
import re
import subprocess


_quote_pos = re.compile('(?=[^-0-9a-zA-Z_./\n])')

def quote(arg):
    r"""
    >>> quote('\t')
    '\\\t'
    >>> quote('foo bar')
    'foo\\ bar'
    """

    # This is the logic emacs uses
    if arg:
        return _quote_pos.sub('\\\\', arg).replace('\n', "'\n'")
    else:
        return "''"


class ShellCommand:
    def __init__(self, *args, **kwargs) -> None:
        self.verbose = kwargs.pop('verbose', False)

        for arg in "stdout", "stderr":
            if arg not in kwargs:
                kwargs[arg] = subprocess.PIPE

        self.cmdline = " ".join([quote(x) for x in args])

        if self.verbose:
            print(f'---- running: {self.cmdline}')

        self.proc = subprocess.run(args, **kwargs)

    def status(self) -> bool:
        try:
            self.proc.check_returncode()
            return True
        except Exception as e:
            return False

    def check(self, what: str) -> bool:
        if self.status():
            return True
        else:
            print(f"==== COMMAND FAILED: {what}")
            print("---- command line ----")
            print(self.cmdline)
            print("---- stdout ----")
            print(self.stdout)
            print("")
            print("---- stderr ----")
            print(self.stderr)

            return False

    @property
    def stdout(self) -> str:
        return self.proc.stdout.decode("utf-8")

    @property
    def stderr(self) -> str:
        return self.proc.stderr.decode("utf-8")

    @classmethod
    def run(cls, what: str, *args, **kwargs) -> bool:
        return ShellCommand(*args, **kwargs).check(what)

def namespace_manifest(namespace):
    ret = f"""
---
apiVersion: v1
kind: Namespace
metadata:
  name: {namespace}
"""

    if os.environ.get("DEV_USE_IMAGEPULLSECRET", None):
        dockercfg = {
            "auths": {
                os.path.dirname(os.environ['DEV_REGISTRY']): {
                    "auth": b64encode((os.environ['DOCKER_BUILD_USERNAME']+":"+os.environ['DOCKER_BUILD_PASSWORD']).encode("utf-8")).decode("utf-8")
                }
            }
        }
        ret += f"""
---
apiVersion: v1
kind: Secret
metadata:
  name: dev-image-pull-secret
  namespace: {namespace}
type: kubernetes.io/dockerconfigjson
data:
  ".dockerconfigjson": "{b64encode(json.dumps(dockercfg).encode("utf-8")).decode("utf-8")}"
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: default
  namespace: {namespace}
imagePullSecrets:
- name: dev-image-pull-secret
"""

    return ret
