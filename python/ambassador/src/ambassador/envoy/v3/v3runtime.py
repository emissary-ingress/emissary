# Copyright 2023 Ambassador Labs. All rights reserved.
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
    from . import V3Config  # pragma: no cover


class V3Runtime(dict):
    def __init__(self, config: "V3Config") -> None:
        super().__init__()

        static_runtime_layer = {
            "re2.max_program_size.error_level": 200,
            # TODO(lance):
            # the new default is that all filters are looked up using the @type which currently we exclude on a lot of
            # our filters. This will ensure we do not break current config. We can migrate over
            # in a minor release. see here: https://www.envoyproxy.io/docs/envoy/v1.22.0/version_history/current#minor-behavior-changes
            # The biggest impact of this is ensuring that ambex imports all the types because we will need to import many more
            "envoy.reloadable_features.no_extension_lookup_by_name": False,
        }

        user_runtime = config.ir.ambassador_module.get("runtime_flags", None)
        if user_runtime:
            max_io_per_cycle = user_runtime.get("http.max_requests_per_io_cycle", None)
            if max_io_per_cycle:
                if isinstance(max_io_per_cycle, int) and max_io_per_cycle > 0:
                    static_runtime_layer["http.max_requests_per_io_cycle"] = max_io_per_cycle
                else:
                    config.ir.logger.error(
                        f"value: {max_io_per_cycle} is invalid for Module field runtime_flags.max_requests_per_io_cycle. must be an integer greater than zero"
                    )

            rapid_reset_min_stream_lifetime = user_runtime.get(
                "overload.premature_reset_min_stream_lifetime_seconds", None
            )
            if rapid_reset_min_stream_lifetime:
                if (
                    isinstance(rapid_reset_min_stream_lifetime, int)
                    and rapid_reset_min_stream_lifetime > 0
                ):
                    static_runtime_layer["overload.premature_reset_min_stream_lifetime_seconds"] = (
                        rapid_reset_min_stream_lifetime
                    )
                else:
                    config.ir.logger.error(
                        f"value: {rapid_reset_min_stream_lifetime} is invalid for Module field overload.premature_reset_min_stream_lifetime_seconds. must be an integer greater than zero"
                    )

            rapid_reset_total_streams = user_runtime.get(
                "overload.premature_reset_total_stream_count", None
            )
            if rapid_reset_total_streams:
                if isinstance(rapid_reset_total_streams, int) and rapid_reset_total_streams > 0:
                    static_runtime_layer["overload.premature_reset_total_stream_count"] = (
                        rapid_reset_total_streams
                    )
                else:
                    config.ir.logger.error(
                        f"value: {rapid_reset_total_streams} invalid for Module field overload.premature_reset_total_stream_count. must be an integer greater than zero"
                    )

            use_rapid_reset_goaway = user_runtime.get(
                "envoy.restart_features.send_goaway_for_premature_rst_streams", None
            )
            if use_rapid_reset_goaway:
                if isinstance(rapid_reset_total_streams, bool):
                    static_runtime_layer[
                        "envoy.restart_features.send_goaway_for_premature_rst_streams"
                    ] = use_rapid_reset_goaway
                else:
                    config.ir.logger.error(
                        f"value: {use_rapid_reset_goaway} is invalid for Module field envoy.restart_features.send_goaway_for_premature_rst_streams. This field must be true/false"
                    )

        self.update({"layers": [{"name": "static_layer", "static_layer": static_runtime_layer}]})

    @classmethod
    def generate(cls, config: "V3Config") -> None:
        config.layered_runtime = config.save_element(
            "layered_runtime", config.ir.ambassador_module, V3Runtime(config)
        )
