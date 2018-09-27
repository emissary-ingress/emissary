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

from typing import Dict, TYPE_CHECKING

from ...ir.ircluster import IRCluster

if TYPE_CHECKING:
    from . import V1Config


class V1GRPCService(dict):
    def __init__(self, config: 'V1Config', cluster: IRCluster) -> None:
        super().__init__()

        self['config'] = {
            'cluster_name': cluster.name
        }

        self['type'] = 'grpc_service'

    @classmethod
    def generate(self, config: 'V1Config') -> None:
        config.grpc_services = {}

        for svc, cluster in config.ir.grpc_services.items():
            config.grpc_services[svc] = config.save_element('grpc_service', cluster, V1GRPCService(config, cluster))
