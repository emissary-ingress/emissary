import sys

import base64
import os
import pytest
import shutil
import subprocess
import yaml

yaml_loader = yaml.SafeLoader
yaml_dumper = yaml.SafeDumper

try:
    yaml_loader = yaml.CSafeLoader
    yaml_dumper = yaml.CSafeDumper
except AttributeError:
    pass

from typing import Any, ClassVar, Dict, List, Optional, Sequence
from typing import cast as typecast

from kat.harness import abstract_test, sanitize, Name, Node, Test, Query, load_manifest
from kat.utils import ShellCommand

RBAC_CLUSTER_SCOPE = load_manifest("rbac_cluster_scope")
RBAC_NAMESPACE_SCOPE = load_manifest("rbac_namespace_scope")
AMBASSADOR = load_manifest("ambassador")
BACKEND = load_manifest("backend")
GRPC_ECHO_BACKEND = load_manifest("grpc_echo_backend")
AUTH_BACKEND = load_manifest("auth_backend")
GRPC_AUTH_BACKEND = load_manifest("grpc_auth_backend")

AMBASSADOR_LOCAL = """
---
apiVersion: v1
kind: Secret
metadata:
  name: {self.path.k8s}
  annotations:
    kubernetes.io/service-account.name: {self.path.k8s}
type: kubernetes.io/service-account-token
"""


def assert_default_errors(errors, include_ingress_errors=True):
    default_errors = [
        ["",
         "Ambassador could not find core CRD definitions. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored..."],
        ["",
         "Ambassador could not find Resolver type CRD definitions. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored..."],
        ["",
         "Ambassador could not find the Host CRD definition. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored..."],
        ["",
         "Ambassador could not find the LogService CRD definition. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored..."]
    ]

    if include_ingress_errors:
        default_errors.append(
            ["",
             "Ambassador is not permitted to read Ingress resources. Please visit https://www.getambassador.io/user-guide/ingress-controller/ for more information. You can continue using Ambassador, but Ingress resources will be ignored..."
            ]
        )

    number_of_default_errors = len(default_errors)
    assert errors[:number_of_default_errors] == default_errors

    for error in errors[number_of_default_errors:]:
        assert 'found invalid port' in error[1], "Could not find 'found invalid port' in the error {}".format(error[1])


DEV = os.environ.get("AMBASSADOR_DEV", "0").lower() in ("1", "yes", "true")


