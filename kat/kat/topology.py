from typing import Any, Dict, List, Optional, Tuple, Union, TYPE_CHECKING

import os

from .manifests import SUPERPOD_POD

from .parser import load, Tag

if TYPE_CHECKING:
    from .harness import Node

PortSpec = List[Tuple[str, int, int]]
EnvSpec = Dict[str, str]
ConfigSpec = Dict[str, str]
ConfigList = List[dict]

class Superpod:
    def __init__(self, namespace: str) -> None:
        self.name = 'superpod'
        self.namespace = namespace
        self.next_clear = 8080
        self.next_tls = 8443
        self.service_names: Dict[int, str] = {}

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
    def __init__(self, node: 'Node', name: str, namespace: str, path: str, image: str,
                 ambassador_id: Optional[str]=None, is_ambassador: Optional[bool]=False,
                 envs: Optional[EnvSpec]=None, ports: Optional[PortSpec]=None,
                 configs: Optional[ConfigList]=None, crds: Optional[ConfigList]=None) -> None:
        self.node = node
        self.name = name
        self.namespace = namespace
        self.path = path
        self.image = image
        self.ambassador_id = ambassador_id
        self.is_ambassador = is_ambassador
        self.envs = envs
        self.ports = ports or []
        self.configs = configs
        self.crds = crds

    def as_dict(self) -> Dict[str, Any]:
        rd = {
            'name': self.name,
            'namespace': self.namespace,
            'path': self.path,
            'image': self.image,
            'ports': self.ports,
        }

        if self.envs:
            rd['envs'] = self.envs

        if self.ambassador_id:
            rd['ambassador_id'] = self.ambassador_id

        if self.is_ambassador:
            rd['is_ambassador'] = self.is_ambassador

        if self.configs:
            rd['configs'] = self.configs

        if self.crds:
            rd['crds'] = self.crds

        return rd

    def set_ip(self, ip: str) -> None:
        self.ip = ip

    def all_configs(self):
        if self.configs:
            yield from self.configs

        if self.crds:
            yield from self.crds


class AmbassadorContainer(Container):
    def __init__(self, node: 'Node', name: str, namespace: str, path: str, image: str,
                 ambassador_id: Optional[str]=None,
                 envs: Optional[EnvSpec]=None, ports: Optional[PortSpec]=None,
                 configs: Optional[ConfigList]=None, crds: Optional[ConfigList]=None) -> None:
        super().__init__(node=node, name=name, namespace=namespace, path=path,
                         image=image, ambassador_id=ambassador_id, is_ambassador=True,
                         envs=envs, ports=ports, configs=configs, crds=crds)

