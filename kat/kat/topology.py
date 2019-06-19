from typing import Any, Dict, List, Optional, Tuple, Union, TYPE_CHECKING

import os

from .manifests import SUPERPOD_POD

from .parser import load, Tag

if TYPE_CHECKING:
    from .harness import Node

PortSpec = List[Tuple[str, int, int]]
EnvSpec = Dict[str, str]
ConfigSpec = Dict[str, str]


class Superpod:
    def __init__(self, namespace: str) -> None:
        self.namespace = namespace
        self.next_clear = 8080
        self.next_tls = 8443
        self.service_names: Dict[int, str] = {}
        self.name = 'superpod-%s' % (self.namespace or 'default')

    def allocate(self, service_name) -> List[int]:
        ports = [ self.next_clear, self.next_tls ]
        self.service_names[self.next_clear] = service_name
        self.service_names[self.next_tls] = service_name

        self.next_clear += 1
        self.next_tls += 1

        return ports

    def get_manifest_list(self) -> List[Dict[str, Any]]:
        manifest = load('superpod', SUPERPOD_POD, Tag.MAPPING)

        assert len(manifest) == 1, "SUPERPOD manifest must have exactly one object"

        m = manifest[0]

        template = m['spec']['template']

        ports: List[Dict[str, int]] = []
        envs: List[Dict[str, Union[str, int]]] = template['spec']['containers'][0]['env']

        for p in sorted(self.service_names.keys()):
            ports.append({ 'containerPort': p })
            envs.append({ 'name': f'BACKEND_{p}', 'value': self.service_names[p] })

        template['spec']['containers'][0]['ports'] = ports

        if 'metadata' not in m:
            m['metadata'] = {}

        metadata = m['metadata']
        metadata['name'] = self.name

        m['spec']['selector']['matchLabels']['backend'] = self.name
        template['metadata']['labels']['backend'] = self.name

        if self.namespace:
            # Fix up the namespace.
            if 'namespace' not in metadata:
                metadata['namespace'] = self.namespace

        return list(manifest)


class Container:
    def __init__(self, name: str, namespace: str, image: str,
                 envs: Optional[EnvSpec], ports: Optional[PortSpec]=None,
                 configs: Optional[str]=None, crds: Optional[str]=None) -> None:
        self.name = name
        self.namespace = namespace
        self.image = image
        self.envs = envs
        self.ports = ports or []
        self.configs = configs
        self.crds = crds

    def as_dict(self) -> Dict[str, Any]:
        rd = {
            'name': self.name,
            'namespace': self.namespace,
            'image': self.image,
            'envs': self.envs,
            'ports': self.ports,
        }

        if self.configs:
            cfgs = load(f'{self.name}.{self.namespace} configs', self.configs, Tag.MAPPING)
            rd['configs'] = cfgs.as_python()
            # rd['configs'] = self.configs

        if self.crds:
            crds = load(f'{self.name}.{self.namespace} CRDs', self.crds, Tag.MAPPING)
            rd['crds'] = crds.as_python()
            # rd['crds'] = self.crds

        return rd

    def set_ip(self, ip: str) -> None:
        self.ip = ip