@abstract_test
class AmbassadorTest(Test):

    """
    AmbassadorTest is a top level ambassador test.
    """

    OFFSET: ClassVar[int] = 0
    IMAGE_BUILT: ClassVar[bool] = False

    _index: Optional[int] = None
    _ambassador_id: Optional[str] = None
    single_namespace: bool = False
    disable_endpoints: bool = False
    name: Name
    path: Name
    extra_ports: Optional[List[int]] = None
    debug_diagd: bool = False
    manifest_envs = ""
    is_ambassador = True
    allow_edge_stack_redirect = False

    env = []

    def manifests(self) -> str:
        rbac = RBAC_CLUSTER_SCOPE

        if self.debug_diagd:
            self.manifest_envs += """
    - name: AMBASSADOR_DEBUG
      value: "diagd"
"""

        if self.ambassador_id:
            self.manifest_envs += f"""
    - name: AMBASSADOR_LABEL_SELECTOR
      value: "kat-ambassador-id={self.ambassador_id}"
"""

        if self.single_namespace:
            self.manifest_envs += """
    - name: AMBASSADOR_SINGLE_NAMESPACE
      value: "yes"
"""
            rbac = RBAC_NAMESPACE_SCOPE

        if self.disable_endpoints:
            self.manifest_envs += """
    - name: AMBASSADOR_DISABLE_ENDPOINTS
      value: "yes"
"""
        if not self.allow_edge_stack_redirect:
            self.manifest_envs += """
    - name: AMBASSADOR_NO_HOST_REDIRECT
      value: "yes"
"""

        eports = ""

        if self.extra_ports:
            for port in self.extra_ports:
                eports += f"""
  - name: extra-{port}
    protocol: TCP
    port: {port}
    targetPort: {port}
"""

        if DEV:
            return self.format(rbac + AMBASSADOR_LOCAL, extra_ports=eports)
        else:
            return self.format(rbac + AMBASSADOR,
                               image=os.environ["AMBASSADOR_DOCKER_IMAGE"], envs=self.manifest_envs, extra_ports=eports, capabilities_block = "")

    # Will tear this out of the harness shortly
    @property
    def ambassador_id(self) -> str:
        if self._ambassador_id is None:
            return self.name.k8s
        else:
            return typecast(str, self._ambassador_id)

    @ambassador_id.setter
    def ambassador_id(self, val: str) -> None:
        self._ambassador_id = val

    @property
    def index(self) -> int:
        if self._index is None:
            # lock here?
            self._index = AmbassadorTest.OFFSET
            AmbassadorTest.OFFSET += 1

        return typecast(int, self._index)

    def post_manifest(self):
        if not DEV:
            return

        if os.environ.get('KAT_SKIP_DOCKER'):
            return

        image = os.environ["AMBASSADOR_DOCKER_IMAGE"]
        cached_image = os.environ["BASE_PY_IMAGE"]
        ambassador_base_image = os.environ["BASE_GO_IMAGE"]

        if not AmbassadorTest.IMAGE_BUILT:
            AmbassadorTest.IMAGE_BUILT = True

            cmd = ShellCommand('docker', 'ps', '-a', '-f', 'label=kat-family=ambassador', '--format', '{{.ID}}')

            if cmd.check('find old docker container IDs'):
                ids = cmd.stdout.split('\n')

                while ids:
                    if ids[-1]:
                        break

                    ids.pop()

                if ids:
                    print("Killing old containers...")
                    ShellCommand.run('kill old containers', 'docker', 'kill', *ids, verbose=True)
                    ShellCommand.run('rm old containers', 'docker', 'rm', *ids, verbose=True)

            context = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))

            print("Starting docker build...", end="")
            sys.stdout.flush()

            cmd = ShellCommand("docker", "build", "--build-arg", "BASE_PY_IMAGE={}".format(cached_image), "--build-arg", "BASE_GO_IMAGE={}".format(ambassador_base_image), context, "-t", image)

            if cmd.check("docker build Ambassador image"):
                print("done.")
            else:
                pytest.exit("container failed to build")

        fname = "/tmp/k8s-%s.yaml" % self.path.k8s
        if os.path.exists(fname):
            with open(fname) as fd:
                content = fd.read()
        else:
            nsp = getattr(self, 'namespace', None) or 'default'

            cmd = ShellCommand("kubectl", "get", "-n", nsp, "-o", "yaml", "secret", self.path.k8s)

            if not cmd.check(f'fetch secret for {self.path.k8s}'):
                pytest.exit(f'could not fetch secret for {self.path.k8s}')

            content = cmd.stdout

            with open(fname, "wb") as fd:
                fd.write(content.encode('utf-8'))

        try:
            secret = yaml.load(content, Loader=yaml_loader)
        except Exception as e:
            print("could not parse YAML:\n%s" % content)
            raise e

        data = secret['data']
        # secret_dir = tempfile.mkdtemp(prefix=self.path.k8s, suffix="secret")
        secret_dir = "/tmp/%s-ambassadormixin-%s" % (self.path.k8s, 'secret')

        shutil.rmtree(secret_dir, ignore_errors=True)
        os.mkdir(secret_dir, 0o777)

        for k, v in data.items():
            with open(os.path.join(secret_dir, k), "wb") as f:
                f.write(base64.decodebytes(bytes(v, "utf8")))
        print("Launching %s container." % self.path.k8s)
        command = ["docker", "run", "-d", "-l", "kat-family=ambassador", "--name", self.path.k8s]

        envs = [ "KUBERNETES_SERVICE_HOST=kubernetes",
                 "KUBERNETES_SERVICE_PORT=443",
                 "AMBASSADOR_SNAPSHOT_COUNT=1",
                 "AMBASSADOR_CONFIG_BASE_DIR=/tmp/ambassador",
                 "AMBASSADOR_ID=%s" % self.ambassador_id]

        if self.namespace:
            envs.append("AMBASSADOR_NAMESPACE=%s" % self.namespace)

        if self.single_namespace:
            envs.append("AMBASSADOR_SINGLE_NAMESPACE=yes")

        if self.disable_endpoints:
            envs.append("AMBASSADOR_DISABLE_ENDPOINTS=yes")

        if self.debug_diagd:
            envs.append("AMBASSADOR_DEBUG=diagd")

        envs.extend(self.env)
        [command.extend(["-e", env]) for env in envs]

        ports = ["%s:8877" % (8877 + self.index), "%s:8001" % (8001 + self.index), "%s:8080" % (8080 + self.index), "%s:8443" % (8443 + self.index)]

        if self.extra_ports:
            for port in self.extra_ports:
                ports.append(f'{port}:{port}')

        [command.extend(["-p", port]) for port in ports]

        volumes = ["%s:/var/run/secrets/kubernetes.io/serviceaccount" % secret_dir]
        [command.extend(["-v", volume]) for volume in volumes]

        command.append(image)

        if os.environ.get('KAT_SHOW_DOCKER'):
            print(" ".join(command))

        cmd = ShellCommand(*command)

        if not cmd.check(f'start container for {self.path.k8s}'):
            pytest.exit(f'could not start container for {self.path.k8s}')

    def queries(self):
        if DEV:
            cmd = ShellCommand("docker", "ps", "-qf", "name=%s" % self.path.k8s)

            if not cmd.check(f'docker check for {self.path.k8s}'):
                if not cmd.stdout.strip():
                    log_cmd = ShellCommand("docker", "logs", self.path.k8s, stderr=subprocess.STDOUT)

                    if log_cmd.check(f'docker logs for {self.path.k8s}'):
                        print(cmd.stdout)

                    pytest.exit(f'container failed to start for {self.path.k8s}')

        return ()

    def scheme(self) -> str:
        return "http"

    def url(self, prefix, scheme=None, port=None) -> str:
        if scheme is None:
            scheme = self.scheme()

        if DEV:
            if not port:
                port = 8443 if scheme == 'https' else 8080
                port += self.index

            return "%s://%s/%s" % (scheme, "localhost:%s" % port, prefix)
        else:
            host_and_port = self.path.fqdn

            if port:
                host_and_port += f':{port}'

            return "%s://%s/%s" % (scheme, host_and_port, prefix)

    def requirements(self):
        yield ("url", Query(self.url("ambassador/v0/check_ready")))
        yield ("url", Query(self.url("ambassador/v0/check_alive")))


