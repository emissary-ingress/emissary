from typing import Dict, List, Optional, Union, Tuple, TYPE_CHECKING

import json
import logging
import re
import urllib.parse

from ipaddress import ip_address

from multi import multi

from ..config import Config
from ..utils import RichStatus

from .irresource import IRResource
from .irtlscontext import IRTLSContext

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover
    from .ircluster import IRCluster  # pragma: no cover
    from .irbasemapping import IRBaseMapping  # pragma: no cover

#############################################################################
## irserviceresolver.py -- resolve endpoints for services
##
## IRServiceResolver does the work of looking into Service data structures.
## There are, naturally, some weirdnesses.
##
## Here's the way this goes:
##
## When you create an AConf, you must hand in Service objects and Resolver
## objects. (This will generally happen by virtue of the ResourceFetcher
## finding them someplace.) There can be multiple kinds of Resolver objects
## (e.g. ConsulResolver, KubernetesEndpointResolver, etc.).
##
## When you create an IR from that AConf, the various kinds of Resolvers
## all get turned into IRServiceResolvers, and the IR uses those to handle
## the mechanics of finding the upstream endpoints for a service.

SvcEndpoint = Dict[str, Union[int, str]]
SvcEndpointSet = List[SvcEndpoint]
ClustermapEntry = Dict[str, Union[int, str]]


