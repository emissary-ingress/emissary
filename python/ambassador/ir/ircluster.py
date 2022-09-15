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

import json
import re
import urllib.parse
from typing import TYPE_CHECKING, Any, ClassVar, Dict, List, Optional, Union
from typing import cast as typecast

from ..config import Config
from ..utils import RichStatus
from .irresource import IRResource
from .irtlscontext import IRTLSContext

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover
    from .ir.irserviceresolver import IRServiceResolver  # pragma: no cover

#############################################################################
## ircluster.py -- the ircluster configuration object for Ambassador
##
## IRCluster represents an Envoy cluster: a collection of endpoints that
## provide a single service. IRClusters get used for quite a few different
## things in Ambassador -- they are basically the generic "upstream service"
## entity.


class IRCluster(IRResource):
    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        parent_ir_resource: "IRResource",
        location: str,  # REQUIRED
        service: str,  # REQUIRED
        resolver: Optional[str] = None,
        connect_timeout_ms: Optional[int] = 3000,
        cluster_idle_timeout_ms: Optional[int] = None,
        cluster_max_connection_lifetime_ms: Optional[int] = None,
        marker: Optional[str] = None,  # extra marker for this context name
        stats_name: Optional[str] = None,  # Override the stats name for this cluster
        ctx_name: Optional[Union[str, bool]] = None,
        host_rewrite: Optional[str] = None,
        dns_type: Optional[str] = "strict_dns",
        enable_ipv4: Optional[bool] = None,
        enable_ipv6: Optional[bool] = None,
        lb_type: str = "round_robin",
        grpc: Optional[bool] = False,
        allow_scheme: Optional[bool] = True,
        load_balancer: Optional[dict] = None,
        keepalive: Optional[dict] = None,
        circuit_breakers: Optional[list] = None,
        respect_dns_ttl: Optional[bool] = False,
        rkey: str = "-override-",
        kind: str = "IRCluster",
        apiVersion: str = "getambassador.io/v0",  # Not a typo! See below.
        **kwargs,
    ) -> None:
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
        name_fields: List[str] = ["cluster"]
        ctx: Optional[IRTLSContext] = None
        errors: List[str] = []
        unknown_breakers = 0

        # Do we have a marker?
        if marker:
            name_fields.append(marker)

        # Set this flag to True if you discover something that's grave enough to warrant ignoring this cluster
        self.ignore_cluster = False

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
            service = service[len("https://") :]

            originate_tls = True
            name_fields.append("otls")

        elif allow_scheme and service.lower().startswith("http://"):
            service = service[len("http://") :]

            if ctx:
                errors.append(
                    "Originate-TLS context %s being used even though service %s lists HTTP"
                    % (ctx_name, service)
                )
                originate_tls = True
                name_fields.append("otls")
            else:
                originate_tls = False

        elif ctx:
            # No scheme (or schemes are ignored), but we have a context.
            originate_tls = True
            name_fields.append("otls")
            name_fields.append(ctx.name)

        if "://" in service:
            # WTF is this?
            idx = service.index("://")
            scheme = service[0:idx]

            if allow_scheme:
                errors.append(
                    "service %s has unknown scheme %s, assuming %s"
                    % (service, scheme, "HTTPS" if originate_tls else "HTTP")
                )
            else:
                errors.append(
                    "ignoring scheme %s for service %s, since it is being used for a non-HTTP mapping"
                    % (scheme, service)
                )

            service = service[idx + 3 :]

        # XXX Should this be checking originate_tls? Why does it do that?
        if originate_tls and host_rewrite:
            name_fields.append("hr-%s" % host_rewrite)

        # Parse the service as a URL. Note that we have to supply a scheme to urllib's
        # parser, because it's kind of stupid.

        ir.logger.debug("cluster setup: service %s otls %s ctx %s" % (service, originate_tls, ctx))
        p = urllib.parse.urlparse("random://" + service)

        # Is there any junk after the host?

        if p.path or p.params or p.query or p.fragment:
            errors.append(
                "service %s has extra URL components; ignoring everything but the host and port"
                % service
            )

        # p is read-only, so break stuff out.

        hostname = p.hostname
        namespace = parent_ir_resource.namespace
        # Make sure we save the namespace in the cluster name, to prevent clashes with non-fully qualified service resolution
        name_fields.append(namespace)

        # Do we actually have a hostname?
        if not hostname:
            # We don't. That ain't good.
            errors.append(
                "service %s has no hostname and will be ignored; please re-configure" % service
            )
            self.ignore_cluster = True
            hostname = "unknown"

        try:
            port = p.port
        except ValueError as e:
            errors.append(
                "found invalid port for service {}. Please specify a valid port between 0 and 65535 - {}. Service {} cluster will be ignored, please re-configure".format(
                    service, e, service
                )
            )
            self.ignore_cluster = True
            port = 0

        # If the port is unset, fix it up.
        if not port:
            port = 443 if originate_tls else 80

        # Rebuild the URL with the 'tcp' scheme and our changed info.
        # (Yes, really, TCP. Envoy uses the TLS context to determine whether to originate
        # TLS. Kind of odd, but there we go.)
        url = "tcp://%s:%d" % (hostname, port)

        # Is there a circuit breaker involved here?
        if circuit_breakers:
            for breaker in circuit_breakers:
                name = breaker.get("_name", None)

                if name:
                    name_fields.append(name)
                else:
                    # This is "impossible", but... let it go I guess?
                    errors.append(f"{service}: unvalidated circuit breaker {breaker}!")
                    name_fields.append(f"cbu{unknown_breakers}")
                    unknown_breakers += 1

        # The Ambassador module will always have a load_balancer (which may be None).
        global_load_balancer = ir.ambassador_module.load_balancer

        if not load_balancer:
            load_balancer = global_load_balancer

        self.logger.debug(f"Load balancer for {url} is {load_balancer}")

        enable_endpoints = False

        if self.endpoints_required(load_balancer):
            if not Config.enable_endpoints:
                # Bzzt.
                errors.append(
                    f"{service}: endpoint routing is not enabled, falling back to {global_load_balancer}"
                )
                load_balancer = global_load_balancer
            else:
                enable_endpoints = True

                if load_balancer:
                    # This is used only for cluster naming; it doesn't need to be a real
                    # load balancer policy.

                    lb_type = load_balancer.get("policy", "default")

                    key_fields = ["er", lb_type.lower()]

                    # XXX Should we really include these things?
                    if "header" in load_balancer:
                        key_fields.append("hdr")
                        key_fields.append(load_balancer["header"])

                    if "cookie" in load_balancer:
                        key_fields.append("cookie")
                        key_fields.append(load_balancer["cookie"]["name"])

                    if "source_ip" in load_balancer:
                        key_fields.append("srcip")

                    name_fields.append("-".join(key_fields))

        # Finally we can construct the cluster name.
        name = "_".join(name_fields)
        name = re.sub(r"[^0-9A-Za-z_]", "_", name)

        # OK. Build our default args.
        #
        # XXX We should really save the hostname and the port, not the URL.

        if enable_ipv4 is None:
            enable_ipv4 = ir.ambassador_module.enable_ipv4
            ir.logger.debug(
                "%s: copying enable_ipv4 %s from Ambassador Module" % (name, enable_ipv4)
            )

        if enable_ipv6 is None:
            enable_ipv6 = ir.ambassador_module.enable_ipv6
            ir.logger.debug(
                "%s: copying enable_ipv6 %s from Ambassador Module" % (name, enable_ipv6)
            )

        new_args: Dict[str, Any] = {
            "type": dns_type,
            "lb_type": lb_type,
            "urls": [url],  # TODO: Should we completely eliminate `urls` in favor of `targets`?
            "load_balancer": load_balancer,
            "keepalive": keepalive,
            "circuit_breakers": circuit_breakers,
            "service": service,
            "enable_ipv4": enable_ipv4,
            "enable_ipv6": enable_ipv6,
            "enable_endpoints": enable_endpoints,
            "connect_timeout_ms": connect_timeout_ms,
            "cluster_idle_timeout_ms": cluster_idle_timeout_ms,
            "cluster_max_connection_lifetime_ms": cluster_max_connection_lifetime_ms,
            "respect_dns_ttl": respect_dns_ttl,
        }

        # If we have a stats_name, use it. If not, default it to the service to make life
        # easier for people trying to find stats later -- but translate unusual characters
        # to underscores, just in case.
        if stats_name:
            new_args["stats_name"] = stats_name
        else:
            new_args["stats_name"] = re.sub(r"[^0-9A-Za-z_]", "_", service)

        if grpc:
            new_args["grpc"] = True

        if host_rewrite:
            new_args["host_rewrite"] = host_rewrite

        if originate_tls:
            if ctx:
                new_args["tls_context"] = typecast(IRTLSContext, ctx)
            else:
                new_args["tls_context"] = IRTLSContext.null_context(ir=ir)

        if rkey == "-override-":
            rkey = name

        # Stash the resolver, hostname, and port for setup.
        self._resolver = resolver
        self._hostname = hostname
        self._namespace = namespace
        self._port = port
        self._is_sidecar = False

        if self._hostname == "127.0.0.1" and self._port == 8500:
            self._is_sidecar = True

        super().__init__(
            ir=ir,
            aconf=aconf,
            rkey=rkey,
            location=location,
            kind=kind,
            name=name,
            apiVersion=apiVersion,
            **new_args,
        )

        if ctx:
            ctx.referenced_by(self)

        if errors:
            for error in errors:
                ir.post_error(error, resource=self)

    def setup(self, ir: "IR", aconf: Config) -> bool:
        self._cache_key = f"Cluster-{self.name}"

        if self.ignore_cluster:
            return False

        # Resolve our actual targets.
        targets = ir.resolve_targets(
            self, self._resolver, self._hostname, self._namespace, self._port
        )

        self.targets = targets

        if not targets:
            self.ir.logger.debug("accepting cluster with no endpoints: %s" % self.name)

        return True

    def is_edge_stack_sidecar(self) -> bool:
        return self.is_active() and self._is_sidecar

    def endpoints_required(self, load_balancer) -> bool:
        required = False

        if load_balancer:
            lb_policy = load_balancer.get("policy")

            if lb_policy in ["round_robin", "least_request", "ring_hash", "maglev"]:
                self.logger.debug(
                    "Endpoints are required for load balancing policy {}".format(lb_policy)
                )
                required = True

        return required

    def add_url(self, url: str) -> List[str]:
        self.urls.append(url)

        return self.urls

    def merge(self, other: "IRCluster") -> bool:
        # Is this mergeable?

        mismatches = []

        for key in [
            "type",
            "lb_type",
            "host_rewrite",
            "tls_context",
            "originate_tls",
            "grpc",
            "connect_timeout_ms",
            "cluster_idle_timeout_ms",
            "cluster_max_connection_lifetime_ms",
        ]:
            if self.get(key, None) != other.get(key, None):
                mismatches.append(key)

        if mismatches:
            self.post_error(
                RichStatus.fromError(
                    "cannot merge cluster %s: mismatched attributes %s"
                    % (other.name, ", ".join(mismatches))
                )
            )
            return False

        # All good.
        if other.urls:
            self.referenced_by(other)

            for url in other.urls:
                self.add_url(url)

        if other.targets:
            self.referenced_by(other)
            if self.targets == None:
                self.targets = other.targets
            else:
                self.targets = (
                    typecast(List[Dict[str, Union[int, str]]], self.targets) + other.targets
                )

        return True

    def get_resolver(self) -> "IRServiceResolver":
        return self.ir.resolve_resolver(self, self._resolver)

    def clustermap_entry(self) -> Dict:
        return self.get_resolver().clustermap_entry(
            self.ir, self, self._hostname, self._namespace, self._port
        )
