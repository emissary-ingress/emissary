from typing import Any, Dict, List, Optional, Tuple, TYPE_CHECKING
# from typing import cast as typecast

import json
import logging
import os
import yaml

from .config import Config
from .acresource import ACResource

from ..utils import parse_yaml, dump_yaml

AnyDict = Dict[str, Any]
HandlerResult = Optional[Tuple[str, List[AnyDict]]]

# Some thoughts:
# - loading a bunch of Ambassador resources is different from loading a bunch of K8s
#   services, because we should assume that if we're being a fed a bunch of Ambassador
#   resources, we'll get a full set. The whole 'secret loader' thing needs to have the
#   concept of a TLSSecret resource that can be force-fed to us, or that can be fetched
#   through the loader if needed.
# - If you're running a debug-loop Ambassador, you should just have a flat (or
#   recursive, I don't care) directory full of Ambassador YAML, including TLSSecrets
#   and Endpoints and whatnot, as needed. All of it will get read by
#   load_from_filesystem and end up in the elements array.
# - If you're running expecting to be fed by kubewatch, at present kubewatch will
#   send over K8s Service records, and anything annotated in there will end up in
#   elements. This may include TLSSecrets or Endpoints. Any TLSSecret mentioned that
#   isn't already in elements will need to be fetched.
# - Ambassador resources do not have namespaces. They have the ambassador_id. That's
#   it. The ambassador_id is completely orthogonal to the namespace. No element with
#   the wrong ambassador_id will end up in elements. It would be nice if they were
#   never sent by kubewatch, but, well, y'know.
# - TLSSecret resources are not TLSContexts. TLSSecrets only have a name, a private
#   half, and a public half. They do _not_ have other TLSContext information.
# - Endpoint resources probably have just a name, a service name, and an endpoint
#   address.

