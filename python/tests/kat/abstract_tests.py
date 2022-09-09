import base64
import os
import shutil
import subprocess
import sys
from typing import Any, ClassVar, Generator, List, Optional, Sequence, Tuple, Union
from typing import cast as typecast

import pytest
import yaml

# These type: ignores are because, weirdly, the yaml.CSafe* variants don't share
# a type with their non-C variants. No clue why not.
yaml_loader = yaml.SafeLoader  # type: ignore
yaml_dumper = yaml.SafeDumper  # type: ignore

try:
    yaml_loader = yaml.CSafeLoader  # type: ignore
    yaml_dumper = yaml.CSafeDumper  # type: ignore
except AttributeError:
    pass

import tests.integration.manifests as integration_manifests
from kat.harness import Name, Node, Query, Test, abstract_test, sanitize
from kat.utils import ShellCommand

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
        [
            "",
            "Ambassador could not find core CRD definitions. Please visit https://www.getambassador.io/docs/edge-stack/latest/topics/install/upgrade-to-edge-stack/#5-update-and-restart for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored...",
        ],
        [
            "",
            "Ambassador could not find Resolver type CRD definitions. Please visit https://www.getambassador.io/docs/edge-stack/latest/topics/install/upgrade-to-edge-stack/#5-update-and-restart for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored...",
        ],
        [
            "",
            "Ambassador could not find the Host CRD definition. Please visit https://www.getambassador.io/docs/edge-stack/latest/topics/install/upgrade-to-edge-stack/#5-update-and-restart for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored...",
        ],
        [
            "",
            "Ambassador could not find the LogService CRD definition. Please visit https://www.getambassador.io/docs/edge-stack/latest/topics/install/upgrade-to-edge-stack/#5-update-and-restart for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored...",
        ],
    ]

    if include_ingress_errors:
        default_errors.append(
            [
                "",
                "Ambassador is not permitted to read Ingress resources. Please visit https://www.getambassador.io/docs/edge-stack/latest/topics/running/ingress-controller/#ambassador-as-an-ingress-controller for more information. You can continue using Ambassador, but Ingress resources will be ignored...",
            ]
        )

    number_of_default_errors = len(default_errors)

    if errors[:number_of_default_errors] != default_errors:
        assert False, f"default error table mismatch: got\n{errors}"

    for error in errors[number_of_default_errors:]:
        assert (
            "found invalid port" in error[1]
        ), "Could not find 'found invalid port' in the error {}".format(error[1])


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
    name: str
    path: Name
    extra_ports: Optional[List[int]] = None
    debug_diagd: bool = True
    debug_envoy: bool = False
    manifest_envs = ""
    is_ambassador = True
    allow_edge_stack_redirect = False
    edge_stack_cleartext_host = True
    envoy_api_version: Optional[str] = None

    env: List[str] = []

    def manifests(self) -> str:
        rbac = integration_manifests.load("rbac_cluster_scope")

        self.manifest_envs += """
    - name: POLL_EVERY_SECS
      value: "0"
    - name: CONSUL_WATCHER_PORT
      value: "8500"
"""

        if os.environ.get("AMBASSADOR_FAST_RECONFIGURE", "true").lower() == "false":
            self.manifest_envs += """
    - name: AMBASSADOR_FAST_RECONFIGURE
      value: "false"
"""

        amb_debug = []
        if self.debug_diagd:
            amb_debug.append("diagd")
        if self.debug_envoy:
            amb_debug.append("envoy")
        if amb_debug:
            self.manifest_envs += """
    - name: AMBASSADOR_DEBUG
      value: "%s"
    - name: AES_LOG_LEVEL
      value: "debug"
""" % ":".join(
                amb_debug
            )

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
            rbac = integration_manifests.load("rbac_namespace_scope")

        if self.disable_endpoints:
            self.manifest_envs += """
    - name: AMBASSADOR_DISABLE_ENDPOINTS
      value: "yes"
"""
        if not self.allow_edge_stack_redirect:
            self.manifest_envs += """
    - name: AMBASSADOR_NO_TLS_REDIRECT
      value: "yes"
"""

        if self.envoy_api_version is not None:
            self.manifest_envs += f"""
    - name: AMBASSADOR_ENVOY_API_VERSION
      value: "{self.envoy_api_version}"
"""
        elif os.environ.get("AMBASSADOR_ENVOY_API_VERSION", "") != "":
            self.manifest_envs += (
                """
    - name: AMBASSADOR_ENVOY_API_VERSION
      value: "%s"
"""
                % os.environ["AMBASSADOR_ENVOY_API_VERSION"]
            )

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
            return self.format(
                rbac + integration_manifests.load("ambassador"),
                envs=self.manifest_envs,
                extra_ports=eports,
                capabilities_block="",
            )

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

        if os.environ.get("KAT_SKIP_DOCKER"):
            return

        image = os.environ["AMBASSADOR_DOCKER_IMAGE"]
        cached_image = os.environ["BASE_PY_IMAGE"]
        ambassador_base_image = os.environ["BASE_GO_IMAGE"]

        if not AmbassadorTest.IMAGE_BUILT:
            AmbassadorTest.IMAGE_BUILT = True

            cmd = ShellCommand(
                "docker", "ps", "-a", "-f", "label=kat-family=ambassador", "--format", "{{.ID}}"
            )

            if cmd.check("find old docker container IDs"):
                ids = cmd.stdout.split("\n")

                while ids:
                    if ids[-1]:
                        break

                    ids.pop()

                if ids:
                    print("Killing old containers...")
                    ShellCommand.run("kill old containers", "docker", "kill", *ids, verbose=True)
                    ShellCommand.run("rm old containers", "docker", "rm", *ids, verbose=True)

            context = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))

            print("Starting docker build...", end="")
            sys.stdout.flush()

            cmd = ShellCommand(
                "docker",
                "build",
                "--build-arg",
                "BASE_PY_IMAGE={}".format(cached_image),
                "--build-arg",
                "BASE_GO_IMAGE={}".format(ambassador_base_image),
                context,
                "-t",
                image,
            )

            if cmd.check("docker build Ambassador image"):
                print("done.")
            else:
                pytest.exit("container failed to build")

        fname = "/tmp/k8s-%s.yaml" % self.path.k8s
        if os.path.exists(fname):
            with open(fname) as fd:
                content = fd.read()
        else:
            nsp = getattr(self, "namespace", None) or "default"

            cmd = ShellCommand(
                "tools/bin/kubectl", "get", "-n", nsp, "-o", "yaml", "secret", self.path.k8s
            )

            if not cmd.check(f"fetch secret for {self.path.k8s}"):
                pytest.exit(f"could not fetch secret for {self.path.k8s}")

            content = cmd.stdout

            with open(fname, "wb") as fd:
                fd.write(content.encode("utf-8"))

        try:
            secret = yaml.load(content, Loader=yaml_loader)
        except Exception as e:
            print("could not parse YAML:\n%s" % content)
            raise e

        data = secret["data"]
        # secret_dir = tempfile.mkdtemp(prefix=self.path.k8s, suffix="secret")
        secret_dir = "/tmp/%s-ambassadormixin-%s" % (self.path.k8s, "secret")

        shutil.rmtree(secret_dir, ignore_errors=True)
        os.mkdir(secret_dir, 0o777)

        for k, v in data.items():
            with open(os.path.join(secret_dir, k), "wb") as f:
                f.write(base64.decodebytes(bytes(v, "utf8")))
        print("Launching %s container." % self.path.k8s)
        command = ["docker", "run", "-d", "-l", "kat-family=ambassador", "--name", self.path.k8s]

        envs = [
            "KUBERNETES_SERVICE_HOST=kubernetes",
            "KUBERNETES_SERVICE_PORT=443",
            "AMBASSADOR_SNAPSHOT_COUNT=1",
            "AMBASSADOR_CONFIG_BASE_DIR=/tmp/ambassador",
            "POLL_EVERY_SECS=0",
            "CONSUL_WATCHER_PORT=8500",
            "AMBASSADOR_UPDATE_MAPPING_STATUS=false",
            "AMBASSADOR_ID=%s" % self.ambassador_id,
        ]

        if self.namespace:
            envs.append("AMBASSADOR_NAMESPACE=%s" % self.namespace)

        if self.single_namespace:
            envs.append("AMBASSADOR_SINGLE_NAMESPACE=yes")

        if self.disable_endpoints:
            envs.append("AMBASSADOR_DISABLE_ENDPOINTS=yes")

        amb_debug = []
        if self.debug_diagd:
            amb_debug.append("diagd")
        if self.debug_envoy:
            amb_debug.append("envoy")
        if amb_debug:
            envs.append("AMBASSADOR_DEBUG=%s" % ":".join(amb_debug))

        envs.extend(self.env)
        [command.extend(["-e", env]) for env in envs]

        ports = [
            "%s:8877" % (8877 + self.index),
            "%s:8001" % (8001 + self.index),
            "%s:8080" % (8080 + self.index),
            "%s:8443" % (8443 + self.index),
        ]

        if self.extra_ports:
            for port in self.extra_ports:
                ports.append(f"{port}:{port}")

        [command.extend(["-p", port]) for port in ports]

        volumes = ["%s:/var/run/secrets/kubernetes.io/serviceaccount" % secret_dir]
        [command.extend(["-v", volume]) for volume in volumes]

        command.append(image)

        if os.environ.get("KAT_SHOW_DOCKER"):
            print(" ".join(command))

        cmd = ShellCommand(*command)

        if not cmd.check(f"start container for {self.path.k8s}"):
            pytest.exit(f"could not start container for {self.path.k8s}")

    def queries(self):
        if DEV:
            cmd = ShellCommand("docker", "ps", "-qf", "name=%s" % self.path.k8s)

            if not cmd.check(f"docker check for {self.path.k8s}"):
                if not cmd.stdout.strip():
                    log_cmd = ShellCommand(
                        "docker", "logs", self.path.k8s, stderr=subprocess.STDOUT
                    )

                    if log_cmd.check(f"docker logs for {self.path.k8s}"):
                        print(cmd.stdout)

                    pytest.exit(f"container failed to start for {self.path.k8s}")

        return ()

    def scheme(self) -> str:
        return "http"

    def url(self, prefix, scheme=None, port=None) -> str:
        if scheme is None:
            scheme = self.scheme()

        if DEV:
            if not port:
                port = 8443 if scheme == "https" else 8080
                port += self.index

            return "%s://%s/%s" % (scheme, "localhost:%s" % port, prefix)
        else:
            host_and_port = self.path.fqdn

            if port:
                host_and_port += f":{port}"

            return "%s://%s/%s" % (scheme, host_and_port, prefix)

    def requirements(self):
        yield ("url", Query(self.url("ambassador/v0/check_ready")))
        yield ("url", Query(self.url("ambassador/v0/check_alive")))