class Namespace:
    def __init__(self, name: str):
        self.name = name
        self.containers = {}

        self.superpod = Superpod(self.name)
        self.superpod_container = self.add_container(Container(name=self.superpod.name,
                                                               namespace=self.name,
                                                               image='quay.io/datawire/kat-backend:13',
                                                               envs={ 'INCLUDE_EXTAUTH_HEADER': 'yes' }))

        self.routes = {}

    def add_container(self, c: Container) -> Container:
        if c.name in self.containers:
            raise Exception(f'Namespace {self.name}: container name {c.name} is already in use')

        self.containers[c.name] = c

        for protocol, src_port, dest_port in c.ports:
            self.add_route(protocol, c.name, src_port, c, dest_port)

        return c

    def register_superpod(self, svc_name: str, svc_type: str, configs: Optional[str]) -> None:
        del svc_type       # silence typing error

        clear, tls = self.superpod.allocate(svc_name)

        self.superpod_container.envs[f'BACKEND_{clear}'] = svc_name
        self.superpod_container.envs[f'BACKEND_{tls}'] = svc_name

        self.superpod_container.ports.append(( 'tcp', clear, clear ))
        self.superpod_container.ports.append(( 'tcp', tls, tls ))

        if configs:
            configs = configs.strip()

            if not configs.startswith('---'):
                configs = '---\n' + configs

            cfg = self.superpod_container.configs or ''
            cfg += configs

            self.superpod_container.configs = cfg

        self.add_route('tcp', svc_name, 80, self.superpod_container, clear)
        self.add_route('tcp', svc_name, 443, self.superpod_container, tls)

    def add_route(self, protocol: str, src_name: str, src_port: int, dest: Container, dest_port: int) -> None:
        key = f'{src_name}:{src_port}'

        rdict = {
            'protocol': protocol,
            'src_name': src_name,
            'src_port': src_port,
            'dest_name': dest.name,
            'dest_port': dest_port
        }

        if key not in self.routes:
            self.routes[key] = rdict
        elif self.routes[key] != rdict:
            raise Exception(f'Topology {self.name}: route collision for {key}')

    def as_dict(self) -> dict:
        return {
            'name': self.name,
            'containers': { name: c.as_dict() for name, c in self.containers.items() },
            'routes': self.routes
        }


def formatted_config(n: 'Node', configs: Dict[str, str], key: str) -> Optional[str]:
    v = configs.get(key)

    if v:
        return n.format(v)

    return None

class Topology:
    def __init__(self):
        self.namespaces = {}

    def process_node(self, n: 'Node') -> None:
        # We need at least upstreams and configs here

        upstreams: dict = getattr(n, 'upstreams', None)
        configs: dict = getattr(n, 'configs', None)

        if not upstreams or not configs:
            return

        node_environment: Optional[Dict[str, str]] = None

        _env = getattr(n, '_environ', None)

        if _env:
            node_environment = {}

            for k, v in _env.items():
                node_environment[k] = n.format(v)

        print(f'...{n.name}')

        cfgs = formatted_config(n, configs, 'self')
        crds = formatted_config(n, configs, 'CRD')

        self.add_container(
            Container(
                name=n.name,
                namespace=n.namespace,
                image=os.environ['AMBASSADOR_DOCKER_IMAGE'],
                envs=node_environment,
                ports=[
                    ('tcp', 8080, 8080),
                    ('tcp', 8443, 8443)
                ],
                configs=cfgs,
                crds=crds
            )
        )

        for pod_name, pod_info in upstreams.items():
            target_name = pod_name
            tgt_configs = formatted_config(n, configs, target_name)

            target = getattr(n, pod_name, None)

            if target:
                target_name = target.path.k8s

            print(f'    {target_name}: {pod_info}')

            svctype = pod_info.get('servicetype', None)

            if svctype:
                self.register_superpod(n.namespace, target_name, svctype, tgt_configs)
            else:
                self.add_container(Container(namespace=n.namespace, name=target_name,
                                             configs=tgt_configs, **pod_info))

    def get_namespace(self, name: str) -> Namespace:
        if name not in self.namespaces:
            self.namespaces[name] = Namespace(name)

        return self.namespaces[name]

    def add_container(self, c: Container) -> None:
        self.get_namespace(c.namespace).add_container(c)

    def register_superpod(self, namespace: str, svc_name: str, svc_type: str, configs: Optional[ConfigSpec]) -> None:
        self.get_namespace(namespace).register_superpod(svc_name=svc_name, svc_type=svc_type, configs=configs)

    def as_dict(self) -> dict:
        return {
            'namespaces': { name: n.as_dict() for name, n in self.namespaces.items() }
        }
