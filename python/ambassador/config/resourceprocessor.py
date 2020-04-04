from __future__ import annotations
from typing import Any, ClassVar, ContextManager, Dict, FrozenSet, Iterator, List, Mapping, Optional, Set

import collections
import collections.abc
import contextlib
import dataclasses
import datetime
import itertools
import json
import logging

import durationpy

from .config import Config
from .acresource import ACResource

from ..utils import dump_yaml


@dataclasses.dataclass(frozen=True)
class ResourceKind:
    """
    Represents a Kubernetes resource type (API version and kind).
    """

    api_version: str
    kind: str

    @property
    def api_group(self) -> Optional[str]:
        # These are backward-indexed to support apiVersion: v1, which has a
        # version but no group.
        try:
            return self.api_version.split('/', 1)[-2]
        except IndexError:
            return None

    @property
    def version(self) -> str:
        return self.api_version.split('/', 1)[-1]

    @property
    def domain(self) -> str:
        if self.api_group:
            return f'{self.kind.lower()}.{self.api_group}'
        else:
            return self.kind.lower()

    @classmethod
    def for_ambassador(cls, kind: str) -> ResourceKind:
        return cls('getambassador.io/v2', kind)

    @classmethod
    def for_knative_networking(cls, kind: str) -> ResourceKind:
        return cls('networking.internal.knative.dev/v1alpha1', kind)


@dataclasses.dataclass(frozen=True)
class ResourceIdent:
    """
    Represents a single Kubernetes resource by kind and name.
    """

    kind: ResourceKind
    namespace: Optional[str]
    name: str


@dataclasses.dataclass
class Location:
    """
    Represents a location for parsing.
    """

    filename: Optional[str] = None
    ocount: int = 1

    def mark_annotation(self) -> None:
        if self.filename is None:
            return
        elif self.filename.endswith(':annotation'):
            return

        self.filename += ':annotation'

    def __str__(self) -> str:
        return f"{self.filename or 'anonymous YAML'}.{self.ocount}"


class LocationManager:
    """
    Manages locations contextually.
    """

    previous: List[Location]
    current: Location

    def __init__(self) -> None:
        self.previous = []
        self.current = Location()

    def push(self, filename: Optional[str] = None, ocount: int = 1) -> ContextManager[Location]:
        self.previous.append(self.current)
        self.current = Location(filename, ocount)

        # This trick lets you use the return value of this method in a `with`
        # statement. At the conclusion of the statement block, the location will
        # automatically be popped from the stack.
        @contextlib.contextmanager
        def popper():
            yield
            self.pop()

        return popper()

    def push_reset(self) -> ContextManager[Location]:
        """
        Like push, but simply resets ocount keeping the current filename. Useful
        for changing resource types.
        """
        return self.push(filename=self.current.filename)

    def pop(self) -> Location:
        current = self.current
        self.current = self.previous.pop()
        return current


class ResourceDict(collections.abc.Mapping):
    """
    Represents a raw resource from Kubernetes.
    """

    default_namespace: Optional[str]

    def __init__(self, delegate: Dict[str, Any], default_namespace: Optional[str] = None) -> None:
        self.delegate = delegate
        self.default_namespace = default_namespace

        try:
            self.kind
            self.name
        except KeyError:
            raise ValueError('delegate is not a valid Kubernetes resource')

    def __getitem__(self, key: str) -> Any:
        return self.delegate[key]

    def __iter__(self) -> Iterator[str]:
        return iter(self.delegate)

    def __len__(self) -> int:
        return len(self.delegate)

    @property
    def kind(self) -> ResourceKind:
        return ResourceKind(self['apiVersion'], self['kind'])

    @property
    def metadata(self) -> Dict[str, Any]:
        return self['metadata']

    @property
    def namespace(self) -> Optional[str]:
        val = self.metadata.get('namespace', self.default_namespace)
        if val == '_automatic_':
            val = Config.ambassador_namespace

        return val

    @property
    def name(self) -> str:
        return self.metadata['name']

    @property
    def ident(self) -> ResourceIdent:
        return ResourceIdent(self.kind, self.namespace, self.name)

    @property
    def generation(self) -> int:
        return self.metadata.get('generation', 1)

    @property
    def annotations(self) -> Dict[str, str]:
        return self.metadata.get('annotations', {})

    @property
    def ambassador_id(self) -> str:
        return self.annotations.get('getambassador.io/ambassador-id', 'default')

    @property
    def labels(self) -> Dict[str, str]:
        return self.metadata.get('labels', {})

    @property
    def spec(self) -> Dict[str, Any]:
        return self.get('spec', {})

    @property
    def status(self) -> Dict[str, Any]:
        return self.get('status', {})


