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

import re
from typing import TYPE_CHECKING, Any, Dict, List, Optional, Union
from typing import cast as typecast

from ...cache import Cacheable
from ...ir.irhttpmapping import IRHTTPMapping
from ...ir.irhttpmappinggroup import IRHTTPMappingGroup
from ...ir.irutils import are_mapping_group_fixes_disabled, hostglob_matches
from ..common import EnvoyRoute
from .v3ratelimitaction import V3RateLimitAction

if TYPE_CHECKING:
    from . import V3Config  # pragma: no cover


# This is the root of a certain amount of ugliness in this file -- it's a V3Route
# that's been turned into a plain old dict, so it can be easily JSONified. The
# problem is that that currently happens earlier than it should; I'm hoping to fix
# that shortly.
DictifiedV3Route = Dict[str, Any]


def v3prettyroute(route: DictifiedV3Route) -> str:
    match = route["match"]

    key = "PFX"
    value = match.get("prefix", None)

    if not value:
        key = "SRX"
        value = match.get("safe_regex", {}).get("regex", None)

    if not value:
        key = "???"
        value = "-none-"

    match_str = f"{key} {value}"

    headers = match.get("headers", {})
    xfp: Optional[str] = None
    host: Optional[str] = None

    for header in headers:
        name = header.get("name", None).lower()
        exact = header.get("exact_match", None)

        if header == ":authority":
            if exact:
                host = exact
            elif "prefix_match" in header:
                host = header["prefix_match"] + "*"
            elif "suffix_match" in header:
                host = "*" + header["suffix_match"]
            elif "safe_regex_match" in header:
                host = header["safe_regex_match"]["regex"]
        elif name == "x-forwarded-proto":
            xfp = exact

    if xfp:
        match_str += f" XFP {xfp}"
    else:
        match_str += " ALWAYS"

    if host:
        match_str += f" HOST {host}"

    target_str = "-none-"

    if route.get("route"):
        target_str = f"ROUTE {route['route']['cluster']}"
    elif route.get("redirect"):
        target_str = f"REDIRECT"

    hcstr = route.get("_host_constraints") or "{i'*'}"

    return f"<V3Route {hcstr}: {match_str} -> {target_str}>"


# regex_matcher generates Envoy configuration to do a regex match in a Route. It's complex
# here because, even though we don't have to deal with safe and unsafe regexes, it's simpler
# to keep the weird baroqueness of this stuff wrapped in a function.
def regex_matcher(config: "V3Config", regex: str, key="regex", safe_key=None) -> Dict[str, Any]:
    max_size = int(config.ir.ambassador_module.get("regex_max_size", 200))

    if not safe_key:
        safe_key = "safe_" + key

    return {safe_key: {"google_re2": {"max_program_size": max_size}, "regex": regex}}


