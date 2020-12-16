from typing import Any, Dict, List, Optional, Tuple, Union

import json
import logging
import os
import yaml
import re

from ..config import ACResource, Config
from ..utils import parse_yaml, parse_json, dump_json

from .resource import NormalizedResource, ResourceManager
from .k8sobject import KubernetesGVK, KubernetesObject
from .k8sprocessor import (
    KubernetesProcessor,
    AggregateKubernetesProcessor,
    DeduplicatingKubernetesProcessor,
    CountingKubernetesProcessor,
)
from .ambassador import AmbassadorProcessor
from .service import ServiceProcessor
from .knative import KnativeIngressProcessor

AnyDict = Dict[str, Any]
HandlerResult = Optional[Tuple[str, List[AnyDict]]]

# XXX ALL OF THE BELOW COMMENT IS PROBABLY OUT OF DATE. (Flynn, 2019-10-29)
#
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

k8sLabelMatcher = re.compile(r'([\w\-_./]+)=\"(.+)\"')


class ResourceFetcher:
    manager: ResourceManager
    k8s_processor: KubernetesProcessor

    def __init__(self, logger: logging.Logger, aconf: 'Config',
                 skip_init_dir: bool=False, watch_only=False) -> None:
        self.aconf = aconf
        self.logger = logger
        self.manager = ResourceManager(self.logger, self.aconf)

        self.k8s_processor = DeduplicatingKubernetesProcessor(AggregateKubernetesProcessor([
            CountingKubernetesProcessor(self.aconf, KubernetesGVK.for_knative_networking('Ingress'), 'knative_ingress'),
            AmbassadorProcessor(self.manager),
            ServiceProcessor(self.manager, watch_only=watch_only),
            KnativeIngressProcessor(self.manager),
        ]))

        self.alerted_about_labels = False

        # For deduplicating objects coming in from watt
        self.k8s_parsed: Dict[str, bool] = {}

        # Deltas, for managing the cache.
        self.deltas: List[Dict[str, Union[str, Dict[str, str]]]] = []

        # HACK
        # If AGENT_SERVICE is set, skip the init dir: we'll force some defaults later
        # instead.
        #
        # XXX This is rather a hack. We can do better.

        if os.environ.get("AGENT_SERVICE", "").lower() != "":
            logger.debug("Intercept agent active: skipping init dir")
            skip_init_dir = True

        if not skip_init_dir:
            # Check /ambassador/init-config for initialization resources -- note NOT
            # $AMBASSADOR_CONFIG_BASE_DIR/init-config! This is compile-time stuff that
            # doesn't move around if you change the configuration base.
            init_dir = '/ambassador/init-config'

            if os.path.isdir(init_dir):
                self.load_from_filesystem(init_dir, k8s=True, recurse=True, finalize=False)

    @property
    def elements(self) -> List[ACResource]:
        return self.manager.elements

    @property
    def location(self) -> str:
        return str(self.manager.locations.current)

    def load_from_filesystem(self, config_dir_path, recurse: bool=False,
                             k8s: bool=False, finalize: bool=True):
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
            self.logger.debug("reading %s (%s)" % (filename, filepath))

            try:
                serialization = open(filepath, "r").read()
                self.parse_yaml(serialization, k8s=k8s, filename=filename, finalize=False)
            except IOError as e:
                self.aconf.post_error("could not read YAML from %s: %s" % (filepath, e))

        if finalize:
            self.finalize()

    def parse_yaml(self, serialization: str, k8s=False, rkey: Optional[str]=None,
                   filename: Optional[str]=None, finalize: bool=True, metadata_labels: Optional[Dict[str, str]]=None) -> None:
        # self.logger.info(f"RF YAML: {serialization}")

        # Expand environment variables allowing interpolation in manifests.
        serialization = os.path.expandvars(serialization)

        try:
            # UGH. This parse_yaml is the one we imported from utils. XXX This needs to be fixed.
            objects = parse_yaml(serialization)
            self.parse_object(objects=objects, k8s=k8s, rkey=rkey, filename=filename)
        except yaml.error.YAMLError as e:
            self.aconf.post_error("%s: could not parse YAML: %s" % (self.location, e))

        if finalize:
            self.finalize()

    def parse_json(self, serialization: str, k8s=False, rkey: Optional[str]=None,
                   filename: Optional[str]=None, finalize: bool=True) -> None:
        # self.logger.debug("%s: parsing %d byte%s of YAML:\n%s" %
        #                   (self.location, len(serialization), "" if (len(serialization) == 1) else "s",
        #                    serialization))

        # Expand environment variables allowing interpolation in manifests.
        serialization = os.path.expandvars(serialization)

        try:
            # This parse_json is the one we imported from utils. XXX This (also?) needs to be fixed.
            objects = parse_json(serialization)
            self.parse_object(objects=objects, k8s=k8s, rkey=rkey, filename=filename)
        except json.decoder.JSONDecodeError as e:
            self.aconf.post_error("%s: could not parse YAML: %s" % (self.location, e))

        if finalize:
            self.finalize()

    def parse_watt(self, serialization: str, finalize: bool=True) -> None:
        basedir = os.environ.get('AMBASSADOR_CONFIG_BASE_DIR', '/ambassador')

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds')):
            self.aconf.post_error("Ambassador could not find core CRD definitions. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds_2')):
            self.aconf.post_error("Ambassador could not find Resolver type CRD definitions. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador, but ConsulResolver, KubernetesEndpointResolver, and KubernetesServiceResolver resources will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds_3')):
            self.aconf.post_error("Ambassador could not find the Host CRD definition. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador, but Host resources will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds_4')):
            self.aconf.post_error("Ambassador could not find the LogService CRD definition. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador, but LogService resources will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds_5')):
            self.aconf.post_error("Ambassador could not find the DevPortal CRD definition. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador, but DevPortal resources will be ignored...")

        # We could be posting errors about the missing IngressClass resource, but given it's new in K8s 1.18
        # and we assume most users would be worried about it when running on older clusters, we'll rely on
        # Ambassador logs "Ambassador does not have permission to read IngressClass resources" for the moment.
        #if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_ingress_class')):
        #    self.aconf.post_error("Ambassador is not permitted to read IngressClass resources. Please visit https://www.getambassador.io/user-guide/ingress-controller/ for more information. You can continue using Ambassador, but IngressClass resources will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_ingress')):
            self.aconf.post_error("Ambassador is not permitted to read Ingress resources. Please visit https://www.getambassador.io/user-guide/ingress-controller/ for more information. You can continue using Ambassador, but Ingress resources will be ignored...")

        # Expand environment variables allowing interpolation in manifests.
        serialization = os.path.expandvars(serialization)

        self.load_pod_labels()

        try:
            watt_dict = parse_json(serialization)

            # Grab deltas if they're present...
            self.deltas = watt_dict.get('Deltas', [])

            # ...then it's off to deal with Kubernetes.
            watt_k8s = watt_dict.get('Kubernetes', {})

            # First, though, let's fold any invalid objects into the main watt_k8s
            # tree. They're in the "Invalid" dict simply because we don't fully trust
            # round-tripping an invalid object through our Golang parsers for Ambassador
            # configuration objects.
            #
            # Why, you may ask, do we want to dump invalid objects back in to be
            # processed??? It's because they have error information that we need to
            # propagate to the user, and this is the simplest way to do that.

            invalid: List[Dict] = watt_dict.get('Invalid') or []

            for obj in invalid:
                kind = obj.get('kind', None)

                if not kind:
                    # Can't work with this at _all_.
                    self.logger.error(f"skipping invalid object with no kind: {obj}")
                    continue
            
                # We can't use watt_k8s.setdefault() here because many keys have
                # explicit null values -- they'll need to be turned into empty lists
                # and re-saved, and setdefault() won't do that for an explicit null.
                watt_list = watt_k8s.get(kind)

                if not watt_list:
                    watt_list = []
                    watt_k8s[kind] = watt_list

                watt_list.append(obj)

            # These objects have to be processed first, in order, as they depend
            # on each other.
            watt_k8s_keys = ['service', 'endpoints', 'secret', 'ingressclasses', 'ingresses']

            # Then we add everything else to be processed.
            watt_k8s_keys += watt_k8s.keys()

            # `dict.fromkeys(iterable)` is a convenient way to work around the
            # lack of an ordered set collection type in Python. As Python 3.7,
            # dicts are guaranteed to be insertion-ordered.
            for key in dict.fromkeys(watt_k8s_keys):
                for obj in watt_k8s.get(key) or []:
                    # self.logger.debug(f"Handling Kubernetes {key}...")
                    self.handle_k8s(obj)

            watt_consul = watt_dict.get('Consul', {})
            consul_endpoints = watt_consul.get('Endpoints', {})

            for consul_rkey, consul_object in consul_endpoints.items():
                result = self.handle_consul_service(consul_rkey, consul_object)

                if result:
                    rkey, parsed_objects = result

                    self.parse_object(parsed_objects, k8s=False, rkey=rkey)
        except json.decoder.JSONDecodeError as e:
            self.aconf.post_error("%s: could not parse WATT: %s" % (self.location, e))

        if finalize:
            self.finalize()

    def load_pod_labels(self):
        pod_labels_path = '/tmp/ambassador-pod-info/labels'
        if not os.path.isfile(pod_labels_path):
            if not self.alerted_about_labels:
                self.aconf.post_error(f"Pod labels are not mounted in the Ambassador container; Kubernetes Ingress support is likely to be limited")
                self.alerted_about_labels = True

            return False

        with open(pod_labels_path) as pod_labels_file:
            pod_labels = pod_labels_file.readlines()

        self.logger.debug(f"Found pod labels: {pod_labels}")
        for pod_label in pod_labels:
            pod_label_kv = k8sLabelMatcher.findall(pod_label)
            if len(pod_label_kv) != 1 or len(pod_label_kv[0]) != 2:
                self.aconf.post_notice(f"Dropping pod label {pod_label}")
            else:
                self.aconf.pod_labels[pod_label_kv[0][0]] = pod_label_kv[0][1]
        self.logger.debug(f"Parsed pod labels: {self.aconf.pod_labels}")

    def check_k8s_dup(self, kind: str, namespace: Optional[str], name: str) -> bool:
        key = f"{kind}/{name}.{namespace}"

        if key in self.k8s_parsed:
            # self.logger.debug(f"dropping K8s dup {key}")
            return False

        # self.logger.info(f"remembering K8s {key}")
        self.k8s_parsed[key] = True
        return True

    def handle_k8s(self, raw_obj: dict) -> None:
        # self.logger.debug("handle_k8s obj %s" % dump_json(obj, pretty=True))

        try:
            obj = KubernetesObject(raw_obj)
        except ValueError:
            # The object doesn't contain a kind, API version, or name, so we
            # can't process it.
            return

        with self.manager.locations.push_reset():
            if self.k8s_processor.try_process(obj):
                # Nothing else to do.
                return

        handler_name = f'handle_k8s_{obj.kind.lower()}'
        # self.logger.debug(f"looking for handler {handler_name} for K8s {kind} {name}")
        handler = getattr(self, handler_name, None)

        if not handler:
            self.logger.debug(f"{self.location}: skipping K8s {obj.gvk}")
            return

        if not self.check_k8s_dup(obj.kind, obj.namespace, obj.name):
            return

        result = handler(raw_obj)

        if result:
            rkey, parsed_objects = result

            self.parse_object(parsed_objects, k8s=False, rkey=rkey)

    def parse_object(self, objects, k8s=False, rkey: Optional[str] = None, filename: Optional[str] = None):
        if not filename:
            filename = self.manager.locations.current.filename

        self.manager.locations.push(filename=filename)

        # self.logger.debug("PARSE_OBJECT: incoming %d" % len(objects))

        for obj in objects:
            # self.logger.debug("PARSE_OBJECT: checking %s" % obj)

            if k8s:
                self.handle_k8s(obj)
            else:
                # if not obj:
                #     self.logger.debug("%s: empty object from %s" % (self.location, serialization))

                self.manager.emit(NormalizedResource(obj, rkey=rkey))

        self.manager.locations.pop()

    def sorted(self, key=lambda x: x.rkey):  # returns an iterator, probably
        return sorted(self.elements, key=key)

    def handle_k8s_ingressclass(self, k8s_object: AnyDict) -> HandlerResult:
        metadata = k8s_object.get('metadata', None)
        ingress_class_name = metadata.get('name') if metadata else None
        ingress_class_spec = k8s_object.get('spec', None)

        # Important: IngressClass is not namespaced!
        resource_identifier = f'{ingress_class_name}'

        skip = False
        if not metadata:
            self.logger.debug('ignoring K8s IngressClass with no metadata')
            skip = True
        if not ingress_class_name:
            self.logger.debug('ignoring K8s IngressClass with no name')
            skip = True
        if not ingress_class_spec:
            self.logger.debug('ignoring K8s IngressClass with no spec')
            skip = True

        # We only want to deal with IngressClasses that belong to "spec.controller: getambassador.io/ingress-controller"
        if ingress_class_spec.get('controller', '').lower() != 'getambassador.io/ingress-controller':
            self.logger.debug(f'ignoring IngressClass {ingress_class_name} without controller - getambassador.io/ingress-controller')
            skip = True

        if skip:
            return None

        annotations = metadata.get('annotations', {})
        ambassador_id = annotations.get('getambassador.io/ambassador-id', 'default')

        # We don't want to deal with non-matching Ambassador IDs
        if ambassador_id != Config.ambassador_id:
            self.logger.debug(f'IngressClass {ingress_class_name} does not have Ambassador ID {Config.ambassador_id}, ignoring...')
            return None

        # TODO: Do we intend to use this parameter in any way?
        # `parameters` is of type TypedLocalObjectReference,
        # meaning it links to another k8s resource in the same namespace.
        # https://godoc.org/k8s.io/api/core/v1#TypedLocalObjectReference
        #
        # In this case, the resource referenced by TypedLocalObjectReference
        # should not be namespaced, as IngressClass is a non-namespaced resource.
        #
        # It was designed to reference a CRD for this specific ingress-controller
        # implementation... although usage is optional and not prescribed.
        ingress_parameters = ingress_class_spec.get('parameters', {})

        self.logger.debug(f'Handling IngressClass {ingress_class_name} with parameters {ingress_parameters}...')

        # Don't return this as we won't handle IngressClass downstream.
        # Instead, save it in self.aconf.k8s_ingress_classes for reference in handle_k8s_ingress
        self.aconf.k8s_ingress_classes[resource_identifier] = ingress_parameters

        return None

    def handle_k8s_ingress(self, k8s_object: AnyDict) -> HandlerResult:
        if 'metadata' not in k8s_object:
            self.logger.debug("ignoring K8s Ingress with no metadata")
            return None

        metadata = k8s_object['metadata']

        if 'name' not in metadata:
            self.logger.debug("ignoring K8s Ingress with no name")
            return None

        ingress_name = metadata['name']

        if 'spec' not in k8s_object:
            self.logger.debug(f"ignoring K8s Ingress {ingress_name} with no spec")
            return None

        ingress_spec = k8s_object['spec']
        ingress_namespace = metadata.get('namespace') or 'default'

        metadata_labels: Optional[Dict[str, str]] = metadata.get('labels')

        resource_identifier = f'{ingress_name}.{ingress_namespace}'

        # we don't need an ingress without ingress class set to ambassador
        annotations = metadata.get('annotations', {})
        ingress_class_name = ingress_spec.get('ingressClassName', '')

        ingress_class = self.aconf.k8s_ingress_classes.get(ingress_class_name, None)
        has_ambassador_ingress_class_annotation = annotations.get('kubernetes.io/ingress.class', '').lower() == 'ambassador'

        # check the Ingress resource has either:
        #  - a `kubernetes.io/ingress.class: "ambassador"` annotation
        #  - a `spec.ingressClassName` that references an IngressClass with
        #      `spec.controller: getambassador.io/ingress-controller`
        #
        # also worth noting, the kube-apiserver might assign the `spec.ingressClassName` if unspecified
        # and only 1 IngressClass has the following annotation:
        #   annotations:
        #     ingressclass.kubernetes.io/is-default-class: "true"
        if (not has_ambassador_ingress_class_annotation) and (ingress_class is None):
            self.logger.debug(f'ignoring Ingress {ingress_name} without annotation (kubernetes.io/ingress.class: "ambassador") or IngressClass controller (getambassador.io/ingress-controller)')
            return None

        # Let's see if our Ingress resource has Ambassador annotations on it
        annotations = metadata.get('annotations', {})
        ambassador_annotations = annotations.get('getambassador.io/config', None)

        parsed_ambassador_annotations = None
        if ambassador_annotations is not None:
            self.manager.locations.mark_annotated()

            try:
                parsed_ambassador_annotations = parse_yaml(ambassador_annotations)
            except yaml.error.YAMLError as e:
                self.logger.debug("could not parse YAML: %s" % e)

        ambassador_id = annotations.get('getambassador.io/ambassador-id', 'default')

        # We don't want to deal with non-matching Ambassador IDs
        if ambassador_id != Config.ambassador_id:
            self.logger.debug(f"Ingress {ingress_name} does not have Ambassador ID {Config.ambassador_id}, ignoring...")
            return None

        self.logger.debug(f"Handling Ingress {ingress_name}...")
        # We will translate the Ingress resource into Hosts and Mappings,
        # but keep a reference to the k8s resource in aconf for debugging and stats
        self.aconf.k8s_ingresses[resource_identifier] = k8s_object

        ingress_tls = ingress_spec.get('tls', [])
        for tls_count, tls in enumerate(ingress_tls):

            tls_secret = tls.get('secretName', None)
            if tls_secret is not None:

                for host_count, host in enumerate(tls.get('hosts', ['*'])):
                    tls_unique_identifier = f"{ingress_name}-{tls_count}-{host_count}"

                    ingress_host: Dict[str, Any] = {
                        'apiVersion': 'getambassador.io/v2',
                        'kind': 'Host',
                        'metadata': {
                            'name': tls_unique_identifier,
                            'namespace': ingress_namespace
                        },
                        'spec': {
                            'ambassador_id': [ambassador_id],
                            'hostname': host,
                            'acmeProvider': {
                                'authority': 'none'
                            },
                            'tlsSecret': {
                                'name': tls_secret
                            },
                            'requestPolicy': {
                                'insecure': {
                                    'action': 'Route'
                                }
                            }
                        }
                    }

                    if metadata_labels:
                        ingress_host['metadata']['labels'] = metadata_labels

                    self.logger.debug(f"Generated Host from ingress {ingress_name}: {ingress_host}")
                    self.handle_k8s(ingress_host)

        # parse ingress.spec.defaultBackend
        # using ingress.spec.backend as a fallback, for older versions of the Ingress resource.
        default_backend = ingress_spec.get('defaultBackend', ingress_spec.get('backend', {}))
        db_service_name = default_backend.get('serviceName', None)
        db_service_port = default_backend.get('servicePort', None)
        if db_service_name is not None and db_service_port is not None:
            db_mapping_identifier = f"{ingress_name}-default-backend"

            default_backend_mapping: AnyDict = {
                'apiVersion': 'getambassador.io/v2',
                'kind': 'Mapping',
                'metadata': {
                    'name': db_mapping_identifier,
                    'namespace': ingress_namespace
                },
                'spec': {
                    'ambassador_id': ambassador_id,
                    'prefix': '/',
                    'service': f'{db_service_name}.{ingress_namespace}:{db_service_port}'
                }
            }

            if metadata_labels:
                default_backend_mapping['metadata']['labels'] = metadata_labels

            self.logger.debug(f"Generated mapping from Ingress {ingress_name}: {default_backend_mapping}")
            self.handle_k8s(default_backend_mapping)

        # parse ingress.spec.rules
        ingress_rules = ingress_spec.get('rules', [])
        for rule_count, rule in enumerate(ingress_rules):
            rule_http = rule.get('http', {})

            rule_host = rule.get('host', None)

            http_paths = rule_http.get('paths', [])
            for path_count, path in enumerate(http_paths):
                path_backend = path.get('backend', {})
                path_type = path.get('pathType', 'ImplementationSpecific')

                service_name = path_backend.get('serviceName', None)
                service_port = path_backend.get('servicePort', None)
                path_location = path.get('path', '/')

                if not service_name or not service_port or not path_location:
                    continue

                unique_suffix = f"{rule_count}-{path_count}"
                mapping_identifier = f"{ingress_name}-{unique_suffix}"

                # For cases where `pathType: Exact`,
                # otherwise `Prefix` and `ImplementationSpecific` are handled as regular Mapping prefixes
                is_exact_prefix = True if path_type == 'Exact' else False

                path_mapping: Dict[str, Any] = {
                    'apiVersion': 'getambassador.io/v2',
                    'kind': 'Mapping',
                    'metadata': {
                        'name': mapping_identifier,
                        'namespace': ingress_namespace
                    },
                    'spec': {
                        'ambassador_id': ambassador_id,
                        'prefix': path_location,
                        'prefix_exact': is_exact_prefix,
                        'precedence': 1 if is_exact_prefix else 0,  # Make sure exact paths are evaluated before prefix
                        'service': f'{service_name}.{ingress_namespace}:{service_port}'
                    }
                }

                if metadata_labels:
                    path_mapping['metadata']['labels'] = metadata_labels

                if rule_host is not None:
                    if rule_host.startswith('*.'):
                        # Ingress allow specifying hosts with a single wildcard as the first label in the hostname.
                        # Transform the rule_host into a host_regex:
                        # *.star.com  becomes  ^[a-z0-9]([-a-z0-9]*[a-z0-9])?\.star\.com$
                        path_mapping['spec']['host'] = rule_host\
                            .replace('.', '\\.')\
                            .replace('*', '^[a-z0-9]([-a-z0-9]*[a-z0-9])?', 1) + '$'
                        path_mapping['spec']['host_regex'] = True
                    else:
                        path_mapping['spec']['host'] = rule_host

                self.logger.debug(f"Generated mapping from Ingress {ingress_name}: {path_mapping}")
                self.handle_k8s(path_mapping)

        # let's make arrangements to update Ingress' status now
        if not self.manager.ambassador_service:
            self.logger.error(f"Unable to update Ingress {ingress_name}'s status, could not find Ambassador service")
        else:
            ingress_status = self.manager.ambassador_service.status

            if ingress_status:
                kind = k8s_object.get('kind')
                assert(kind)

                ingress_status_update = (kind, ingress_namespace, ingress_status)
                self.logger.debug(f"Updating Ingress {ingress_name} status to {ingress_status_update}")
                self.aconf.k8s_status_updates[f'{ingress_name}.{ingress_namespace}'] = ingress_status_update

        if parsed_ambassador_annotations is not None:
            # Copy metadata_labels to parsed annotations, if need be.
            if metadata_labels:
                for p in parsed_ambassador_annotations:
                    if p.get('metadata_labels') is None:
                        p['metadata_labels'] = metadata_labels

            # Force validation for all of these objects.
            for p in parsed_ambassador_annotations:
                p['_force_validation'] = True

            return resource_identifier, parsed_ambassador_annotations

        return None

    # Handler for K8s Secret resources.
    def handle_k8s_secret(self, k8s_object: AnyDict) -> HandlerResult:
        # XXX Another one where we shouldn't be saving everything.

        secret_type = k8s_object.get('type', None)
        metadata = k8s_object.get('metadata', None)
        metadata_labels: Optional[Dict[str, str]] = metadata.get('labels')
        resource_name = metadata.get('name') if metadata else None
        resource_namespace = metadata.get('namespace', 'default') if metadata else None
        data = k8s_object.get('data', None)

        skip = False

        if (secret_type != 'kubernetes.io/tls') and (secret_type != 'Opaque') and (secret_type != 'istio.io/key-and-cert'):
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

        if not skip and (Config.single_namespace and (resource_namespace != Config.ambassador_namespace) and Config.certs_single_namespace):
            # This should never happen in actual usage, since we shouldn't be given things
            # in the wrong namespace. However, in development, this can happen a lot.
            self.logger.debug("ignoring K8s Secret in wrong namespace")
            skip = True

        if skip:
            return None

        # This resource identifier is useful for log output since filenames can be duplicated (multiple subdirectories)
        resource_identifier = f'{resource_name}.{resource_namespace}'

        found_any = False

        for key in [ 'tls.crt', 'tls.key', 'user.key', 'cert-chain.pem', 'key.pem', 'root-cert.pem' ]:
            if data.get(key, None):
                found_any = True
                break

        if not found_any:
            # Uh. WTFO?
            self.logger.debug(f'ignoring K8s Secret {resource_identifier} with no keys')
            return None

        # No need to muck about with resolution later, just immediately turn this
        # into an Ambassador Secret resource.
        secret_info = {
            'apiVersion': 'getambassador.io/v2',
            'ambassador_id': Config.ambassador_id,
            'kind': 'Secret',
            'name': resource_name,
            'namespace': resource_namespace,
            'secret_type': secret_type
        }

        if metadata_labels:
            secret_info['metadata_labels'] = metadata_labels

        for key, value in data.items():
            secret_info[key.replace('.', '_')] = value

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

        # We can turn this directly into an Ambassador Service resource, since Consul keeps
        # services and endpoints together (as it should!!).
        #
        # Note that we currently trust the association ID to contain the datacenter name.
        # That's a function of the watch_hook putting it there.

        normalized_endpoints: Dict[str, List[Dict[str, Any]]] = {}

        for ep in endpoints:
            ep_addr = ep.get('Address')
            ep_port = ep.get('Port')

            if not ep_addr or not ep_port:
                self.logger.debug(f"ignoring Consul service {name} endpoint {ep['ID']} missing address info")
                continue

            # Consul services don't have the weird indirections that Kube services do, so just
            # lump all the endpoints together under the same source port of '*'.
            svc_eps = normalized_endpoints.setdefault('*', [])
            svc_eps.append({
                'ip': ep_addr,
                'port': ep_port,
                'target_kind': 'Consul'
            })

        spec = {
            'ambassador_id': Config.ambassador_id,
            'datacenter': consul_object.get('Id') or 'dc1',
            'endpoints': normalized_endpoints,
        }

        self.manager.emit(NormalizedResource.from_data(
            kind='Service',
            name=name,
            spec=spec,
            rkey=f"consul-{name}-{spec['datacenter']}",
        ))

        return None

    def finalize(self) -> None:
        self.k8s_processor.finalize()
