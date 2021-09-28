# Copyright 2021 Datawire. All rights reserved.
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

from typing import Any, Dict

import logging

################
## hostglob_matches is a utility for host globbing.

def hostglob_matches(glob: str, value: str) -> bool:
    """
    Does a host glob match a given value?
    """

    rc = False

    if ('*' in value) and not ('*' in glob):
        # Swap.
        tmp = value
        value = glob
        glob = tmp

    if glob == "*": # special wildcard
        rc=True
    elif glob.endswith("*"): # prefix match
        rc=value.startswith(glob[:-1])
    elif glob.startswith("*"): # suffix match
        rc=value.endswith(glob[1:])
    else: # exact match
        rc=value == glob

    # sys.stderr.write(f"hostglob_matches: {value} gl~ {glob} == {rc}\n")
    return rc

################
## selector_matches is a utility for doing K8s label selector matching.

def selector_matches(logger: logging.Logger, selector: Dict[str, Any], labels: Dict[str, str]) -> bool:
    match: Dict[str, str] = selector.get("matchLabels") or {}

    if not match:
        # If there's no matchLabels to match, return True.
        logger.debug("    no matchLabels in selector => True")
        return True

    # If we have stuff to match on, but no labels to actually match them, we
    # can short-circuit (and skip a weirder conditional down in the loop).
    if not labels:
        logger.debug("    no incoming labels => False")
        return False

    selmatch = False

    for k, v in match.items():
        if labels.get(k) == v:
            logger.debug("    selector match for %s=%s => True", k, v)
            return True

        logger.debug("    selector miss on %s=%s", k, v)

    logger.debug("    all selectors miss => False")
    return False
