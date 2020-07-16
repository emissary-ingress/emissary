from __future__ import annotations
from typing import Any, Dict, List, Optional

import dataclasses
import json
import logging

from ..config import ACResource, Config
from ..utils import dump_yaml

from .k8sobject import KubernetesObject
from .location import LocationManager


@dataclasses.dataclass
class NormalizedResource:
    """
    Represents an Ambassador resource emitted after processing fetched data.
    """

    object: dict
    rkey: Optional[str] = None

    @classmethod
    def from_data(cls, kind: str, name: str, namespace: str = 'default', generation: int = 1,
                  version: str = 'v2', labels: Dict[str, Any] = None, spec: Dict[str, Any] = None,
                  errors: Optional[str]=None) -> NormalizedResource:
        rkey = f'{name}.{namespace}'

        ir_obj = {}
        if spec:
            ir_obj.update(spec)

        ir_obj['apiVersion'] = f'getambassador.io/{version}'
        ir_obj['name'] = name
        ir_obj['namespace'] = namespace
        ir_obj['kind'] = kind
        ir_obj['generation'] = generation
        ir_obj['metadata_labels'] = labels or {}

        if errors:
            ir_obj['errors'] = errors

        return cls(ir_obj, rkey)

    @classmethod
    def from_kubernetes_object(cls, obj: KubernetesObject) -> NormalizedResource:
        if obj.gvk.api_group != 'getambassador.io':
            raise ValueError(f'Cannot construct resource from non-Ambassador Kubernetes object with API version {obj.gvk.api_version}')
        if obj.namespace is None:
            raise ValueError(f'Cannot construct resource from Kubernetes object {obj.key} without namespace')

        labels = dict(obj.labels)
        labels['ambassador_crd'] = f"{obj.name}.{obj.namespace}"

        # When creating an Ambassador object from a Kubernetes object, we have to make
        # sure that we pay attention to 'errors', which will be set IFF watt's validation
        # finds errors.

        return cls.from_data(
            obj.kind,
            obj.name,
            errors=obj.get('errors'),
            namespace=obj.namespace,
            generation=obj.generation,
            version=obj.gvk.version,
            labels=labels,
            spec=obj.spec,
        )


class ResourceManager:
    """
    Holder for managed resources before they are processed and emitted as IR.
    """

    logger: logging.Logger
    aconf: Config
    locations: LocationManager
    ambassador_service: Optional[KubernetesObject]
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

    def _emit(self, resource: NormalizedResource) -> bool:
        obj = resource.object
        rkey = resource.rkey

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

    def emit(self, resource: NormalizedResource):
        if self._emit(resource):
            self.locations.current.ocount += 1
