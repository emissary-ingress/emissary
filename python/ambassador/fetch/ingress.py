from typing import ClassVar, FrozenSet, Optional

from ..config import Config
from .dependency import IngressClassesDependency, SecretDependency, ServiceDependency
from .k8sobject import KubernetesGVK, KubernetesObject, KubernetesObjectKey
from .k8sprocessor import ManagedKubernetesProcessor
from .resource import NormalizedResource, ResourceManager


class IngressClassProcessor(ManagedKubernetesProcessor):

    CONTROLLER: ClassVar[str] = "getambassador.io/ingress-controller"

    ingress_classes_dep: IngressClassesDependency

    def __init__(self, manager: ResourceManager) -> None:
        super().__init__(manager)

        self.ingress_classes_dep = self.deps.provide(IngressClassesDependency)

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        return frozenset(
            [
                KubernetesGVK("networking.k8s.io/v1beta1", "IngressClass"),
                KubernetesGVK("networking.k8s.io/v1", "IngressClass"),
            ]
        )

    def _process(self, obj: KubernetesObject) -> None:
        # We only want to deal with IngressClasses that belong to "spec.controller: getambassador.io/ingress-controller"
        if obj.spec.get("controller", "").lower() != self.CONTROLLER:
            self.logger.debug(
                f"ignoring IngressClass {obj.name} without controller - getambassador.io/ingress-controller"
            )
            return

        if obj.ambassador_id != Config.ambassador_id:
            self.logger.debug(
                f"IngressClass {obj.name} does not have Ambassador ID {Config.ambassador_id}, ignoring..."
            )
            return

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
        ingress_parameters = obj.spec.get("parameters", {})

        self.logger.debug(
            f"Handling IngressClass {obj.name} with parameters {ingress_parameters}..."
        )
        self.aconf.incr_count("k8s_ingress_class")

        # Don't emit this directly. We use it when we handle ingresses below. If
        # we want to use the parameters, we should add them to this dependency
        # type.
        self.ingress_classes_dep.ingress_classes.add(obj.name)


