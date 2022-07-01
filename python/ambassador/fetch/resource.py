from __future__ import annotations
from typing import Any, ClassVar, Dict, List, Optional

import dataclasses
import logging
import os

from ..config import ACResource, Config
from ..utils import dump_yaml, parse_yaml, parse_bool, dump_json

from .dependency import DependencyManager
from .k8sobject import KubernetesObjectScope, KubernetesObject
from .location import LocationManager


@dataclasses.dataclass
class NormalizedResource:
    """
    Represents an Ambassador resource emitted after processing fetched data.
    """

    object: dict
    rkey: Optional[str] = None
    log_resources: ClassVar[bool] = parse_bool(os.environ.get("AMBASSADOR_LOG_RESOURCES"))

    @classmethod
    def from_data(
        cls,
        kind: str,
        name: str,
        namespace: Optional[str] = None,
        generation: Optional[int] = None,
        version: str = "v3alpha1",
        api_group="getambassador.io",
        labels: Optional[Dict[str, Any]] = None,
        spec: Dict[str, Any] = None,
        errors: Optional[str] = None,
        rkey: Optional[str] = None,
    ) -> NormalizedResource:
        if rkey is None:
            rkey = f"{name}.{namespace}"

        ir_obj = {}
        if spec:
            ir_obj.update(spec)

        ir_obj["apiVersion"] = f"{api_group}/{version}"
        ir_obj["kind"] = kind
        ir_obj["name"] = name

        if namespace is not None:
            ir_obj["namespace"] = namespace

        if generation is not None:
            ir_obj["generation"] = generation

        ir_obj["metadata_labels"] = labels or {}

        if errors:
            ir_obj["errors"] = errors

        return cls(ir_obj, rkey)

    @classmethod
    def from_kubernetes_object(
        cls, obj: KubernetesObject, rkey: Optional[str] = None
    ) -> NormalizedResource:
        if obj.namespace is None:
            raise ValueError(
                f"Cannot construct resource from Kubernetes object {obj.key} without namespace"
            )

        labels = dict(obj.labels)

        if not rkey:  # rkey is only set for annotations
            # Default rkey for native Kubernetes resources
            rkey = f"{obj.name}.{obj.namespace}"
            # Some other code uses the 'ambassador_crd' label to know which resource to update
            # .status for with the apiserver.  Which is (IMO) a horrible hack, but I'm not up for
            # changing it at the moment.
            labels["ambassador_crd"] = rkey
        else:
            # Don't let it think that an annotation can have its status updated.
            labels.pop("ambassador_crd", None)

        # When creating an Ambassador object from a Kubernetes object, we have to make
        # sure that we pay attention to 'errors', which will be set IFF watt's validation
        # finds errors.

        return cls.from_data(
            obj.kind,
            obj.name,
            errors=obj.get("errors"),
            namespace=obj.namespace,
            generation=obj.generation,
            version=obj.gvk.version,
            api_group=obj.gvk.api_group,
            labels=labels,
            spec=obj.spec,
            rkey=rkey,
        )


class ResourceManager:
    """
    Holder for managed resources before they are processed and emitted as IR.
    """

    logger: logging.Logger
    aconf: Config
    deps: DependencyManager
    locations: LocationManager
    elements: List[ACResource]

    def __init__(self, logger: logging.Logger, aconf: Config, deps: DependencyManager):
        self.logger = logger
        self.aconf = aconf
        self.deps = deps
        self.locations = LocationManager()
        self.elements = []

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
                self.aconf.post_error(
                    "%s is not a dictionary? %s" % (self.location, dump_json(obj, pretty=True))
                )
            return True

        if not self.aconf.good_ambassador_id(obj):
            self.logger.debug("%s ignoring object with mismatched ambassador_id" % self.location)
            return True

        if "kind" not in obj:
            # Bug!!
            self.aconf.post_error(
                "%s is missing 'kind'?? %s" % (self.location, dump_json(obj, pretty=True))
            )
            return True

        # Is this a pragma object?
        if obj["kind"] == "Pragma":
            # Why did I think this was a good idea? [ :) ]
            new_source = obj.get("source", None)

            if new_source:
                # We don't save the old self.filename here, so this change will last until
                # the next input source (or the next Pragma).
                self.locations.current.filename = new_source

            # Don't count Pragma objects, since the user generally doesn't write them.
            return False

        if not rkey:
            rkey = self.locations.current.filename_default("unknown")

        if obj["kind"] != "Service":
            # Services are unique and don't get an object count appended to
            # them.
            rkey = "%s.%d" % (rkey, self.locations.current.ocount)

        serialization = dump_yaml(obj, default_flow_style=False)

        try:
            r = ACResource.from_dict(rkey, rkey, serialization, obj)
            self.elements.append(r)
        except Exception as e:
            self.aconf.post_error(e.args[0])

        if NormalizedResource.log_resources:
            self.logger.debug(
                "%s PROCESS %s save %s: %s", self.location, obj["kind"], rkey, serialization
            )

        return True

    def emit(self, resource: NormalizedResource):
        if self._emit(resource):
            self.locations.current.ocount += 1
