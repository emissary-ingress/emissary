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

import json
from typing import TYPE_CHECKING, Any, Dict, List, Optional, Tuple, Union

from ...cache import Cache, NullCache
from ..common import EnvoyConfig, sanitize_pre_json
from .v3_static_resources import V3StaticResources
from .v3admin import V3Admin
from .v3bootstrap import V3Bootstrap
from .v3cluster import V3Cluster
from .v3listener import V3Listener
from .v3ratelimit import V3RateLimit
from .v3route import V3Route, V3RouteVariants
from .v3tracing import V3Tracing

if TYPE_CHECKING:
    from ...ir import IR  # pragma: no cover
    from ...ir.irserviceresolver import ClustermapEntry  # pragma: no cover


# #############################################################################
# ## v3config.py -- the Envoy V3 configuration engine
#
#
class V3Config(EnvoyConfig):
    admin: V3Admin
    tracing: Optional[V3Tracing]
    ratelimit: Optional[V3RateLimit]
    bootstrap: V3Bootstrap
    routes: List[V3Route]
    route_variants: List[V3RouteVariants]
    listeners: List[V3Listener]
    clusters: List[V3Cluster]
    static_resources: V3StaticResources
    clustermap: Dict[str, Any]

    def __init__(self, ir: "IR", cache: Optional[Cache] = None) -> None:
        ir.logger.info("EnvoyConfig: Generating V3")

        # Init our superclass...
        super().__init__(ir)

        # ...then make sure we have a cache (which might be a NullCache).
        self.cache = cache or NullCache(self.ir.logger)

        V3Admin.generate(self)
        V3Tracing.generate(self)

        V3RateLimit.generate(self)
        V3Route.generate(self)
        V3Listener.generate(self)
        V3Cluster.generate(self)
        V3StaticResources.generate(self)
        V3Bootstrap.generate(self)

    def has_listeners(self) -> bool:
        return len(self.listeners) > 0

    def as_dict(self) -> Dict[str, Any]:
        bootstrap_config, ads_config, clustermap = self.split_config()

        d = {"bootstrap": bootstrap_config, "clustermap": clustermap, **ads_config}

        return d

    def split_config(self) -> Tuple[Dict[str, Any], Dict[str, Any], Dict[str, "ClustermapEntry"]]:
        ads_config = {
            "@type": "/envoy.config.bootstrap.v3.Bootstrap",
            "static_resources": self.static_resources,
            "layered_runtime": {
                "layers": [
                    {
                        "name": "static_layer",
                        "static_layer": {
                            "envoy.reloadable_features.enable_deprecated_v2_api": True,
                            "envoy.deprecated_features:envoy.config.trace.v3.ZipkinConfig.hidden_envoy_deprecated_HTTP_JSON_V1": True,
                            "re2.max_program_size.error_level": 200,
                        },
                    }
                ]
            },
        }

        bootstrap_config = dict(self.bootstrap)

        return bootstrap_config, ads_config, self.clustermap
