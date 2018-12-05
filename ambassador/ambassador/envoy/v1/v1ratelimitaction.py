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

from typing import Any, List, Dict, TYPE_CHECKING

from ...utils import RichStatus

if TYPE_CHECKING:
    from . import V1Config


class V1RateLimitAction(dict):
    def __init__(self, config: 'V1Config', rate_limit: Dict[str, Any]) -> None:
        super().__init__()

        self.valid = False
        self.stage = 0
        self.actions: List[Any] = []

        if rate_limit == []:
            # If empty, we can be done.
            return

        config.ir.logger.debug("V1RateLimitAction translating %s" % rate_limit)

        lkeys = rate_limit.keys()
        if len(lkeys) > 1:
            # "Impossible". This should've been caught earlier.
            err = RichStatus.fromError("ratelimit has multiple entries (%s) instead of just one" %
                                       lkeys)
            config.ir.aconf.post_error(err)
            return

        lkey = list(lkeys)[0]

        if not lkey.startswith('v0_ratelimit_'):
            # This isn't from a V0 rate_limit element. Skip it.
            return

        actions = rate_limit[lkey]

        for action in actions:
            config.ir.logger.debug("V1RateLimitAction working on '%s'" % action)

            if ((action == "source_cluster") or
                (action == "destination_cluster") or
                (action == "remote_address")):
                self.save_action({ 'type': action })
            elif isinstance(action, dict):
                # This should be a dict with a single key.
                keylist = list(action.keys())

                if len(keylist) != 1:
                    config.ir.logger.error("V1RateLimitAction '%s' has invalid custom header '%s'" % (rate_limit, action))
                    continue

                dkey = keylist[0]

                if dkey == 'generic_key':
                    self.save_action({ 'type': 'generic_key',
                                       'descriptor_value': action[dkey] })
                else:
                    # This is a header block.
                    hdr_action = action[dkey]
                    hdr_name = hdr_action['header']

                    self.save_action({ 'type': 'request_headers',
                                       'header_name': hdr_name,
                                       'descriptor_key': dkey })
            elif isinstance(action, str):
                # This is a shorthand for a generic_key.
                self.save_action({ 'type': 'generic_key',
                                   'descriptor_value': action })
            else:
                # WTF.
                config.ir.logger.error("V1RateLimitAction: invalid action '%s'" % action)

    def save_action(self, action):
        self.actions.append(action)
        self.valid = True

    def to_dict(self):
        return {
            'stage': self.stage,
            'actions': self.actions
        }