@abstract_test
class ServiceType(Node):

    path: Name
    _manifests: Optional[str]
    use_superpod: bool = True

    def __init__(self, service_manifests: str=None, namespace: str=None, *args, **kwargs) -> None:
        super().__init__(namespace=namespace, *args, **kwargs)

        self._manifests = service_manifests

        if self._manifests:
            self.use_superpod = False

    def config(self):
        yield from ()

    def manifests(self):
        if self.use_superpod:
            return None

        return self.format(self._manifests)

    def requirements(self):
        if self.use_superpod:
            yield from ()

        yield ("url", Query("http://%s" % self.path.fqdn))
        yield ("url", Query("https://%s" % self.path.fqdn))


@abstract_test
class ServiceTypeGrpc(Node):

    path: Name

    def __init__(self, service_manifests: str=None, *args, **kwargs) -> None:
        super().__init__(*args, **kwargs)
        self._manifests = service_manifests or BACKEND

    def config(self):
        yield from ()

    def manifests(self):
        return self.format(self._manifests)

    def requirements(self):
        yield ("url", Query("http://%s" % self.path.fqdn))
        yield ("url", Query("https://%s" % self.path.fqdn))


class HTTP(ServiceType):
    pass


class GRPC(ServiceType):
    pass

class EGRPC(ServiceType):
    skip_variant: ClassVar[bool] = True

    def __init__(self, *args, **kwargs) -> None:
        super().__init__(*args, service_manifests=GRPC_ECHO_BACKEND, **kwargs)

    def requirements(self):
        yield ("url", Query("http://%s/echo.EchoService/Echo" % self.path.fqdn,
                            headers={ "content-type": "application/grpc",
                                      "requested-status": "0" },
                            expected=200,
                            grpc_type="real"))

class AHTTP(ServiceType):
    skip_variant: ClassVar[bool] = True

    def __init__(self, *args, **kwargs) -> None:
        super().__init__(*args, service_manifests=AUTH_BACKEND, **kwargs)


class AGRPC(ServiceType):
    skip_variant: ClassVar[bool] = True

    def __init__(self, *args, **kwargs) -> None:
        super().__init__(*args, service_manifests=GRPC_AUTH_BACKEND, **kwargs)

    def requirements(self):
        yield ("pod", self.path.k8s)

@abstract_test
class MappingTest(Test):

    target: ServiceType
    options: Sequence['OptionTest']
    parent: AmbassadorTest

    no_local_mode = True
    skip_local_instead_of_xfail = "Plain (MappingTest)"

    def init(self, target: ServiceType, options=()) -> None:
        self.target = target
        self.options = list(options)
        self.is_ambassador = True

@abstract_test
class OptionTest(Test):

    VALUES: ClassVar[Any] = None
    value: Any
    parent: Test

    no_local_mode = True
    skip_local_instead_of_xfail = "Plain (OptionTests)"

    @classmethod
    def variants(cls):
        if cls.VALUES is None:
            yield cls()
        else:
            for val in cls.VALUES:
                yield cls(val, name=sanitize(val))

    def init(self, value=None):
        self.value = value