@abstract_test
class ServiceType(Node):

    path: Name
    _manifests: Optional[str]
    use_superpod: bool = True

    def __init__(
        self, service_manifests: str = None, namespace: str = None, *args, **kwargs
    ) -> None:
        super().__init__(namespace=namespace, *args, **kwargs)

        self._manifests = service_manifests

        if self._manifests:
            self.use_superpod = False

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
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

    def __init__(self, service_manifests: str = None, *args, **kwargs) -> None:
        super().__init__(*args, **kwargs)
        self._manifests = service_manifests or integration_manifests.load("backend")

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
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
        # Do this unconditionally, because that's the point of this class.
        kwargs["service_manifests"] = integration_manifests.load("grpc_echo_backend")
        super().__init__(*args, **kwargs)

    def requirements(self):
        yield (
            "url",
            Query(
                "http://%s/echo.EchoService/Echo" % self.path.fqdn,
                headers={"content-type": "application/grpc", "requested-status": "0"},
                expected=200,
                grpc_type="real",
            ),
        )


class AHTTP(ServiceType):
    skip_variant: ClassVar[bool] = True

    def __init__(self, *args, **kwargs) -> None:
        # Do this unconditionally, because that's the point of this class.
        kwargs["service_manifests"] = integration_manifests.load("auth_backend")
        super().__init__(*args, **kwargs)