@dataclasses.dataclass
class ResourceEmission:
    """
    Represents an object emitted after processing a resource.
    """

    object: dict
    rkey: Optional[str] = None

    @classmethod
    def from_data(cls, kind: ResourceKind, name: str, namespace: str = 'default',
                  generation: int = 1, labels: Dict[str, Any] = None, spec: Dict[str, Any] = None) -> ResourceEmission:
        rkey = f'{name}.{namespace}'

        ir_obj = {}
        if spec:
            ir_obj.update(spec)

        ir_obj['apiVersion'] = kind.api_version
        ir_obj['name'] = name
        ir_obj['namespace'] = namespace
        ir_obj['kind'] = kind.kind
        ir_obj['generation'] = generation

        ir_obj['labels'] = {}
        if labels:
            ir_obj['labels'].update(labels)

        ir_obj['labels']['ambassador_crd'] = rkey

        return cls(ir_obj, rkey)

    @classmethod
    def from_resource(cls, obj: ResourceDict) -> ResourceEmission:
        return cls.from_data(
            obj.kind,
            obj.name,
            namespace=obj.namespace or 'default',
            generation=obj.generation,
            labels=obj.labels,
            spec=obj.spec,
        )


class ResourceManager:
    """
    Holder for managed resources before they are processed and emitted as IR.
    """

    logger: logging.Logger
    aconf: Config
    locations: LocationManager
    ambassador_service: Optional[ResourceDict]
    elements: List[ACResource]
    services: Dict[str, Dict[str, Any]]

    def __init__(self, logger: logging.Logger, aconf: Config):
        self.logger = logger
        self.aconf = aconf
        self.locations = LocationManager()
        self.ambassador_service = None
        self.elements = []
        self.services = {}

    @property
    def location(self) -> str:
        return str(self.locations.current)

    def _emit(self, emission: ResourceEmission) -> bool:
        obj = emission.object
        rkey = emission.rkey

        if not isinstance(obj, dict):
            # Bug!!
            if not obj:
                self.aconf.post_error("%s is empty" % self.location)
            else:
                self.aconf.post_error("%s is not a dictionary? %s" %
                                      (self.location, json.dumps(obj, indent=4, sort_keys=4)))
            return True

        if not self.aconf.good_ambassador_id(obj):
            self.logger.debug("%s ignoring object with mismatched ambassador_id" % self.location)
            return True

        if 'kind' not in obj:
            # Bug!!
            self.aconf.post_error("%s is missing 'kind'?? %s" %
                                  (self.location, json.dumps(obj, indent=4, sort_keys=True)))
            return True

        # self.logger.debug("%s PROCESS %s initial rkey %s" % (self.location, obj['kind'], rkey))

        # Is this a pragma object?
        if obj['kind'] == 'Pragma':
            # Why did I think this was a good idea? [ :) ]
            new_source = obj.get('source', None)

            if new_source:
                # We don't save the old self.filename here, so this change will last until
                # the next input source (or the next Pragma).
                self.locations.current.filename = new_source

            # Don't count Pragma objects, since the user generally doesn't write them.
            return False

        if not rkey:
            rkey = self.locations.current.filename

        rkey = "%s.%d" % (rkey, self.locations.current.ocount)

        # self.logger.debug("%s PROCESS %s updated rkey to %s" % (self.location, obj['kind'], rkey))

        # Force the namespace and metadata_labels, if need be.
        # TODO(impl): Remove this?
        # if namespace and not obj.get('namespace', None):
        #     obj['namespace'] = namespace

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

        return True

    def emit(self, emission: ResourceEmission):
        if self._emit(emission):
            self.locations.current.ocount += 1


class ResourceProcessor:
    """
    An abstract processor for Kubernetes resources that emit configuration
    resources.
    """

    def kinds(self) -> FrozenSet[ResourceKind]:
        # Override kinds to describe the types of resources this processor wants
        # to process.
        return frozenset()

    def _process(self, obj: ResourceDict) -> None:
        # Override _process to handle a single resource. Note that the entry
        # point for _process is try_process; _process should not be called
        # directly.
        pass

    def try_process(self, obj: ResourceDict) -> bool:
        if obj.kind not in self.kinds():
            return False

        self._process(obj)
        return True

    def finalize(self) -> None:
        # Override finalize to do processing at the end of the configuration
        # fetching.
        pass


class ManagedResourceProcessor (ResourceProcessor):
    """
    An abstract processor that provides access to a resource manager.
    """

    manager: ResourceManager

    def __init__(self, manager: ResourceManager):
        self.manager = manager

    @property
    def aconf(self) -> Config:
        return self.manager.aconf

    @property
    def logger(self) -> logging.Logger:
        return self.manager.logger


