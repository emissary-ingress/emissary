import dataclasses
from typing import Dict, FrozenSet, List, Optional

from ..config import Config
from ..utils import dump_json
from .dependency import ServiceDependency
from .k8sobject import KubernetesGVK, KubernetesObject, KubernetesObjectKey
from .k8sprocessor import AggregateKubernetesProcessor, ManagedKubernetesProcessor
from .resource import NormalizedResource, ResourceManager


@dataclasses.dataclass(frozen=True)
class EndpointAddress:
    """
    A single address of an endpoint used internally to map back to Service
    resources.
    """

    ip: str
    node: Optional[str]
    target: Optional[KubernetesObjectKey]


@dataclasses.dataclass(frozen=True)
class Endpoints:
    """
    A discovered set of endpoint addresses and ports used internally to map back
    to Service resources.
    """

    addresses: List[EndpointAddress]
    ports: Dict[str, int]
    labels: Dict[str, str]


class InternalServiceProcessor(ManagedKubernetesProcessor):
    """
    An internal Kubernetes object processor for services. Used by the
    ServiceProcessor aggregate class.
    """

    service_dep: ServiceDependency
    helm_chart: Optional[str]
    discovered_services: Dict[KubernetesObjectKey, KubernetesObject]

    def __init__(self, manager: ResourceManager) -> None:
        super().__init__(manager)

        self.service_dep = self.deps.provide(ServiceDependency)
        self.helm_chart = None
        self.discovered_services = {}

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        return frozenset([KubernetesGVK("v1", "Service")])

    def _is_ambassador_service(self, obj: KubernetesObject) -> bool:
        selector = obj.spec.get("selector", {})
        # self.logger.info(f"is_ambassador_service checking {obj.labels} - {selector}")

        # Every Ambassador service must have the label 'app.kubernetes.io/component: ambassador-service'
        if obj.labels.get("app.kubernetes.io/component", "").lower() != "ambassador-service":
            return False

        # This service must be in the same namespace as the Ambassador deployment.
        if obj.namespace != Config.ambassador_namespace:
            return False

        # Now that we have the Ambassador label, let's verify that this Ambassador service routes to this very
        # Ambassador pod.
        # We do this by checking that the pod's labels match the selector in the service.
        if len(selector) == 0:
            return False

        for key, value in selector.items():
            pod_label_value = self.aconf.pod_labels.get(key)
            if pod_label_value != value:
                return False

        return True

    def _process(self, obj: KubernetesObject) -> None:
        # The annoying bit about K8s Service resources is that not only do we have to look
        # inside them for Ambassador resources, but we also have to save their info for
        # later endpoint resolution too.
        #
        # Again, we're trusting that the input isn't overly bloated on that latter bit.

        chart_version = obj.labels.get("helm.sh/chart")
        if chart_version and not self.helm_chart:
            self.helm_chart = chart_version

        if not obj.spec.get("ports"):
            self.logger.debug(
                f"not saving Kubernetes Service {obj.name}.{obj.namespace} with no ports"
            )
        else:
            self.discovered_services[obj.key] = obj

            if self._is_ambassador_service(obj):
                self.logger.debug(f"Found Ambassador service: {obj.name}")
                self.service_dep.ambassador_service = obj


