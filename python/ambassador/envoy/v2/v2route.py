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

from typing import Any, Dict, List, Set, Union, TYPE_CHECKING
from typing import cast as typecast

from ..common import EnvoyRoute
from ...cache import Cacheable
from ...ir.irhttpmappinggroup import IRHTTPMappingGroup
from ...ir.irbasemapping import IRBaseMapping

from .v2ratelimitaction import V2RateLimitAction

if TYPE_CHECKING:
    from . import V2Config # pragma: no cover


def regex_matcher(config: 'V2Config', regex: str, key="regex", safe_key=None, re_type=None) -> Dict[str, Any]:
        # If re_type is specified explicitly, do not query its value from config
        if re_type is None:
            re_type = config.ir.ambassador_module.get('regex_type', 'safe').lower()

        config.ir.logger.debug(f"re_type {re_type}")

        # 'safe' is the default. You must explicitly say "unsafe" to get the unsafe
        # regex matcher.
        if re_type != 'unsafe':
            max_size = int(config.ir.ambassador_module.get('regex_max_size', 200))

            if not safe_key:
                safe_key = "safe_" + key

            return {
                safe_key: {
                    "google_re2": {
                        "max_program_size": max_size
                    },
                    "regex": regex
                }
            }
        else:
            return {
                key: regex
            }


def hostglob_matches(glob: str, value: str) -> bool:
    if glob == "*": # special wildcard
        return True
    elif glob.endswith("*"): # prefix match
        return value.startswith(glob[:-1])
    elif glob.startswith("*"): # suffix match
        return value.endswith(glob[1:])
    else: # exact match
        return value == glob