class Namespace:
    def __init__(self, name: str):
        self.name = name
        self.containers = {}
        self.dns = {}

        self.superpod = Superpod(self.name)
        self.superpod_container = self.add_container(Container(node=None,
                                                               name=self.superpod.name,
                                                               path=f'{self.superpod.name}.{self.name}',
                                                               namespace=self.name,
                                                               image='quay.io/datawire/kat-backend:13',
                                                               envs={ 'INCLUDE_EXTAUTH_HEADER': 'yes' }))

        self.routes = {}

    def all_containers(self) -> List[Container]:
        for k in sorted(self.containers.keys()):
            yield self.containers[k]

    def all_routes(self) -> List[Dict[str, Union[int, str]]]:
        for k in sorted(self.routes.keys()):
            yield self.routes[k]

    def add_dns(self, name: str, c: Container) -> None:
        self.dns[name] = c

    def get_ip(self, name: str) -> str:
        return self.dns[name].ip

    def add_container(self, c: Container) -> Container:
        if c.name in self.containers:
            raise Exception(f'Namespace {self.name}: container name {c.name} is already in use')

        self.containers[c.name] = c

        self.add_dns(c.path, c)

        for protocol, src_port, dest_port in c.ports:
            self.add_route(protocol, c.path, src_port, c, dest_port)

        return c

    def register_superpod(self, svc_name: str, svc_type: str, configs: Optional[ConfigList]) -> None:
        del svc_type       # silence typing error

        clear, tls = self.superpod.allocate(svc_name)

        self.superpod_container.envs[f'BACKEND_{clear}'] = svc_name
        self.superpod_container.envs[f'BACKEND_{tls}'] = svc_name

        self.superpod_container.ports.append(( 'tcp', clear, clear ))
        self.superpod_container.ports.append(( 'tcp', tls, tls ))

        if configs:
            extant_cfgs = self.superpod_container.configs or []

            self.superpod_container.configs = extant_cfgs + configs

        self.add_dns(svc_name, self.superpod_container)
        self.add_route('tcp', svc_name, 80, self.superpod_container, clear)
        self.add_route('tcp', f'{svc_name}.{self.name}', 80, self.superpod_container, clear)
        self.add_route('tcp', svc_name, 443, self.superpod_container, tls)
        self.add_route('tcp', f'{svc_name}.{self.name}', 443, self.superpod_container, clear)

    def add_route(self, protocol: str, src_name: str, src_port: int, dest: Container, dest_port: int) -> None:
        key = f'{src_name}:{src_port}'

        rdict = {
            'protocol': protocol,
            'src_name': src_name,
            'src_port': src_port,
            'dest_name': dest.path,
            'dest_port': dest_port
        }

        if key not in self.routes:
            self.routes[key] = rdict
        elif self.routes[key] != rdict:
            raise Exception(f'Topology {self.name}: route collision for {key}')

    def as_dict(self) -> dict:
        return {
            'name': self.name,
            'dns': { name: ( c.path, getattr(c, 'ip', '-no IP yet-') ) for name, c in self.dns.items() },
            'containers': { name: c.as_dict() for name, c in self.containers.items() },
            'routes': self.routes
        }


def parsed_configs(n: 'Node', configs: Dict[str, str], key: str,
                   ambassador_id: Optional[str]) -> Optional[List[dict]]:
    v = configs.get(key)

    if v:
        v = n.format(v)

        cfgs = load(f'{n.name}.{n.namespace} {key}', v, Tag.MAPPING)

        if ambassador_id:
            for el in cfgs:
                if not 'ambassador_id' in el:
                    el['ambassador_id'] = ambassador_id

        return cfgs.as_python()

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

        # print(f'...{n.name}')

        ambassador_id = getattr(n, 'ambassador_id', None)

        cfgs = parsed_configs(n, configs, 'self', ambassador_id)
        crds = parsed_configs(n, configs, 'CRD', ambassador_id)

        # This is an Ambassador pod.
        self.add_container(
            AmbassadorContainer(
                node=n,
                name=n.path.k8s,
                path=n.path.fqdn,
                ambassador_id=ambassador_id,
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
            target_path = pod_name
            tgt_configs = parsed_configs(n, configs, target_name, ambassador_id)

            target = getattr(n, pod_name, None)

            if target:
                target_name = target.path.k8s
                target_path = target.path.fqdn

            # print(f'    {target_name}: {pod_info}')

            svctype = pod_info.get('servicetype', None)

            if svctype:
                self.register_superpod(n.namespace, target_name, svctype, tgt_configs)
            else:
                self.add_container(Container(node=n, namespace=n.namespace, name=target_name,
                                             path=target_path, configs=tgt_configs, **pod_info))

    def get_namespace(self, name: str) -> Namespace:
        if name not in self.namespaces:
            self.namespaces[name] = Namespace(name)

        return self.namespaces[name]

    def all_namespaces(self) -> List[Namespace]:
        for k in sorted(self.namespaces.keys()):
            yield self.namespaces[k]

    def all_containers(self) -> List[Container]:
        for namespace in self.all_namespaces():
            yield from namespace.all_containers()

    def all_routes(self) -> List[Dict[str, Union[int, str]]]:
        for namespace in self.all_namespaces():
            yield from namespace.all_routes()

    def add_container(self, c: Container) -> None:
        self.get_namespace(c.namespace).add_container(c)

    def register_superpod(self, namespace: str, svc_name: str, svc_type: str, configs: Optional[ConfigSpec]) -> None:
        self.get_namespace(namespace).register_superpod(svc_name=svc_name, svc_type=svc_type, configs=configs)

    def as_dict(self) -> dict:
        return {
            'namespaces': { name: n.as_dict() for name, n in self.namespaces.items() }
        }
