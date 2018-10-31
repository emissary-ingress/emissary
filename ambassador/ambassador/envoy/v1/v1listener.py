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

import json

from ..common import EnvoyRoute
from ...ir.irlistener import IRListener
# from ...ir.irmapping import IRMapping
from ...ir.irfilter import IRFilter

from .v1tls import V1TLSContext
from .v1ratelimitaction import V1RateLimitAction

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

        if 'tls_contexts' in listener:
            envoy_ctx = V1TLSContext()

            for ctx_name, ctx in listener.tls_contexts.items():
                envoy_ctx.add_context(ctx)

            if envoy_ctx:
                self['ssl_context'] = dict(envoy_ctx)

        vhost = {
            "name": "backend",
            "domains": [ "*" ],
            "routes": config.routes
        }

        if listener.get("require_tls", False):
            vhost["require_ssl"] = "all"

        hcm_config = {
            "codec_type": "auto",
            "stat_prefix": "ingress_http",
            "access_log": [
                {
                    "format": "ACCESS [%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(USER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\"\n",
                    "path": "/dev/fd/1"
                }
            ],
            "route_config": {
                "virtual_hosts": [ vhost ]
            },
            "filters": [ V1Filter(filter) for filter in config.ir.filters ]
        }

        if 'use_remote_address' in config.ir.ambassador_module:
            hcm_config["use_remote_address"] = config.ir.ambassador_module.use_remote_address

        if config.ir.tracing:
            hcm_config["generate_request_id"] = True
            hcm_config["tracing"] = {
                "operation_name": "egress",
                "request_headers_for_tags": config.ir.tracing.get('tag_headers', [])
            }

        self["filters"] = [
            {
                "type": "read",
                "name": "http_connection_manager",
                "config": hcm_config,
            }
        ]

    @classmethod
    def generate(cls, config: 'V1Config') -> None:
        config.listeners = []

        for irlistener in config.ir.listeners:
            listener = config.save_element('listener', irlistener, V1Listener(config, irlistener))
            config.listeners.append(listener)