class AmbassadorResourceProcessor (ManagedResourceProcessor):
    """
    A resource processor that emits direct IR from an Ambassador CRD.
    """

    def kinds(self) -> FrozenSet[ResourceKind]:
        kinds = [
            'AuthService',
            'ConsulResolver',
            'Host',
            'KubernetesEndpointResolver',
            'KubernetesServiceResolver',
            'LogService',
            'Mapping',
            'Module',
            'RateLimitService',
            'TCPMapping',
            'TLSContext',
            'TracingService',
        ]

        return frozenset([ResourceKind.for_ambassador(kind) for kind in kinds])

    def _process(self, obj: ResourceDict) -> None:
        self.manager.emit(ResourceEmission.from_resource(obj))


class KnativeIngressResourceProcessor (ManagedResourceProcessor):
    """
    A resource processor that emits mappings from Knative Ingresses.
    """

    INGRESS_CLASS: ClassVar[str] = 'ambassador.ingress.networking.knative.dev'

    def kinds(self) -> FrozenSet[ResourceKind]:
        kinds = [
            'Ingress',
            'ClusterIngress',
        ]

        return frozenset([ResourceKind.for_knative_networking(kind) for kind in kinds])

    def _has_required_annotations(self, obj: ResourceDict) -> bool:
        annotations = obj.annotations

        # Let's not parse KnativeIngress if it's not meant for us. We only need
        # to ignore KnativeIngress iff networking.knative.dev/ingress.class is
        # present in annotation. If it's not there, then we accept all ingress
        # classes.
        ingress_class = annotations.get('networking.knative.dev/ingress.class', self.INGRESS_CLASS)
        if ingress_class.lower() != self.INGRESS_CLASS:
            self.logger.debug(f'Ignoring Knative {obj.kind.kind} {obj.name}; set networking.knative.dev/ingress.class '
                              f'annotation to {self.INGRESS_CLASS} for ambassador to parse it.')
            return False

        # We don't want to deal with non-matching Ambassador IDs
        if obj.ambassador_id != Config.ambassador_id:
            self.logger.info(f"Knative {obj.kind.kind} {obj.name} does not have Ambassador ID {Config.ambassador_id}, ignoring...")
            return False

        return True

    def _emit_mapping(self, obj: ResourceDict, rule_count: int, rule: Dict[str, Any]) -> None:
        hosts = rule.get('hosts', [])

        split_mapping_specs: List[Dict[str, Any]] = []

        paths = rule.get('http', {}).get('paths', [])
        for path in paths:
            global_headers = path.get('appendHeaders', {})

            splits = path.get('splits', [])
            for split in splits:
                service_name = split.get('serviceName')
                if not service_name:
                    continue

                service_namespace = split.get('serviceNamespace', obj.namespace)
                service_port = split.get('servicePort', 80)

                headers = split.get('appendHeaders', {})
                headers = {**global_headers, **headers}

                split_mapping_specs.append({
                    'service': f"{service_name}.{service_namespace}:{service_port}",
                    'add_request_headers': headers,
                    'weight': split.get('percent', 100),
                    'prefix': path.get('path', '/'),
                    'prefix_regex': True,
                    'timeout_ms': int(durationpy.from_str(path.get('timeout', '15s')).total_seconds() * 1000),
                })

        for split_count, (host, split_mapping_spec) in enumerate(itertools.product(hosts, split_mapping_specs)):
            mapping_identifier = f"{obj.name}-{rule_count}-{split_count}"

            spec = {
                'ambassador_id': obj.ambassador_id,
                'host': host,
            }
            spec.update(split_mapping_spec)

            mapping = ResourceEmission.from_data(
                ResourceKind.for_ambassador('Mapping'),
                mapping_identifier,
                namespace=obj.namespace or 'default',
                generation=obj.generation,
                labels=obj.labels,
                spec=spec,
            )

            self.logger.debug(f"Generated mapping from Knative {obj.kind.kind}: {mapping}")
            self.manager.emit(mapping)

    def _make_status(self, generation: int = 1, lb_domain: Optional[str] = None) -> Dict[str, Any]:
        utcnow = datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
        status = {
            "observedGeneration": generation,
            "conditions": [
                {
                    "lastTransitionTime": utcnow,
                    "status": "True",
                    "type": "LoadBalancerReady"
                },
                {
                    "lastTransitionTime": utcnow,
                    "status": "True",
                    "type": "NetworkConfigured"
                },
                {
                    "lastTransitionTime": utcnow,
                    "status": "True",
                    "type": "Ready"
                }
            ]
        }

        if lb_domain:
            load_balancer = {
                "ingress": [
                    {
                        "domainInternal": lb_domain,
                    }
                ]
            }

            status['loadBalancer'] = load_balancer
            status['privateLoadBalancer'] = load_balancer

        return status

    def _update_status(self, obj: ResourceDict) -> None:
        current_generation = obj.spec.get('generation', 1)
        has_new_generation = current_generation > obj.status.get('observedGeneration', 0)

        # Knative expects the load balancer information on the ingress, which it
        # then propagates to an ExternalName service for intra-cluster use. We
        # pull that information here. Otherwise, it will continue to use the DNS
        # name configured by the Knative service and go through an
        # out-of-cluster ingress to access the service.
        current_lb_domain = None

        if not self.manager.ambassador_service or not self.manager.ambassador_service.name:
            self.logger.warning(f"Unable to set Knative {obj.kind.kind} {obj.name}'s load balancer, could not find Ambassador service")
        else:
            # TODO: It is technically possible to use a domain other than
            # cluster.local (common-ish on bare metal clusters). We can resolve
            # the relevant domain by doing a DNS lookup on
            # kubernetes.default.svc, but this problem appears elsewhere in the
            # code as well and probably should just be fixed all at once.
            current_lb_domain = f"{self.manager.ambassador_service.name}.{self.manager.ambassador_service.namespace or 'default'}.svc.cluster.local"

        observed_ingress: Dict[str, Any] = next(iter(obj.status.get('privateLoadBalancer', {}).get('ingress', [])), {})
        observed_lb_domain = observed_ingress.get('domainInternal')

        has_new_lb_domain = current_lb_domain != observed_lb_domain

        if has_new_generation or has_new_lb_domain:
            status = self._make_status(generation=current_generation, lb_domain=current_lb_domain)
            status_update = (obj.kind.domain, obj.namespace, status)

            self.logger.info(f"Updating Knative {obj.kind.kind} {obj.name} status to {status_update}")
            self.aconf.k8s_status_updates[f"{obj.name}.{obj.namespace}"] = status_update
        else:
            self.logger.debug(f"Not reconciling Knative {obj.kind.kind} {obj.name}: observed and current generations are in sync")

    def _process(self, obj: ResourceDict) -> None:
        if not self._has_required_annotations(obj):
            return

        rules = obj.spec.get('rules', [])
        for rule_count, rule in enumerate(rules):
            self._emit_mapping(obj, rule_count, rule)

        self._update_status(obj)


