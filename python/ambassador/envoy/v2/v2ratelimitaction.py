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

from typing import Any, ClassVar, Dict, List, TYPE_CHECKING

# from ...utils import RichStatus

if TYPE_CHECKING:
    from . import V2Config


class V2RateLimitAction(dict):
    already_errored: ClassVar[bool] = False

    def __init__(self, config: 'V2Config', rate_limit: Dict[str, Any]) -> None:
        super().__init__()

        self.valid = False
        self.stage = 0
        self.actions: List[dict] = []

        # if rate_limit == {}:
        #     rate_limit = []

        config.ir.logger.debug("V2RateLimitAction translating %s" % rate_limit)

        lkeys = rate_limit.keys()
        if len(lkeys) > 1:
            # "Impossible". This should've been caught earlier.
            config.ir.post_error("Label for RateLimit has multiple entries instead of just one: %s" % rate_limit)
            return

        lkey = list(lkeys)[0]
        actions = rate_limit[lkey]

        for action in actions:
            config.ir.logger.debug("V2RateLimitAction working on '%s'" % action)

            if ((action == "source_cluster") or
                (action == "destination_cluster") or
                (action == "remote_address")):
                self.save_action({ action: {} })
            elif isinstance(action, dict):
                # This should be a dict with a single key.
                keylist = list(action.keys())

                if len(keylist) != 1:
                    config.ir.post_error("Label for RateLimit has invalid custom header '%s' (%s)" %
                                         (action, rate_limit))
                    continue

                dkey = keylist[0]

                if dkey == 'generic_key':
                    self.save_action({
                        'generic_key': {
                            'descriptor_value': action[dkey]
                        }
                    })
                else:
                    # This is a header block.
                    hdr_action = action[dkey]

                    hdr_name = hdr_action['header']
                    # hdr_omit = hdr_action.get('omit_if_not_present', False)

                    self.save_action({
                        'request_headers': {
                            'header_name': hdr_name,
                            'descriptor_key': dkey
                        }
                    })

                    ### This whole bit doesn't work with the existing RateLimit filter. We're
                    ### going to have to tweak it to allow request_headers with a default value.
                    # if not hdr_omit:
                    #     if 'default' not in hdr_action:
                    #         config.ir.logger.error("V2RateLimitAction '%s' is missing a default value" % rate_limit)
                    #     else:
                    #         hdr_default = hdr_action['default']
                    #
                    #         self.save_action({
                    #             'header_value_match': {
                    #                 'headers': [{
                    #                     'name': hdr_name,
                    #                     'present_match': True
                    #                 }],
                    #                 'expect_match': False,
                    #                 'descriptor_value': hdr_default
                    #             }
                    #         })
            elif isinstance(action, str):
                # This is a shorthand for a generic_key.
                self.save_action({
                    'generic_key': {
                        'descriptor_value': action
                    }
                })
            else:
                # WTF.
                config.ir.post_error("Label for RateLimit is not valid: %s" % action)

    def save_action(self, action):
        self.actions.append(action)
        self.valid = True

    def to_dict(self):
        return {
            'stage': self.stage,
            'actions': self.actions
        }
