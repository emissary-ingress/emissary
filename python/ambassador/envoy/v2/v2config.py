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

        V2RateLimit.generate(self)
        V2Route.generate(self)
        V2Listener.generate(self)
        V2Cluster.generate(self)
        V2StaticResources.generate(self)
        V2Bootstrap.generate(self)

    def as_dict(self) -> Dict[str, Any]:
        bootstrap_config, ads_config = self.split_config()

        d = {
            'bootstrap': bootstrap_config,
            **ads_config
        }

        return d

    def split_config(self) -> Tuple[Dict[str, Any], Dict[str, Any]]:
        ads_config = {
            '@type': '/envoy.config.bootstrap.v2.Bootstrap',
            'static_resources': self.static_resources,
            'layered_runtime': {
                'layers': [
                    {
                        'name': 'static_layer',
                        'static_layer': {
                            # For now, we enable the deprecated & disallowed_by_default "HTTP_JSON_V1" Zipkin
                            # collector_endpoint_version because it repesents the Zipkin v1 API, while the
                            # non-deprecated options HTTP_JSON and HTTP_PROTO are the Zipkin v2 API; switching
                            # top one of them would change how Envoy talks to the outside world.
                            'envoy.deprecated_features:envoy.config.trace.v2.ZipkinConfig.HTTP_JSON_V1': True,
                            # Give our users more time to migrate to v2; we've said that we'll continue
                            # supporting both for a while even after we change the default.
                            'envoy.deprecated_features:envoy.config.filter.http.ext_authz.v2.ExtAuthz.use_alpha': True,
                            # We haven't yet told users that we'll be deprecating `regex_type: unsafe`.
                            'envoy.deprecated_features:envoy.api.v2.route.RouteMatch.regex': True,         # HTTP path
                            'envoy.deprecated_features:envoy.api.v2.route.HeaderMatcher.regex_match': True # HTTP header
                        }
                    }
                ]
            }
        }

        bootstrap_config = dict(self.bootstrap)

        return bootstrap_config, ads_config

