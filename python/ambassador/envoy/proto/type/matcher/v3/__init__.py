# Copyright 2022 Datawire.  All rights reserved.
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

"""Package ambassador.envoy.type.matcher.v3 provides types for the `envoy.type.matcher.v3` Protobuf
package.

This is hand-written, and thus incomplete.  We should figure out how to generate it.

"""

from typing import TypedDict


class GoogleRE2(TypedDict, total=False):
    """GoogleRE2 is a `envoy.type.matcher.v3.RegexMatcher.GoogleRE2`."""

    max_program_size: int  # = 1


class RegexMatcher(TypedDict, total=False):
    """RegexMatcher is a `envoy.type.matcher.v3.RegexMatcher`."""

    google_re2: GoogleRE2  # = 1
    regex: str  # = 2
