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

from typing import List, TYPE_CHECKING

from ..common import EnvoyRoute
from ...ir.irhttpmappinggroup import IRHTTPMappingGroup

from .v2ratelimitaction import V2RateLimitAction

if TYPE_CHECKING:
    from . import V2Config


class V2Route(dict):
    def __init__(self, config: 'V2Config', group: IRHTTPMappingGroup) -> None:
        super().__init__()

        envoy_route = EnvoyRoute(group).envoy_route

        match = {
            envoy_route: group.get('prefix'),
            'case_sensitive': group.get('case_sensitive', True),
        }

        headers = self.generate_headers(group)

        if len(headers) > 0:
            match['headers'] = headers

        self['match'] = match

        # `per_filter_config` is used for customization of an Envoy filter
        per_filter_config = {}

        if group.get('bypass_auth', False):
            per_filter_config['envoy.ext_authz'] = {'disabled': True}

        if per_filter_config:
            self['per_filter_config'] = per_filter_config

        request_headers_to_add = group.get('add_request_headers', None)

        if request_headers_to_add:
            self['request_headers_to_add'] = [
                {
                    'header': {
                        'key': k,
                        'value': v
                    },
                    'append': True  # ???
                } for k, v in request_headers_to_add.items()
            ]

        response_headers_to_add = group.get('add_response_headers', None)

        if response_headers_to_add:
            self['response_headers_to_add'] = [
                {
                    'header': {
                        'key': k,
                        'value': v
                    },
                    'append': True  # ???
                } for k, v in response_headers_to_add.items()
            ]

        # If a host_redirect is set, we won't do a 'route' entry.
        host_redirect = group.get('host_redirect', None)

        if host_redirect:
            # We have a host_redirect. Deal with it.
            self['redirect'] = {
                'host_redirect': host_redirect.service
            }

            path_redirect = host_redirect.get('path_redirect', None)

            if path_redirect:
                self['redirect']['path_redirect'] = path_redirect
        else:
            # No host_redirect. Do route stuff.
            clusters = [ {
                'name': mapping.cluster.name,
                'weight': mapping.weight
            } for mapping in group.mappings ]

            route = {
                'priority': group.get('priority'),
                'timeout': "%0.3fs" % (group.get('timeout_ms', 3000) / 1000.0),
                'weighted_clusters': {
                    'clusters': clusters
                }
            }

            if group.get('rewrite', None):
                route['prefix_rewrite'] = group['rewrite']

            if 'host_rewrite' in group:
                route['host_rewrite'] = group['host_rewrite']

            if 'auto_host_rewrite' in group:
                route['auto_host_rewrite'] = group['auto_host_rewrite']

            hash_policy = self.generate_hash_policy(group)
            if len(hash_policy) > 0:
                route['hash_policy'] = [ hash_policy ]

            cors = None

            if "cors" in group:
                cors = group.cors.as_dict()
            elif "cors" in config.ir.ambassador_module:
                cors = config.ir.ambassador_module.cors.as_dict()

            if cors:
                for key in [ "_active", "_errored", "_referenced_by", "_rkey", "kind", "location", "name" ]:
                    cors.pop(key, None)

                route['cors'] = cors

            # Is shadowing enabled?
            shadow = group.get("shadows", None)

            if shadow:
                shadow = shadow[0]

                route['request_mirror_policy'] = {
                    'cluster': shadow.cluster.name
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

            self['route'] = route

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.routes = []

        for irgroup in config.ir.ordered_groups():
            if not isinstance(irgroup, IRHTTPMappingGroup):
                # We only want HTTP mapping groups here.
                continue

            # It's an HTTP group. Great.
            route = config.save_element('route', irgroup, V2Route(config, irgroup))

            if irgroup.get('sni'):
                info = {
                    'hosts': irgroup['tls_context']['hosts'],
                    'secret_info': irgroup['tls_context']['secret_info']
                }
                config.sni_routes.append({'route': route, 'info': info})
            else:
                config.routes.append(route)

    @staticmethod
    def generate_headers(mapping_group: IRHTTPMappingGroup) -> List[dict]:
        headers = []

        group_headers = mapping_group.get('headers', [])

        for group_header in group_headers:
            header = { 'name': group_header.get('name') }

            if group_header.get('regex'):
                header['regex_match'] = group_header.get('value')
            else:
                header['exact_match'] = group_header.get('value')

            headers.append(header)

        return headers

    @staticmethod
    def generate_hash_policy(mapping_group: IRHTTPMappingGroup) -> dict:
        hash_policy = {}
        load_balancer = mapping_group.get('load_balancer', None)
        if load_balancer is not None:
            lb_policy = load_balancer.get('policy')
            if lb_policy == 'ring_hash':
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