class V3RouteVariants:
    """
    A "route variant" is a version of a V3Route that's been modified to
    enforce a certain kind of match, and to take a particular action if the
    match is good.

    For example, a V3Route might look like:

    {
        "match": {
            "prefix": "/foo/",
        },
        "route": {
            "cluster": "cluster_foo_default",
        }
    }

    The variant of that route that redirects to HTTPS if XFP is HTTP would
    be

    {
        "match": {
            "headers": [
                {
                    "name": "x-forwarded-proto"
                    "exact_match": "http",
                }
            ],
            "prefix": "/foo/",
        },
        "redirect": {
            "https_redirect": true
        }
    }

    which can be mechanically constructed from the primary V3Route.

    Note that V3Routes and their variants are independent of any Host,
    Listener, etc. -- they depend only on the matcher and the action, so
    they can be lazily constructed and then cached. V3RouteVariants is
    such a lazy collection of route variants for a given V3Route.
    """

    route: "V3Route"
    variants: Dict[str, DictifiedV3Route]

    def __init__(self, route: "V3Route") -> None:
        self.route = route
        self.variants = {}

    # get_variant might return a cached variant, or it might make a new one.
    # Whatever. The important thing is that you can specify a matcher name
    # and an action name, to mechanically turn a route into a variant of that route.
    def get_variant(self, matcher: str, action: str) -> DictifiedV3Route:
        matcher = matcher.lower()
        action = action.lower()

        # First, check the cache.
        key = f"{matcher}-{action}"

        variant: Optional[DictifiedV3Route] = self.variants.get(key, None)

        if variant:
            # Hit! Good to go.
            return variant

        # The matcher and the action are strings; valid matchers and actions
        # have handler methods in this class with derivable names. To dispatch
        # the matcher and action, we derive the names and call the methods.
        matcher_handler = getattr(self, f"matcher_{matcher.replace('-', '_')}")

        if not matcher_handler:
            raise Exception(f"Invalid route matcher {matcher} requested")

        variant = dict(self.route)
        matcher_handler(variant)

        # Repeat for the action.
        action_handler = getattr(self, f"action_{action.replace('-', '_')}")

        if not action_handler:
            raise Exception(f"Invalid route action {action} requested")

        action_handler(variant)

        self.variants[key] = variant
        return self.variants[key]

    # Always match: don't add anything to the route.
    def matcher_always(self, variant: DictifiedV3Route) -> None:
        pass

    # Match XFP=https.
    def matcher_xfp_https(self, variant: DictifiedV3Route) -> None:
        self.matcher_xfp(variant, "https")

    # Match XFP=http... but we turn that into "don't match XFP at all"
    # because if XFP isn't set (somehow?), we want that case to match
    # here. It's really "not https" as opposed to "equals http".
    #
    # (We could also have done this as "invert XFP=https" but this is a
    # better fit for what we'e done historically.)
    def matcher_xfp_http(self, variant: DictifiedV3Route) -> None:
        self.matcher_xfp(variant, None)

    # Heavy lifting for the XFP matchers.
    def matcher_xfp(self, variant: DictifiedV3Route, value: Optional[str]) -> None:
        # We're going to create a new XFP match, so start by making a
        # copy of the match struct...
        match_copy = dict(variant["match"])
        variant["match"] = match_copy

        # ...then make a copy of match["headers"], but don't include
        # any existing XFP header match.
        headers = match_copy.get("headers") or []
        headers_copy = [h for h in headers if h.get("name", "").lower() != "x-forwarded-proto"]

        # OK, if the new XFP value is anything, write a match for it. If not,
        # we'll just match any XFP.

        if value:
            headers_copy.append({"name": "x-forwarded-proto", "exact_match": value})

        # Don't bother writing headers_copy back if it's empty.
        if headers_copy:
            match_copy["headers"] = headers_copy

    # Route a request -- really this means "do what the rule asks for", which might
    # mean a host redirect or the like. So we don't change anything here.
    def action_route(self, variant) -> None:
        pass

    # Redirect a request. Drop any previous "route" element and force a redirect
    # instead.
    def action_redirect(self, variant) -> None:
        variant.pop("route", None)
        variant["redirect"] = {"https_redirect": True}
        for filter in self.route._group.ir.filters:
            if filter.kind == "IRAuth":
                # Required to ensure that the redirect occurs prior to calling ext_authz
                # when an AuthService is applied
                variant["typed_per_filter_config"] = {
                    "envoy.filters.http.ext_authz": {
                        "@type": "type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthzPerRoute",
                        "disabled": True,
                    }
                }


# Model an Envoy route.
#
# This is where the magic happens to actually route an HTTP request. There's a
# lot going on here because the Envoy route element is actually pretty complex.
#
# Of particular note is the ["_host_constraints"] element: the Mapping can
# either specify a single host glob, or nothing (which means "*"). All the
# context-matching madness happens up at the chain level, so we only need to
# mess with the one host glob at this point.


