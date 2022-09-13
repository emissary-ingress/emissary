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

"""Package ambassador.envoy.proto.config.listener.v3 provides types for the
`envoy.config.listener.v3` Protobuf package.

This is hand-written, and thus incomplete.  We should figure out how to generate it.

"""

from typing import Any, Dict, List, Literal, TypedDict

import ambassador.envoy.proto.config.core.v3 as corev3


class Filter(TypedDict):
    """Filter is a (subset of) `envoy.config.listener.v3.Filter`."""

    name: str  # = 1
    typed_config: Any  # = 4


class FilterChainMatch(TypedDict, total=False):
    """FilterChainMatch is a (subset of) `envoy.config.listener.v3.FilterChainMatch`."""

    transport_protocol: str  # = 9
    server_names: List[str]  # = 11


class FilterChain(TypedDict, total=False):
    """FilterChain is a (subset of) `envoy.config.listener.v3.FilterChain`."""

    filter_chain_match: FilterChainMatch  # = 1
    filters: List[Filter]  # = 3
    transport_socket: corev3.TransportSocket  # = 6
    name: str # = 7


# TrafficDirection is a `envoy.config.core.TrafficDirection`.
TrafficDirection = Literal["UNSPECIFIED", "INBOUND", "OUTBOUND"]


class ListenerFilter(TypedDict, total=False):
    """ListenerFilter is a (subset of) `envoy.config.listener.v3.ListenerFilter`."""

    name: str  # = 1
    typed_config: Any  # = 3


class Listener(TypedDict, total=False):
    """_Listener is a (subset of) `envoy.config.listener.v3.Listener`."""

    name: str  # = 1
    address: corev3.Address  # = 2
    filter_chains: List[FilterChain]  # = 3

    per_connection_buffer_limit_bytes: int  # = 5

    listener_filters: List[ListenerFilter]  # = 9

    traffic_direction: TrafficDirection  # = 16
