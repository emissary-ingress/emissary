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

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from . import V2Config # pragma: no cover


class V2StaticResources(dict):
    def __init__(self, config: 'V2Config') -> None:
        super().__init__()

        self.update({
            'listeners': [ l.as_dict() for l in config.listeners ],
            'clusters': config.clusters,
        })

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        # We needn't use config.save_element here -- this is just a wrapper element.
        config.static_resources = V2StaticResources(config)
