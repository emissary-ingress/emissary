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

from typing import List, Tuple, TYPE_CHECKING

from ..common import EnvoyRoute
from ...ir import IRResource
from ...ir.irmapping import IRMappingGroup

# from .v2ratelimit import V2RateLimits

if TYPE_CHECKING:
    from . import V2Config


class V2Route(dict):
    def __init__(self, config: 'V2Config', group: IRMappingGroup) -> None:
        super().__init__()

        envoy_route = EnvoyRoute(group).envoy_route

        match = {
            envoy_route: group.get('prefix'),
            'case_sensitive': group.get('case_sensitive', True),
            'headers': group.get('headers') if len(group.get('headers', [])) > 0 else None
        }

        route = {
            'priority': group.get('priority'),
            'use_websocket': group.get('use_websocket'),
            'weighted_clusters': {
                'clusters': [
                    {
                        'name': mapping.cluster.name,
                        'weight': mapping.weight,
                        'request_headers_to_add': group.get('request_headers_to_add')
                    } for mapping in group.mappings
                ],
            },
            'prefix_rewrite': group.get('rewrite'),
        }

        if 'host_rewrite' in group:
            route['host_rewrite'] = group['host_rewrite']

        if 'auto_host_rewrite' in group:
            route['auto_host_rewrite'] = group['auto_host_rewrite']

        self['match'] = match
        self['route'] = route

        request_headers_to_add = []
        for mapping in group.mappings:
            for k, v in mapping.get('add_request_headers', {}).items():
                request_headers_to_add.append({
                    'header': {'key': k, 'value': v},
                    'append': True, # ???
                    })
        if request_headers_to_add:
            self['request_headers_to_add'] = request_headers_to_add

        host_redirect = group.get('host_redirect', None)

        if host_redirect:
            self['redirect']['host_redirect'] = host_redirect.service
            self['redirect']['path_redirect'] = host_redirect.path_redirect

        cors = None

        if "cors" in group:
            cors = group.cors.as_dict()
        elif "cors" in config.ir.ambassador_module:
            cors = config.ir.ambassador_module.cors.as_dict()

        if cors:
            for key in [ "_active", "_referenced_by", "_rkey", "kind", "location", "name" ]:
                cors.pop(key, None)

            self['cors'] = cors

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.routes = []

        for irgroup in config.ir.ordered_groups():
            route = config.save_element('route', irgroup, V2Route(config, irgroup))
            config.routes.append(route)
