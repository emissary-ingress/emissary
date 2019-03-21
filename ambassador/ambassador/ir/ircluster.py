# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

from typing import Any, ClassVar, Dict, List, Optional, Union, TYPE_CHECKING
from typing import cast as typecast

import json
import re
import urllib.parse

from ..config import Config
from ..utils import RichStatus

from .irresource import IRResource
from .irtlscontext import IRTLSContext

if TYPE_CHECKING:
    from .ir import IR

#############################################################################
## ircluster.py -- the ircluster configuration object for Ambassador
##
## IRCluster represents an Envoy cluster: a collection of endpoints that
## provide a single service. IRClusters get used for quite a few different
## things in Ambassador -- they are basically the generic "upstream service"
## entity.


class IRCluster (IRResource):
    # TransparentRouteKeys: ClassVar[Dict[str, bool]] = {
    #     "auto_host_rewrite": True,
    #     "case_sensitive": True,
    #     "enable_ipv4": True,
    #     "enable_ipv6": True,
    #     "envoy_override": True,
    #     "host_rewrite": True,
    #     "path_redirect": True,
    #     "priority": True,
    #     "timeout_ms": True,
    # }

    def __init__(self, ir: 'IR', aconf: Config,
                 location: str,  # REQUIRED

                 service: str,   # REQUIRED

                 marker: Optional[str] = None,  # extra marker for this context name

                 ctx_name: Optional[Union[str, bool]]=None,
                 host_rewrite: Optional[str]=None,

                 dns_type: str="strict_dns",
                 enable_ipv4: Optional[bool]=None,
                 enable_ipv6: Optional[bool]=None,
                 lb_type: str="round_robin",
                 grpc: Optional[bool] = False,
                 allow_scheme: Optional[bool] = True,
                 load_balancer: Optional[dict] = None,

                 cb_name: Optional[str]=None,
                 od_name: Optional[str]=None,

                 rkey: str="-override-",
                 kind: str="IRCluster",
                 apiVersion: str="ambassador/v0",   # Not a typo! See below.
                 **kwargs) -> None:
        # Step one: look at the service and such and figure out a cluster name
        # and TLS origination info.

        # Here's how it goes:
        # - If allow_scheme is True and the service starts with https://, it is forced
        #   to originate TLS.
        # - Else, if allow_scheme is True and the service starts with http://, it is
        #   forced to _not_ originate TLS.
        # - Else, if we have a context (either a string that names a valid context,
        #   or the boolean value True), it will originate TLS.
        #
        # After figuring that out, if we have a context which is a string value,
        # we try to use that context name to look up certs to use. If we can't
        # find any, we won't send any originating cert.
        #
        # Finally, if no port is present in the service already, we force port 443
        # if we're originating TLS, 80 if not.

        originate_tls: bool = False
        name_fields: List[str] = [ 'cluster' ]
        ctx: Optional[IRTLSContext] = None
        errors: List[str] = []

        # Do we have a marker?
        if marker:
            name_fields.append(marker)

        self.logger = ir.logger

        # Toss in the original service before we mess with it, too.
        name_fields.append(service)

        # If we have a ctx_name, does it match a real context?
        if ctx_name:
            if ctx_name is True:
                ir.logger.debug("using null context")
                ctx = IRTLSContext.null_context(ir=ir)
            else:
                ir.logger.debug("seeking named context %s" % ctx_name)
                ctx = ir.get_tls_context(typecast(str, ctx_name))

            if not ctx:
                ir.logger.debug("no named context %s" % ctx_name)
                errors.append("Originate-TLS context %s is not defined" % ctx_name)
            else:
                ir.logger.debug("found context %s" % ctx)

        # TODO: lots of duplication of here, need to replace with broken down functions

        if allow_scheme and service.lower().startswith("https://"):
            service = service[len("https://"):]

            originate_tls = True
            name_fields.append('otls')

        elif allow_scheme and service.lower().startswith("http://"):
            service = service[ len("http://"): ]

            if ctx:
                errors.append("Originate-TLS context %s being used even though service %s lists HTTP" %
                              (ctx_name, service))
                originate_tls = True
                name_fields.append('otls')
            else:
                originate_tls = False

        elif ctx:
            # No scheme (or schemes are ignored), but we have a context.
            originate_tls = True
            name_fields.append('otls')
            name_fields.append(ctx.name)

        if '://' in service:
            # WTF is this?
            idx = service.index('://')
            scheme = service[0:idx]

            if allow_scheme:
                errors.append("service %s has unknown scheme %s, assuming %s" %
                              (service, scheme, "HTTPS" if originate_tls else "HTTP"))
            else:
                errors.append("ignoring scheme %s for service %s, since it is being used for a non-HTTP mapping" %
                              (scheme, service))

            service = service[idx + 3:]

        # XXX Should this be checking originate_tls? Why does it do that? 
        if originate_tls and host_rewrite:
            name_fields.append("hr-%s" % host_rewrite)

        name = "_".join(name_fields)
        name = re.sub(r'[^0-9A-Za-z_]', '_', name)

        # Parse the service as a URL. Note that we have to supply a scheme to urllib's
        # parser, because it's kind of stupid.

        ir.logger.debug("cluster %s service %s otls %s ctx %s" % (name, service, originate_tls, ctx))
        p = urllib.parse.urlparse('random://' + service)

        # Is there any junk after the host?

        if p.path or p.params or p.query or p.fragment:
            errors.append("service %s has extra URL components; ignoring everything but the host and port" % service)

        # p is read-only, so break stuff out.

        hostname = p.hostname
        port = p.port

        # If the port is unset, fix it up.
        if not port:
            port = 443 if originate_tls else 80

        if rkey == '-override-':
            rkey = name

        # Rebuild the URL with the 'tcp' scheme and our changed info.
        # (Yes, really, TCP. Envoy uses the TLS context to determine whether to originate
        # TLS. Kind of odd, but there we go.)
        url = "tcp://%s:%d" % (hostname, port)

        # The Ambassador module will always have a load_balancer.
        global_load_balancer = ir.ambassador_module.load_balancer

        if not load_balancer:
            load_balancer = global_load_balancer

        self.logger.info("Load balancer for {} is {}".format(url, load_balancer))

        endpoint = {}
        enable_endpoints = False

        if load_balancer is not None:
            if self.endpoints_required(load_balancer):
                if not Config.enable_endpoints:
                    errors.append(f"{name}: endpoint routing is not enabled, falling back to {global_load_balancer}")
                    load_balancer = global_load_balancer
                else:
                    self.logger.debug("fetching endpoint information for {}".format(hostname))
                    endpoint = self.get_endpoint(hostname, port,
                                                 ir.service_info.get(service, None),
                                                 ir.endpoints.get(hostname, None))
                    if len(endpoint) > 0:
                        # We want to enable endpoints and change the load balancer policy only if endpoint routing
                        # is configured correctly and we're getting endpoints for the given service
                        enable_endpoints = True
                        lb_type = load_balancer.get('policy')
                    else:
                        self.logger.debug("No endpoints found. Endpoint routing misconfigured, not enabling endpoint routing")

        # OK. Build our default args.
        #
        # XXX We should really save the hostname and the port, not the URL.

        if enable_ipv4 is None:
            enable_ipv4 = ir.ambassador_module.enable_ipv4
            ir.logger.debug("%s: copying enable_ipv4 %s from Ambassador Module" % (name, enable_ipv4))

        if enable_ipv6 is None:
            enable_ipv6 = ir.ambassador_module.enable_ipv6
            ir.logger.debug("%s: copying enable_ipv6 %s from Ambassador Module" % (name, enable_ipv6))

        new_args: Dict[str, Any] = {
            "type": dns_type,
            "lb_type": lb_type,
            "urls": [ url ],
            "endpoint": endpoint,
            "load_balancer": load_balancer,
            "service": service,
            'enable_ipv4': enable_ipv4,
            'enable_ipv6': enable_ipv6,
            'enable_endpoints': enable_endpoints
        }

        if grpc:
            new_args['grpc'] = True

        if host_rewrite:
            new_args['host_rewrite'] = host_rewrite

        if originate_tls:
            if ctx:
                new_args['tls_context'] = typecast(IRTLSContext, ctx)
            else:
                new_args['tls_context'] = IRTLSContext.null_context(ir=ir)

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location,
            kind=kind, name=name, apiVersion=apiVersion,
            **new_args
        )

        if ctx:
            ctx.referenced_by(self)

        if errors:
            for error in errors:
                ir.post_error(error, resource=self)

    def endpoints_required(self, load_balancer) -> bool:
        required = False
        lb_policy = load_balancer.get('policy')
        if lb_policy in ['round_robin', 'ring_hash']:
            self.logger.debug("Endpoints are required for load balancing policy {}".format(lb_policy))
            required = True
        return required

    def get_endpoint(self, hostname, port, service_info, endpoint):
        if endpoint is None:
            self.logger.debug("no relevant endpoint found for hostname {}".format(hostname))
            return {}

        if service_info is None:
            self.logger.debug("no service found for hostname {}".format(hostname))
            return {}

        ip = []
        for address in endpoint['addresses']:
            ip.append(address['ip'])

        ep_port = None

        # service_port_name  is only required if there are multiple ports in a given service, to match
        # the port in Endpoint resource
        service_port_name = ""
        service_ports = service_info.get('ports', [])
        num_service_ports = len(service_ports)
        if num_service_ports > 1:
            for service_port in service_ports:
                if port == service_port.get('port'):
                    service_port_name = service_port.get('name')
                    break
        elif num_service_ports == 0:
            self.logger.debug("no service port found for service: {}".format(service_info))
            return {}

        self.logger.debug("service port name is '{}'".format(service_port_name))

        if len(service_port_name) == 0:
            # this means there is only one port
            endpoint_ports = endpoint.get('ports', [])
            if len(endpoint_ports) != 1:
                self.logger.debug("no or more than one endpoint ports found {}, not enabling endpoint routing".format(endpoint_ports))
                return {}
            ep_port = endpoint_ports[0].get('port', None)
        else:
            # there are more than one service ports, so we need to match on name
            endpoint_ports = endpoint.get('ports', [])
            for ep in endpoint_ports:
                name = ep.get('name', "")
                if name == service_port_name:
                    ep_port = ep.get('port', None)
                    break

        if ep_port is None:
            self.logger.debug("could not discover any relevant endpoint port for hostname {}, not enabling endpoint routing".format(hostname))
            return {}

        if len(ip) == 0:
            self.logger.debug("no IP addresses found for endpoint for hostname {}, not enabling endpoint routing".format(hostname))
            return {}

        generated_endpoint = {
            'ip': ip,
            'port': ep_port
        }
        self.logger.debug("generated endpoint for hostname {}: {}".format(hostname, generated_endpoint))

        return generated_endpoint

    def add_url(self, url: str) -> List[str]:
        self.urls.append(url)

        return self.urls

    def merge(self, other: 'IRCluster') -> bool:
        # Is this mergeable?

        mismatches = []

        for key in [ 'type', 'lb_type', 'host_rewrite',
                     'tls_context', 'originate_tls', 'grpc' ]:
            if self.get(key, None) != other.get(key, None):
                mismatches.append(key)

        if mismatches:
            self.post_error(RichStatus.fromError("cannot merge cluster %s: mismatched attributes %s" %
                                                 (other.name, ", ".join(mismatches))))
            return False

        # All good.
        for url in other.urls:
            self.add_url(url)
            self.referenced_by(other)

        return True
