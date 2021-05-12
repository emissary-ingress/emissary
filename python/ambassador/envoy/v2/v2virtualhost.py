# Copyright 2021 Datawire. All rights reserved.
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

from typing import Any, Dict, List, Optional, TYPE_CHECKING

import logging

from ...utils import dump_json

from .v2route import V2Route, DictifiedV2Route, v2prettyroute
from .v2tls import V2TLSContext

if TYPE_CHECKING:
    from ...ir.irtlscontext import IRTLSContext
    from .v2listener import V2Listener
    from . import V2Config


def jsonify(x) -> str:
    return dump_json(x, pretty=True)


class V2VirtualHost:
    def __init__(self, config: 'V2Config', listener: 'V2Listener',
                 name: str, hostname: str, ctx: Optional['IRTLSContext'],
                 secure: bool, action: Optional[str], insecure_action: Optional[str]) -> None:
        super().__init__()

        self._config = config
        self._listener = listener
        self._name = name
        self._hostname = hostname
        self._ctx = ctx
        self._secure = secure
        self._action = action
        self._insecure_action = insecure_action
        self.tls_context = V2TLSContext(ctx)
        self.routes: List[DictifiedV2Route] = []

    def finalize(self) -> None:
        # It's important from a performance perspective to wrap debug log statements
        # with this check so we don't end up generating log strings (or even JSON
        # representations) that won't get logged anyway.
        log_debug = self._config.ir.logger.isEnabledFor(logging.DEBUG)

        # Even though this is called V2VirtualHost, we track the filter_chain_match here,
        # because it makes more sense, because this is where we have the domain information.
        # The 1:1 correspondence that this implies between filters and domains may need to
        # change later, of course...
        if log_debug:
            self._config.ir.logger.debug(f"V2VirtualHost finalize {jsonify(self.pretty())}")

        match: Dict[str, Any] = {}

        if self._ctx:
            match["transport_protocol"] = "tls"

        # Make sure we include a server name match if the hostname isn't "*".
        if self._hostname and (self._hostname != '*'):
                match["server_names"] = [ self._hostname ]

        self.filter_chain_match = match

        # If we're on Edge Stack and we're not an intercept agent, punch a hole for ACME
        # challenges, for every listener.
        if self._config.ir.edge_stack_allowed and not self._config.ir.agent_active:
            found_acme = False

            for route in self.routes:
                if route["match"].get("prefix", None) == "/.well-known/acme-challenge/":
                    found_acme = True
                    break

            if not found_acme:
                # The target cluster doesn't actually matter -- the auth service grabs the
                # challenge and does the right thing. But we do need a cluster that actually
                # exists, so use the sidecar cluster.

                if not self._config.ir.sidecar_cluster_name:
                    # Uh whut? how is Edge Stack running exactly?
                    raise Exception("Edge Stack claims to be running, but we have no sidecar cluster??")

                if log_debug:
                    self._config.ir.logger.debug(f"V2VirtualHost finalize punching a hole for ACME")

                self.routes.insert(0, {
                    "match": {
                        "case_sensitive": True,
                        "prefix": "/.well-known/acme-challenge/"
                    },
                    "route": {
                        "cluster": self._config.ir.sidecar_cluster_name,
                        "prefix_rewrite": "/.well-known/acme-challenge/",
                        "timeout": "3.000s"
                    }
                })

        if log_debug:
            for route in self.routes:
                self._config.ir.logger.debug(f"VHost Route {v2prettyroute(route)}")

    def pretty(self) -> str:
        ctx_name = "-none-"

        if self.tls_context:
            ctx_name = self.tls_context.pretty()

        route_count = len(self.routes)
        route_plural = "" if (route_count == 1) else "s"

        return "<VHost %s ctx %s a %s ia %s %d route%s>" % \
               (self._hostname, ctx_name, self._action, self._insecure_action,
                route_count, route_plural)

    def verbose_dict(self) -> dict:
        return {
            "_name": self._name,
            "_hostname": self._hostname,
            "_secure": self._secure,
            "_action": self._action,
            "_insecure_action": self._insecure_action,
            "tls_context": self.tls_context,
            "routes": self.routes,
        }
