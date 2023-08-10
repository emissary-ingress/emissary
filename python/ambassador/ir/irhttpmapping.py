import hashlib
from typing import TYPE_CHECKING, Any, ClassVar, Dict, List, Optional, Type, Union

from ambassador.utils import ParsedService as Service
from ambassador.utils import RichStatus

from ..config import Config
from .irbasemapping import IRBaseMapping, normalize_service_name
from .irbasemappinggroup import IRBaseMappingGroup
from .ircors import IRCORS
from .irerrorresponse import IRErrorResponse
from .irhttpmappinggroup import IRHTTPMappingGroup
from .irretrypolicy import IRRetryPolicy
from .irutils import selector_matches

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


# Kind of cheating here so that it's easy to json-serialize key-value pairs (including with regex)
class KeyValueDecorator(dict):
    def __init__(
        self, name: str, value: Optional[str] = None, regex: Optional[bool] = False
    ) -> None:
        super().__init__()
        self.name = name
        self.value = value
        self.regex = regex

    def __getattr__(self, key: str) -> Any:
        return self[key]

    def __setattr__(self, key: str, value: Any) -> None:
        self[key] = value

    def _get_value(self) -> str:
        return self.value or "*"

    def length(self) -> int:
        return len(self.name) + len(self._get_value()) + (1 if self.regex else 0)

    def key(self) -> str:
        return self.name + "-" + self._get_value()