class V2Route(Cacheable):
    def __init__(self, config: 'V2Config', group: IRHTTPMappingGroup, mapping: IRBaseMapping) -> None:
        super().__init__()

        # Stash SNI and precedence info where we can find it later.
        if group.get('sni'):
            self['_sni'] = {
                'hosts': group['tls_context']['hosts'],
                'secret_info': group['tls_context']['secret_info']
            }

        if group.get('precedence'):
            self['_precedence'] = group['precedence']

        envoy_route = EnvoyRoute(group).envoy_route

        mapping_prefix = mapping.get('prefix', None)
        route_prefix = mapping_prefix if mapping_prefix is not None else group.get('prefix')

        mapping_case_sensitive = mapping.get('case_sensitive', None)
        case_sensitive = mapping_case_sensitive if mapping_case_sensitive is not None else group.get('case_sensitive', True)

        runtime_fraction: Dict[str, Union[dict, str]] = {
            'default_value': {
                'numerator': mapping.get('weight', 100),
                'denominator': 'HUNDRED'
            }
        }

        if len(mapping) > 0:
            if not 'cluster' in mapping:
                config.ir.logger.error("%s: Mapping %s has no cluster? %s", mapping.rkey, route_prefix, mapping.as_json())
                self['_failed'] = True
            else:
                runtime_fraction['runtime_key'] = f'routing.traffic_shift.{mapping.cluster.envoy_name}'

        match = {
            'case_sensitive': case_sensitive,
            'runtime_fraction': runtime_fraction
        }

        if envoy_route == 'prefix':
            match['prefix'] = route_prefix
        elif envoy_route == 'path':
            match['path'] = route_prefix
        else:
            # Cheat.
            if config.ir.edge_stack_allowed and (self.get('_precedence', 0) == -1000000):
                # Force the safe_regex engine.
                match.update({
                    "safe_regex": {
                        "google_re2": {
                            "max_program_size": 200,
                        },
                        "regex": route_prefix
                    }
                })
            else:
                match.update(regex_matcher(config, route_prefix))

        headers = self.generate_headers(config, group)
        if len(headers) > 0:
            match['headers'] = headers

        query_parameters = self.generate_query_parameters(config, group)
        if len(query_parameters) > 0:
            match['query_parameters'] = query_parameters

        self['match'] = match

        # `typed_per_filter_config` is used to pass typed configuration to Envoy filters
        typed_per_filter_config = {}

        if mapping.get('bypass_error_response_overrides', False):
            typed_per_filter_config['envoy.filters.http.response_map'] = {
                '@type': 'type.googleapis.com/envoy.extensions.filters.http.response_map.v3.ResponseMapPerRoute',
                'disabled': True,
            }
        else:
            # The error_response_overrides field is set on the Mapping as input config
            # via kwargs in irhttpmapping.py. Later, in setup(), we replace it with an
            # IRErrorResponse object, which itself returns None if setup failed. This
            # is a similar pattern to IRCors and IRRetrYPolicy.
            #
            # Therefore, if the field is present at this point, it means it's a valid
            # IRErrorResponse with a 'config' field, since setup must have succeded.
            error_response_overrides = mapping.get('error_response_overrides', None)
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
                    assert 'mappers' in filter_config
                    assert len(filter_config['mappers']) > 0
                    typed_per_filter_config['envoy.filters.http.response_map'] = {
                        '@type': 'type.googleapis.com/envoy.extensions.filters.http.response_map.v3.ResponseMapPerRoute',
                        # The ResponseMapPerRoute Envoy config is similar to the ResponseMap filter
                        # config, except that it is wrapped in another object with key 'response_map'.
                        'response_map': {
                            'mappers': filter_config['mappers']
                        }
                    }

        if mapping.get('bypass_auth', False):
            typed_per_filter_config['envoy.filters.http.ext_authz'] = {
                '@type': 'type.googleapis.com/envoy.config.filter.http.ext_authz.v2.ExtAuthzPerRoute',
                'disabled': True,
            }
        else:
            # Additional ext_auth configuration only makes sense when not bypassing auth.
            auth_context_extensions = mapping.get('auth_context_extensions', False)
            if auth_context_extensions:
                typed_per_filter_config['envoy.filters.http.ext_authz'] = {
                    '@type': 'type.googleapis.com/envoy.config.filter.http.ext_authz.v2.ExtAuthzPerRoute',
                    'check_settings': {'context_extensions': auth_context_extensions}
                }

        if len(typed_per_filter_config) > 0:
            self['typed_per_filter_config'] = typed_per_filter_config

        request_headers_to_add = group.get('add_request_headers', None)
        if request_headers_to_add:
            self['request_headers_to_add'] = self.generate_headers_to_add(request_headers_to_add)

        response_headers_to_add = group.get('add_response_headers', None)
        if response_headers_to_add:
            self['response_headers_to_add'] = self.generate_headers_to_add(response_headers_to_add)

        request_headers_to_remove = group.get('remove_request_headers', None)
        if request_headers_to_remove:
            if type(request_headers_to_remove) != list:
                request_headers_to_remove = [ request_headers_to_remove ]
            self['request_headers_to_remove'] = request_headers_to_remove

        response_headers_to_remove = group.get('remove_response_headers', None)
        if response_headers_to_remove:
            if type(response_headers_to_remove) != list:
                response_headers_to_remove = [ response_headers_to_remove ]
            self['response_headers_to_remove'] = response_headers_to_remove

        host_redirect = group.get('host_redirect', None)

        if host_redirect:
            # We have a host_redirect. Deal with it.
            self['redirect'] = {
                'host_redirect': host_redirect.service
            }

            path_redirect = host_redirect.get('path_redirect', None)
            prefix_redirect = host_redirect.get('prefix_redirect', None)
            regex_redirect = host_redirect.get('regex_redirect', None)
            response_code = host_redirect.get('redirect_response_code', None)

            # We enforce that only one of path_redirect or prefix_redirect is set in the IR.
            # But here, we just prefer path_redirect if that's set.
            if path_redirect:
                self['redirect']['path_redirect'] = path_redirect
            elif prefix_redirect:
                # In Envoy, it's called prefix_rewrite.
                self['redirect']['prefix_rewrite'] = prefix_redirect
            elif regex_redirect:
                # In Envoy, it's called regex_rewrite.
                self['redirect']['regex_rewrite'] = {
                    'pattern': {
                        'google_re2': {},
                        'regex': regex_redirect.get('pattern', '')
                    },
                    'substitution': regex_redirect.get('substitution', '')
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
                            f"Unknown redirect_response_code={response_code}, must be one of [301, 302, 303,307, 308]. Using default redirect_response_code=301")
                    enum_code = 0
                self['redirect']['response_code'] = enum_code

            return

        # Take the default `timeout_ms` value from the Ambassador module using `cluster_request_timeout_ms`.
        # If that isn't set, use 3000ms. The mapping below will override this if its own `timeout_ms` is set.
        default_timeout_ms = config.ir.ambassador_module.get('cluster_request_timeout_ms', 3000)
        route = {
            'priority': group.get('priority'),
            'timeout': "%0.3fs" % (mapping.get('timeout_ms', default_timeout_ms) / 1000.0),
            'cluster': mapping.cluster.envoy_name
        }

        idle_timeout_ms = mapping.get('idle_timeout_ms', None)

        if idle_timeout_ms is not None:
            route['idle_timeout'] = "%0.3fs" % (idle_timeout_ms / 1000.0)

        regex_rewrite = self.generate_regex_rewrite(config, group)
        if len(regex_rewrite) > 0:
            route['regex_rewrite'] =  regex_rewrite
        elif mapping.get('rewrite', None):
            route['prefix_rewrite'] = mapping['rewrite']

        if 'host_rewrite' in mapping:
            route['host_rewrite'] = mapping['host_rewrite']

        if 'auto_host_rewrite' in mapping:
            route['auto_host_rewrite'] = mapping['auto_host_rewrite']

        hash_policy = self.generate_hash_policy(group)
        if len(hash_policy) > 0:
            route['hash_policy'] = [ hash_policy ]

        cors = None

        if "cors" in group:
            cors = group.cors
        elif "cors" in config.ir.ambassador_module:
            cors = config.ir.ambassador_module.cors

        if cors:
            # Duplicate this IRCORS, then set its group ID correctly.
            cors = cors.dup()
            cors.set_id(group.group_id)

            route['cors'] = cors.as_dict()

        retry_policy = None

        if "retry_policy" in group:
            retry_policy = group.retry_policy.as_dict()
        elif "retry_policy" in config.ir.ambassador_module:
            retry_policy = config.ir.ambassador_module.retry_policy.as_dict()

        if retry_policy:
            route['retry_policy'] = retry_policy

        # Is shadowing enabled?
        shadow = group.get("shadows", None)

        if shadow:
            shadow = shadow[0]

            weight = shadow.get('weight', 100)

            route['request_mirror_policy'] = {
                'cluster': shadow.cluster.envoy_name,
                'runtime_fraction': {
                    'default_value': {
                        'numerator': weight,
                        'denominator': 'HUNDRED'
                    }
                }
            }

        # Is RateLimit a thing?
        rlsvc = config.ir.ratelimit

        if rlsvc:
            # Yup. Build our labels into a set of RateLimitActions (remember that default
            # labels have already been handled, as has translating from v0 'rate_limits' to
            # v1 'labels').

            if "labels" in group:
                # The Envoy RateLimit filter only supports one domain, so grab the configured domain
                # from the RateLimitService and use that to look up the labels we should use.

                rate_limits = []

                for rl in group.labels.get(rlsvc.domain, []):
                    action = V2RateLimitAction(config, rl)

                    if action.valid:
                        rate_limits.append(action.to_dict())

                if rate_limits:
                    route["rate_limits"] = rate_limits

        # Save upgrade configs.
        if group.get('allow_upgrade'):
            route["upgrade_configs"] = [ { 'upgrade_type': proto } for proto in group.get('allow_upgrade', []) ]

        self['route'] = route


    def host_constraints(self, prune_unreachable_routes: bool) -> Set[str]:
        """Return a set of hostglobs that match (a superset of) all hostnames that this route can
        apply to.

        An emtpy set means that this route cannot possibly apply to any hostnames.

        This considers SNI information and (if prune_unreachable_routes) HeaderMatchers that
        `exact_match` on the `:authority` header.  There are other things that could narrow the set
        down more, but that we don't consider (like regex matches on `:authority`), leading to it
        possibly returning a set that is too broad.  That's OK for correctness, it just means that
        we'll emit an Envoy config that contains extra work for Envoy.

        """
        # Start by grabbing a list of all the SNI host globs for this route. If there aren't any,
        # default to "*".
        hostglobs = set(self.get('_sni', {}).get('hosts', ['*']))

        # If we're going to do any aggressive pruning here...
        if prune_unreachable_routes:
            # Note: We're *pruning*; the hostglobs set will only ever get *smaller*, it will never
            # grow.  If it gets down to the empty set, then we can safely bail early.

            # Take all the HeaderMatchers...
            header_matchers = self.get("match", {}).get("headers", [])
            for header in header_matchers:
                # ... and look for ones that exact_match on :authority.
                if header.get("name") == ":authority" and "exact_match" in header:
                    exact_match = header["exact_match"]

                    if "*" in exact_match:
                        # A real :authority header will never contain a "*", so if this route has an
                        # exact_match looking for one, then this route is unreachable.
                        hostglobs = set()
                        break # hostglobs is empty, no point in doing more work

                    elif any(hostglob_matches(glob, exact_match) for glob in hostglobs):
                        # The exact_match that this route is looking for is matched by one or more
                        # of the hostglobs; so this route is reachable (so far).  Set hostglobs to
                        # just match that route.  Because we already checked if the exact_match
                        # contains a "*", we don't need to worry about it possibly being interpreted
                        # incorrectly as a glob.
                        hostglobs = set([exact_match])
                        # Don't "break" here--if somehow this route has multiple disagreeing
                        # HeaderMatchers on :authority, then it's unreachable and we want the next
                        # iteration of the loop to trigger the "else" clause and prune hostglobs
                        # down to the empty set.

                    else:
                        # The exact_match that this route is looking for isn't matched by any of the
                        # hostglobs; so this route is unreachable.
                        hostglobs = set()
                        break # hostglobs is empty, no point in doing more work

        return hostglobs


    @classmethod
    def get_route(cls, config: 'V2Config', cache_key: str,
                  irgroup: IRHTTPMappingGroup, mapping: IRBaseMapping) -> 'V2Route':
        route: 'V2Route'

        cached_route = config.cache[cache_key]

        if cached_route is None:
            # Cache miss.
            # config.ir.logger.info(f"V2Route: cache miss for {cache_key}, synthesizing route")

            route = V2Route(config, irgroup, mapping)

            # Cheat a bit and force the route's cache_key.
            route.cache_key = cache_key

            config.cache.add(route)
            config.cache.link(irgroup, route)
        else:
            # Cache hit. We know a priori that it's a V2Route, but let's assert that
            # before casting.
            assert(isinstance(cached_route, V2Route))
            route = cached_route

            # config.ir.logger.info(f"V2Route: cache hit for {cache_key}")

        # One way or another, we have a route now.
        return route

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.routes = []

        for irgroup in config.ir.ordered_groups():
            if not isinstance(irgroup, IRHTTPMappingGroup):
                # We only want HTTP mapping groups here.
                continue

            if irgroup.get('host_redirect') is not None and len(irgroup.get('mappings', [])) == 0:
                # This is a host-redirect-only group, which is weird, but can happen. Do we
                # have a cached route for it?
                key = f"Route-{irgroup.group_id}-hostredirect"

                # Casting an empty dict to an IRBaseMapping may look weird, but in fact IRBaseMapping
                # is (ultimately) a subclass of dict, so it's the cleanest way to pass in a completely
                # empty IRBaseMapping to V2Route().
                #
                # (We could also have written V2Route to allow the mapping to be Optional, but that
                # makes a lot of its constructor much uglier.)
                route = config.save_element('route', irgroup, cls.get_route(config, key, irgroup, typecast(IRBaseMapping, {})))
                config.routes.append(route)

            # Repeat for our real mappings.
            for mapping in irgroup.mappings:
                key = f"Route-{irgroup.group_id}-{mapping.cache_key}"

                route = cls.get_route(config, key, irgroup, mapping)

                if not route.get('_failed', False):
                    config.routes.append(config.save_element('route', irgroup, route))

    @staticmethod
    def generate_headers(config: 'V2Config', mapping_group: IRHTTPMappingGroup) -> List[dict]:
        headers = []

        group_headers = mapping_group.get('headers', [])

        for group_header in group_headers:
            header = { 'name': group_header.get('name') }

            if group_header.get('regex'):
                header.update(regex_matcher(config, group_header.get('value'), key='regex_match'))
            else:
                header['exact_match'] = group_header.get('value')

            headers.append(header)

        return headers

    @staticmethod
    def generate_query_parameters(config: 'V2Config', mapping_group: IRHTTPMappingGroup) -> List[dict]:
        query_parameters = []

        group_query_parameters = mapping_group.get('query_parameters', [])

        for group_query_parameter in group_query_parameters:
            query_parameter = { 'name': group_query_parameter.get('name') }

            if group_query_parameter.get('regex'):
                query_parameter.update({
                    'string_match': regex_matcher(
                        config,
                        group_query_parameter.get('value'),
                        key='regex'
                    )
                })
            else:
                value = group_query_parameter.get('value', None)
                if value is not None:
                    query_parameter.update({
                        'string_match': {
                            'exact': group_query_parameter.get('value')
                        }
                    })
                else:
                    query_parameter.update({
                        'present_match': True
                    })

            query_parameters.append(query_parameter)

        return query_parameters

    @staticmethod
    def generate_hash_policy(mapping_group: IRHTTPMappingGroup) -> dict:
        hash_policy = {}
        load_balancer = mapping_group.get('load_balancer', None)
        if load_balancer is not None:
            lb_policy = load_balancer.get('policy')
            if lb_policy in ['ring_hash', 'maglev']:
                cookie = load_balancer.get('cookie')
                header = load_balancer.get('header')
                source_ip = load_balancer.get('source_ip')

                if cookie is not None:
                    hash_policy['cookie'] = {
                        'name': cookie.get('name')
                    }
                    if 'path' in cookie:
                        hash_policy['cookie']['path'] = cookie['path']
                    if 'ttl' in cookie:
                        hash_policy['cookie']['ttl'] = cookie['ttl']
                elif header is not None:
                    hash_policy['header'] = {
                        'header_name': header
                    }
                elif source_ip is not None:
                    hash_policy['connection_properties'] = {
                        'source_ip': source_ip
                    }

        return hash_policy

    @staticmethod
    def generate_headers_to_add(header_dict: dict) -> List[dict]:
        headers = []
        for k, v in header_dict.items():
                append = True
                if isinstance(v,dict):
                    if 'append' in v:
                        append = bool(v['append'])
                    headers.append({
                        'header': {
                            'key': k,
                            'value': v['value']
                        },
                        'append': append
                    })
                else:
                    headers.append({
                        'header': {
                            'key': k,
                            'value': v
                        },
                        'append': append  # Default append True, for backward compatability
                    })
        return headers

    @staticmethod
    def generate_regex_rewrite(config: 'V2Config', mapping_group: IRHTTPMappingGroup) -> dict:
        regex_rewrite = {}
        group_regex_rewrite = mapping_group.get('regex_rewrite', None)
        if group_regex_rewrite is not None:
            pattern = group_regex_rewrite.get('pattern', None)
            if (pattern is not None):
                regex_rewrite.update(regex_matcher(config, pattern, key='regex',safe_key='pattern', re_type='safe')) # regex_rewrite should never ever be unsafe
        substitution = group_regex_rewrite.get('substitution', None)
        if (substitution is not None):
            regex_rewrite["substitution"] = substitution
        return regex_rewrite