class IRServiceResolver(IRResource):
    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str = "ir.resolver",
        kind: str = "IRServiceResolver",
        name: str = "ir.resolver",
        location: str = "--internal--",
        **kwargs,
    ) -> None:
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name, location=location, **kwargs
        )

    def setup(self, ir: "IR", aconf: Config) -> bool:
        if self.kind == "ConsulResolver":
            self.resolve_with = "consul"

            if not self.get("datacenter"):
                self.post_error("ConsulResolver is required to have a datacenter")
                return False
        elif self.kind == "KubernetesServiceResolver":
            self.resolve_with = "k8s"
        elif self.kind == "KubernetesEndpointResolver":
            self.resolve_with = "k8s"
        else:
            self.post_error(f"Resolver kind {self.kind} unknown")
            return False

        return True

    @multi
    def valid_mapping(self, ir: "IR", mapping: "IRBaseMapping") -> str:
        del ir
        del mapping

        return self.kind

    @valid_mapping.when("KubernetesServiceResolver")
    def _k8s_svc_valid_mapping(self, ir: "IR", mapping: "IRBaseMapping"):
        # You're not allowed to specific a load balancer with a KubernetesServiceResolver.
        if mapping.get("load_balancer"):
            mapping.post_error(
                "No load_balancer setting is allowed with the KubernetesServiceResolver"
            )
            return False

        return True

    @valid_mapping.when("KubernetesEndpointResolver")
    def _k8s_valid_mapping(self, ir: "IR", mapping: "IRBaseMapping"):
        # There's no real validation to do here beyond what the Mapping already does.
        return True

    @valid_mapping.when("ConsulResolver")
    def _consul_valid_mapping(self, ir: "IR", mapping: "IRBaseMapping"):
        # Mappings using the Consul resolver can't use service names with '.', or port
        # override. We currently do this the cheap & sleazy way.

        valid = True

        if mapping.service.find(".") >= 0:
            mapping.post_error("The Consul resolver does not allow dots in service names")
            valid = False

        if mapping.service.find(":") >= 0:
            # This is not an _error_ per se -- we'll accept the mapping and just ignore the port.
            ir.aconf.post_notice(
                "The Consul resolver does not allow overriding service port; ignoring requested port",
                resource=mapping,
            )

        return valid

    @multi
    def resolve(
        self, ir: "IR", cluster: "IRCluster", svc_name: str, svc_namespace: str, port: int
    ) -> str:
        del ir  # silence warnings
        del cluster
        del svc_name
        del svc_namespace
        del port

        return self.kind

    @resolve.when("KubernetesServiceResolver")
    def _k8s_svc_resolver(
        self, ir: "IR", cluster: "IRCluster", svc_name: str, svc_namespace: str, port: int
    ) -> Optional[SvcEndpointSet]:
        # The K8s service resolver always returns a single endpoint.
        return [{"ip": svc_name, "port": port, "target_kind": "DNSname"}]

    @resolve.when("KubernetesEndpointResolver")
    def _k8s_resolver(
        self, ir: "IR", cluster: "IRCluster", svc_name: str, svc_namespace: str, port: int
    ) -> Optional[SvcEndpointSet]:
        svc, namespace = self.parse_service(ir, svc_name, svc_namespace)
        # Find endpoints, and try for a port match!
        return self.get_endpoints(ir, f"k8s-{svc}-{namespace}", port)

    def parse_service(self, ir: "IR", svc_name: str, svc_namespace: str) -> Tuple[str, str]:
        # K8s service names can be 'svc' or 'svc.namespace'. Which does this look like?
        svc = svc_name
        namespace = Config.ambassador_namespace

        if "." in svc and not is_ip_address(svc):
            # OK, cool. Peel off the service and the namespace.
            #
            # Note that some people may use service.namespace.cluster.svc.local or
            # some such crap. The [0:2] is to restrict this to just the first two
            # elements if there are more, but still work if there are not.

            (svc, namespace) = svc.split(".", 2)[0:2]
        elif (
            not ir.ambassador_module.use_ambassador_namespace_for_service_resolution
            and svc_namespace
        ):
            namespace = svc_namespace
            ir.logger.debug(
                "KubernetesEndpointResolver use_ambassador_namespace_for_service_resolution %s, upstream key %s"
                % (
                    ir.ambassador_module.use_ambassador_namespace_for_service_resolution,
                    f"{svc}-{namespace}",
                )
            )

        return svc, namespace

    @resolve.when("ConsulResolver")
    def _consul_resolver(
        self, ir: "IR", cluster: "IRCluster", svc_name: str, svc_namespace: str, port: int
    ) -> Optional[SvcEndpointSet]:
        # For Consul, we look things up with the service name and the datacenter at present.
        # We ignore the port in the lookup (we should've already posted a warning about the port
        # being present, actually).

        return self.get_endpoints(ir, f"consul-{svc_name}-{self.datacenter}", None)

    def get_endpoints(self, ir: "IR", key: str, port: Optional[int]) -> Optional[SvcEndpointSet]:
        # OK. Do we have a Service by this key?
        service = ir.services.get(key)

        if not service:
            self.logger.debug(f"Resolver {self.name}: {key} matches no Service for endpoints")
            return None

        self.logger.debug(f"Resolver {self.name}: {key} matches %s" % service.as_json())

        endpoints = service.get("endpoints")

        if not endpoints:
            self.logger.debug(f"Resolver {self.name}: {key} has no endpoints")
            return None

        # Do we have a match for the port they're asking for (y'know, if they're asking for one)?

        targets = endpoints.get(port or "*")

        if targets:
            # Yes!
            tstr = ", ".join([f'{x["ip"]}:{x["port"]}' for x in targets])

            self.logger.debug(f"Resolver {self.name}: {key}:{port} matches {tstr}")

            return targets
        else:
            hrtype = "Kubernetes" if (self.resolve_with == "k8s") else self.resolve_with

            # This is ugly. We're almost certainly being called from _within_ the initialization
            # of the cluster here -- so I guess we'll report the error against the service. Sigh.
            self.ir.aconf.post_error(
                f"Service {service.name}: {key}:{port} matches no endpoints from {hrtype}",
                resource=service,
            )

            return None

    @multi
    def clustermap_entry(
        self, ir: "IR", cluster: "IRCluster", svc_name: str, svc_namespace: str, port: int
    ) -> str:
        del ir  # silence warnings
        del cluster
        del svc_name
        del svc_namespace
        del port

        return self.kind

    @clustermap_entry.when("KubernetesServiceResolver")
    def _k8s_svc_clustermap_entry(
        self, ir: "IR", cluster: "IRCluster", svc_name: str, svc_namespace: str, port: int
    ) -> ClustermapEntry:
        # The K8s service resolver always returns a single endpoint.
        svc, namespace = self.parse_service(ir, svc_name, svc_namespace)
        return {"port": port, "kind": self.kind, "service": svc, "namespace": namespace}

    @clustermap_entry.when("KubernetesEndpointResolver")
    def _k8s_clustermap_entry(
        self, ir: "IR", cluster: "IRCluster", svc_name: str, svc_namespace: str, port: int
    ) -> ClustermapEntry:
        # Fallback to the KubernetesServiceResolver for IP addresses or if the service doesn't exist.
        if is_ip_address(svc_name):
            return {
                "service": svc_name,
                "namespace": svc_namespace,
                "port": port,
                "kind": "KubernetesServiceResolver",
            }

        if port:
            portstr = "/%s" % port
        else:
            portstr = ""
        svc, namespace = self.parse_service(ir, svc_name, svc_namespace)
        # Find endpoints, and try for a port match!
        return {
            "service": svc,
            "namespace": namespace,
            "port": port,
            "kind": self.kind,
            "endpoint_path": "k8s/%s/%s%s" % (namespace, svc, portstr),
        }

    @clustermap_entry.when("ConsulResolver")
    def _consul_clustermap_entry(
        self, ir: "IR", cluster: "IRCluster", svc_name: str, svc_namespace: str, port: int
    ) -> ClustermapEntry:
        # Fallback to the KubernetesServiceResolver for ip addresses.
        if is_ip_address(svc_name):
            return {
                "service": svc_name,
                "namespace": svc_namespace,
                "port": port,
                "kind": "KubernetesServiceResolver",
            }

        # For Consul, we look things up with the service name and the datacenter at present.
        # We ignore the port in the lookup (we should've already posted a warning about the port
        # being present, actually).
        return {
            "service": svc_name,
            "datacenter": self.datacenter,
            "kind": self.kind,
            "endpoint_path": "consul/%s/%s" % (self.datacenter, svc_name),
        }