class V3Route(Cacheable):
    _group: IRHTTPMappingGroup

    def __init__(
        self, config: "V3Config", group: IRHTTPMappingGroup, mapping: IRHTTPMapping
    ) -> None:
        super().__init__()

        # Save the logger and the group.
        self.logger = group.logger
        self._group = group

        # To start, get all the route matching settings from the group since every Mapping within it shares the same
        # matching settings

        # Passing a list to set is _very important_ here, lest you get a set of
        # the individual characters in group.host!
        self["_host_constraints"] = set([group.get("host") or "*"])

        if group.get("precedence"):
            self["_precedence"] = group["precedence"]

        envoy_route = EnvoyRoute(group).envoy_route

        seed_mapping = mapping
        if are_mapping_group_fixes_disabled():
            seed_mapping = group.seed_mapping

        mapping_prefix = mapping.get("prefix", None)
        route_prefix = mapping_prefix if mapping_prefix is not None else group.get("prefix")

        mapping_case_sensitive = mapping.get("case_sensitive", None)
        case_sensitive = (
            mapping_case_sensitive
            if mapping_case_sensitive is not None
            else group.get("case_sensitive", True)
        )

        runtime_fraction: Dict[str, Union[dict, str]] = {
            "default_value": {"numerator": mapping.get("_weight", 100), "denominator": "HUNDRED"}
        }

        match = {"case_sensitive": case_sensitive, "runtime_fraction": runtime_fraction}

        if len(mapping) > 0:
            if "cluster" in mapping:
                runtime_fraction[
                    "runtime_key"
                ] = f"routing.traffic_shift.{mapping.cluster.envoy_name}"
            # If it's a redirect mapping we don't have a cluster, otherwise this is an error
            elif mapping.get("host_redirect", None) == None:
                config.ir.logger.error(
                    "%s: Mapping %s has no cluster? %s",
                    mapping.rkey,
                    route_prefix,
                    mapping.as_json(),
                )
                self["_failed"] = True

        match = {"case_sensitive": case_sensitive, "runtime_fraction": runtime_fraction}

        if envoy_route == "prefix":
            match["prefix"] = route_prefix
        elif envoy_route == "path":
            match["path"] = route_prefix
        else:
            # Cheat.
            if config.ir.edge_stack_allowed and (self.get("_precedence", 0) == -1000000):
                # Force the safe_regex engine.
                match.update(
                    {
                        "safe_regex": {
                            "google_re2": {
                                "max_program_size": 200,
                            },
                            "regex": route_prefix,
                        }
                    }
                )
            else:
                match.update(regex_matcher(config, route_prefix))

        headers_match = self.generate_headers_match(config, group)
        if len(headers_match) > 0:
            match["headers"] = headers_match

        query_params_match = self.generate_query_params_match(config, group)
        if len(query_params_match) > 0:
            match["query_parameters"] = query_params_match

        self["match"] = match

        # Ok, now we move on to getting all of the traffic transformations and routing settings that can be different for
        # each Mapping within the group

        # `typed_per_filter_config` is used to pass typed configuration to Envoy filters
        typed_per_filter_config = {}

        if mapping.get("bypass_error_response_overrides", False):
            typed_per_filter_config["envoy.filters.http.response_map"] = {
                "@type": "type.googleapis.com/envoy.extensions.filters.http.response_map.v3.ResponseMapPerRoute",
                "disabled": True,
            }
        else:
            # The error_response_overrides field is set on the Mapping as input config
            # via kwargs in irhttpmapping.py. Later, in setup(), we replace it with an
            # IRErrorResponse object, which itself returns None if setup failed. This
            # is a similar pattern to IRCors and IRRetrYPolicy.
            #
            # Therefore, if the field is present at this point, it means it's a valid
            # IRErrorResponse with a 'config' field, since setup must have succeded.
            error_response_overrides = mapping.get("error_response_overrides", None)
            if error_response_overrides:
                # The error reponse IR only has optional response map config to use.
                # On this particular code path, we're protected by both Mapping schema
                # and CRD validation so we're reasonable confident there is going to
                # be a valid config here. However the source of this config is theoretically
                # not guaranteed and we need to use the config() method safely, so check
                # first before using it.
                filter_config = error_response_overrides.config()
                if filter_config:
                    # The error response IR itself guarantees that any resulting config() has
                    # at least one mapper in 'mappers', so assert on that here.
                    assert "mappers" in filter_config
                    assert len(filter_config["mappers"]) > 0
                    typed_per_filter_config["envoy.filters.http.response_map"] = {
                        "@type": "type.googleapis.com/envoy.extensions.filters.http.response_map.v3.ResponseMapPerRoute",
                        # The ResponseMapPerRoute Envoy config is similar to the ResponseMap filter
                        # config, except that it is wrapped in another object with key 'response_map'.
                        "response_map": {"mappers": filter_config["mappers"]},
                    }

        if mapping.get("bypass_auth", False) or (mapping.get("host_redirect", None) != None):
            typed_per_filter_config["envoy.filters.http.ext_authz"] = {
                "@type": "type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthzPerRoute",
                "disabled": True,
            }
        else:
            # Additional ext_auth configuration only makes sense when not bypassing auth.
            auth_context_extensions = mapping.get("auth_context_extensions", False)
            if auth_context_extensions:
                typed_per_filter_config["envoy.filters.http.ext_authz"] = {
                    "@type": "type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthzPerRoute",
                    "check_settings": {"context_extensions": auth_context_extensions},
                }

        if len(typed_per_filter_config) > 0:
            self["typed_per_filter_config"] = typed_per_filter_config

        request_headers_to_add = seed_mapping.get("add_request_headers", None)
        if request_headers_to_add:
            self["request_headers_to_add"] = self.generate_headers_to_add(request_headers_to_add)

        response_headers_to_add = seed_mapping.get("add_response_headers", None)
        if response_headers_to_add:
            self["response_headers_to_add"] = self.generate_headers_to_add(response_headers_to_add)

        request_headers_to_remove = seed_mapping.get("remove_request_headers", None)
        if request_headers_to_remove:
            if type(request_headers_to_remove) != list:
                request_headers_to_remove = [request_headers_to_remove]
            self["request_headers_to_remove"] = request_headers_to_remove

        response_headers_to_remove = seed_mapping.get("remove_response_headers", None)
        if response_headers_to_remove:
            if type(response_headers_to_remove) != list:
                response_headers_to_remove = [response_headers_to_remove]
            self["response_headers_to_remove"] = response_headers_to_remove

        host_redirect = seed_mapping.get("host_redirect", None)

        if host_redirect:
            # We have a host_redirect. Deal with it.
            self["redirect"] = {"host_redirect": mapping.service}

            path_redirect = mapping.get("path_redirect", None)
            prefix_redirect = mapping.get("prefix_redirect", None)
            regex_redirect = mapping.get("regex_redirect", None)
            response_code = mapping.get("redirect_response_code", None)

            # We enforce that only one of path_redirect or prefix_redirect is set in the IR.
            # But here, we just prefer path_redirect if that's set.
            if path_redirect:
                self["redirect"]["path_redirect"] = path_redirect
            elif prefix_redirect:
                # In Envoy, it's called prefix_rewrite.
                self["redirect"]["prefix_rewrite"] = prefix_redirect
            elif regex_redirect:
                # In Envoy, it's called regex_rewrite.
                self["redirect"]["regex_rewrite"] = {
                    "pattern": {"google_re2": {}, "regex": regex_redirect.get("pattern", "")},
                    "substitution": regex_redirect.get("substitution", ""),
                }

            # In Ambassador, we express the redirect_reponse_code as the actual
            # HTTP response code for operator simplicity. In Envoy, those codes
            # are represented as an enum, so do the translation here.
            if response_code:
                if response_code == 301:
                    enum_code = 0
                elif response_code == 302:
                    enum_code = 1
                elif response_code == 303:
                    enum_code = 2
                elif response_code == 307:
                    enum_code = 3
                elif response_code == 308:
                    enum_code = 4
                else:
                    config.ir.post_error(
                        f"Unknown redirect_response_code={response_code}, must be one of [301, 302, 303,307, 308]. Using default redirect_response_code=301"
                    )
                    enum_code = 0
                self["redirect"]["response_code"] = enum_code

            return

        # Take the default `timeout_ms` value from the Ambassador module using `cluster_request_timeout_ms`.
        # If that isn't set, use 3000ms. The mapping below will override this if its own `timeout_ms` is set.
        default_timeout_ms = config.ir.ambassador_module.get("cluster_request_timeout_ms", 3000)
        route = {
            "priority": group.get("priority"),
            "timeout": "%0.3fs" % (mapping.get("timeout_ms", default_timeout_ms) / 1000.0),
            "cluster": mapping.cluster.envoy_name,
        }

        idle_timeout_ms = mapping.get("idle_timeout_ms", None)

        if idle_timeout_ms is not None:
            route["idle_timeout"] = "%0.3fs" % (idle_timeout_ms / 1000.0)

        regex_rewrite = self.generate_regex_rewrite(config, mapping)
        if len(regex_rewrite) > 0:
            route["regex_rewrite"] = regex_rewrite
        elif mapping.get("rewrite", None):
            route["prefix_rewrite"] = mapping["rewrite"]

        if "host_rewrite" in mapping:
            route["host_rewrite_literal"] = mapping["host_rewrite"]

        if "auto_host_rewrite" in mapping:
            route["auto_host_rewrite"] = mapping["auto_host_rewrite"]

        hash_policy = self.generate_hash_policy(seed_mapping)
        if len(hash_policy) > 0:
            route["hash_policy"] = [hash_policy]

        cors = None

        if "cors" in seed_mapping:
            cors = seed_mapping.cors
        elif "cors" in config.ir.ambassador_module:
            cors = config.ir.ambassador_module.cors

        if cors:
            cors = cors.dup()
            cors.set_id(mapping.cache_key)

            route["cors"] = cors.as_dict()

        retry_policy = None

        if "retry_policy" in seed_mapping:
            retry_policy = seed_mapping.retry_policy.as_dict()
        elif "retry_policy" in config.ir.ambassador_module:
            retry_policy = config.ir.ambassador_module.retry_policy.as_dict()

        if retry_policy:
            route["retry_policy"] = retry_policy

        # Are we doing any traffic shadowing?
        # Shadow mappings get added to all generated routes for a mapping group since it is an addition to the routing action, and not a replacement for the
        # routing action like a redirect would be. This is why we keep them separate from the regular Mappings.
        if len(group.shadow_mappings) > 0:
            mirror_policies = []
            for shadow_mapping in group.shadow_mappings:
                mirror_policies.append(
                    {
                        "cluster": shadow_mapping.cluster.envoy_name,
                        "runtime_fraction": {
                            "default_value": {
                                "numerator": shadow_mapping.get("weight", 100),
                                "denominator": "HUNDRED",
                            }
                        },
                    }
                )
            route["request_mirror_policies"] = mirror_policies

        # Is RateLimit a thing?
        rlsvc = config.ir.ratelimit

        if rlsvc:
            # Yup. Build our labels into a set of RateLimitActions (remember that default
            # labels have already been handled, as has translating from v0 'rate_limits' to
            # v1 'labels').

            if "labels" in mapping:
                # The Envoy RateLimit filter only supports one domain, so grab the configured domain
                # from the RateLimitService and use that to look up the labels we should use.

                rate_limits = []

                for rl in mapping.labels.get(rlsvc.domain, []):
                    action = V3RateLimitAction(config, rl)

                    if action.valid:
                        rate_limits.append(action.to_dict())

                if rate_limits:
                    route["rate_limits"] = rate_limits

        # Save upgrade configs.
        if seed_mapping.get("allow_upgrade"):
            route["upgrade_configs"] = [
                {"upgrade_type": proto} for proto in seed_mapping.get("allow_upgrade", [])
            ]

        self["route"] = route

    # matches_domain and matches_domains are both still written assuming a _host_constraints
    # with more than element. Not changing that yet.
    def matches_domain(self, domain: str) -> bool:
        return any(hostglob_matches(route_glob, domain) for route_glob in self["_host_constraints"])

    def matches_domains(self, domains: List[str]) -> bool:
        route_hosts = self["_host_constraints"]

        self.logger.debug(f"    - matches_domains: route_hosts {', '.join(sorted(route_hosts))}")
        self.logger.debug(f"    - matches_domains: domains {', '.join(sorted(domains))}")

        if (not route_hosts) or ("*" in route_hosts):
            self.logger.debug(f"    - matches_domains: nonspecific route_hosts")
            return True

        if "*" in domains:
            self.logger.debug(f"    - matches_domains: nonspecific domains")
            return True

        if any([self.matches_domain(domain) for domain in domains]):
            self.logger.debug(f"    - matches_domains: domain match")
            return True

        self.logger.debug(f"    - matches_domains: nothing matches")
        return False

    @classmethod
    def get_route(
        cls,
        config: "V3Config",
        cache_key: str,
        mapping_group: IRHTTPMappingGroup,
        mapping: IRHTTPMapping,
    ) -> "V3Route":
        route: "V3Route"

        cached_route = config.cache[cache_key]

        if cached_route is None:
            # Cache miss.
            route = V3Route(config, mapping_group, mapping)

            # Cheat a bit and force the route's cache_key.
            route.cache_key = cache_key
            config.cache.add(route)
            # config.cache.link(mapping, route)
            config.cache.link(mapping_group, route)
            config.cache.dump("V3Route synth %s: %s", cache_key, v3prettyroute(route))
        else:
            # Cache hit. We know a priori that it's a V3Route, but let's assert that
            # before casting.
            assert isinstance(cached_route, V3Route)
            route = cached_route

        # One way or another, we have a route now.
        return route

    @classmethod
    def generate(cls, config: "V3Config") -> None:
        config.routes = []

        # Build a route for each Mapping we have
        for mapping_group in config.ir.ordered_http_mapping_groups():
            for mapping in mapping_group.mappings:
                assert isinstance(mapping, IRHTTPMapping)
                key = f"Route-{mapping_group.group_id}-{mapping.cache_key}"
                route = cls.get_route(config, key, mapping_group, mapping)
                if not route.get("_failed", False):
                    config.routes.append(config.save_element("route", mapping_group, route))
                else:
                    config.ir.post_error(f'Route with key "{key}" failed to build')

        # Once that's done, go build the variants on each route.
        config.route_variants = []
        for route in config.routes:
            # Set up a currently-empty set of variants for this route.
            config.route_variants.append(V3RouteVariants(route))

    @staticmethod
    def generate_headers_match(config: "V3Config", mapping_group: IRHTTPMappingGroup) -> List[dict]:
        headers = []

        group_headers = mapping_group.get("headers", [])

        for group_header in group_headers:
            header_name = group_header.get("name")
            header_value = group_header.get("value")

            header = {"name": header_name}

            # Is this a regex?
            if group_header.get("regex"):
                header.update(regex_matcher(config, header_value, key="regex_match"))
            else:
                if header_name == ":authority":
                    # The authority header is special, because its value is a glob.
                    # (This works without the user marking it as such because '*' isn't
                    # valid in DNS names, so we know that treating a name with a '*' as
                    # as exact match will always fail.)
                    if header_value == "*":
                        # This is actually a noop, so just don't include this header.
                        continue
                    elif header_value.startswith("*"):
                        header["suffix_match"] = header_value[1:]
                    elif header_value.endswith("*"):
                        header["prefix_match"] = header_value[:-1]
                    else:
                        # But wait! What about 'foo.*.com'?? Turns out Envoy doesn't
                        # support that in the places it actually does host globbing,
                        # so we won't either for the moment.
                        header["exact_match"] = header_value
                else:
                    header["exact_match"] = header_value

            headers.append(header)

        return headers

    @staticmethod
    def generate_query_params_match(
        config: "V3Config", mapping_group: IRHTTPMappingGroup
    ) -> List[dict]:
        query_parameters = []

        group_query_parameters = mapping_group.get("query_parameters", [])

        for group_query_parameter in group_query_parameters:
            query_parameter = {"name": group_query_parameter.get("name")}

            if group_query_parameter.get("regex"):
                query_parameter.update(
                    {
                        "string_match": regex_matcher(
                            config, group_query_parameter.get("value"), key="regex"
                        )
                    }
                )
            else:
                value = group_query_parameter.get("value", None)
                if value is not None:
                    query_parameter.update(
                        {"string_match": {"exact": group_query_parameter.get("value")}}
                    )
                else:
                    query_parameter.update({"present_match": True})

            query_parameters.append(query_parameter)

        return query_parameters

    @staticmethod
    def generate_hash_policy(mapping: IRHTTPMapping) -> dict:
        hash_policy = {}
        load_balancer = mapping.get("load_balancer", None)
        if load_balancer is not None:
            lb_policy = load_balancer.get("policy")
            if lb_policy in ["ring_hash", "maglev"]:
                cookie = load_balancer.get("cookie")
                header = load_balancer.get("header")
                source_ip = load_balancer.get("source_ip")

                if cookie is not None:
                    hash_policy["cookie"] = {"name": cookie.get("name")}
                    if "path" in cookie:
                        hash_policy["cookie"]["path"] = cookie["path"]
                    if "ttl" in cookie:
                        hash_policy["cookie"]["ttl"] = cookie["ttl"]
                elif header is not None:
                    hash_policy["header"] = {"header_name": header}
                elif source_ip is not None:
                    hash_policy["connection_properties"] = {"source_ip": source_ip}

        return hash_policy

    @staticmethod
    def generate_headers_to_add(header_dict: dict) -> List[dict]:
        headers = []
        for k, v in header_dict.items():
            append = True
            if isinstance(v, dict):
                if "append" in v:
                    append = bool(v["append"])
                headers.append({"header": {"key": k, "value": v["value"]}, "append": append})
            else:
                headers.append(
                    {
                        "header": {"key": k, "value": v},
                        "append": append,  # Default append True, for backward compatability
                    }
                )
        return headers

    @staticmethod
    def generate_regex_rewrite(config: "V3Config", mapping: IRHTTPMapping) -> dict:
        regex_rewrite = {}
        group_regex_rewrite = mapping.get("regex_rewrite", None)
        if group_regex_rewrite is not None:
            pattern = group_regex_rewrite.get("pattern", None)
            if pattern is not None:
                regex_rewrite.update(
                    regex_matcher(config, pattern, key="regex", safe_key="pattern")
                )  # regex_rewrite should never ever be unsafe
        substitution = group_regex_rewrite.get("substitution", None)
        if substitution is not None:
            regex_rewrite["substitution"] = substitution
        return regex_rewrite
