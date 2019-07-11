from typing import Any, ClassVar, Dict, List, Optional, Sequence, Tuple
from typing import cast as typecast

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

from kat.harness import abstract_test, sanitize, Name, Node, Test, Query
from kat import manifests
from kat.utils import ShellCommand, KAT_FAMILY
from kat.dockerdriver import DockerDriver

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


def assert_default_errors(errors):
    default_errors = [
        ["",
         "Ambassador could not find core CRD definitions. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored..."],
        ["",
         "Ambassador could not find Resolver type CRD definitions. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored..."]
    ]

    number_of_default_errors = len(default_errors)
    assert errors[:number_of_default_errors] == default_errors

    for error in errors[number_of_default_errors:]:
        assert 'found invalid port' in error[1], "Could not find 'found invalid port' in the error {}".format(error[1])


#DEV = os.environ.get("AMBASSADOR_DEV", "0").lower() in ("1", "yes", "true")
DEV = False


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

    skip_in_dev: bool = False                   # set to True to skip in dev shells
    envs: Dict[str, str] = {}                   # set extra environment variables
    configs: Dict[str, str] = {}                # configuration elements
    namespaces: List[str] = []                  # list of namespaces to create
    extra_ports: Optional[List[int]] = None     # list of additional ports to expose
    upstreams: Dict[str, dict] = {}             # list of additional pods to create
    debug_diagd: bool = False                   # should we debug diagd?

    _environ: Dict[str, str] = {}   # __init__ builds up the full environment here

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)

        self._environ = dict(self.envs)

        if self.single_namespace:
            self._environ['AMBASSADOR_SINGLE_NAMESPACE'] = 'yes'

        if self.disable_endpoints:
            self._environ['AMBASSADOR_DISABLE_ENDPOINTS'] = 'yes'

        if self.debug_diagd:
            self._environ['AMBASSADOR_DEBUG'] = 'diagd'

        if self.skip_in_dev and DEV:
            self.skip_node = True

    def manifests(self) -> str:
        crd_array = []
        crds = ''

        if self.configs:
            crd_configs = self.format(self.configs.pop('CRD', ""))

            if crd_configs:
                for config in yaml.load_all(crd_configs, Loader=yaml_loader):
                    spec = dict(config)
                    api_version = spec.pop('apiVersion', None)
                    kind = spec.pop('kind', None)
                    name = spec.pop('name', None)

                    if not api_version or not kind or not name:
                        raise Exception(f'{self.name}: CRD config must have apiVersion, kind, and name')

                    spec['ambassador_id'] = self.ambassador_id

                    crd_array.append({
                        'apiVersion': api_version,
                        'kind': kind,
                        'metadata': {
                            'name': name
                        },
                        'spec': spec
                    })

                crds = '---\n' + yaml.dump_all(crd_array, Dumper=yaml_dumper)

        ns = ""

        for namespace in self.namespaces:
            ns += f'''
---
apiVersion: v1
kind: Namespace
metadata:
  name: {namespace}
'''

        rbac = manifests.RBAC_CLUSTER_SCOPE

        if self.single_namespace:
            rbac = manifests.RBAC_NAMESPACE_SCOPE

        eports = ""

        if self.extra_ports:
            for port in self.extra_ports:
                eports += f"""
  - name: extra-{port}
    protocol: TCP
    port: {port}
    targetPort: {port}
"""

        epods = ''

        if self.upstreams:
            for pod_name, pod_info in self.upstreams.items():
                # Defer servicetype stuff
                if pod_info.get('servicetype'):
                    continue

                pod_def = {
                    'apiVersion': 'v1',
                    'kind': 'Pod',
                    'metadata': {
                        'name': pod_name,
                        'labels': {
                            'backend': pod_name
                        }
                    },
                }

                container = {
                    'name': pod_name,
                    'image': pod_info['image'],
                    'imagePullPolicy': 'Always'
                }

                pod_ports = []
                svc_ports = []

                if pod_info.get('ports'):
                    for protocol, svc_port, container_port in pod_info['ports']:
                        protocol = protocol.upper()

                        pod_ports.append({
                            'name': f'port-{svc_port}',
                            'containerPort': container_port,
                            'protocol': protocol
                        })

                        svc_ports.append({
                            'name': f'port-{svc_port}',
                            'port': svc_port,
                            'targetPort': f'port-{svc_port}',
                            'protocol': protocol
                        })

                if pod_ports:
                    container['ports'] = pod_ports

                pod_env_info = pod_info.get('envs', {})

                if pod_env_info:
                    container['env'] = [
                        {
                            'name': name,
                            'value': value
                        }
                        for name, value in pod_env_info.items()
                    ]

                pod_def['spec'] = {
                    'containers': [ container ]
                }

                svc_def = {
                    'apiVersion': 'v1',
                    'kind': 'Service',
                    'metadata': {
                        'name': pod_name
                    },
                    'spec': {
                        'selector': {
                            'backend': pod_name
                        },
                    }
                }

                if svc_ports:
                    svc_def['spec']['ports'] = svc_ports

                epods += '---\n' + yaml.dump(svc_def, Dumper=yaml_dumper)
                epods += '---\n' + yaml.dump(pod_def, Dumper=yaml_dumper)

        envs = ''

        if self._environ:
            for key, value in self._environ.items():
                envs += f'''
    - name: {key}
      value: "{value}"
'''

        base = ns + rbac + epods + crds

        if DEV:
            return self.format(base + AMBASSADOR_LOCAL, extra_ports=eports)
        else:
            return self.format(base + manifests.AMBASSADOR,
                               image=os.environ["AMBASSADOR_DOCKER_IMAGE"], envs=envs, extra_ports=eports)

    def init_upstreams(self):
        if self.upstreams:
            for pod_name, pod_info in self.upstreams.items():
                svctype = pod_info.get('servicetype', None)

                if svctype:
                    # OK, this is us.
                    extras = dict(pod_info)
                    extras.pop('servicetype')

                    typeclass = globals().get(svctype, None)

                    if not typeclass:
                        raise Exception(f'{self} wants {pod_name} of unknown type {svctype}')

                    setattr(self, pod_name, typeclass(**extras))
                else:
                    setattr(self, pod_name, UpstreamService(name=pod_name, _no_classname=True, **pod_info))

                    print(f'{self}: adding {pod_name}: {getattr(self, pod_name)}')

    # Subclasses can override this. Of course.
    def config(self):
        for attr, value in self.configs.items():
            element = self if (attr == 'self') else getattr(self, attr)

            yield element, self.format(value)

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
        cached_image = os.environ["AMBASSADOR_DOCKER_IMAGE_CACHED"]
        ambassador_base_image = os.environ["AMBASSADOR_BASE_IMAGE"]

        if not AmbassadorTest.IMAGE_BUILT:
            AmbassadorTest.IMAGE_BUILT = True

            DockerDriver.kill_old_containers()

            context = os.path.dirname(os.path.dirname(os.path.dirname(__file__)))

            print("Starting docker build...", end="")
            sys.stdout.flush()

            if ShellCommand.run("docker build Ambassador image",
                                "docker", "build",
                                "--build-arg", "CACHED_CONTAINER_IMAGE={}".format(cached_image),
                                "--build-arg", "AMBASSADOR_BASE_IMAGE={}".format(ambassador_base_image),
                                context, "-t", image):
                print("done.")
            else:
                pytest.exit("container failed to build")

        fname = "/tmp/k8s-%s.yaml" % self.path.k8s
        if os.path.exists(fname):
            with open(fname) as fd:
                content = fd.read()
        else:
            nsp = getattr(self, 'namespace', None) or 'default'

            if not ShellCommand.run(f'fetch secret for {self.path.k8s}',
                                    "kubectl", "get", "-n", nsp, "-o", "yaml", "secret", self.path.k8s):
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
        command = ["docker", "run", "-d", "-l", f'kat-family={KAT_FAMILY}', "--name", self.path.k8s]

        envs = [ f'{key}={value}' for key, value in self._environ.items() ]

        envs += [
            "KUBERNETES_SERVICE_HOST=kubernetes",
            "KUBERNETES_SERVICE_PORT=443",
            "AMBASSADOR_SNAPSHOT_COUNT=1",
            "AMBASSADOR_CONFIG_BASE_DIR=/tmp/ambassador",
            f"AMBASSADOR_ID={self.ambassador_id}"
        ]

        if self.namespace:
            envs.append("AMBASSADOR_NAMESPACE=%s" % self.namespace)

        for env in envs:
            command.extend([ "-e", env ])

        ports = ["%s:8877" % (8877 + self.index), "%s:8080" % (8080 + self.index), "%s:8443" % (8443 + self.index)]

        if self.extra_ports:
            for port in self.extra_ports:
                ports.append(f'{port}:{port}')

        for port in ports:
            command.extend([ "-p", port ])

        volumes = ["%s:/var/run/secrets/kubernetes.io/serviceaccount" % secret_dir]

        for volume in volumes:
            command.extend([ "-v", volume ])

        command.append(image)

        if os.environ.get('KAT_SHOW_DOCKER'):
            print(" ".join(command))

        if not ShellCommand.run(f'start container for {self.path.k8s}', *command):
            pytest.exit(f'could not start container for {self.path.k8s}')

    def queries(self):
        if DEV:
            cmd = ShellCommand(f'docker check for {self.path.k8s}',
                               "docker", "ps", "-qf", "name=%s" % self.path.k8s)

            if not cmd.check():
                if not cmd.stdout.strip():
                    log_cmd = ShellCommand(f'docker logs for {self.path.k8s}',
                                           "docker", "logs", self.path.k8s, stderr=subprocess.STDOUT)

                    if log_cmd.check():
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

            if not port:
                port = 8443 if scheme == 'https' else 8080

            if port:
                host_and_port += f':{port}'

            return "%s://%s/%s" % (scheme, host_and_port, prefix)

    def requirements(self):
        yield ("url", Query(self.url("ambassador/v0/check_ready")))
        yield ("url", Query(self.url("ambassador/v0/check_alive")))