class AGRPC(ServiceType):
    skip_variant: ClassVar[bool] = True

    def __init__(self, protocol_version: str = "v2", *args, **kwargs) -> None:
        self.protocol_version = protocol_version

        # Do this unconditionally, because that's the point of this class.
        kwargs["service_manifests"] = integration_manifests.load("grpc_auth_backend")
        super().__init__(*args, **kwargs)

    def requirements(self):
        yield ("pod", self.path.k8s)


class RLSGRPC(ServiceType):
    skip_variant: ClassVar[bool] = True

    def __init__(self, protocol_version: str = "v2", *args, **kwargs) -> None:
        self.protocol_version = protocol_version

        # Do this unconditionally, because that's the point of this class.
        kwargs["service_manifests"] = integration_manifests.load("grpc_rls_backend")
        super().__init__(*args, **kwargs)

    def requirements(self):
        yield ("pod", self.path.k8s)


class ALSGRPC(ServiceType):
    skip_variant: ClassVar[bool] = True

    def __init__(self, *args, **kwargs) -> None:
        # Do this unconditionally, because that's the point of this class.
        kwargs["service_manifests"] = integration_manifests.load("grpc_als_backend")
        super().__init__(*args, **kwargs)

    def requirements(self):
        yield ("pod", self.path.k8s)


@abstract_test
class MappingTest(Test):

    target: ServiceType
    options: Sequence["OptionTest"]
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
    def variants(cls) -> Generator[Node, None, None]:
        if cls.VALUES is None:
            yield cls()
        else:
            for val in cls.VALUES:
                yield cls(val, name=sanitize(val))

    def init(self, value=None):
        self.value = value
