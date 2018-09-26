import os

import shutil

from abc import abstractmethod
from typing import Any, Iterable, Optional, Sequence, Type

from kat.harness import abstract_test, sanitize, variant, variants, Node, Test
from kat import manifests

AMBASSADOR_LOCAL = """
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRole
metadata:
  name: {self.path.k8s}
rules:
- apiGroups: [""]
  resources:
  - services
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources:
  - configmaps
  verbs: ["create", "update", "patch", "get", "list", "watch"]
- apiGroups: [""]
  resources:
  - secrets
  verbs: ["get", "list", "watch"]
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {self.path.k8s}
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: {self.path.k8s}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: {self.path.k8s}
subjects:
- kind: ServiceAccount
  name: {self.path.k8s}
  namespace: default
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}
  annotations:
    kubernetes.io/service-account.name: {self.path.k8s}
type: kubernetes.io/service-account-token
"""

import base64, subprocess, sys, tempfile, time, yaml

def run(*args, **kwargs):
    for arg in "stdout", "stderr":
        if arg not in kwargs:
            kwargs[arg] = subprocess.PIPE
    return subprocess.run(args, **kwargs)

DEV = os.environ.get("AMBASSADOR_DEV", "0").lower() in ("1", "yes", "true")

@abstract_test
class AmbassadorTest(Test):

    OFFSET = 0

    @classmethod
    def variants(cls):
        yield variant(variants(MappingTest))

    def __init__(self, mappings = ()):
        self.mappings = list(mappings)
        self.index = AmbassadorTest.OFFSET
        AmbassadorTest.OFFSET += 1

    def manifests(self) -> str:
        if DEV:
            return self.format(AMBASSADOR_LOCAL)
        else:
            return self.format(manifests.AMBASSADOR, image=os.environ["AMBASSADOR_DOCKER_IMAGE"])

    IMAGE_BUILT = False

    def pre_query(self):
        if not DEV: return

        run("docker", "kill", self.path.k8s)
        run("docker", "rm", self.path.k8s)

        image = os.environ["AMBASSADOR_DOCKER_IMAGE"]

        if not AmbassadorTest.IMAGE_BUILT:
            AmbassadorTest.IMAGE_BUILT = True
            context = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))
            print("Starting docker build...", end="")
            sys.stdout.flush()
            result = run("docker", "build", context, "-t", image)
            try:
                result.check_returncode()
                print("done.")
            except Exception as e:
                raise RuntimeError(result.stdout + b"\n" + result.stderr) from e

        fname = "/tmp/k8s-%s.yaml" % self.path.k8s
        if os.path.exists(fname):
            with open(fname) as fd:
                content = fd.read()
        else:
            result = run("kubectl", "get", "-o", "yaml", "secret", self.path.k8s)
            result.check_returncode()
            with open(fname, "wb") as fd:
                fd.write(result.stdout)
            content = result.stdout
        secret = yaml.load(content)
        data = secret['data']
        # dir = tempfile.mkdtemp(prefix=self.path.k8s, suffix="secret")
        dir = "/tmp/%s-screwoff-%s" % (self.path.k8s, 'secret')

        shutil.rmtree(dir, ignore_errors=True)
        os.mkdir(dir, 0o777)

        for k, v in data.items():
            with open(os.path.join(dir, k), "wb") as f:
                f.write(base64.decodebytes(bytes(v, "utf8")))
        print("Launching %s container." % self.path.k8s)
        result = run("docker", "run", "-d", "--name", self.path.k8s,
                     "-p", "%s:8877" % (8877 + self.index),
                     "-p", "%s:80" % (8080 + self.index),
                     "-v", "%s:/var/run/secrets/kubernetes.io/serviceaccount" % dir,
                     "-e", "KUBERNETES_SERVICE_HOST=kubernetes",
                     "-e", "KUBERNETES_SERVICE_PORT=443",
                     "-e", "AMBASSADOR_ID=%s" % self.path.k8s,
                     image)
        result.check_returncode()
        self.deadline = time.time() + 3

    def queries(self):
        if DEV:
            now = time.time()
            if now < self.deadline:
                time.sleep(self.deadline - now)
        return ()

    @abstractmethod
    def scheme(self) -> str:
        pass

    def url(self, prefix) -> str:
        if DEV:
            return "%s://%s/%s" % (self.scheme(), "localhost:%s" % (8080 + self.index), prefix)
        else:
            return "%s://%s/%s" % (self.scheme(), self.path.k8s, prefix)

    def requirements(self):
        if not DEV:
            yield ("pod", "%s" % self.name.k8s)

@abstract_test
class ServiceType(Node):

    def config(self):
        if False: yield

    def manifests(self):
        return self.format(manifests.BACKEND)

    def requirements(self):
        yield ("pod", self.path.k8s)

class HTTP(ServiceType):
    pass

class GRPC(ServiceType):
    pass

@abstract_test
class MappingTest(Test):

    target: ServiceType
    options: Sequence['OptionTest']

    def __init__(self, target: ServiceType, options = ()) -> None:
        self.target = target
        self.options = list(options)

@abstract_test
class OptionTest(Test):

    VALUES: Any = None

    @classmethod
    def variants(cls):
        if cls.VALUES is None:
            yield variant()
        else:
            for val in cls.VALUES:
                yield variant(val, name=sanitize(val))

    def __init__(self, value = None):
        self.value = value