@abstract_test
class UpstreamService(Node):
    path: Name

    def __init__(self, *args, **kwargs) -> None:
        super().__init__(*args, **kwargs)

    def config(self):
        yield from ()

    def manifests(self):
        return None

    def requirements(self):
        yield from ()


@abstract_test
class IsolatedServiceType(Node):

    path: Name

    def __init__(self, service_manifests: str=None, *args, **kwargs) -> None:
        super().__init__(*args, **kwargs)
        self._manifests = service_manifests or manifests.ISOLATED_BACKEND

    def config(self):
        yield from ()

    def manifests(self):
        return self.format(self._manifests)

    def requirements(self):
        yield ("url", Query("http://%s" % self.path.fqdn))
        yield ("url", Query("https://%s" % self.path.fqdn))


@abstract_test
class ServiceType(Node):
    _manifests: Optional[str]
    use_superpod: bool = True

    def __init__(self,
                 namespace: Optional[str] = None,
                 service_manifests: Optional[str] = None,
                 *args, **kwargs) -> None:
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
        self._manifests = service_manifests or manifests.BACKEND

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
        super().__init__(*args, service_manifests=manifests.GRPC_ECHO_BACKEND, **kwargs)

    def requirements(self):
        yield ("pod", self.path.k8s)

class AHTTP(ServiceType):
    skip_variant: ClassVar[bool] = True

    def __init__(self, *args, **kwargs) -> None:
        super().__init__(*args, service_manifests=manifests.AUTH_BACKEND, **kwargs)


class AGRPC(ServiceType):
    skip_variant: ClassVar[bool] = True

    def __init__(self, *args, **kwargs) -> None:
        super().__init__(*args, service_manifests=manifests.GRPC_AUTH_BACKEND, **kwargs)

    def requirements(self):
        yield ("pod", self.path.k8s)

@abstract_test
class MappingTest(Test):

    target: ServiceType
    options: Sequence['OptionTest']
    parent: AmbassadorTest

    def init(self, target: ServiceType, options=()) -> None:
        self.target = target
        self.options = list(options)


@abstract_test
class OptionTest(Test):

    VALUES: ClassVar[Any] = None
    value: Any
    parent: Test

    @classmethod
    def variants(cls):
        if cls.VALUES is None:
            yield cls()
        else:
            for val in cls.VALUES:
                yield cls(val, name=sanitize(val))

    def init(self, value=None):
        self.value = value
