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

from typing import Any, ClassVar, Dict, List, Optional, Tuple, Union

import json
import logging

from ...ir import IR
from ..common import EnvoyConfig

from .v1admin import V1Admin
from .v1statsd import V1Statsd
from .v1route import V1Route
from .v1listener import V1Listener
from .v1cluster import V1Cluster
from .v1clustermanager import V1ClusterManager
from .v1tracing import V1Tracing
from .v1grpcservice import V1GRPCService

#############################################################################
## v1config.py -- the Envoy V1 configuration engine


class V1Config (EnvoyConfig):
    admin: V1Admin
    statsd: V1Statsd
    routes: List[V1Route]
    listeners: List[V1Listener]
    clusters: List[V1Cluster]
    clustermgr: V1ClusterManager
    tracing: Optional[V1Tracing]
    grpc_services: Optional[Dict[str, V1GRPCService]]

    def __init__(self, ir: IR) -> None:
        super().__init__(ir)

        V1Admin.generate(self)
        V1Statsd.generate(self)
        V1Route.generate(self)
        V1Listener.generate(self)
        V1Cluster.generate(self)
        V1ClusterManager.generate(self)
        V1Tracing.generate(self)
        V1GRPCService.generate(self)

    def as_dict(self):
        d = {
            'admin': self.admin,
            'listeners': self.listeners,
            'cluster_manager': self.clustermgr,
        }

        if self.tracing:
            d['tracing'] = self.tracing

        for svc_name in sorted(self.grpc_services.keys()):
            d[svc_name] = dict(self.grpc_services[svc_name])

        if self.statsd and self.statsd.get('enabled', False):
            d['stats_flush_interval_ms'] = 1000
            d['statsd_udp_ip_address'] = '127.0.0.1:8125'

        return d

    def as_json(self):
        return json.dumps(self.as_dict(), sort_keys=True, indent=4)
