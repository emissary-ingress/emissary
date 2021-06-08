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

import sys

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
