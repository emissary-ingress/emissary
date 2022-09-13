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

"""Package ambassador.envoy.proto.config.route.v3 provides types for the `envoy.config.route.v3`
Protobuf package.

This is hand-written, and thus incomplete.  We should figure out how to generate it.

"""

from typing import List, TypedDict

import ambassador.envoy.proto.type.matcher.v3 as matcherv3
import ambassador.envoy.proto.wkt as wkt


class HeaderMatcher(TypedDict, total=False):
    name: str  # = 1

    # oneof header_match_specifier {
    exact_match: str  # = 4
    safe_regex_match: matcherv3.RegexMatcher  # = 11
    # range_match: typev3.Int64Range = 6
    present_match: bool  # = 7
    prefix_match: str  # = 9
    suffix_match: str  # = 10
    contains_match: str  # = 12
    # }


class RouteMatch(TypedDict, total=False):
    """RouteMatch is a (subset of) `envoy.config.route.v3.RouteMatch`."""

    # oneof path_specifier {
    prefix: str  # = 1
    safe_regex: matcherv3.RegexMatcher  # = 10
    # }

    case_sensitive: bool  # = 4
    headers: List[HeaderMatcher]  # = 6


class RouteAction(TypedDict, total=False):
    """RouteAction is a (subset of) `envoy.config.route.v3.RouteAction`."""

    cluster: str  # = 1
    prefix_rewrite: str  # = 5
    timeout: wkt.Duration  # = 8


class Route(TypedDict, total=False):
    """Route is a (subset of) `envoy.config.route.v3.Route`."""

    name: str  # = 15
    match: RouteMatch  # = 1
    route: RouteAction  # = 2


class VirtualHost(TypedDict):
    """VirtualHost is a (subset of) `envoy.config.route.v3.VirtualHost`."""

    name: str  # = 1
    domains: List[str]  # = 2
    routes: List[Route]  # = 3
