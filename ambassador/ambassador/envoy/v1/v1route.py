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

from .v1ratelimit import V1RateLimits

if TYPE_CHECKING:
    from . import V1Config


class V1Route(dict):
    def __init__(self, config: 'V1Config', group: IRMappingGroup) -> None:
        super().__init__()

        self["timeout_ms"] = group.get("timeout_ms", 3000)

        envoy_route = EnvoyRoute(group).envoy_route
        self[envoy_route] = group.get('prefix')

        if "regex" in group:
            self["regex"] = group.regex

        if "case_sensitive" in group:
            self["case_sensitive"] = group.case_sensitive

        cors = None

        if "cors" in group:
            cors = group.cors.as_dict()
        elif "cors" in config.ir.ambassador_module:
            cors = config.ir.ambassador_module.cors.as_dict()

        if cors:
            for key in [ "_active", "_referenced_by", "_rkey", "kind", "location", "name" ]:
                cors.pop(key, None)

            self['cors'] = cors

        if "rate_limits" in group:
            self["rate_limits"] = V1RateLimits(group.rate_limits)

        if "priority" in group:
            self["priority"] = group.priority

        if "use_websocket" in group:
            self["use_websocket"] = group.use_websocket

        if len(group.get('headers', [])) > 0:
            self["headers"] = group.headers
        # print(len(group.get('headers', [])) > 0)

        if group.get("host_redirect", None):
            self["host_redirect"] = group.host_redirect.service

            if "path_redirect" in group.host_redirect:
                self["path_redirect"] = group.host_redirect.path_redirect
        else:
            # Don't include prefix_rewrite unless group.rewrite is present and not
            # empty. The special handling is so that using rewrite: "" in an
            # Ambassador mapping doesn't rewrite the path at all, which can be
            # important in regex mappings.
            rewrite = group.get("rewrite", None)

            if rewrite:
                self["prefix_rewrite"] = rewrite

            if "host_rewrite" in group:
                self["host_rewrite"] = group.host_rewrite

            if "auto_host_rewrite" in group:
                self["auto_host_rewrite"] = group.auto_host_rewrite

            if "add_request_headers" in group:
                self["request_headers_to_add"] = [
                    {"key": k, "value": v}
                    for k, v in group.add_request_headers.items()
                ]

            if "use_websocket" in group:
                self["cluster"] = group.mappings[0].cluster.name
            else:
                self["weighted_clusters"] = {
                    "clusters": [ {
                            "name": mapping.cluster.name,
                            "weight": mapping.weight
                        } for mapping in sorted(group.mappings, key=lambda x: x.name)
                    ]
                }
                # print("WEIGHTED_CLUSTERS %s" % route["weighted_clusters"])

            if group.get("shadows", []):
                self["shadow"] = {
                    "cluster": group.shadows[0].cluster.name
                }

        if "envoy_override" in group:
            for key in group.envoy_override.keys():
                self[key] = group.envoy_override[key]

    @classmethod
    def generate(cls, config: 'V1Config') -> None:
        config.routes = []

        for irgroup in config.ir.ordered_groups():
            route = config.save_element('route', irgroup, V1Route(config, irgroup))
            config.routes.append(route)
