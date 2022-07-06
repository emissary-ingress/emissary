from __future__ import annotations

import datetime
import itertools
from typing import Any, ClassVar, Dict, FrozenSet, List, Optional

import durationpy

from ..config import Config
from .dependency import ServiceDependency
from .k8sobject import KubernetesGVK, KubernetesObject
from .k8sprocessor import ManagedKubernetesProcessor
from .resource import NormalizedResource, ResourceManager


class KnativeIngressProcessor(ManagedKubernetesProcessor):
    """
    A Kubernetes object processor that emits mappings from Knative Ingresses.
    """

    INGRESS_CLASS: ClassVar[str] = "ambassador.ingress.networking.knative.dev"

    service_dep: ServiceDependency

    def __init__(self, manager: ResourceManager):
        super().__init__(manager)

        self.service_dep = self.deps.want(ServiceDependency)

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        return frozenset([KubernetesGVK.for_knative_networking("Ingress")])

    def _has_required_annotations(self, obj: KubernetesObject) -> bool:
        annotations = obj.annotations

        # Let's not parse KnativeIngress if it's not meant for us. We only need
        # to ignore KnativeIngress iff networking.knative.dev/ingress.class is
        # present in annotation. If it's not there, then we accept all ingress
        # classes.
        ingress_class = annotations.get("networking.knative.dev/ingress.class", self.INGRESS_CLASS)
        if ingress_class.lower() != self.INGRESS_CLASS:
            self.logger.debug(
                f"Ignoring Knative {obj.kind} {obj.name}; set networking.knative.dev/ingress.class "
                f"annotation to {self.INGRESS_CLASS} for ambassador to parse it."
            )
            return False

        # We don't want to deal with non-matching Ambassador IDs
        if obj.ambassador_id != Config.ambassador_id:
            self.logger.info(
                f"Knative {obj.kind} {obj.name} does not have Ambassador ID {Config.ambassador_id}, ignoring..."
            )
            return False

        return True

    def _emit_mapping(self, obj: KubernetesObject, rule_count: int, rule: Dict[str, Any]) -> None:
        hosts = rule.get("hosts", [])

        split_mapping_specs: List[Dict[str, Any]] = []

        paths = rule.get("http", {}).get("paths", [])
        for path in paths:
            global_headers = path.get("appendHeaders", {})

            splits = path.get("splits", [])
            for split in splits:
                service_name = split.get("serviceName")
                if not service_name:
                    continue

                service_namespace = split.get("serviceNamespace", obj.namespace)
                service_port = split.get("servicePort", 80)

                headers = split.get("appendHeaders", {})
                headers = {**global_headers, **headers}

                split_mapping_specs.append(
                    {
                        "service": f"{service_name}.{service_namespace}:{service_port}",
                        "add_request_headers": headers,
                        "weight": split.get("percent", 100),
                        "prefix": path.get("path", "/"),
                        "timeout_ms": int(
                            durationpy.from_str(path.get("timeout", "15s")).total_seconds() * 1000
                        ),
                    }
                )

        for split_count, (host, split_mapping_spec) in enumerate(
            itertools.product(hosts, split_mapping_specs)
        ):
            mapping_identifier = f"{obj.name}-{rule_count}-{split_count}"

            spec = {
                "ambassador_id": obj.ambassador_id,
                "host": host,
            }
            spec.update(split_mapping_spec)

            mapping = NormalizedResource.from_data(
                "Mapping",
                mapping_identifier,
                namespace=obj.namespace,
                generation=obj.generation,
                labels=obj.labels,
                spec=spec,
            )

            self.logger.debug(f"Generated Mapping from Knative {obj.kind}: {mapping}")
            self.manager.emit(mapping)

    def _make_status(self, generation: int = 1, lb_domain: Optional[str] = None) -> Dict[str, Any]:
        utcnow = datetime.datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
        status = {
            "observedGeneration": generation,
            "conditions": [
                {"lastTransitionTime": utcnow, "status": "True", "type": "LoadBalancerReady"},
                {"lastTransitionTime": utcnow, "status": "True", "type": "NetworkConfigured"},
                {"lastTransitionTime": utcnow, "status": "True", "type": "Ready"},
            ],
        }

        if lb_domain:
            load_balancer = {
                "ingress": [
                    {
                        "domainInternal": lb_domain,
                    }
                ]
            }

            status["loadBalancer"] = load_balancer
            status["privateLoadBalancer"] = load_balancer

        return status

    def _update_status(self, obj: KubernetesObject) -> None:
        has_new_generation = obj.generation > obj.status.get("observedGeneration", 0)

        # Knative expects the load balancer information on the ingress, which it
        # then propagates to an ExternalName service for intra-cluster use. We
        # pull that information here. Otherwise, it will continue to use the DNS
        # name configured by the Knative service and go through an
        # out-of-cluster ingress to access the service.
        current_lb_domain = None

        if not self.service_dep.ambassador_service or not self.service_dep.ambassador_service.name:
            self.logger.warning(
                f"Unable to set Knative {obj.kind} {obj.name}'s load balancer, could not find Ambassador service"
            )
        else:
            # TODO: It is technically possible to use a domain other than
            # cluster.local (common-ish on bare metal clusters). We can resolve
            # the relevant domain by doing a DNS lookup on
            # kubernetes.default.svc, but this problem appears elsewhere in the
            # code as well and probably should just be fixed all at once.
            current_lb_domain = f"{self.service_dep.ambassador_service.name}.{self.service_dep.ambassador_service.namespace}.svc.cluster.local"

        observed_ingress: Dict[str, Any] = next(
            iter(obj.status.get("privateLoadBalancer", {}).get("ingress", [])), {}
        )
        observed_lb_domain = observed_ingress.get("domainInternal")

        has_new_lb_domain = current_lb_domain != observed_lb_domain

        if has_new_generation or has_new_lb_domain:
            status = self._make_status(generation=obj.generation, lb_domain=current_lb_domain)

            if status:
                status_update = (obj.gvk.domain, obj.namespace, status)
                self.logger.info(
                    f"Updating Knative {obj.kind} {obj.name} status to {status_update}"
                )
                self.aconf.k8s_status_updates[f"{obj.name}.{obj.namespace}"] = status_update
        else:
            self.logger.debug(
                f"Not reconciling Knative {obj.kind} {obj.name}: observed and current generations are in sync"
            )

    def _process(self, obj: KubernetesObject) -> None:
        if not self._has_required_annotations(obj):
            return

        rules = obj.spec.get("rules", [])
        for rule_count, rule in enumerate(rules):
            self._emit_mapping(obj, rule_count, rule)

        self._update_status(obj)