class IRServiceResolverFactory:
    @classmethod
    def load_all(cls, ir: "IR", aconf: Config) -> None:
        config_info = aconf.get_config("resolvers")

        if config_info:
            assert len(config_info) > 0  # really rank paranoia on my part...

            for config in config_info.values():
                cdict = config.as_dict()
                cdict["rkey"] = config.rkey
                cdict["location"] = config.location

                ir.add_resolver(IRServiceResolver(ir, aconf, **cdict))

        if not ir.get_resolver("kubernetes-service"):
            # Default the K8s service resolver.
            resolver_config = {
                "apiVersion": "getambassador.io/v3alpha1",
                "kind": "KubernetesServiceResolver",
                "name": "kubernetes-service",
            }

            if Config.single_namespace:
                resolver_config["namespace"] = Config.ambassador_namespace

            ir.add_resolver(IRServiceResolver(ir, aconf, **resolver_config))

        # Ugh, the aliasing for the K8s and Consul endpoint resolvers is annoying.
        res_e = ir.get_resolver("endpoint")
        res_k_e = ir.get_resolver("kubernetes-endpoint")

        if not res_e and not res_k_e:
            # Neither exists. Create them from scratch.

            resolver_config = {
                "apiVersion": "getambassador.io/v3alpha1",
                "kind": "KubernetesEndpointResolver",
                "name": "kubernetes-endpoint",
            }

            if Config.single_namespace:
                resolver_config["namespace"] = Config.ambassador_namespace

            ir.add_resolver(IRServiceResolver(ir, aconf, **resolver_config))

            resolver_config["name"] = "endpoint"

            ir.add_resolver(IRServiceResolver(ir, aconf, **resolver_config))
        else:
            cls.check_aliases(ir, aconf, "endpoint", res_e, "kubernetes-endpoint", res_k_e)

        res_c = ir.get_resolver("consul")
        res_c_e = ir.get_resolver("consul-endpoint")

        if not res_c and not res_c_e:
            # Neither exists. Create them from scratch.

            resolver_config = {
                "apiVersion": "getambassador.io/v3alpha1",
                "kind": "ConsulResolver",
                "name": "consul-endpoint",
                "datacenter": "dc1",
            }

            ir.add_resolver(IRServiceResolver(ir, aconf, **resolver_config))

            resolver_config["name"] = "consul"

            ir.add_resolver(IRServiceResolver(ir, aconf, **resolver_config))
        else:
            cls.check_aliases(ir, aconf, "consul", res_c, "consul-endpoint", res_c_e)

    @classmethod
    def check_aliases(
        cls,
        ir: "IR",
        aconf: Config,
        n1: str,
        r1: Optional[IRServiceResolver],
        n2: str,
        r2: Optional[IRServiceResolver],
    ) -> None:
        source = None
        name = None

        if not r1:
            # r2 must exist to be here.
            source = r2
            name = n1
        elif not r2:
            # r1 must exist to be here.
            source = r1
            name = n2

        if source:
            config = dict(**source.as_dict())

            # Fix up this dict. Sigh.
            config["rkey"] = config.pop("_rkey", config.get("rkey", None))  # Kludge, I know...
            config.pop("_errored", None)
            config.pop("_active", None)
            config.pop("resolve_with", None)

            config["name"] = name

            ir.add_resolver(IRServiceResolver(ir, aconf, **config))


def is_ip_address(addr: str) -> bool:
    try:
        x = ip_address(addr)
        return True
    except ValueError:
        return False
