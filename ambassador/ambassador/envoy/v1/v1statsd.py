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
    from . import V1Config


class V1Statsd(dict):
    def __init__(self, config: 'V1Config') -> None:
        super().__init__()

        statsd = config.ir.ambassador_module.get('statsd', {})

        if statsd and ('enabled' in statsd):
            self['enabled'] = statsd['enabled']
        else:
            self['enabled'] = False

    @classmethod
    def generate(self, config: 'V1Config') -> None:
        # Save this under the Ambassador module as well.
        config.statsd = config.save_element('statsd', config.ir.ambassador_module, V1Statsd(config))
