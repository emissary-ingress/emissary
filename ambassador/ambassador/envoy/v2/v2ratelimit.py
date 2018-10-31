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

from typing import Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from . import V2Config


class V2RateLimit(dict):
    def __init__(self, config: 'V2Config') -> None:
        super().__init__()

        self['use_data_plane_proto'] = config.ir.ratelimit.data_plane_proto
        self['grpc_service'] = {
            'envoy_grpc': {
                'cluster_name': config.ir.ratelimit.cluster.name
            },
            'timeout': "5s"
        }

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.ratelimit = None

        if config.ir.ratelimit:
            config.ratelimit = config.save_element('ratelimit', config.ir.ratelimit, V2RateLimit(config))
