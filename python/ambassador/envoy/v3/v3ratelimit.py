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
from typing import cast as typecast

from ...ir.irratelimit import IRRateLimit

if TYPE_CHECKING:
    from . import V3Config  # pragma: no cover


class V3RateLimit(dict):
    def __init__(self, config: "V3Config") -> None:
        # We should never be instantiated unless there is, in fact, defined ratelimit stuff.
        assert config.ir.ratelimit

        super().__init__()

        ratelimit = typecast(IRRateLimit, config.ir.ratelimit)

        assert ratelimit.cluster.envoy_name

        protocol_version = ratelimit.protocol_version
        self["transport_api_version"] = protocol_version.replace("alpha", "").upper()
        self["grpc_service"] = {"envoy_grpc": {"cluster_name": ratelimit.cluster.envoy_name}}

    @classmethod
    def generate(cls, config: "V3Config") -> None:
        config.ratelimit = None

        if config.ir.ratelimit:
            config.ratelimit = config.save_element(
                "ratelimit", config.ir.ratelimit, V3RateLimit(config)
            )