class InternalEndpointsProcessor(ManagedKubernetesProcessor):
    """
    This processor discovers endpoints, extracts information we care about, and
    stores the data for referencing by another processor.
    """

    discovered_endpoints: Dict[KubernetesObjectKey, Endpoints]

    def __init__(self, manager: ResourceManager) -> None:
        super().__init__(manager)

        self.discovered_endpoints = {}

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        return frozenset([KubernetesGVK("v1", "Endpoints")])

    def _process(self, obj: KubernetesObject) -> None:
        resource_subsets = obj.get("subsets")
        if not resource_subsets:
            self.logger.debug(
                f"ignoring Kubernetes Endpoints {obj.name}.{obj.namespace} with no subsets"
            )
            return

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

            addresses: List[EndpointAddress] = []

            for address in subset.get("addresses", []):
                ip = address.get("ip")
                if not ip:
                    continue

                target_ref: Optional[KubernetesObjectKey] = None
                try:
                    target_ref = KubernetesObjectKey.from_object_reference(
                        address.get("targetRef", {})
                    )
                except KeyError:
                    pass

                addresses.append(
                    EndpointAddress(ip, node=address.get("nodeName"), target=target_ref)
                )

            # If we got no addresses, there's no point in messing with ports.
            if len(addresses) == 0:
                continue

            ports = subset.get("ports", [])

            # A service can reference a port either by name or by port number.
            port_dict: Dict[str, int] = {}

            for port in ports:
                port_name = port.get("name", None)
                port_number = port.get("port", None)
                port_proto = port.get("protocol", "TCP").upper()

                if port_proto != "TCP":
                    continue

                if port_number is None:
                    # WTFO.
                    continue

                port_dict[str(port_number)] = port_number

                if port_name:
                    port_dict[port_name] = port_number

            if not port_dict:
                self.logger.debug(
                    f"ignoring K8s Endpoints {obj.name}.{obj.namespace} with no routable ports"
                )
                continue

            self.discovered_endpoints[obj.key] = Endpoints(addresses, port_dict, obj.labels)


