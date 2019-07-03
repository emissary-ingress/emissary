from typing import Dict, List, Optional, Sequence, Tuple, TYPE_CHECKING

import json
import os

from kat.parser import dump
from kat.utils import ShellCommand, KAT_FAMILY

if TYPE_CHECKING:
    from kat.topology import Topology, Container

class DockerDriver:
    """
    DockerDriver: set up the world to run Ambassador tests completely within Docker,
    with Kubernetes at all. In general, this is will not be a good way to test discovery
    mechanisms -- it's about fast feature testing. It also does not support anything
    except endpoint routing.

    The Docker driver works as follows:

    - We create a Docker network named 'kat-dmz' which is where the test client will run.

    - For each namespace in our test topology, we create a Docker network named
      'kat-$namespace' which is where the upstream test services will run.

    - All of our Ambassadors are dual-homed, with an interface on the 'kat-dmz' network
      and an interface on the 'kat-$namespace' network.

    - Since the Ambassadors all have separate IP addresses on kat-dmz, and since
      user-defined Docker bridge networks have a DNS that can resolve container names,
      the test client can talk to the Ambassadors by hostname (with a port number).

    - We can include Service records in the Ambassador configurations that explicitly
      define the port mappings and IP addresses on kat-$namespace for upstream services.«
    """

    @classmethod
    def kill_old_world(cls) -> None:
        """
        Kill old containers and networks.

        :return: None.
        """
        cls.kill_old_containers()
        cls.kill_old_networks()

    @classmethod
    def kill_old_containers(cls) -> None:
        """
        Kill and remove old containers, which is to say all containers with the label
        'kat-family=ambassador'. The assumption here is that you're getting ready for
        a new test run, but this method can be used for cleanup as well.

        :return: None
        """

        cmd = ShellCommand('find old docker container IDs',
                           'docker', 'ps', '-a', '-f', f'label=kat-family={KAT_FAMILY}', '--format', '{{.ID}}')

        if cmd.check():
            ids = cmd.stdout.split('\n')

            while ids:
                if ids[-1]:
                    break

                ids.pop()

            if ids:
                print("Killing old containers...")
                ShellCommand.run('kill old containers', 'docker', 'kill', *ids, verbose=True, may_fail=True)
                ShellCommand.run('rm old containers', 'docker', 'rm', *ids, verbose=True, may_fail=True)

    @classmethod
    def kill_old_networks(cls) -> None:
        """
        Kill and remove old networks, which is to say all networks with the label
        'kat-family=ambassador'. The assumption here is that you're getting ready for
        a new test run, but this method can be used for cleanup as well.

        :return: None
        """

        cmd = ShellCommand('find old docker network IDs',
                           'docker', 'network', 'ls', '-f', f'label=kat-family={KAT_FAMILY}', '--format', '{{.ID}}')

        if cmd.check():
            ids = cmd.stdout.split('\n')

            while ids:
                if ids[-1]:
                    break

                ids.pop()

            if ids:
                print("Killing old networks...")
                ShellCommand.run('kill old networks', 'docker', 'network', 'rm', *ids, verbose=True, may_fail=True)

    def create_network_cmd(self, name: str, subnet: Optional[str]=None, internal: Optional[bool]=True) -> ShellCommand:
        command = [ "docker", "network", "create",
                    "--label", f'kat-family={KAT_FAMILY}' ]

        if internal:
            command.append("--internal")

        if subnet:
            command.extend([ "--subnet", subnet ])

        command.append(name)

        return ShellCommand(f'create network {name}', *command, verbose=True, defer=True)

    def start_container_cmd(self, container: 'Container',
                            network: str, ip: Optional[str]=None,
                            host_ports: Optional[List[Tuple[int, int]]]=None,
                            mounts: Optional[List[Tuple[str, str]]]=None) -> ShellCommand:
        command = [ "docker", "run", "-d",
                    "-l", f'kat-family={KAT_FAMILY}',
                    "--name", container.path,
                    "--network", network ]

        if container.is_ambassador:
            command.extend([ '-l', 'kat-type=ambassador' ])

        if ip:
            command.extend([ "--ip", ip ])

        envs = dict(container.envs) if container.envs else {}

        if container.ambassador_id:
            envs['AMBASSADOR_ID'] = container.ambassador_id

        if container.namespace:
            envs['AMBASSADOR_NAMESPACE'] = container.namespace

        for key, value in envs.items():
            command.extend([ '-e', f'{key}={value}' ])

        if host_ports:
            for src_port, dst_port in host_ports:
                command.extend([ '-p', f'{src_port}:{dst_port}' ])

        if mounts:
            for src_dir, dst_dir in mounts:
                command.extend([ '-v', f'{src_dir}:{dst_dir}' ])

        command.append(container.image)

        return ShellCommand(f'start container {container.path}', *command, verbose=True, defer=True)

    def attach_container_cmd(self, container: 'Container', attach_to: str) -> ShellCommand:
        return ShellCommand(f'attach container {container.path} to {attach_to}',
                            'docker', 'network', 'connect', attach_to, container.path, verbose=True, defer=True)

    def __init__(self, topology: 'Topology') -> None:
        """
        Initialize a DockerDriver object. Don't alter Docker configuration yet.

        :param topology: The incoming test Topology
        """

        # Assemble all configurations for every container -- in Kube, they
        # would all see the same thing, so we'll do that here.

        self.topology = topology
        self.ok = True

        self.network_cmds: List[ShellCommand] = []
        self.container_cmds: List[ShellCommand] = []
        self.attach_cmds: List[ShellCommand] = []

        self.cmds = [
            self.network_cmds,
            self.container_cmds,
            self.attach_cmds
        ]

    def setup(self) -> bool:
        """
        Tear down any old Docker networks and containers, and set up the Docker world
        for our test topology.

        *Calling setup kills any old DockerDriver setup.*

        :return: True if all is ready to go, False if we couldn't get things running
        """

        if os.environ.get('KAT_DOCKER_READY', None):
            print(f'DockerDriver assuming Docker is already set up')
            return True

        DockerDriver.kill_old_world()

        subnet = 1

        print(f'DockerDriver: starting DMZ...')

        cmd = self.create_network_cmd('kat-dmz', internal=False)
        cmd.start()

        if not cmd.check():
            return False

        for namespace in self.topology.all_namespaces():
            print(f'DockerDriver: working on {namespace.name}...')

            namespace_net = f'kat-{namespace.name}'

            print(f'    ...prepping for network {namespace_net}')

            ip_prefix = f'10.106.{subnet}'
            ip_host = 2

            self.network_cmds.append(self.create_network_cmd(namespace_net, subnet=f'{ip_prefix}.0/24'))

            print(f'    ...assembling configs')

            configs = []

            for container in namespace.all_containers():
                ip = f'{ip_prefix}.{ip_host}'
                container.ip = ip
                ip_host += 1

                amod = None
                ambassador_id = 'default'

                if container.is_ambassador:
                    ambassador_id = container.ambassador_id

                for config in container.all_configs():
                    if (container.is_ambassador and
                        (config['kind'] == 'Module') and
                        (config['name'] == 'ambassador')):
                        amod = config
                    else:
                        configs.append(config)

                if container.is_ambassador:
                    if not amod:
                        amod = {
                            'apiVersion': 'getambassador.io/v1',
                            'kind': 'Module',
                            'name': 'ambassador',
                            'ambassador_id': ambassador_id,
                            'config': {}
                        }

                    amod_config = amod.get('config', {})

                    if 'resolver' not in amod_config:
                        amod_config['resolver'] = 'endpoint'

                    if 'load_balancer' not in amod_config:
                        amod_config['load_balancer'] = {}

                    if 'policy' not in amod_config['load_balancer']:
                        amod_config['load_balancer']['policy'] = 'round_robin'

                    print(f'    ...{container.name} has Ambassador module: {json.dumps(amod, sort_keys=True)}')
                    configs.append(amod)

            # Ew.
            for route in namespace.all_routes():
                protocol = route['protocol']

                if protocol != 'tcp':
                    continue

                src_name = route['src_name']
                src_port = route['src_port']
                dest_name = route['dest_name']
                dest_port = route['dest_port']

                target_container = namespace.dns[dest_name]
                target_ip = target_container.ip

                print(f'    ...routing {src_name}:{src_port} to {dest_name} [{target_ip}] port {dest_port}')

                for container in namespace.all_containers():
                    if container.is_ambassador:
                        entry = {
                            'apiVersion': 'getambassador.io/v1',
                            'kind': 'Service',
                            'name': f'k8s-{src_name}-{namespace.name}',
                            'ambassador_id': container.ambassador_id,
                            'endpoints': {
                                f'{src_port}': [ {
                                    'ip': target_ip,
                                    'port': dest_port
                                } ]
                            }
                        }

                        configs.append(entry)

            config_path = None

            if configs:
                config_path = f'/tmp/kat-DockerDriver-{namespace.name}.yaml'

                with open(config_path, "w") as config_yaml:
                    config_yaml.write(dump(configs))

            for container in namespace.all_containers():
                print(f'    ...prepping for container {container.path} @ {container.ip} {"(Ambassador)" if container.is_ambassador else ""}')

                mounts = None

                if container.is_ambassador and config_path:
                    mounts = [ ( config_path, '/ambassador/ambassador-config/kat.yaml' ) ]

                self.container_cmds.append(self.start_container_cmd(container=container,
                                                                    network=namespace_net,
                                                                    ip=container.ip,
                                                                    mounts=mounts))

                self.attach_cmds.append(self.attach_container_cmd(container=container,
                                                                  attach_to='kat-dmz'))

            subnet += 1

        print('DockerDriver: prepping for DMZ container...')

        os.makedirs("/tmp/kat-DockerDriver-hostinfo", mode=0o755, exist_ok=True)

        self.container_cmds.append(ShellCommand('starting DMZ container',
                                                'docker', 'run', '--detach', '--rm', '--name', 'kat',
                                                '-l', f'kat-family={KAT_FAMILY}',
                                                '--network', 'kat-dmz',
                                                '-v', '/tmp/kat-DockerDriver-hostinfo:/tmp/hostinfo',
                                                '--entrypoint', '/bin/sh',
                                                'dwflynn/tzone:1',
                                                '-c', 'sleep 3600', verbose=True, defer=True))

        # OK. Fire it all up.
        for cmdset in self.cmds:
            if not self.run_all(cmdset):
                return False

        return True

    def run_all(self, cmdset: List[ShellCommand]) -> bool:
        all_good = True

        for cmd in cmdset:
            cmd.start()

        for cmd in cmdset:
            if not cmd.check():
                all_good = False
        
        return all_good

    def pods_ready(self, requirements) -> Tuple[bool, List[str]]:
        return (True, None)

    def run_queries(self, queries: Sequence[dict]) -> Sequence[dict]:
        # print('DockerDriver: running %d queries' % len(queries))

        with open("/tmp/kat-DockerDriver-hostinfo/urls.json", "w") as f:
            json.dump(queries, f)

        if not ShellCommand.run('DockerDriver: running queries',
                                'docker', 'exec', 'kat', './kat-client',
                                '-input', '/tmp/hostinfo/urls.json',
                                '-output', '/tmp/hostinfo/results.json'):
            return None

        with open("/tmp/kat-DockerDriver-hostinfo/results.json") as f:
            json_results = json.load(f)

        return json_results



