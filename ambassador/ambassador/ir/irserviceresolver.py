from ipaddress import ip_address
from typing import Any, ClassVar, Dict, List, Optional, Union, TYPE_CHECKING
from typing import cast as typecast

import json
import logging
import re
import urllib.parse

from ..config import Config
from ..utils import RichStatus

from .irresource import IRResource
from .irtlscontext import IRTLSContext

if TYPE_CHECKING:
    from .ir import IR
    from .ircluster import IRCluster

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

IREndpointSet = Any


class IRServiceResolver(IRResource):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str = "ir.resolver",
                 kind: str = "IRServiceResolver",
                 name: str = "ir.resolver",
                 location: str = "--internal--",
                 **kwargs) -> None:
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            location=location,
            **kwargs)

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        if self.kind == 'ConsulResolver':
            self.resolve_with = 'consul'

            if not self.datacenter:
                self.post_error("ConsulResolver is required to have a datacenter")
                return False
        elif self.kind == 'KubernetesServiceResolver':
            self.resolve_with = 'k8s'
        elif self.kind == 'KubernetesEndpointResolver':
            self.resolve_with = 'k8s'
        else:
            self.post_error(f"Resolver kind {self.kind} unknown")
            return False

        return True

    def resolve(self, ir: 'IR', cluster: 'IRCluster', svc_name: str, port: int, lb: str) -> IREndpointSet:
        keyfields = [ self.resolve_with ]

        # Is this already an IP address?
        is_ip_address = False

        try:
            x = ip_address(svc_name)
            is_ip_address = True
        except ValueError:
            pass

        if is_ip_address:
            # Already an IP address, great.
            self.logger.debug(f'Resolver {self.name}: {svc_name} is already an IP address')
            return [
                {
                    'ip': svc_name,
                    'port': port,
                    'target_kind': 'IPaddr'
                }
            ]


        if self.resolve_with == 'k8s':
            # K8s service names can be 'svc' or 'svc.namespace'. Which does this look like?

            svc = svc_name
            namespace = Config.ambassador_namespace

            if '.' in svc:
                # OK, cool. Peel off the service and the namespace.
                #
                # Note that some people may use service.namespace.cluster.svc.local or
                # some such crap. The [0:2] is to restrict this to just the first two
                # elements if there are more, but still work if there are not.

                ( svc, namespace ) = svc.split(".", 2)[0:2]

            keyfields.append(svc)
            keyfields.append(namespace)
        elif self.resolve_with == 'consul':
            keyfields.append(svc_name)
            keyfields.append(self.datacenter)
        else:
            # "Impossible."
            self.post_error(f'resolver {self.name} is neither Kubernetes nor Consul?')
            return None

        # OK. Do we have a Service by this key?
        key = "-".join(keyfields)

        service = ir.services.get(key)

        if not service:
            self.logger.debug(f'Resolver {self.name}: {key} matches no Service')
            return None

        self.logger.debug(f'Resolver {self.name}: {key} matches %s' % service.as_json())

        endpoints = service.get('endpoints')

        if not endpoints:
            self.logger.debug(f'Resolver {self.name}: {key} has no endpoints')
            return None

        # If this is a Kubernetes resolver, try to find a port match. If not, assume
        # that we don't need to.

        search_port = port if (self.resolve_with == 'k8s') else '*'

        # Do we have a match for the port they're asking for?

        targets = endpoints.get(search_port)

        if targets:
            # Yes!
            tstr = ", ".join([ f'{x["ip"]}:{x["port"]}' for x in targets ])

            self.logger.debug(f'Resolver {self.name}: {key}:{port} matches {tstr}')

            return targets
        else:
            hrtype = 'Kubernetes' if (self.resolve_with == 'k8s') else self.resolve_with

            # This is ugly. We're almost certainly being called from _within_ the initialization
            # of the cluster here -- so I guess we'll report the error against the service. Sigh.
            self.ir.post_error(f'Service {service.name}: {key}:{port} matches no endpoints from {hrtype}',
                               resource=service)

            return None


class IRServiceResolverFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        config_info = aconf.get_config('resolvers')

        if config_info:
            assert(len(config_info) > 0)    # really rank paranoia on my part...

            for config in config_info.values():
                cdict = config.as_dict()
                cdict['rkey'] = config.rkey
                cdict['location'] = config.location

                ir.add_resolver(IRServiceResolver(ir, aconf, **cdict))

        if not ir.get_resolver('kubernetes-service'):
            # Default the K8s service resolver.
            config = {
                'apiVersion': 'getambassador.io/v2',
                'kind': 'KubernetesServiceResolver',
                'name': 'kubernetes-service'
            }

            if Config.single_namespace:
                config['namespace'] = Config.ambassador_namespace

            ir.add_resolver(IRServiceResolver(ir, aconf, **config))

        # Ugh, the aliasing for the K8s and Consul endpoint resolvers is annoying.
        res_e = ir.get_resolver('endpoint')
        res_k_e = ir.get_resolver('kubernetes-endpoint')

        if not res_e and not res_k_e:
            # Neither exists. Create them from scratch.

            config = {
                'apiVersion': 'getambassador.io/v2',
                'kind': 'KubernetesEndpointResolver',
                'name': 'kubernetes-endpoint'
            }

            if Config.single_namespace:
                config['namespace'] = Config.ambassador_namespace

            ir.add_resolver(IRServiceResolver(ir, aconf, **config))

            config['name'] = 'endpoint'

            ir.add_resolver(IRServiceResolver(ir, aconf, **config))
        else:
            cls.check_aliases(ir, aconf, 'endpoint', res_e, 'kubernetes-endpoint', res_k_e)

        res_c = ir.get_resolver('consul')
        res_c_e = ir.get_resolver('consul-endpoint')

        if not res_c and not res_c_e:
            # Neither exists. Create them from scratch.

            config = {
                'apiVersion': 'getambassador.io/v2',
                'kind': 'ConsulResolver',
                'name': 'consul-endpoint',
                'datacenter': 'dc1'
            }

            ir.add_resolver(IRServiceResolver(ir, aconf, **config))

            config['name'] = 'consul'

            ir.add_resolver(IRServiceResolver(ir, aconf, **config))
        else:
            cls.check_aliases(ir, aconf, 'consul', res_c, 'consul-endpoint', res_c_e)

    @classmethod
    def check_aliases(cls, ir: 'IR', aconf: Config,
                      n1: str, r1: Optional[IRServiceResolver],
                      n2: str, r2: Optional[IRServiceResolver]) -> None:
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
            config['rkey'] = config.pop('_rkey', config.get('rkey', None))  # Kludge, I know...
            config.pop('_errored', None)
            config.pop('_active', None)
            config.pop('resolve_with', None)

            config['name'] = name

            ir.add_resolver(IRServiceResolver(ir, aconf, **config))
