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

"""Package ambassador.envoy.proto.config.core.v3 provides types for the `envoy.config.core.v3`
Protobuf package.

This is hand-written, and thus incomplete.  We should figure out how to generate it.

"""

from typing import Any, Literal, TypedDict


class SocketAddress(TypedDict):
    """SocketAddress is a (subset of) `envoy.config.core.v3.SocketAddress`."""

    protocol: Literal["TCP", "UDP"]  # = 1
    address: str  # = 2
    port_value: int  # = 3


class Address(TypedDict):
    """Address is a (subeset of) `envoy.config.core.v3.Address`."""

    socket_address: SocketAddress  # = 1


class TransportSocket(TypedDict):
    """Address is a (subeset of) `envoy.config.core.v3.TransportSocket`."""

    name: str  # = 1
    typed_config: Any  # = 3