class IngressProcessor(ManagedKubernetesProcessor):

    service_dep: ServiceDependency
    ingress_classes_dep: IngressClassesDependency

    def __init__(self, manager: ResourceManager) -> None:
        super().__init__(manager)

        self.deps.want(SecretDependency)
        self.service_dep = self.deps.want(ServiceDependency)
        self.ingress_classes_dep = self.deps.want(IngressClassesDependency)

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        return frozenset(
            [
                KubernetesGVK("extensions/v1beta1", "Ingress"),
                KubernetesGVK("networking.k8s.io/v1beta1", "Ingress"),
                KubernetesGVK("networking.k8s.io/v1", "Ingress"),
            ]
        )

    def _update_status(self, obj: KubernetesObject) -> None:
        service_status = None

        if not self.service_dep.ambassador_service or not self.service_dep.ambassador_service.name:
            self.logger.error(
                f"Unable to set Ingress {obj.name}'s load balancer, could not find Ambassador service"
            )
        else:
            service_status = self.service_dep.ambassador_service.status

        if obj.status != service_status:
            if service_status:
                status_update = (obj.gvk.kind, obj.namespace, service_status)
                self.logger.debug(f"Updating Ingress {obj.name} status to {status_update}")
                self.aconf.k8s_status_updates[f"{obj.name}.{obj.namespace}"] = status_update
        else:
            self.logger.debug(
                f"Not reconciling Ingress {obj.name}: observed and current statuses are in sync"
            )

    def _resolve_service_port_number(self, namespace, service_name, service_port):
        self.logger.debug(f"Resolving named port '{service_port}' in service '{service_name}'")

        key = KubernetesObjectKey(KubernetesGVK("v1", "Service"), namespace, service_name)
        k8s_svc: Optional[KubernetesObject]
        k8s_svc = self.service_dep.discovered_services.get(key, None)
        if not k8s_svc:
            self.logger.debug(f"Could not find service '{service_name}'")
            return service_port

        for port in k8s_svc.spec.get("ports", []):
            if service_port == port.get("name", None):
                return port.get("port", service_port)

        self.logger.debug(f"Could not find port '{service_port}' in service '{service_name}'")
        return service_port

    def _process(self, obj: KubernetesObject) -> None:
        ingress_class_name = obj.spec.get("ingressClassName", "")

        has_ingress_class = ingress_class_name in self.ingress_classes_dep.ingress_classes
        has_ambassador_ingress_class_annotation = (
            obj.annotations.get("kubernetes.io/ingress.class", "").lower() == "ambassador"
        )

        # check the Ingress resource has either:
        #  - a `kubernetes.io/ingress.class: "ambassador"` annotation
        #  - a `spec.ingressClassName` that references an IngressClass with
        #      `spec.controller: getambassador.io/ingress-controller`
        #
        # also worth noting, the kube-apiserver might assign the `spec.ingressClassName` if unspecified
        # and only 1 IngressClass has the following annotation:
        #   annotations:
        #     ingressclass.kubernetes.io/is-default-class: "true"
        if not (has_ingress_class or has_ambassador_ingress_class_annotation):
            self.logger.debug(
                f'ignoring Ingress {obj.name} without annotation (kubernetes.io/ingress.class: "ambassador") or IngressClass controller (getambassador.io/ingress-controller)'
            )
            return

        # We don't want to deal with non-matching Ambassador IDs
        if obj.ambassador_id != Config.ambassador_id:
            self.logger.debug(
                f"Ingress {obj.name} does not have Ambassador ID {Config.ambassador_id}, ignoring..."
            )
            return

        self.logger.debug(f"Handling Ingress {obj.name}...")
        self.aconf.incr_count("k8s_ingress")

        # We'll generate an ingress_id to match up this Ingress with its Mappings, but
        # only if this Ingress defines a Host. If no Host is defined, ingress_id will stay
        # None.
        ingress_id: Optional[str] = None

        ingress_tls = obj.spec.get("tls", [])
        for tls_count, tls in enumerate(ingress_tls):
            # Use the name and namespace to make a unique ID for this Ingress. We'll use
            # this for matching up this Ingress with its Mappings.
            ingress_id = f"a10r-ingress-{obj.name}-{obj.namespace}"

            tls_secret = tls.get("secretName", None)
            if tls_secret is not None:

                for host_count, host in enumerate(tls.get("hosts", ["*"])):
                    tls_unique_identifier = f"{obj.name}-{tls_count}-{host_count}"

                    spec = {
                        "ambassador_id": [obj.ambassador_id],
                        "hostname": host,
                        "acmeProvider": {"authority": "none"},
                        "tlsSecret": {"name": tls_secret},
                        "selector": {"matchLabels": {"a10r-k8s-ingress": ingress_id}},
                        "requestPolicy": {"insecure": {"action": "Route"}},
                    }

                    ingress_host = NormalizedResource.from_data(
                        "Host",
                        tls_unique_identifier,
                        namespace=obj.namespace,
                        labels=obj.labels,
                        spec=spec,
                    )

                    self.logger.debug(f"Generated Host from ingress {obj.name}: {ingress_host}")
                    self.manager.emit(ingress_host)

        # parse ingress.spec.defaultBackend
        # using ingress.spec.backend as a fallback, for older versions of the Ingress resource.
        default_backend = obj.spec.get("defaultBackend", obj.spec.get("backend", {}))
        db_service_name = default_backend.get("serviceName", None)
        db_service_port = default_backend.get("servicePort", None)
        if db_service_name is not None and db_service_port is not None:
            db_mapping_identifier = f"{obj.name}-default-backend"

            mapping_labels = dict(obj.labels)

            if ingress_id:
                mapping_labels["a10r-k8s-ingress"] = ingress_id

            default_backend_mapping = NormalizedResource.from_data(
                "Mapping",
                db_mapping_identifier,
                namespace=obj.namespace,
                labels=mapping_labels,
                spec={
                    "ambassador_id": obj.ambassador_id,
                    "hostname": "*",
                    "prefix": "/",
                    "service": f"{db_service_name}.{obj.namespace}:{db_service_port}",
                },
            )

            self.logger.debug(
                f"Generated Mapping from Ingress {obj.name}: {default_backend_mapping}"
            )
            self.manager.emit(default_backend_mapping)

        # parse ingress.spec.rules
        ingress_rules = obj.spec.get("rules", [])
        for rule_count, rule in enumerate(ingress_rules):
            rule_http = rule.get("http", {})

            rule_host = rule.get("host", None)

            http_paths = rule_http.get("paths", [])
            for path_count, path in enumerate(http_paths):
                path_backend = path.get("backend", {})
                path_type = path.get("pathType", "ImplementationSpecific")

                service_name = path_backend.get("serviceName", None)
                service_port = path_backend.get("servicePort", None)
                path_location = path.get("path", "/")

                try:
                    service_port = int(service_port)
                except:
                    service_port = self._resolve_service_port_number(
                        obj.namespace, service_name, service_port
                    )

                if not service_name or not service_port or not path_location:
                    continue

                unique_suffix = f"{rule_count}-{path_count}"
                mapping_identifier = f"{obj.name}-{unique_suffix}"

                # For cases where `pathType: Exact`,
                # otherwise `Prefix` and `ImplementationSpecific` are handled as regular Mapping prefixes
                is_exact_prefix = True if path_type == "Exact" else False

                spec = {
                    "ambassador_id": obj.ambassador_id,
                    "prefix": path_location,
                    "prefix_exact": is_exact_prefix,
                    "precedence": 1
                    if is_exact_prefix
                    else 0,  # Make sure exact paths are evaluated before prefix
                    "service": f"{service_name}.{obj.namespace}:{service_port}",
                }

                if rule_host is not None:
                    if rule_host.startswith("*."):
                        # Ingress allow specifying hosts with a single wildcard as the first label in the hostname.
                        # Transform the rule_host into a host_regex:
                        # *.star.com  becomes  ^[a-z0-9]([-a-z0-9]*[a-z0-9])?\.star\.com$
                        spec["host"] = (
                            rule_host.replace(".", "\\.").replace(
                                "*", "^[a-z0-9]([-a-z0-9]*[a-z0-9])?", 1
                            )
                            + "$"
                        )
                        spec["host_regex"] = True
                    else:
                        # Use hostname since this can be a hostname of "*" too.
                        spec["hostname"] = rule_host
                else:
                    # If there's no rule_host, and we don't have an ingress_id, force a hostname
                    # of "*" so that the Mapping we generate doesn't get dropped.
                    if not ingress_id:
                        spec["hostname"] = "*"

                mapping_labels = dict(obj.labels)

                if ingress_id:
                    mapping_labels["a10r-k8s-ingress"] = ingress_id

                path_mapping = NormalizedResource.from_data(
                    "Mapping",
                    mapping_identifier,
                    namespace=obj.namespace,
                    labels=mapping_labels,
                    spec=spec,
                )

                self.logger.debug(f"Generated Mapping from Ingress {obj.name}: {path_mapping}")
                self.manager.emit(path_mapping)

        # let's make arrangements to update Ingress' status now
        self._update_status(obj)
