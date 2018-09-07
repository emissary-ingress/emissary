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

from ..ir.irmapping import IRMapping


class EnvoyRoute:
    def __init__(self, mapping: IRMapping):
        self.prefix = 'prefix'
        self.path = 'path'
        self.regex = 'regex'
        self.envoy_route = self._get_envoy_route(mapping)

    def _get_envoy_route(self, mapping: IRMapping) -> str:
        if mapping.get('prefix_regex', False):
            return self.regex
        else:
            return self.prefix


def sanitize_pre_json(input):
    # Removes all potential null values
    if isinstance(input, dict):
        for key, value in list(input.items()):
            if value is None:
                del input[key]
            else:
                sanitize_pre_json(value)
    elif isinstance(input, list):
        for item in input:
            sanitize_pre_json(item)
    return input