class ResourceFetcher:
    def __init__(self, logger: logging.Logger, aconf: 'Config') -> None:
        self.aconf = aconf
        self.logger = logger
        self.elements: List[ACResource] = []
        self.filename: Optional[str] = None
        self.ocount: int = 1
        self.saved: List[Tuple[Optional[str], int]] = []

    @property
    def location(self):
        return "%s.%d" % (self.filename or "anonymous YAML", self.ocount)

    def push_location(self, filename: Optional[str], ocount: int) -> None:
        self.saved.append((self.filename, self.ocount))
        self.filename = filename
        self.ocount = ocount

    def pop_location(self) -> None:
        self.filename, self.ocount = self.saved.pop()

    def load_from_filesystem(self, config_dir_path, recurse: bool=False, k8s: bool=False):
        inputs: List[Tuple[str, str]] = []

        if os.path.isdir(config_dir_path):
            dirs = [ config_dir_path ]

            while dirs:
                dirpath = dirs.pop(0)

                for filename in os.listdir(dirpath):
                    filepath = os.path.join(dirpath, filename)

                    if recurse and os.path.isdir(filepath):
                        # self.logger.debug("%s: RECURSE" % filepath)
                        dirs.append(filepath)
                        continue

                    if not os.path.isfile(filepath):
                        # self.logger.debug("%s: SKIP non-file" % filepath)
                        continue

                    if not filename.lower().endswith('.yaml'):
                        # self.logger.debug("%s: SKIP non-YAML" % filepath)
                        continue

                    # self.logger.debug("%s: SAVE configuration file" % filepath)
                    inputs.append((filepath, filename))

        else:
            # this allows a file to be passed into the ambassador cli
            # rather than just a directory
            inputs.append((config_dir_path, os.path.basename(config_dir_path)))

        for filepath, filename in inputs:
            self.logger.info("reading %s (%s)" % (filename, filepath))

            try:
                serialization = open(filepath, "r").read()
                self.parse_yaml(serialization, k8s=k8s, filename=filename)
            except IOError as e:
                self.aconf.post_error("could not read YAML from %s: %s" % (filepath, e))

    def parse_yaml(self, serialization: Optional[str], k8s=False, rkey: Optional[str]=None,
                   filename: Optional[str]=None) -> None:
        # self.logger.debug("%s: parsing %d byte%s of YAML:\n%s" %
        #                   (self.location, len(serialization), "" if (len(serialization) == 1) else "s",
        #                    serialization))

        try:
            objects = parse_yaml(serialization)
            self.parse_object(objects=objects, k8s=k8s, rkey=rkey, filename=filename)
        except yaml.error.YAMLError as e:
            self.aconf.post_error("%s: could not parse YAML: %s" % (self.location, e))

    def parse_watt(self, serialization: Optional[str]) -> None:
        try:
            watt_dict = parse_yaml(serialization)[0]

            watt_k8s = watt_dict.get('Kubernetes', {})

            for key in [ 'service', 'endpoints', 'secret' ]:
                for obj in watt_k8s.get(key, []):
                    self.handle_k8s(obj)

            watt_consul = watt_dict.get('Consul', {})
            consul_endpoints = watt_consul.get('Endpoints', {})

            for consul_rkey, consul_object in consul_endpoints.items():
                result = self.handle_consul_service(consul_rkey, consul_object)

                if result:
                    rkey, parsed_objects = result

                    self.parse_object(parsed_objects, k8s=False,
                                      filename=self.filename, rkey=rkey)
        except yaml.error.YAMLError as e:
            self.aconf.post_error("%s: could not parse WATT: %s" % (self.location, e))

    def handle_k8s(self, obj: dict) -> None:
        # self.logger.debug("handle_k8s obj %s" % json.dumps(obj, indent=4, sort_keys=True))

        kind = obj.get('kind')

        if not kind:
            # self.logger.debug("%s: ignoring K8s object, no kind" % self.location)
            return

        handler_name = f'handle_k8s_{kind.lower()}'
        handler = getattr(self, handler_name, None)

        if not handler:
            # self.logger.debug("%s: ignoring K8s object, no kind" % self.location)
            return

        result = handler(obj)

        if result:
            rkey, parsed_objects = result

            self.parse_object(parsed_objects, k8s=False,
                              filename=self.filename, rkey=rkey)

    def parse_object(self, objects, k8s=False, rkey: Optional[str]=None, filename: Optional[str]=None):
        self.push_location(filename, 1)

        # self.logger.debug("PARSE_OBJECT: incoming %d" % len(objects))

        for obj in objects:
            self.logger.debug("PARSE_OBJECT: checking %s" % obj)

            if k8s:
                self.handle_k8s(obj)
            else:
                # if not obj:
                #     self.logger.debug("%s: empty object from %s" % (self.location, serialization))

                self.process_object(obj, rkey=rkey)
                self.ocount += 1

        self.pop_location()

    def process_object(self, obj: dict, rkey: Optional[str]=None) -> None:
        if not isinstance(obj, dict):
            # Bug!!
            if not obj:
                self.aconf.post_error("%s is empty" % self.location)
            else:
                self.aconf.post_error("%s is not a dictionary? %s" %
                                      (self.location, json.dumps(obj, indent=4, sort_keys=4)))
            return

        if not self.aconf.good_ambassador_id(obj):
            # self.logger.debug("%s ignoring K8s Service with mismatched ambassador_id" % self.location)
            return

        if 'kind' not in obj:
            # Bug!!
            self.aconf.post_error("%s is missing 'kind'?? %s" %
                                  (self.location, json.dumps(obj, indent=4, sort_keys=True)))
            return

        # self.logger.debug("%s PROCESS %s initial rkey %s" % (self.location, obj['kind'], rkey))

        # Is this a pragma object?
        if obj['kind'] == 'Pragma':
            # Why did I think this was a good idea? [ :) ]
            new_source = obj.get('source', None)

            if new_source:
                # We don't save the old self.filename here, so this change will last until
                # the next input source (or the next Pragma).
                self.filename = new_source

            # Don't count Pragma objects, since the user generally doesn't write them.
            self.ocount -= 1
            return

        if not rkey:
            rkey = self.filename

        rkey = "%s.%d" % (rkey, self.ocount)

        # self.logger.debug("%s PROCESS %s updated rkey to %s" % (self.location, obj['kind'], rkey))

        # Fine. Fine fine fine.
        serialization = dump_yaml(obj, default_flow_style=False)

        r = ACResource.from_dict(rkey, rkey, serialization, obj)
        self.elements.append(r)

        # self.logger.debug("%s PROCESS %s save %s: %s" % (self.location, obj['kind'], rkey, serialization))

    def sorted(self, key=lambda x: x.rkey):  # returns an iterator, probably
        return sorted(self.elements, key=key)

    def handle_k8s_endpoints(self, k8s_object: AnyDict) -> HandlerResult:
        # Don't include Endpoints unless endpoint routing is enabled.
        if not Config.enable_endpoints:
            return None

        metadata = k8s_object.get('metadata', None)
        resource_name = metadata.get('name')
        resource_namespace = metadata.get('namespace', 'default')

        subsets = []
        for subset in k8s_object.get('subsets', []):
            addresses = []
            for address in subset.get('addresses', []):
                add = {}

                ip = address.get('ip', None)
                if ip is not None:
                    add['ip'] = ip

                node = address.get('nodeName', None)
                if node is not None:
                    add['node'] = node

                target_ref = address.get('targetRef', None)
                if target_ref is not None:
                    target_kind = target_ref.get('kind', None)
                    if target_kind is not None:
                        add['target_kind'] = target_kind

                    target_name = target_ref.get('name', None)
                    if target_name is not None:
                        add['target_name'] = target_name

                    target_namespace = target_ref.get('namespace', None)
                    if target_namespace is not None:
                        add['target_namespace'] = target_namespace

                if len(add) > 0:
                    addresses.append(add)

            if len(addresses) == 0:
                continue

            ports = subset.get('ports', [])

            # XXX LOAD_BALANCER HACK This is horrible: we take _way_ too many endpoints this
            # XXX way! We need to only accept `Endpoints` that match `Mappings`.
            subsets.append({
                'apiVersion': 'ambassador/v1',
                'ambassador_id': Config.ambassador_id,
                'kind': 'Endpoints',
                'name': resource_name,
                'addresses': addresses,
                'ports': ports
            })

        if len(subsets) == 0:
            return None

        resource_identifier = '{name}.{namespace}'.format(namespace=resource_namespace, name=resource_name)

        return resource_identifier, subsets

    def handle_k8s_service(self, k8s_object: AnyDict) -> HandlerResult:
        # XXX Really we shouldn't generate these unless there's a namespace match. Ugh.
        #
        # XXX LOAD_BALANCER HACK This is horrible: we take _way_ too many endpoints this
        # XXX way! We need to only generate `ServiceInfo` resources that match `Mappings`.

        service_info = {
            'apiVersion': 'ambassador/v1',
            'ambassador_id': Config.ambassador_id,
            'kind': 'ServiceInfo'
        }
        spec = k8s_object.get('spec', None)
        if spec is not None:
            ports = spec.get('ports', None)
            if ports is not None:
                service_info['ports'] = []
                for port in ports:
                    service_info['ports'].append(port)

        metadata = k8s_object.get('metadata', None)
        resource_name = metadata.get('name') if metadata else None
        resource_namespace = metadata.get('namespace', 'default') if metadata else None
        service_info['name'] = resource_name
        annotations = metadata.get('annotations', None) if metadata else None
        if annotations:
            annotations = annotations.get('getambassador.io/config', None)

        skip = False

        if not metadata:
            self.logger.debug("ignoring K8s Service with no metadata")
            skip = True

        if not resource_name:
            self.logger.debug("ignoring K8s Service with no name")
            skip = True

        if not skip and (Config.single_namespace and (resource_namespace != Config.ambassador_namespace)):
            # This should never happen in actual usage, since we shouldn't be given things
            # in the wrong namespace. However, in development, this can happen a lot.
            self.logger.debug("ignoring K8s Service in wrong namespace")
            skip = True

        if skip:
            return None

        # This resource identifier is useful for log output since filenames can be duplicated (multiple subdirectories)
        resource_identifier = '{name}.{namespace}'.format(namespace=resource_namespace, name=resource_name)

        objects = []

        if annotations:
            if (self.filename is not None) and (not self.filename.endswith(":annotation")):
                self.filename += ":annotation"

            try:
                objects = parse_yaml(annotations)
            except yaml.error.YAMLError as e:
                self.logger.debug("could not parse YAML: %s" % e)

        # Don't include service_info unless endpoint routing is enabled.
        if Config.enable_endpoints:
            objects.append(service_info)

        return resource_identifier, objects

    # Handler for K8s Secret resources.
    def handle_k8s_secret(self, k8s_object: AnyDict) -> HandlerResult:
        # XXX Another one where we shouldn't be saving everything.

        secret_type = k8s_object.get('type', None)
        metadata = k8s_object.get('metadata', None)
        resource_name = metadata.get('name') if metadata else None
        resource_namespace = metadata.get('namespace', 'default') if metadata else None
        data = k8s_object.get('data', None)

        skip = False

        if (secret_type != 'kubernetes.io/tls') and (secret_type != 'Opaque'):
            self.logger.debug("ignoring K8s Secret with unknown type %s" % secret_type)
            skip = True

        if not data:
            self.logger.debug("ignoring K8s Secret with no data")
            skip = True

        if not metadata:
            self.logger.debug("ignoring K8s Secret with no metadata")
            skip = True

        if not resource_name:
            self.logger.debug("ignoring K8s Secret with no name")
            skip = True

        if not skip and (Config.single_namespace and (resource_namespace != Config.ambassador_namespace)):
            # This should never happen in actual usage, since we shouldn't be given things
            # in the wrong namespace. However, in development, this can happen a lot.
            self.logger.debug("ignoring K8s Secret in wrong namespace")
            skip = True

        if skip:
            return None

        # This resource identifier is useful for log output since filenames can be duplicated (multiple subdirectories)
        resource_identifier = f'{resource_name}.{resource_namespace}'

        tls_crt = data.get('tls.crt', None)
        tls_key = data.get('tls.key', None)

        if not tls_crt and not tls_key:
            # Uh. WTFO?
            self.logger.debug(f'ignoring K8s Secret {resource_identifier} with no keys')
            return None

        secret_info = {
            'apiVersion': 'ambassador/v1',
            'ambassador_id': Config.ambassador_id,
            'kind': 'Secret',
            'name': resource_name,
            'namespace': resource_namespace
        }

        if tls_crt:
            secret_info['tls_crt'] = tls_crt

        if tls_key:
            secret_info['tls_key'] = tls_key

        return resource_identifier, [ secret_info ]

    # Handler for Consul services
    def handle_consul_service(self,
                              consul_rkey: str, consul_object: AnyDict) -> HandlerResult:
        # resource_identifier = f'consul-{consul_rkey}'

        endpoints = consul_object.get('Endpoints', [])
        name = consul_object.get('Service', consul_rkey)

        if len(endpoints) < 1:
            # Bzzt.
            self.logger.debug(f"ignoring Consul service {name} with no Endpoints")
            return None

        # We need to generate a ServiceInfo object and an Endpoints object. Ew.
        #
        # XXX ...we need port info here, don't we? For now we'll hardcode it
        # to port 80. <cough>

        service_info = {
            'apiVersion': 'ambassador/v1',
            'ambassador_id': Config.ambassador_id,
            'kind': 'ServiceInfo',
            'name': name,
            'ports': [
                {
                    'name': 'http',
                    'port': 80,
                    'protocol': 'TCP'
                }
            ]
        }

        objects = [ service_info ]

        addresses = []
        ports_dict = {}

        for ep in endpoints:
            ep_addr = ep.get('Address')
            ep_port = ep.get('Port')

            if not ep_addr or not ep_port:
                self.logger.debug(f"ignoring Consul service {name} endpoint {ep['ID']} missing address info")
                continue

            addresses.append({
                'ip': ep_addr,
                'target_kind': 'Consul'
            })

            ports_dict[ep_port] = True

        ports = [ { 'port': port, 'protocol': 'TCP' } for port in ports_dict.keys() ]

        endpoint = {
            'apiVersion': 'ambassador/v1',
            'ambassador_id': Config.ambassador_id,
            'kind': 'Endpoints',
            'name': name,
            'addresses': addresses,
            'ports': ports
        }

        objects.append(endpoint)

        return name, objects
