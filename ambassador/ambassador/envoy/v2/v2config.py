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

from typing import Any, Dict, List, Optional, Tuple, Union

import json

from ...ir import IR

from ..common import EnvoyConfig, sanitize_pre_json
from .v2admin import V2Admin
from .v2bootstrap import V2Bootstrap
from .v2route import V2Route
from .v2listener import V2Listener
from .v2cluster import V2Cluster
from .v2_static_resources import V2StaticResources
from .v2tracing import V2Tracing
from .v2ratelimit import V2RateLimit


# #############################################################################
# ## v2config.py -- the Envoy V2 configuration engine
#
#
class V2Config (EnvoyConfig):
    admin: V2Admin
    tracing: Optional[V2Tracing]
    ratelimit: Optional[V2RateLimit]
    bootstrap: V2Bootstrap
    routes: List[V2Route]
    listeners: List[V2Listener]
    clusters: List[V2Cluster]
    static_resources: V2StaticResources

    def __init__(self, ir: IR) -> None:
        super().__init__(ir)
        V2Admin.generate(self)
        V2Tracing.generate(self)

        self.sni_routes = []
        V2RateLimit.generate(self)
        V2Route.generate(self)
        V2Listener.generate(self)
        V2Cluster.generate(self)
        V2StaticResources.generate(self)
        V2Bootstrap.generate(self)

#         # print("v1.admin %s" % self.admin)
#
#         self.listeners: List[V1Listener] = V1Listener.generate(self)
#
#         # print("v1.listeners %s" % self.listeners)
#
#         self.clustermgr: V1ClusterManager = V1ClusterManager.generate(self)
#
#         # print("v1.clustermgr %s" % self.clustermgr)

    def as_dict(self):
        d = {
            'bootstrap': self.bootstrap,
            'static_resources': self.static_resources
        }

        return d

    def as_json(self):
        return json.dumps(sanitize_pre_json(self.as_dict()), sort_keys=True, indent=4)
