from typing import Any, Dict, List, Optional, Tuple, TYPE_CHECKING
# from typing import cast as typecast

import json
import logging
import os
import yaml
import re

from .config import Config
from .acresource import ACResource

from ..utils import parse_yaml, dump_yaml

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

CRDTypes = frozenset([
    'AuthService', 'ConsulResolver', 'Host',
    'KubernetesEndpointResolver', 'KubernetesServiceResolver',
    'LogService', 'Mapping', 'Module', 'RateLimitService',
    'TCPMapping', 'TLSContext', 'TracingService',
    'clusteringresses.networking.internal.knative.dev',
    'ingresses.networking.internal.knative.dev'
])
k8sLabelMatcher = re.compile(r'([\w\-_./]+)=\"(.+)\"')

class ResourceFetcher:
    def __init__(self, logger: logging.Logger, aconf: 'Config',
                 skip_init_dir: bool=False, watch_only=False) -> None:
        self.aconf = aconf
        self.logger = logger
        self.elements: List[ACResource] = []
        self.filename: Optional[str] = None
        self.ocount: int = 1
        self.saved: List[Tuple[Optional[str], int]] = []
        self.watch_only = watch_only

        self.k8s_endpoints: Dict[str, AnyDict] = {}
        self.k8s_services: Dict[str, AnyDict] = {}
        self.services: Dict[str, AnyDict] = {}
        self.ambassador_service_raw: AnyDict = {}

        self.alerted_about_labels = False

        # Ugh. Should we worry about multiple Helm charts for a single Ambassador?
        self.helm_chart: Optional[str] = None

        # For deduplicating objects coming in from watt
        self.k8s_parsed: Dict[str, bool] = {}

        # HACK
        # If AGENT_SERVICE is set, skip the init dir: we'll force some defaults later
        # instead.
        #
        # XXX This is rather a hack. We can do better.

        if os.environ.get("AGENT_SERVICE", "").lower() != "":
            logger.info("Intercept agent active: skipping init dir")
            skip_init_dir = True

        if not skip_init_dir:
            # Check /ambassador/init-config for initialization resources -- note NOT
            # $AMBASSADOR_CONFIG_BASE_DIR/init-config! This is compile-time stuff that
            # doesn't move around if you change the configuration base.
            init_dir = '/ambassador/init-config'

            if os.path.isdir(init_dir):
                self.load_from_filesystem(init_dir, k8s=True, recurse=True, finalize=False)

    @property
    def location(self):
        return "%s.%d" % (self.filename or "anonymous YAML", self.ocount)

    def push_location(self, filename: Optional[str], ocount: int) -> None:
        self.saved.append((self.filename, self.ocount))
        self.filename = filename
        self.ocount = ocount

    def pop_location(self) -> None:
        self.filename, self.ocount = self.saved.pop()

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
            self.logger.info("reading %s (%s)" % (filename, filepath))

            try:
                serialization = open(filepath, "r").read()
                self.parse_yaml(serialization, k8s=k8s, filename=filename, finalize=False)
            except IOError as e:
                self.aconf.post_error("could not read YAML from %s: %s" % (filepath, e))

        if finalize:
            self.finalize()

    def parse_yaml(self, serialization: str, k8s=False, rkey: Optional[str]=None,
                   filename: Optional[str]=None, finalize: bool=True, namespace: Optional[str]=None,
                   metadata_labels: Optional[Dict[str, str]]=None) -> None:
        # self.logger.info(f"RF YAML: {serialization}")

        # Expand environment variables allowing interpolation in manifests.
        serialization = os.path.expandvars(serialization)

        try:
            # UGH. This parse_yaml is the one we imported from utils. XXX This needs to be fixed.
            objects = parse_yaml(serialization)
            self.parse_object(objects=objects, k8s=k8s, rkey=rkey, filename=filename,
                              namespace=namespace)
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
            objects = json.loads(serialization)
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
            self.aconf.post_error("Ambassador could not find Resolver type CRD definitions. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds_3')):
            self.aconf.post_error("Ambassador could not find the Host CRD definition. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds_4')):
            self.aconf.post_error("Ambassador could not find the LogService CRD definition. Please visit https://www.getambassador.io/reference/core/crds/ for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored...")

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
            watt_dict = json.loads(serialization)

            watt_k8s = watt_dict.get('Kubernetes', {})

            # Handle normal Kube objects... the order is important here as ingresses depend on ingressclasses
            for key in [ 'service', 'endpoints', 'secret', 'ingressclasses', 'ingresses' ]:
                for obj in watt_k8s.get(key) or []:
                    # self.logger.debug(f"Handling Kubernetes {key}...")
                    self.handle_k8s(obj)

            # ...then handle Ambassador CRDs.
            for key in CRDTypes:
                for obj in watt_k8s.get(key) or []:
                    # self.logger.debug(f"Handling CRD {key}...")
                    self.handle_k8s_crd(obj)

            watt_consul = watt_dict.get('Consul', {})
            consul_endpoints = watt_consul.get('Endpoints', {})

            for consul_rkey, consul_object in consul_endpoints.items():
                result = self.handle_consul_service(consul_rkey, consul_object)

                if result:
                    rkey, parsed_objects = result

                    self.parse_object(parsed_objects, k8s=False,
                                      filename=self.filename, rkey=rkey)
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
                self.logger.warning(f"Dropping pod label {pod_label}")
            else:
                self.aconf.pod_labels[pod_label_kv[0][0]] = pod_label_kv[0][1]
        self.logger.info(f"Parsed pod labels: {self.aconf.pod_labels}")

    def check_k8s_dup(self, kind: str, namespace: str, name: str) -> bool:
        key = f"{kind}/{name}.{namespace}"

        if key in self.k8s_parsed:
            self.logger.info(f"dropping K8s dup {key}")
            return False

        # self.logger.info(f"remembering K8s {key}")
        self.k8s_parsed[key] = True
        return True

    def handle_k8s(self, obj: dict) -> None:
        # self.logger.debug("handle_k8s obj %s" % json.dumps(obj, indent=4, sort_keys=True))

        kind = obj.get('kind')

        if not kind:
            # self.logger.debug("%s: ignoring K8s object, no kind" % self.location)
            return

        metadata = obj.get('metadata') or {}
        name = metadata.get('name') or '(no name?)'
        namespace = metadata.get('namespace') or 'default'

        handler = None
        check_dup = True

        if kind in CRDTypes:
            handler = self.handle_k8s_crd
            # self.handle_k8s_crd will do its own dup checking.
            check_dup = False
        else:
            handler_name = f'handle_k8s_{kind.lower()}'
            # self.logger.debug(f"looking for handler {handler_name} for K8s {kind} {name}")
            handler = getattr(self, handler_name, None)

        if not handler:
            self.logger.debug(f"{self.location}: skipping K8s {kind}")
            return

        if check_dup:
            if not self.check_k8s_dup(kind, namespace, name):
                return

        result = handler(obj)

        if result:
            rkey, parsed_objects = result

            self.parse_object(parsed_objects, k8s=False, filename=self.filename, rkey=rkey)

    def handle_k8s_crd(self, obj: dict) -> None:
        # CRDs are _not_ allowed to have embedded objects in annotations, because ew.
        # self.logger.debug(f"Handling K8s CRD: {obj}")

        kind = obj.get('kind')

        if not kind:
            self.logger.debug("%s: ignoring K8s CRD, no kind" % self.location)
            return

        apiVersion = obj.get('apiVersion', '')
        metadata = obj.get('metadata') or {}
        name = metadata.get('name')
        namespace = metadata.get('namespace') or 'default'
        metadata_labels: Optional[Dict[str, str]] = metadata.get('labels')
        generation = metadata.get('generation', 1)
        annotations = metadata.get('annotations', {})

        if not self.check_k8s_dup(kind, namespace, name):
            return

        spec = obj.get('spec') or {}

        # Replace a sentinel value with the namespace of this ambassador pod.
        # This allows hard-coded initialization resources to have a useful namespace.
        if namespace == "_automatic_":
            namespace = Config.ambassador_namespace

        if not apiVersion:
            # I think this is impossible.
            self.logger.debug(f'{self.location}: ignoring K8s {kind} CRD, no apiVersion')
            return

        # We do not want to confuse Knative's Ingress with Kubernetes' Ingress
        if apiVersion.startswith('networking.internal.knative.dev') and kind.lower() == 'ingress':
            self.logger.debug(f"Renaming kind {kind} to KnativeIngress")
            kind = 'KnativeIngress'

            # Let's not parse KnativeIngress if it's not meant for us.
            # We only need to ignore KnativeIngress iff networking.knative.dev/ingress.class is present in annotation.
            # If it's not there, then we accept all ingress classes.
            if 'networking.knative.dev/ingress.class' in annotations:
                if annotations.get('networking.knative.dev/ingress.class').lower() != 'ambassador.ingress.networking.knative.dev':
                    self.logger.debug(f'Ignoring KnativeIngress {name}; set networking.knative.dev/ingress.class '
                                      f'annotation to ambassador.ingress.networking.knative.dev for ambassador to '
                                      f'parse it.')
                    return

        if not name:
            self.logger.debug(f'{self.location}: ignoring K8s {kind} CRD, no name')
            return

        if not apiVersion:
            self.logger.debug(f'{self.location}: ignoring K8s {kind} CRD {name}: no apiVersion')
            return

        # if not spec:
        #     self.logger.debug(f'{self.location}: ignoring K8s {kind} CRD {name}: no spec')
        #     return

        # We use this resource identifier as a key into self.k8s_services, and of course for logging .
        resource_identifier = f'{name}.{namespace}'

        # OK. Shallow copy 'spec'...
        amb_object = dict(spec)

        # ...and then stuff in a couple of other things.
        amb_object['apiVersion'] = apiVersion
        amb_object['name'] = name
        amb_object['namespace'] = namespace
        amb_object['kind'] = kind
        amb_object['generation'] = generation
        amb_object['metadata_labels'] = {}

        if metadata_labels:
            amb_object['metadata_labels'] = metadata_labels

        amb_object['metadata_labels']['ambassador_crd'] = resource_identifier

        # Done. Parse it.
        self.parse_object([ amb_object ], k8s=False, filename=self.filename, rkey=resource_identifier)

    def parse_object(self, objects, k8s=False, rkey: Optional[str]=None,
                     filename: Optional[str]=None, namespace: Optional[str]=None):
        self.push_location(filename, 1)

        # self.logger.debug("PARSE_OBJECT: incoming %d" % len(objects))

        for obj in objects:
            # self.logger.debug("PARSE_OBJECT: checking %s" % obj)

            if k8s:
                self.handle_k8s(obj)
            else:
                # if not obj:
                #     self.logger.debug("%s: empty object from %s" % (self.location, serialization))

                self.process_object(obj, rkey=rkey, namespace=namespace)
                self.ocount += 1

        self.pop_location()

    def process_object(self, obj: dict, rkey: Optional[str]=None, namespace: Optional[str]=None) -> None:
        if not isinstance(obj, dict):
            # Bug!!
            if not obj:
                self.aconf.post_error("%s is empty" % self.location)
            else:
                self.aconf.post_error("%s is not a dictionary? %s" %
                                      (self.location, json.dumps(obj, indent=4, sort_keys=4)))
            return

        if not self.aconf.good_ambassador_id(obj):
            self.logger.debug("%s ignoring object with mismatched ambassador_id" % self.location)
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

        # Force the namespace and metadata_labels, if need be.
        if namespace and not obj.get('namespace', None):
            obj['namespace'] = namespace

        # Brutal hackery.
        if obj['kind'] == 'Service':
            self.logger.debug("%s PROCESS saving service %s" % (self.location, obj['name']))
            self.services[obj['name']] = obj
        else:
            # Fine. Fine fine fine.
            serialization = dump_yaml(obj, default_flow_style=False)

            try:
                r = ACResource.from_dict(rkey, rkey, serialization, obj)
                self.elements.append(r)
            except Exception as e:
                self.aconf.post_error(e.args[0])

            self.logger.debug("%s PROCESS %s save %s: %s" % (self.location, obj['kind'], rkey, serialization))

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
            self.logger.info(f'ignoring IngressClass {ingress_class_name} without controller - getambassador.io/ingress-controller')
            skip = True

        if skip:
            return None

        annotations = metadata.get('annotations', {})
        ambassador_id = annotations.get('getambassador.io/ambassador-id', 'default')

        # We don't want to deal with non-matching Ambassador IDs
        if ambassador_id != Config.ambassador_id:
            self.logger.info(f'IngressClass {ingress_class_name} does not have Ambassador ID {Config.ambassador_id}, ignoring...')
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

        self.logger.info(f'Handling IngressClass {ingress_class_name} with parameters {ingress_parameters}...')

        # Don't return this as we won't handle IngressClass downstream.
        # Instead, save it in self.aconf.k8s_ingress_classes for reference in handle_k8s_ingress
        self.aconf.k8s_ingress_classes[resource_identifier] = ingress_parameters

        return None

    def handle_k8s_ingress(self, k8s_object: AnyDict) -> HandlerResult:
        metadata = k8s_object.get('metadata', None)
        metadata_labels: Optional[Dict[str, str]] = metadata.get('labels')
        ingress_name = metadata.get('name') if metadata else None
        ingress_namespace = metadata.get('namespace', 'default') if metadata else None

        resource_identifier = f'{ingress_name}.{ingress_namespace}'

        ingress_spec = k8s_object.get('spec', None)

        skip = False

        if not metadata:
            self.logger.debug("ignoring K8s Ingress with no metadata")
            skip = True

        if not ingress_name:
            self.logger.debug("ignoring K8s Ingress with no name")
            skip = True

        if not ingress_spec:
            self.logger.debug("ignoring K8s Ingress with no spec")
            skip = True

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
            self.logger.info(f'ignoring Ingress {ingress_name} without annotation (kubernetes.io/ingress.class: "ambassador") or IngressClass controller (getambassador.io/ingress-controller)')
            skip = True

        if skip:
            return None

        # Let's see if our Ingress resource has Ambassador annotations on it
        annotations = metadata.get('annotations', {})
        ambassador_annotations = annotations.get('getambassador.io/config', None)

        parsed_ambassador_annotations = None
        if ambassador_annotations is not None:
            if (self.filename is not None) and (not self.filename.endswith(":annotation")):
                self.filename += ":annotation"

            try:
                parsed_ambassador_annotations = parse_yaml(ambassador_annotations, namespace=ingress_namespace)
            except yaml.error.YAMLError as e:
                self.logger.debug("could not parse YAML: %s" % e)

        ambassador_id = annotations.get('getambassador.io/ambassador-id', 'default')

        # We don't want to deal with non-matching Ambassador IDs
        if ambassador_id != Config.ambassador_id:
            self.logger.info(f"Ingress {ingress_name} does not have Ambassador ID {Config.ambassador_id}, ignoring...")
            return None

        self.logger.info(f"Handling Ingress {ingress_name}...")
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

                    self.logger.info(f"Generated Host from ingress {ingress_name}: {ingress_host}")
                    self.handle_k8s_crd(ingress_host)

        # parse ingress.spec.defaultBackend
        # using ingress.spec.backend as a fallback, for older versions of the Ingress resource.
        default_backend = ingress_spec.get('defaultBackend', ingress_spec.get('backend', {}))
        db_service_name = default_backend.get('serviceName', None)
        db_service_port = default_backend.get('servicePort', None)
        if db_service_name is not None and db_service_port is not None:
            db_mapping_identifier = f"{ingress_name}-default-backend"

            default_backend_mapping = {
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

            self.logger.info(f"Generated mapping from Ingress {ingress_name}: {default_backend_mapping}")
            self.handle_k8s_crd(default_backend_mapping)

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

                self.logger.info(f"Generated mapping from Ingress {ingress_name}: {path_mapping}")
                self.handle_k8s_crd(path_mapping)

        # let's make arrangements to update Ingress' status now
        if not self.ambassador_service_raw:
            self.logger.error(f"Unable to update Ingress {ingress_name}'s status, could not find Ambassador service")
        else:
            ingress_status = self.ambassador_service_raw.get('status', {})
            ingress_status_update = (k8s_object.get('kind'), ingress_namespace, ingress_status)
            self.logger.info(f"Updating Ingress {ingress_name} status to {ingress_status_update}")
            self.aconf.k8s_status_updates[f'{ingress_name}.{ingress_namespace}'] = ingress_status_update

        if parsed_ambassador_annotations is not None:
            # Copy metadata_labels to parsed annotations, if need be.
            if metadata_labels:
                for p in parsed_ambassador_annotations:
                    if p.get('metadata_labels') is None:
                        p['metadata_labels'] = metadata_labels

            return resource_identifier, parsed_ambassador_annotations

        return None

    def is_ambassador_service(self, service_labels, service_selector):
        # self.logger.info(f"is_ambassador_service checking {service_labels} - {service_selector}")

        # Every Ambassador service must have the label 'app.kubernetes.io/component: ambassador-service'
        if service_labels is None:
            return False

        if service_labels.get('app.kubernetes.io/component', "").lower() != 'ambassador-service':
            return False

        # Now that we have the Ambassador label, let's verify that this Ambassador service routes to this very
        # Ambassador pod.
        # We do this by checking that the pod's labels match the selector in the service.
        for key, value in service_selector.items():
            pod_label_value = self.aconf.pod_labels.get(key)
            if pod_label_value == value:
                return True

    def handle_k8s_endpoints(self, k8s_object: AnyDict) -> HandlerResult:
        # Don't include Endpoints unless endpoint routing is enabled.
        if not Config.enable_endpoints:
            return None

        metadata = k8s_object.get('metadata', None)
        metadata_labels: Optional[Dict[str, str]] = metadata.get('labels')
        resource_name = metadata.get('name') if metadata else None
        resource_namespace = metadata.get('namespace', 'default') if metadata else None
        resource_subsets = k8s_object.get('subsets', None)

        skip = False

        if not metadata:
            self.logger.debug("ignoring K8s Endpoints with no metadata")
            skip = True

        if not resource_name:
            self.logger.debug("ignoring K8s Endpoints with no name")
            skip = True

        if not resource_subsets:
            self.logger.debug(f"ignoring K8s Endpoints {resource_name}.{resource_namespace} with no subsets")
            skip = True

        if skip:
            return None

        # We use this resource identifier as a key into self.k8s_services, and of course for logging .
        resource_identifier = '{name}.{namespace}'.format(namespace=resource_namespace, name=resource_name)

        # K8s Endpoints resources are _stupid_ in that they give you a vector of
        # IP addresses and a vector of ports, and you have to assume that every
        # IP address listens on every port, and that the semantics of each port
        # are identical. The first is usually a good assumption. The second is not:
        # people routinely list 80 and 443 for the same service, for example,
        # despite the fact that one is HTTP and the other is HTTPS.
        #
        # By the time the ResourceFetcher is done, we want to be working with
        # Ambassador Service resources, which have an array of address:port entries
        # for endpoints. So we're going to extract the address and port numbers
        # as arrays of tuples and stash them for later.
        #
        # In Kubernetes-speak, the Endpoints resource has some metadata and a set
        # of "subsets" (though I've personally never seen more than one subset in
        # one of these things).

        for subset in resource_subsets:
            # K8s subset addresses have some node info in with the IP address.
            # May as well save that too.

            addresses = []

            for address in subset.get('addresses', []):
                addr = {}

                ip = address.get('ip', None)
                if ip is not None:
                    addr['ip'] = ip

                node = address.get('nodeName', None)
                if node is not None:
                    addr['node'] = node

                target_ref = address.get('targetRef', None)
                if target_ref is not None:
                    target_kind = target_ref.get('kind', None)
                    if target_kind is not None:
                        addr['target_kind'] = target_kind

                    target_name = target_ref.get('name', None)
                    if target_name is not None:
                        addr['target_name'] = target_name

                    target_namespace = target_ref.get('namespace', None)
                    if target_namespace is not None:
                        addr['target_namespace'] = target_namespace

                if len(addr) > 0:
                    addresses.append(addr)

            # If we got no addresses, there's no point in messing with ports.
            if len(addresses) == 0:
                continue

            ports = subset.get('ports', [])

            # A service can reference a port either by name or by port number.
            port_dict = {}

            for port in ports:
                port_name = port.get('name', None)
                port_number = port.get('port', None)
                port_proto = port.get('protocol', 'TCP').upper()

                if port_proto != 'TCP':
                    continue

                if port_number is None:
                    # WTFO.
                    continue

                port_dict[str(port_number)] = port_number

                if port_name:
                    port_dict[port_name] = port_number

            if port_dict:
                # We're not going to actually return this: we'll just stash it for our
                # later resolution pass.

                self.k8s_endpoints[resource_identifier] = {
                    'name': resource_name,
                    'namespace': resource_namespace,
                    'addresses': addresses,
                    'ports': port_dict
                }

                if metadata_labels:
                    self.k8s_endpoints[resource_identifier]['metadata_labels'] = metadata_labels

            else:
                self.logger.debug(f"ignoring K8s Endpoints {resource_identifier} with no routable ports")

        return None

    def handle_k8s_service(self, k8s_object: AnyDict) -> HandlerResult:
        # The annoying bit about K8s Service resources is that not only do we have to look
        # inside them for Ambassador resources, but we also have to save their info for
        # later endpoint resolution too.
        #
        # Again, we're trusting that the input isn't overly bloated on that latter bit.

        metadata = k8s_object.get('metadata', None)
        if not metadata:
            self.logger.debug("ignoring K8s Service with no metadata")
            return None

        metadata_labels: Optional[Dict[str, str]] = metadata.get('labels')
        resource_name = metadata.get('name') if metadata else None
        resource_namespace = metadata.get('namespace', 'default') if metadata else None

        annotations = metadata.get('annotations', None) if metadata else None
        if annotations:
            annotations = annotations.get('getambassador.io/config', None)

        labels = metadata.get('labels')

        if labels:
            chart_version = labels.get('helm.sh/chart', None)

            if chart_version and not self.helm_chart:
                self.helm_chart = chart_version

        skip = False

        if not metadata:
            self.logger.debug("ignoring K8s Service with no metadata")
            skip = True

        if not skip and not resource_name:
            self.logger.debug("ignoring K8s Service with no name")
            skip = True

        if not skip and (Config.single_namespace and (resource_namespace != Config.ambassador_namespace)):
            # This should never happen in actual usage, since we shouldn't be given things
            # in the wrong namespace. However, in development, this can happen a lot.
            self.logger.debug(f"ignoring K8s Service {resource_name}.{resource_namespace} in wrong namespace")
            skip = True

        if skip:
            return None

        # We use this resource identifier as a key into self.k8s_services, and of course for logging .
        resource_identifier = f'{resource_name}.{resource_namespace}'

        # Not skipping. First, if we have some actual ports, stash this in self.k8s_services
        # for later resolution.

        spec = k8s_object.get('spec', None)
        ports = spec.get('ports', None) if spec else None

        if spec and ports:
            self.k8s_services[resource_identifier] = {
                'name': resource_name,
                'namespace': resource_namespace,
                'ports': ports
            }

            if metadata_labels:
                self.k8s_services[resource_identifier]['metadata_labels'] = metadata_labels

            selector = spec.get('selector', {})

            if self.is_ambassador_service(labels, selector):
                self.logger.info(f"Found Ambassador service: {resource_name}")
                self.ambassador_service_raw = k8s_object

        else:
            self.logger.debug(f"not saving K8s Service {resource_name}.{resource_namespace} with no ports")

        result: List[Any] = []

        if annotations:
            if (self.filename is not None) and (not self.filename.endswith(":annotation")):
                self.filename += ":annotation"

            try:
                objects = parse_yaml(annotations)

                for obj in objects:
                    if not obj:
                        self.logger.warning(f"empty YAML document found in ambassador service: {resource_name}.{resource_namespace}")
                        continue
                    if obj.get('metadata_labels') is None and metadata_labels:
                        obj['metadata_labels'] = metadata_labels
                    if obj.get('namespace') is None:
                        obj['namespace'] = resource_namespace
                    result.append(obj)

            except yaml.error.YAMLError as e:
                self.logger.debug("could not parse YAML: %s" % e)

        return resource_identifier, result

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

        svc = {
            'apiVersion': 'getambassador.io/v2',
            'ambassador_id': Config.ambassador_id,
            'kind': 'Service',
            'name': name,
            'datacenter': consul_object.get('Id') or 'dc1',
            'endpoints': {}
        }

        for ep in endpoints:
            ep_addr = ep.get('Address')
            ep_port = ep.get('Port')

            if not ep_addr or not ep_port:
                self.logger.debug(f"ignoring Consul service {name} endpoint {ep['ID']} missing address info")
                continue

            # Consul services don't have the weird indirections that Kube services do, so just
            # lump all the endpoints together under the same source port of '*'.
            svc_eps = svc['endpoints'].setdefault('*', [])
            svc_eps.append({
                'ip': ep_addr,
                'port': ep_port,
                'target_kind': 'Consul'
            })

        # Once again: don't return this. Instead, save it in self.services.
        self.services[f"consul-{name}-{svc['datacenter']}"] = svc

        return None

    def finalize(self) -> None:
        # The point here is to sort out self.k8s_services and self.k8s_endpoints and
        # turn them into proper Ambassador Service resources. This is a bit annoying,
        # because of the annoyances of Kubernetes, but we'll give it a go.
        #
        # Here are the rules:
        #
        # 1. By the time we get here, we have a _complete_ set of Ambassador resources that
        #    have passed muster by virtue of having the correct namespace, the correct
        #    ambassador_id, etc. (They may have duplicate names at this point, admittedly.)
        #    Any service not mentioned by name is out. Since the Ambassador resources in
        #    self.elements are in fact AResources, we can farm this out to code for each
        #    resource.
        #
        # 2. The check is, by design, permissive. If in doubt, write the check to leave
        #    the resource in.
        #
        # 3. For any service that stays in, we vet its listed ports against self.k8s_endpoints.
        #    Anything with no matching ports is _not_ dropped; it is assumed to use service
        #    routing rather than endpoint routing.

        # od = {
        #     'elements': [ x.as_dict() for x in self.elements ],
        #     'k8s_endpoints': self.k8s_endpoints,
        #     'k8s_services': self.k8s_services,
        #     'services': self.services
        # }
        #
        # self.logger.debug("==== FINALIZE START\n%s" % json.dumps(od, sort_keys=True, indent=4))

        for key, k8s_svc in self.k8s_services.items():
            k8s_name = k8s_svc['name']
            k8s_namespace = k8s_svc['namespace']
            k8s_metadata_labels = k8s_svc.get('metadata_labels', None)

            target_ports = {}
            target_addrs = []
            svc_endpoints = {}

            if not self.watch_only:
                # If we're not in watch mode, try to find endpoints for this service.

                k8s_ep = self.k8s_endpoints.get(key, None)
                k8s_ep_ports = k8s_ep.get('ports', None) if k8s_ep else None

                # OK, Kube is weird. The way all this works goes like this:
                #
                # 1. When you create a Kube Service, Kube will allocate a clusterIP
                #    for it and update DNS to resolve the name of the service to
                #    that clusterIP.
                # 2. Kube will look over the pods matched by the Service's selectors
                #    and stick those pods' IP addresses into Endpoints for the Service.
                # 3. The Service will have ports listed. These service.port entries can
                #    contain:
                #      port -- a port number you can talk to at the clusterIP
                #      name -- a name for this port
                #      targetPort -- a port number you can talk to at the _endpoint_ IP
                #    We'll call the 'port' entry here the "service-port".
                # 4. If you talk to clusterIP:service-port, you will get magically
                #    proxied by the Kube CNI to a target port at one of the endpoint IPs.
                #
                # The $64K question is: how does Kube decide which target port to use?
                #
                # First, if there's only one endpoint port, that's the one that gets used.
                #
                # If there's more than one, if the Service's port entry has a targetPort
                # number, it uses that. Otherwise it tries to find an endpoint port with
                # the same name as the service port. Otherwise, I dunno, it punts and uses
                # the service-port.
                #
                # So that's how Ambassador is going to do it, for each Service port entry.
                #
                # If we have no endpoints at all, Ambassador will end up routing using
                # just the service name and port per the Mapping's service spec.

                if not k8s_ep or not k8s_ep_ports:
                    # No endpoints at all, so we're done with this service.
                    self.logger.debug(f'{key}: no endpoints at all')
                else:
                    idx = -1

                    for port in k8s_svc['ports']:
                        idx += 1

                        k8s_target: Optional[int] = None

                        src_port = port.get('port', None)

                        if not src_port:
                            # WTFO. This is impossible.
                            self.logger.error(f"Kubernetes service {key} has no port number at index {idx}?")
                            continue

                        if len(k8s_ep_ports) == 1:
                            # Just one endpoint port. Done.
                            k8s_target = list(k8s_ep_ports.values())[0]
                            target_ports[src_port] = k8s_target

                            self.logger.debug(f'{key} port {src_port}: single endpoint port {k8s_target}')
                            continue

                        # Hmmm, we need to try to actually map whatever ports are listed for
                        # this service. Oh well.

                        found_key = False
                        fallback: Optional[int] = None

                        for attr in [ 'targetPort', 'name', 'port' ]:
                            port_key = port.get(attr)   # This could be a name or a number, in general.

                            if port_key:
                                found_key = True

                                if not fallback and (port_key != 'name') and str(port_key).isdigit():
                                    # fallback can only be digits.
                                    fallback = port_key

                                # Do we have a destination port for this?
                                k8s_target = k8s_ep_ports.get(str(port_key), None)

                                if k8s_target:
                                    self.logger.debug(f'{key} port {src_port} #{idx}: {attr} {port_key} -> {k8s_target}')
                                    break
                                else:
                                    self.logger.debug(f'{key} port {src_port} #{idx}: {attr} {port_key} -> miss')

                        if not found_key:
                            # WTFO. This is impossible.
                            self.logger.error(f"Kubernetes service {key} port {src_port} has an empty port spec at index {idx}?")
                            continue

                        if not k8s_target:
                            # This is most likely because we don't have endpoint info at all, so we'll do service
                            # routing.
                            #
                            # It's actually impossible for fallback to be unset, but WTF.
                            k8s_target = fallback or src_port

                            self.logger.debug(f'{key} port {src_port} #{idx}: falling back to {k8s_target}')

                        target_ports[src_port] = k8s_target

                    if not target_ports:
                        # WTFO. This is impossible. I guess we'll fall back to service routing.
                        self.logger.error(f"Kubernetes service {key} has no routable ports at all?")

                    # OK. Once _that's_ done we have to take the endpoint addresses into
                    # account, or just use the service name if we don't have that.

                    k8s_ep_addrs = k8s_ep.get('addresses', None)

                    if k8s_ep_addrs:
                        for addr in k8s_ep_addrs:
                            ip = addr.get('ip', None)

                            if ip:
                                target_addrs.append(ip)

            # OK! If we have no target addresses, just use service routing.
            if not target_addrs:
                if not self.watch_only:
                    self.logger.debug(f'{key} falling back to service routing')
                target_addrs = [ key ]

            for src_port, target_port in target_ports.items():
                svc_endpoints[src_port] = [ {
                    'ip': target_addr,
                    'port': target_port
                } for target_addr in target_addrs ]

            svc_resource = {
                'apiVersion': 'getambassador.io/v2',
                'ambassador_id': Config.ambassador_id,
                'kind': 'Service',
                'name': k8s_name,
                'namespace': k8s_namespace,
                'endpoints': svc_endpoints
            }

            if k8s_metadata_labels:
                svc_resource['metadata_labels'] = k8s_metadata_labels

            self.services[f'k8s-{k8s_name}-{k8s_namespace}'] = svc_resource

        # OK. After all that, go turn all of the things in self.services into Ambassador
        # Service resources.

        for key, svc in self.services.items():
            serialization = dump_yaml(svc, default_flow_style=False)

            r = ACResource.from_dict(key, key, serialization, svc)

            if self.helm_chart:
                r['helm_chart'] = self.helm_chart

            self.elements.append(r)

        # od = {
        #     'elements': [ x.as_dict() for x in self.elements ],
        #     'k8s_endpoints': self.k8s_endpoints,
        #     'k8s_services': self.k8s_services,
        #     'services': self.services
        # }

        # self.logger.debug("==== FINALIZE END\n%s" % json.dumps(od, sort_keys=True, indent=4))
