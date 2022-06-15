from typing import Any, Dict, List, Optional, Tuple, Union

import json
import logging
import os
import yaml
import re

from ..config import ACResource, Config
from ..utils import parse_yaml, parse_json, dump_json, parse_bool

from .dependency import DependencyManager, IngressClassesDependency, SecretDependency, ServiceDependency
from .resource import NormalizedResource, ResourceManager
from .k8sobject import KubernetesGVK, KubernetesObject
from .k8sprocessor import (
    KubernetesProcessor,
    AggregateKubernetesProcessor,
    DeduplicatingKubernetesProcessor,
    CountingKubernetesProcessor,
)
from .ambassador import AmbassadorProcessor
from .secret import SecretProcessor
from .ingress import IngressClassProcessor, IngressProcessor
from .service import ServiceProcessor
from .knative import KnativeIngressProcessor

AnyDict = Dict[str, Any]

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
    invalid: List[Dict]

    def __init__(self, logger: logging.Logger, aconf: 'Config',
                 skip_init_dir: bool=False, watch_only=False) -> None:
        self.aconf = aconf
        self.logger = logger
        self.manager = ResourceManager(self.logger, self.aconf, DependencyManager([
            ServiceDependency(),
            SecretDependency(),
            IngressClassesDependency(),
        ]))

        self.k8s_processor = DeduplicatingKubernetesProcessor(AggregateKubernetesProcessor([
            CountingKubernetesProcessor(self.aconf, KubernetesGVK.for_knative_networking('Ingress'), 'knative_ingress'),
            AmbassadorProcessor(self.manager),
            SecretProcessor(self.manager),
            IngressClassProcessor(self.manager),
            IngressProcessor(self.manager),
            ServiceProcessor(self.manager, watch_only=watch_only),
            KnativeIngressProcessor(self.manager),
        ]))

        self.alerted_about_labels = False

        # Deltas, for managing the cache.
        self.deltas: List[Dict[str, Union[str, Dict[str, str]]]] = []

        # Paranoia: make sure self.invalid is empty.
        #
        # TODO(Flynn): The only reason this is here is because filesystem configuration
        # doesn't use parse_watt. This is broken for many reasons; filesystem configuration
        # should be handled by entrypoint, so that we can make the fetcher _much_ simpler.
        self.invalid = []

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

            automatic_manifests = []
            edge_stack_mappings_path = os.path.join(init_dir, "edge-stack-mappings.yaml")
            if parse_bool(os.environ.get('EDGE_STACK', 'false')) and not os.path.exists(edge_stack_mappings_path):
                # HACK
                # If we're running in Edge Stack via environment variable and the magic "edge-stack-mappings.yaml" file doesn't
                # exist in its well known location, then go ahead and add it. This should _not_ be necessary under
                # normal circumstances where Edge Stack is running in its container. We do this so that tests can
                # run outside of a container with this environment variable set.
                automatic_manifests.append('''
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: ambassador-edge-stack
  namespace: _automatic_
  labels:
    product: aes
    ambassador_diag_class: private
spec:
  hostname: "*"
  ambassador_id: [ "_automatic_" ]
  prefix: /.ambassador/
  rewrite: ""
  service: "127.0.0.1:8500"
  precedence: 1000000
''')

            if os.path.isdir(init_dir) or len(automatic_manifests) > 0:
                self.load_from_filesystem(init_dir, k8s=True, recurse=True, finalize=False, automatic_manifests=automatic_manifests)

    @property
    def elements(self) -> List[ACResource]:
        return self.manager.elements

    @property
    def location(self) -> str:
        return str(self.manager.locations.current)

    def load_from_filesystem(self, config_dir_path, recurse: bool=False,
                             k8s: bool=False, finalize: bool=True,
                             automatic_manifests: List[str]=[]):
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

        elif os.path.isfile(config_dir_path):
            # this allows a file to be passed into the ambassador cli
            # rather than just a directory
            inputs.append((config_dir_path, os.path.basename(config_dir_path)))
        elif len(automatic_manifests) == 0:
            # The config_dir_path wasn't a directory nor a file, and there are
            # no automatic manifests. Nothing to do.
            self.logger.debug("no init directory/file at path %s and no automatic manifests, doing nothing" % config_dir_path)

        for filepath, filename in inputs:
            self.logger.debug("reading %s (%s)" % (filename, filepath))

            try:
                serialization = open(filepath, "r").read()
                self.parse_yaml(serialization, k8s=k8s, filename=filename, finalize=False)
            except IOError as e:
                self.aconf.post_error("could not read YAML from %s: %s" % (filepath, e))

        for manifest in automatic_manifests:
            self.logger.debug("reading automatic manifest: %s" % manifest)
            try:
                self.parse_yaml(manifest, k8s=k8s, filename="_automatic_", finalize=False)
            except IOError as e:
                self.aconf.post_error("could not read automatic manifest: %s\n%s" % (manifest, e))

        if finalize:
            self.finalize()

    def parse_yaml(self, serialization: str, k8s=False, rkey: Optional[str] = None,
                   filename: Optional[str] = None, finalize: bool = True) -> None:
        # self.logger.info(f"RF YAML: {serialization}")

        # Expand environment variables allowing interpolation in manifests.
        serialization = os.path.expandvars(serialization)

        if not filename:
            filename = self.manager.locations.current.filename

        with self.manager.locations.push(filename=filename):
            try:
                # UGH. This parse_yaml is the one we imported from utils. XXX This needs to be fixed.
                for obj in parse_yaml(serialization):
                    if k8s:
                        with self.manager.locations.push_reset():
                            self.handle_k8s(obj)
                    else:
                        self.manager.emit(NormalizedResource(obj, rkey=rkey))
            except yaml.error.YAMLError as e:
                self.aconf.post_error("%s: could not parse YAML: %s" % (self.location, e))

        if finalize:
            self.finalize()

    def parse_watt(self, serialization: str, finalize: bool=True) -> None:
        basedir = os.environ.get('AMBASSADOR_CONFIG_BASE_DIR', '/ambassador')

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds')):
            self.aconf.post_error("Ambassador could not find core CRD definitions. Please visit https://www.getambassador.io/docs/edge-stack/latest/topics/install/upgrade-to-edge-stack/#5-update-and-restart for more information. You can continue using Ambassador via Kubernetes annotations, any configuration via CRDs will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds_2')):
            self.aconf.post_error("Ambassador could not find Resolver type CRD definitions. Please visit https://www.getambassador.io/docs/edge-stack/latest/topics/install/upgrade-to-edge-stack/#5-update-and-restart for more information. You can continue using Ambassador, but ConsulResolver, KubernetesEndpointResolver, and KubernetesServiceResolver resources will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds_3')):
            self.aconf.post_error("Ambassador could not find the Host CRD definition. Please visit https://www.getambassador.io/docs/edge-stack/latest/topics/install/upgrade-to-edge-stack/#5-update-and-restart for more information. You can continue using Ambassador, but Host resources will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds_4')):
            self.aconf.post_error("Ambassador could not find the LogService CRD definition. Please visit https://www.getambassador.io/docs/edge-stack/latest/topics/install/upgrade-to-edge-stack/#5-update-and-restart for more information. You can continue using Ambassador, but LogService resources will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_crds_5')):
            self.aconf.post_error("Ambassador could not find the DevPortal CRD definition. Please visit https://www.getambassador.io/docs/edge-stack/latest/topics/install/upgrade-to-edge-stack/#5-update-and-restart for more information. You can continue using Ambassador, but DevPortal resources will be ignored...")

        # We could be posting errors about the missing IngressClass resource, but given it's new in K8s 1.18
        # and we assume most users would be worried about it when running on older clusters, we'll rely on
        # Ambassador logs "Ambassador does not have permission to read IngressClass resources" for the moment.
        #if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_ingress_class')):
        #    self.aconf.post_error("Ambassador is not permitted to read IngressClass resources. Please visit https://www.getambassador.io/user-guide/ingress-controller/ for more information. You can continue using Ambassador, but IngressClass resources will be ignored...")

        if os.path.isfile(os.path.join(basedir, '.ambassador_ignore_ingress')):
            self.aconf.post_error("Ambassador is not permitted to read Ingress resources. Please visit https://www.getambassador.io/docs/edge-stack/latest/topics/running/ingress-controller/#ambassador-as-an-ingress-controller for more information. You can continue using Ambassador, but Ingress resources will be ignored...")

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

            self.invalid: List[Dict] = watt_dict.get('Invalid') or []

            for obj in self.invalid:
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

            # Remove annotations from the snapshot; we'll process them separately.
            annotations = watt_k8s.pop('annotations', {})

            # These objects have to be processed first, in order, as they depend
            # on each other.
            watt_k8s_keys = list(self.manager.deps.sorted_watt_keys())

            # Then we add everything else to be processed.
            watt_k8s_keys += watt_k8s.keys()

            # `dict.fromkeys(iterable)` is a convenient way to work around the
            # lack of an ordered set collection type in Python. As Python 3.7,
            # dicts are guaranteed to be insertion-ordered.
            for key in dict.fromkeys(watt_k8s_keys):
                for obj in watt_k8s.get(key) or []:
                    # self.logger.debug(f"Handling Kubernetes {key}...")
                    with self.manager.locations.push_reset():
                        self.handle_k8s(obj)
                        if 'errors' not in obj:
                            ann_parent_key = f"{obj['kind']}/{obj['metadata']['name']}.{obj['metadata'].get('namespace')}"
                            for ann_obj in (annotations.get(ann_parent_key) or []):
                                self.handle_annotation(ann_parent_key, ann_obj)

            watt_consul = watt_dict.get('Consul', {})
            consul_endpoints = watt_consul.get('Endpoints', {})

            for consul_rkey, consul_object in consul_endpoints.items():
                self.handle_consul_service(consul_rkey, consul_object)
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

    def sorted(self, key=lambda x: x.rkey):  # returns an iterator, probably
        return sorted(self.elements, key=key)

    def handle_k8s(self, raw_obj: dict) -> None:
        # self.logger.debug("handle_k8s obj %s" % dump_json(raw_obj, pretty=True))

        try:
            obj = KubernetesObject(raw_obj)
        except ValueError:
            # The object doesn't contain a kind, API version, or name, so we
            # can't process it.
            return

        if not self.k8s_processor.try_process(obj):
            self.logger.debug(f"{self.location}: skipping K8s {obj.gvk}")

    def handle_annotation(self, parent_key: str, raw_obj: dict) -> None:
        try:
            obj = KubernetesObject(raw_obj)
        except ValueError:
            # The object doesn't contain a kind, API version, or name, so we
            # can't process it.
            return

        with self.manager.locations.mark_annotated():
            rkey = parent_key.split('/', 1)[1]
            self.manager.emit(NormalizedResource.from_kubernetes_object(obj, rkey=rkey))

    # Handler for Consul services
    def handle_consul_service(self,
                              consul_rkey: str, consul_object: AnyDict) -> None:
        # resource_identifier = f'consul-{consul_rkey}'

        endpoints = consul_object.get('Endpoints', [])
        name = consul_object.get('Service', consul_rkey)

        if len(endpoints) < 1:
            # Bzzt.
            self.logger.debug(f"ignoring Consul service {name} with no Endpoints")
            return

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
            'Service',
            name,
            spec=spec,
            rkey=f"consul-{name}-{spec['datacenter']}",
        ))

    def finalize(self) -> None:
        self.k8s_processor.finalize()
