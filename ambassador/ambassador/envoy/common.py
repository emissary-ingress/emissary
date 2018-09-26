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

from typing import Any, Dict, Optional

from abc import abstractmethod

from ..ir import IR, IRResource
from ..ir.irmapping import IRMappingGroup

class EnvoyConfig:
    """
    Base class for Envoy configuration that permits fetching configuration
    for various elements to show in diagnostics.
    """

    ir: IR
    elements: Dict[str, Dict[str, Any]]

    def __init__(self, ir: IR) -> None:
        self.ir = ir
        self.elements = {}

    def add_element(self, kind: str, key: str, obj: Any) -> None:
        eldict = self.elements.setdefault(kind, {})
        eldict[key] = obj

    def get_element(self, kind: str, key: str, default: Any) -> Optional[Any]:
        eldict = self.elements.get(kind, {})
        return eldict.get(key, default)

    def pop_element(self, kind: str, key: str, default: Any) -> Optional[Any]:
        eldict = self.elements.get(kind, {})
        return eldict.pop(key, default)

    def save_element(self, kind: str, resource: IRResource, obj: Any):
        self.add_element(kind, resource.rkey, obj)
        self.add_element(kind, resource.location, obj)
        return obj

    @classmethod
    def generate(cls, ir: IR, version: str="V2") -> 'EnvoyConfig':
        if version == "V1":
            from . import V1Config
            return V1Config(ir)
        else:
            from . import V2Config
            return V2Config(ir)

class EnvoyRoute:
    def __init__(self, group: IRMappingGroup):
        self.prefix = 'prefix'
        self.path = 'path'
        self.regex = 'regex'
        self.envoy_route = self._get_envoy_route(group)

    def _get_envoy_route(self, group: IRMappingGroup) -> str:
        if group.get('prefix_regex', False):
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