class AggregateResourceProcessor (ResourceProcessor):
    """
    This resource processor aggregates many resource processors into a single
    convenient processor.
    """

    delegates: List[ResourceProcessor]
    mapping: Mapping[ResourceKind, List[ResourceProcessor]]

    def __init__(self, delegates: List[ResourceProcessor]) -> None:
        self.delegates = delegates
        self.mapping = collections.defaultdict(list)

        for proc in self.delegates:
            for kind in proc.kinds():
                self.mapping[kind].append(proc)

    def kinds(self) -> FrozenSet[ResourceKind]:
        return frozenset(iter(self.mapping))

    def _process(self, obj: ResourceDict) -> None:
        procs = self.mapping.get(obj.kind, [])
        for proc in procs:
            proc.try_process(obj)

    def finalize(self):
        for proc in self.delegates:
            proc.finalize()


class DeduplicatingResourceProcessor (ResourceProcessor):
    """
    This resource processor delegates work to another processor but prevents the
    same object from being processed multiple times.
    """

    delegate: ResourceProcessor
    cache: Set[ResourceIdent]

    def __init__(self, delegate: ResourceProcessor) -> None:
        self.delegate = delegate
        self.cache = set()

    def kinds(self) -> FrozenSet[ResourceKind]:
        return self.delegate.kinds()

    def _process(self, obj: ResourceDict) -> None:
        if obj.ident in self.cache:
            return

        self.cache.add(obj.ident)
        self.delegate.try_process(obj)

    def finalize(self):
        self.delegate.finalize()


class CounterResourceProcessor (ResourceProcessor):
    """
    This resource processor increments a given configuration counter when it
    receives a resource.
    """

    aconf: Config
    kind: ResourceKind
    key: str

    def __init__(self, aconf: Config, kind: ResourceKind, key: str) -> None:
        self.aconf = aconf
        self.kind = kind
        self.key = key

    def kinds(self) -> FrozenSet[ResourceKind]:
        return frozenset([self.kind])

    def _process(self, obj: ResourceDict) -> None:
        self.aconf.incr_count(self.key)