class ServiceProcessor(ManagedKubernetesProcessor):
    """
    This processor handles Service and Endpoints objects and creates relevant
    Ambassador service resources.
    """

    services: InternalServiceProcessor
    endpoints: InternalEndpointsProcessor
    delegate: AggregateKubernetesProcessor
    watch_only: bool

    def __init__(self, manager: ResourceManager, watch_only: bool = False):
        super().__init__(manager)

        self.services = InternalServiceProcessor(manager)
        self.endpoints = InternalEndpointsProcessor(manager)
        self.delegate = AggregateKubernetesProcessor([self.services, self.endpoints])
        self.watch_only = watch_only

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        return self.delegate.kinds()

    def _process(self, obj: KubernetesObject) -> None:
        self.delegate.try_process(obj)

    def finalize(self) -> None:
        self.delegate.finalize()

        # The point here is to sort out self.services.discovered_services and
        # self.endpoints.discovered_endpoints and turn them into proper
        # Ambassador Service resources. This is a bit annoying, because of the
        # annoyances of Kubernetes, but we'll give it a go.
        #
        # Here are the rules:
        #
        # 1. By the time we get here, we have a _complete_ set of Ambassador
        #    resources that have passed muster by virtue of having the correct
        #    namespace, the correct ambassador_id, etc. (They may have duplicate
        #    names at this point, admittedly.) Any service not mentioned by name
        #    is out. Since the Ambassador resources in self.elements are in fact
        #    AResources, we can farm this out to code for each resource.
        #
        # 2. The check is, by design, permissive. If in doubt, write the check
        #    to leave the resource in.
        #
        # 3. For any service that stays in, we vet its listed ports against
        #    self.k8s_endpoints. Anything with no matching ports is _not_
        #    dropped; it is assumed to use service routing rather than endpoint
        #    routing.

        # od = {
        #     'elements': [ x.as_dict() for x in self.elements ],
        #     'k8s_endpoints': self.endpoints.discovered_endpoints,
        #     'k8s_services': self.services.discovered_services,
        # }
        #
        # self.logger.debug("==== FINALIZE START\n%s" % dump_json(od, pretty=True))

        for k8s_svc in self.services.discovered_services.values():
            key = f"{k8s_svc.name}.{k8s_svc.namespace}"

            target_ports = {}
            target_addrs = []
            svc_endpoints = {}

            if not self.watch_only:
                # If we're not in watch mode, try to find endpoints for this service.
                k8s_ep_key = KubernetesObjectKey(
                    KubernetesGVK("v1", "Endpoints"), k8s_svc.namespace, k8s_svc.name
                )
                k8s_ep = self.endpoints.discovered_endpoints.get(k8s_ep_key)

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

                if not k8s_ep:
                    # No endpoints at all, so we're done with this service.
                    self.logger.debug(f"{key}: no endpoints at all")
                else:
                    idx = -1

                    for port in k8s_svc.spec.get("ports", []):
                        idx += 1

                        k8s_target: Optional[int] = None

                        src_port = port.get("port", None)

                        if not src_port:
                            # WTFO. This is impossible.
                            self.logger.error(
                                f"Kubernetes service {key} has no port number at index {idx}?"
                            )
                            continue

                        if len(k8s_ep.ports) == 1:
                            # Just one endpoint port. Done.
                            k8s_target = next(iter(k8s_ep.ports.values()))
                            target_ports[src_port] = k8s_target

                            self.logger.debug(
                                f"{key} port {src_port}: single endpoint port {k8s_target}"
                            )
                            continue

                        # Hmmm, we need to try to actually map whatever ports are listed for
                        # this service. Oh well.

                        found_key = False
                        fallback: Optional[int] = None

                        for attr in ["targetPort", "name", "port"]:
                            port_key = port.get(
                                attr
                            )  # This could be a name or a number, in general.

                            if port_key:
                                found_key = True

                                if (
                                    not fallback
                                    and (port_key != "name")
                                    and str(port_key).isdigit()
                                ):
                                    # fallback can only be digits.
                                    fallback = port_key

                                # Do we have a destination port for this?
                                k8s_target = k8s_ep.ports.get(str(port_key), None)

                                if k8s_target:
                                    self.logger.debug(
                                        f"{key} port {src_port} #{idx}: {attr} {port_key} -> {k8s_target}"
                                    )
                                    break
                                else:
                                    self.logger.debug(
                                        f"{key} port {src_port} #{idx}: {attr} {port_key} -> miss"
                                    )

                        if not found_key:
                            # WTFO. This is impossible.
                            self.logger.error(
                                f"Kubernetes service {key} port {src_port} has an empty port spec at index {idx}?"
                            )
                            continue

                        if not k8s_target:
                            # This is most likely because we don't have endpoint info at all, so we'll do service
                            # routing.
                            #
                            # It's actually impossible for fallback to be unset, but WTF.
                            k8s_target = fallback or src_port

                            self.logger.debug(
                                f"{key} port {src_port} #{idx}: falling back to {k8s_target}"
                            )

                        target_ports[src_port] = k8s_target

                    if not target_ports:
                        # WTFO. This is impossible. I guess we'll fall back to service routing.
                        self.logger.error(f"Kubernetes service {key} has no routable ports at all?")

                    # OK. Once _that's_ done we have to take the endpoint addresses into
                    # account, or just use the service name if we don't have that.

                    for addr in k8s_ep.addresses:
                        target_addrs.append(addr.ip)

            # OK! If we have no target addresses, just use service routing.
            if not target_addrs:
                if not self.watch_only:
                    self.logger.debug(f"{key} falling back to service routing")
                target_addrs = [key]

            for src_port, target_port in target_ports.items():
                svc_endpoints[src_port] = [
                    {"ip": target_addr, "port": target_port} for target_addr in target_addrs
                ]

            spec = {
                "ambassador_id": Config.ambassador_id,
                "endpoints": svc_endpoints,
            }

            if self.services.helm_chart:
                spec["helm_chart"] = self.services.helm_chart

            self.manager.emit(
                NormalizedResource.from_data(
                    "Service",
                    k8s_svc.name,
                    namespace=k8s_svc.namespace,
                    labels=k8s_svc.labels,
                    spec=spec,
                    rkey=f"k8s-{k8s_svc.name}-{k8s_svc.namespace}",
                )
            )

        # self.logger.debug("==== FINALIZE END\n%s" % dump_json(od, pretty=True))
