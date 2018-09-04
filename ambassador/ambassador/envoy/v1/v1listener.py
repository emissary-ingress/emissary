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
from typing import cast as typecast

from ...ir.irlistener import IRListener
from ...ir.irmapping import IRMapping
from ...ir.irfilter import IRFilter

if TYPE_CHECKING:
    from . import V1Config


# XXX This is probably going to go away!
class V1Filter(dict):
    def __init__(self, filter: IRFilter) -> None:
        super().__init__()

        self['name'] = filter.name
        self['config'] = filter.config_dict()

        if filter.get('type', None):
           self['type'] = filter.type


class V1Listener(dict):
    def __init__(self, config: 'V1Config', listener: IRListener) -> None:
        super().__init__()

        self["address"] = "tcp://0.0.0.0:%d" % listener.service_port

        if listener.use_proxy_proto:
            self["use_proxy_proto"] = True

        if 'tls_context' in listener:
            ctx = listener.tls_context

            if ctx:
                lctx = {
                    "cert_chain_file": ctx.cert_chain_file,
                    "private_key_file": ctx.private_key_file
                }

                if "alpn_protocols" in ctx:
                    lctx["alpn_protocols"] = ctx["alpn_protocols"]

                if "cacert_chain_file" in ctx:
                    lctx["cacert_chain_file"] = ctx["cacert_chain_file"]

                if "cert_required" in ctx:
                    lctx["require_client_certificate"] = ctx["cert_required"]

        routes = self.get_routes(config, listener)

        vhosts = [
            {
                "name": "backend",
                "domains": [ "*" ],
                "routes": routes
            }
        ]

        if listener.get("require_tls", False):
            vhosts["require_ssl"] = "all"

        hcm_config = {
            "codec_type": "auto",
            "stat_prefix": "ingress_http",
            "use_remote_address": config.ir.ambassador_module.get('use_remote_address', False),
            "access_log": [
                {
                    "format": "ACCESS [%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\"\n",
                    "path": "/dev/fd/1"
                }
            ],
            "route_config": {
                "virtual_hosts": vhosts,
            },
            "filters": [ V1Filter(filter) for filter in config.ir.filters ]
        }

        if "tracing" in listener:
            hcm_config["tracing"] = {
                "generate_request_id": True,
                "tracing": {
                    "operation_name": "egress",
                    "request_headers_for_tags": []
                }
            }

        self["filters"] = [
            {
                "type": "read",
                "name": "http_connection_manager",
                "config": hcm_config,
            }
        ]

    def get_routes(self, config: 'V1Config', listener: 'IRListener') -> List[dict]:
        routes = []

        for group in reversed(sorted(config.ir.groups.values(), key=lambda x: x['group_weight'])):
            route = {
                "timeout_ms": group.get("timeout_ms", 3000),
            }

            if "prefix" in group:
                route["prefix"] = group.prefix

            if "regex" in group:
                route["regex"] = group.regex

            if "case_sensitive" in group:
                route["case_sensitive"] = group.case_sensitive

            if "cors" in group:
                route["cors"] = group.cors
            elif "cors_default" in group:
                route["cors"] = group.cors_default

            if "rate_limits" in group:
                route["rate_limits"] = group.rate_limits

            if "priority" in group:
                route["priority"] = group.priority

            if "use_websocket" in group:
                route["use_websocket"] = group.use_websocket

            if len(group.get('headers', [])) > 0:
                route["headers"] = group.headers
            # print(len(group.get('headers', [])) > 0)

            if group.get("host_redirect", None):
                route["host_redirect"] = typecast(IRMapping, group.host_redirect).service

                if group.get("path_redirect", None):
                    route["path_redirect"] = group.path_redirect
            else:
                if "rewrite" in group:
                    route["prefix_rewrite"] = group.rewrite

                if "host_rewrite" in group:
                    route["host_rewrite"] = group.host_rewrite

                if "auto_host_rewrite " in group:
                    route["auto_host_rewrite"] = group.auto_host_rewrite

                if "request_headers_to_add" in group:
                    route["request_headers_to_add"] = group.request_headers_to_add

                if "use_websocket" in group:
                    route["cluster"] = group.mappings[0].cluster.name
                else:
                    route["weighted_clusters"] = {
                        "clusters": [ {
                                "name": mapping.cluster.name,
                                "weight": mapping.weight
                            } for mapping in group.mappings
                        ]
                    }
                    # print("WEIGHTED_CLUSTERS %s" % route["weighted_clusters"])

                if group.get("shadows", []):
                    route["shadow"] = {
                        "cluster": group.shadows[0].name
                    }

            if "envoy_override" in group:
                for key in group.envoy_override.keys():
                    route[key] = group.envoy_override[key]

            routes.append(route)

        return routes

    @classmethod
    def generate(cls, config: 'V1Config') -> List['V1Listener']:
        listeners: List['V1Listener'] = []

        for listener in config.ir.listeners:
            listeners.append(V1Listener(config, listener))

        return listeners