class IRHTTPMapping(IRBaseMapping):
    prefix: str
    headers: List[KeyValueDecorator]
    add_request_headers: Dict[str, str]
    add_response_headers: Dict[str, str]
    method: Optional[str]
    service: str
    group_id: str
    route_weight: List[Union[str, int]]
    cors: IRCORS
    retry_policy: IRRetryPolicy
    error_response_overrides: Optional[IRErrorResponse]
    query_parameters: List[KeyValueDecorator]
    regex_rewrite: Dict[str, str]

    # Keys that are present in AllowedKeys are allowed to be set from kwargs.
    # If the value is True, we'll look for a default in the Ambassador module
    # if the key is missing. If the value is False, a missing key will simply
    # be unset.
    #
    # Do not include any named parameters (like 'precedence' or 'rewrite').
    #
    # Any key here will be copied into the mapping. Keys where the only
    # processing is to set something else (like 'host' and 'method', whose
    # which only need to set the ':authority' and ':method' headers) must
    # _not_ be included here. Keys that need to be copied _and_ have special
    # processing (like 'service', which must be copied and used to wrangle
    # Linkerd headers) _do_ need to be included.

    AllowedKeys: ClassVar[Dict[str, bool]] = {
        "add_linkerd_headers": False,
        # Do not include add_request_headers and add_response_headers
        "auto_host_rewrite": False,
        "bypass_auth": False,
        "auth_context_extensions": False,
        "bypass_error_response_overrides": False,
        "case_sensitive": False,
        "circuit_breakers": False,
        "cluster_idle_timeout_ms": False,
        "cluster_max_connection_lifetime_ms": False,
        # Do not include cluster_tag
        "connect_timeout_ms": False,
        "cors": False,
        "docs": False,
        "dns_type": False,
        "enable_ipv4": False,
        "enable_ipv6": False,
        "error_response_overrides": False,
        "grpc": False,
        # Do not include headers
        # Do not include host
        # Do not include hostname
        "health_checks": False,
        "host_redirect": False,
        "host_regex": False,
        "host_rewrite": False,
        "idle_timeout_ms": False,
        "keepalive": False,
        "labels": False,  # Not supported in v0; requires v1+; handled in setup
        "load_balancer": False,
        "metadata_labels": False,
        # Do not include method
        "method_regex": False,
        "path_redirect": False,
        "prefix_redirect": False,
        "regex_redirect": False,
        "redirect_response_code": False,
        # Do not include precedence
        "prefix": False,
        "prefix_exact": False,
        "prefix_regex": False,
        "priority": False,
        "rate_limits": False,  # Only supported in v0; replaced by "labels" in v1; handled in setup
        # Do not include regex_headers
        "remove_request_headers": True,
        "remove_response_headers": True,
        "resolver": False,
        "respect_dns_ttl": False,
        "retry_policy": False,
        # Do not include rewrite
        "service": False,  # See notes above
        "shadow": False,
        "stats_name": True,
        "timeout_ms": False,
        "tls": False,
        "use_websocket": False,
        "allow_upgrade": False,
        "weight": False,
        # Include the serialization, too.
        "serialization": False,
    }

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        rkey: str,  # REQUIRED
        name: str,  # REQUIRED
        location: str,  # REQUIRED
        service: str,  # REQUIRED
        namespace: Optional[str] = None,
        metadata_labels: Optional[Dict[str, str]] = None,
        kind: str = "IRHTTPMapping",
        apiVersion: str = "getambassador.io/v3alpha1",  # Not a typo! See below.
        precedence: int = 0,
        rewrite: str = "/",
        cluster_tag: Optional[str] = None,
        **kwargs,
    ) -> None:
        # OK, this is a bit of a pain. We want to preserve the name and rkey and
        # such here, unlike most kinds of IRResource, so we shallow copy the keys
        # we're going to allow from the incoming kwargs.
        #
        # NOTE WELL: things that we explicitly process below should _not_ be listed in
        # AllowedKeys. The point of AllowedKeys is this loop below.

        new_args = {}

        # When we look up defaults, use lookup class "httpmapping"... and yeah, we need the
        # IR, too.
        self.default_class = "httpmapping"
        self.ir = ir

        for key, check_defaults in IRHTTPMapping.AllowedKeys.items():
            # Do we have a keyword arg for this key?
            if key in kwargs:
                # Yes, it wins.
                value = kwargs[key]
                new_args[key] = value
            elif check_defaults:
                # No value in kwargs, but we're allowed to check defaults for it.
                value = self.lookup_default(key)

                if value is not None:
                    new_args[key] = value

        # add_linkerd_headers is special, because we support setting it as a default
        # in the bare Ambassador module. We should really toss this and use the defaults
        # mechanism, but not for 1.4.3.

        if "add_linkerd_headers" not in new_args:
            # They didn't set it explicitly, so check for the older way.
            add_linkerd_headers = self.ir.ambassador_module.get("add_linkerd_headers", None)

            if add_linkerd_headers != None:
                new_args["add_linkerd_headers"] = add_linkerd_headers

        # OK. On to set up the headers (since we need them to compute our group ID).
        hdrs = []
        query_parameters = []
        regex_rewrite = kwargs.get("regex_rewrite", {})

        # Start by assuming that nothing in our arguments mentions hosts (so no host and no host_regex).
        host = None
        host_regex = False

        # Also start self.host as unspecified.
        self.host = None

        # OK. Start by looking for a :authority header match.
        if "headers" in kwargs:
            for hdr_name, hdr_value in kwargs.get("headers", {}).items():
                if hdr_value is True:
                    hdrs.append(KeyValueDecorator(hdr_name))
                else:
                    # An exact match on the :authority header is special -- treat it like
                    # they set the "host" element (but note that we'll allow the actual
                    # "host" element to override it later).
                    if hdr_name.lower() == ":authority":
                        # This is an _exact_ match, so it mustn't contain a "*" -- that's illegal in the DNS.
                        if "*" in hdr_value:
                            # We can't call self.post_error() yet, because we're not initialized yet. So we cheat a bit
                            # and defer the error for later.
                            new_args[
                                "_deferred_error"
                            ] = f":authority exact-match '{hdr_value}' contains *, which cannot match anything."
                            ir.logger.debug(
                                "IRHTTPMapping %s: self.host contains * (%s, :authority)",
                                name,
                                hdr_value,
                            )
                        else:
                            # No globs, just save it. (We'll end up using it as a glob later, in the Envoy
                            # config part of the world, but that's OK -- a glob with no "*" in it will always
                            # match only itself.)
                            host = hdr_value
                            ir.logger.debug(
                                "IRHTTPMapping %s: self.host == %s (:authority)", name, self.host
                            )
                            # DO NOT save the ':authority' match here -- we'll pick it up after we've checked
                            # for hostname, too.
                    else:
                        # It's not an :authority match, so we're good.
                        hdrs.append(KeyValueDecorator(hdr_name, hdr_value))

        if "regex_headers" in kwargs:
            # DON'T do anything special with a regex :authority match: we can't
            # do host-based filtering within the IR for it anyway.
            for name, value in kwargs.get("regex_headers", {}).items():
                hdrs.append(KeyValueDecorator(name, value, regex=True))

        if "host" in kwargs:
            # It's deliberate that we'll allow kwargs['host'] to silently override an exact :authority
            # header match.
            host = kwargs["host"]
            host_regex = kwargs.get("host_regex", False)

            # If it's not a regex, it's an exact match -- make sure it doesn't contain a '*'.
            if not host_regex:
                if "*" in host:
                    # We can't call self.post_error() yet, because we're not initialized yet. So we cheat a bit
                    # and defer the error for later.
                    new_args[
                        "_deferred_error"
                    ] = f"host exact-match {host} contains *, which cannot match anything."
                    ir.logger.debug("IRHTTPMapping %s: self.host contains * (%s, host)", name, host)
                else:
                    ir.logger.debug("IRHTTPMapping %s: self.host == %s (host)", name, self.host)

        # Finally, check for 'hostname'.
        if "hostname" in kwargs:
            # It's deliberate that we allow kwargs['hostname'] to override anything else -- even a regex host.
            # Yell about it, though.
            if host:
                ir.logger.warning(
                    "Mapping %s in namespace %s: both host and hostname are set, using hostname and ignoring host",
                    name,
                    namespace,
                )

            # No need to be so careful about "*" here, since hostname is defined to be a glob.
            host = kwargs["hostname"]
            host_regex = False
            ir.logger.debug("IRHTTPMapping %s: self.host gl~ %s (hostname)", name, self.host)

        # If we have a host, include a ":authority" match. We're treating this as if it were
        # an exact match, but that's because the ":authority" match is handling specially by
        # Envoy.
        if host:
            hdrs.append(KeyValueDecorator(":authority", host, host_regex))

            # Finally, if our host isn't a regex, save it in self.host.
            if not host_regex:
                self.host = host

        if "method" in kwargs:
            hdrs.append(
                KeyValueDecorator(":method", kwargs["method"], kwargs.get("method_regex", False))
            )

        if "use_websocket" in new_args:
            allow_upgrade = new_args.setdefault("allow_upgrade", [])
            if "websocket" not in allow_upgrade:
                allow_upgrade.append("websocket")
            del new_args["use_websocket"]

        # Next up: figure out what headers we need to add to each request. Again, if the key
        # is present in kwargs, the kwargs value wins -- this is important to allow explicitly
        # setting a value of `{}` to override a default!

        add_request_hdrs: dict
        add_response_hdrs: dict

        if "add_request_headers" in kwargs:
            add_request_hdrs = kwargs["add_request_headers"]
        else:
            add_request_hdrs = self.lookup_default("add_request_headers", {})

        if "add_response_headers" in kwargs:
            add_response_hdrs = kwargs["add_response_headers"]
        else:
            add_response_hdrs = self.lookup_default("add_response_headers", {})

        # Remember that we may need to add the Linkerd headers, too.
        add_linkerd_headers = new_args.get("add_linkerd_headers", False)

        # XXX The resolver lookup code is duplicated from IRBaseMapping.setup --
        # needs to be fixed after 1.6.1.
        resolver_name = kwargs.get("resolver") or self.ir.ambassador_module.get(
            "resolver", "kubernetes-service"
        )

        assert resolver_name  # for mypy -- resolver_name cannot be None at this point
        resolver = self.ir.get_resolver(resolver_name)

        if resolver:
            resolver_kind = resolver.kind
        else:
            # In IRBaseMapping.setup, we post an error if the resolver is unknown.
            # Here, we just don't bother; we're only using it for service
            # qualification.
            resolver_kind = "KubernetesBogusResolver"

        service = normalize_service_name(ir, service, namespace, resolver_kind, rkey=rkey)
        self.ir.logger.debug(f"Mapping {name} service qualified to {repr(service)}")

        svc = Service(ir.logger, service)

        if add_linkerd_headers:
            add_request_hdrs["l5d-dst-override"] = svc.hostname_port

        # XXX BRUTAL HACK HERE:
        # If we _don't_ have an origination context, but our IR has an agent_origination_ctx,
        # force TLS origination because it's the agent. I know, I know. It's a hack.
        if ("tls" not in new_args) and ir.agent_origination_ctx:
            ir.logger.debug(
                f"Mapping {name}: Agent forcing origination TLS context to {ir.agent_origination_ctx.name}"
            )
            new_args["tls"] = ir.agent_origination_ctx.name

        if "query_parameters" in kwargs:
            for pname, pvalue in kwargs.get("query_parameters", {}).items():
                if pvalue is True:
                    query_parameters.append(KeyValueDecorator(pname))
                else:
                    query_parameters.append(KeyValueDecorator(pname, pvalue))

        if "regex_query_parameters" in kwargs:
            for pname, pvalue in kwargs.get("regex_query_parameters", {}).items():
                query_parameters.append(KeyValueDecorator(pname, pvalue, regex=True))

        if "regex_rewrite" in kwargs:
            if rewrite and rewrite != "/":
                self.ir.aconf.post_notice(
                    "Cannot specify both rewrite and regex_rewrite: using regex_rewrite and ignoring rewrite"
                )
            rewrite = ""
            rewrite_items = kwargs.get("regex_rewrite", {})
            regex_rewrite = {
                "pattern": rewrite_items.get("pattern", ""),
                "substitution": rewrite_items.get("substitution", ""),
            }

        # ...and then init the superclass.
        super().__init__(
            ir=ir,
            aconf=aconf,
            rkey=rkey,
            location=location,
            service=service,
            kind=kind,
            name=name,
            namespace=namespace,
            metadata_labels=metadata_labels,
            apiVersion=apiVersion,
            headers=hdrs,
            add_request_headers=add_request_hdrs,
            add_response_headers=add_response_hdrs,
            precedence=precedence,
            rewrite=rewrite,
            cluster_tag=cluster_tag,
            query_parameters=query_parameters,
            regex_rewrite=regex_rewrite,
            **new_args,
        )

        if "outlier_detection" in kwargs:
            self.post_error(RichStatus.fromError("outlier_detection is not supported"))

    @staticmethod
    def group_class() -> Type[IRBaseMappingGroup]:
        return IRHTTPMappingGroup

    def _enforce_mutual_exclusion(self, preferred, other):
        if preferred in self and other in self:
            self.ir.aconf.post_error(
                f"Cannot specify both {preferred} and {other}. Using {preferred} and ignoring {other}.",
                resource=self,
            )
            del self[other]

    def setup(self, ir: "IR", aconf: Config) -> bool:
        # First things first: handle any deferred error.
        _deferred_error = self.get("_deferred_error")
        if _deferred_error:
            self.post_error(_deferred_error)
            return False

        if not super().setup(ir, aconf):
            return False

        # If we have CORS stuff, normalize it.
        if "cors" in self:
            self.cors = IRCORS(ir=ir, aconf=aconf, location=self.location, **self.cors)

            if self.cors:
                self.cors.referenced_by(self)
            else:
                return False

        # If we have RETRY_POLICY stuff, normalize it.
        if "retry_policy" in self:
            self.retry_policy = IRRetryPolicy(
                ir=ir, aconf=aconf, location=self.location, **self.retry_policy
            )

            if self.retry_policy:
                self.retry_policy.referenced_by(self)
            else:
                return False

        # If we have error response overrides, generate an IR for that too.
        if "error_response_overrides" in self:
            self.error_response_overrides = IRErrorResponse(
                self.ir, aconf, self.get("error_response_overrides", None), location=self.location
            )
            # if self.error_response_overrides.setup(self.ir, aconf):
            if self.error_response_overrides:
                self.error_response_overrides.referenced_by(self)
            else:
                return False

        if self.get("load_balancer", None) is not None:
            if not self.validate_load_balancer(self["load_balancer"]):
                self.post_error(
                    "Invalid load_balancer specified: {}, invalidating mapping".format(
                        self["load_balancer"]
                    )
                )
                return False

        # All three redirect fields are mutually exclusive.
        #
        # Prefer path_redirect over the other two. If only prefix_redirect and
        # regex_redirect are set, prefer prefix_redirect. There's no exact
        # reason for this, only to arbitrarily prefer "less fancy" features.
        self._enforce_mutual_exclusion("path_redirect", "prefix_redirect")
        self._enforce_mutual_exclusion("path_redirect", "regex_redirect")
        self._enforce_mutual_exclusion("prefix_redirect", "regex_redirect")

        ir.logger.debug(
            "Mapping %s: setup OK: host %s hostname %s regex %s",
            self.name,
            self.get("host"),
            self.get("hostname"),
            self.get("host_regex"),
        )

        return True

    @staticmethod
    def validate_load_balancer(load_balancer) -> bool:
        lb_policy = load_balancer.get("policy", None)

        is_valid = False
        if lb_policy in ["round_robin", "least_request"]:
            if len(load_balancer) == 1:
                is_valid = True
        elif lb_policy in ["ring_hash", "maglev"]:
            if len(load_balancer) == 2:
                if "cookie" in load_balancer:
                    cookie = load_balancer.get("cookie")
                    if "name" in cookie:
                        is_valid = True
                elif "header" in load_balancer:
                    is_valid = True
                elif "source_ip" in load_balancer:
                    is_valid = True

        return is_valid

    # Mappings are grouped by:
    #  - HTTP Method
    #  - Prefix
    #  - Headers
    #  - Query Parameters
    #  - Mapping Label Selectors
    #  - Precedence
    def _group_id(self) -> str:
        # This is an HTTP mapping.
        group_id = "HTTP-".encode("utf-8")

        # Method
        method = self.get("method") or "GET"
        group_id = group_id + method.encode("utf-8")

        # Prefix
        group_id = group_id + self.prefix.encode("utf-8")

        # Headers
        for hdr in self.headers:
            group_id = group_id + hdr.name.encode("utf-8")

            if hdr.value is not None:
                group_id = group_id + hdr.value.encode("utf-8")

        # Query Parameters
        for query_parameter in self.query_parameters:
            group_id = group_id + query_parameter.name.encode("utf-8")

            if query_parameter.value is not None:
                group_id = group_id + query_parameter.value.encode("utf-8")

        # Host Mapping Selector Labels
        if not self.host and self.metadata_labels is not None:
            for host in self.ir.hosts.values():
                mapsel = host.get("mappingSelector")
                if not mapsel:
                    continue

                if selector_matches(self.ir.logger, mapsel, self.metadata_labels):
                    # We care only about the labels that are part of the Host mappingSelector.
                    # For example, let's say there are two Mappings with labels
                    # host=foo;irrelevant-label=1 for one Mapping and host=foo;irrelevant-label=2
                    # for the other Mapping. There exists a Host that contains a mappingSelector
                    # for host=foo. We would only want to group based on the host label and not
                    # the irrelevant label. In this case the two Mappings are part of the same group
                    # assumming method, prefix, headers, etc. all match.
                    for key, val in mapsel.get("matchLabels", {}).items():
                        group_id = group_id + key.encode("utf-8")
                        group_id = group_id + val.encode("utf-8")

        # Precedence
        if self.precedence != 0:
            group_id = group_id + str(self.precedence).encode("utf-8")

        h = hashlib.new("sha1")
        h.update(group_id)
        return h.hexdigest()

    def _route_weight(self) -> List[Union[str, int]]:
        len_headers = 0
        len_query_parameters = 0

        for hdr in self.headers:
            len_headers += hdr.length()

        for query_parameter in self.query_parameters:
            len_query_parameters += query_parameter.length()

        # For calculating the route weight, 'method' defaults to '*' (for historical reasons).

        weight = [
            self.precedence,
            len(self.prefix),
            len_headers,
            len_query_parameters,
            self.prefix,
            self.get("method", "GET"),
        ]
        weight += [hdr.key() for hdr in self.headers]
        weight += [query_parameter.key() for query_parameter in self.query_parameters]

        return weight

    def summarize_errors(self) -> str:
        errors = self.ir.aconf.errors.get(self.rkey, [])
        errstr = "(no errors)"

        if errors:
            errstr = errors[0].get("error") or "unknown error?"

            if len(errors) > 1:
                errstr += " (and more)"

        return errstr

    def status(self) -> Dict[str, str]:
        if not self.is_active():
            return {"state": "Inactive", "reason": self.summarize_errors()}
        else:
            return {"state": "Running"}
