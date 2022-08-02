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

from ...ir.irtracing import IRTracing

if TYPE_CHECKING:
    from . import V3Config  # pragma: no cover


class V3Tracing(dict):
    def __init__(self, config: "V3Config") -> None:
        # We should never be instantiated unless there is, in fact, defined tracing stuff.
        assert config.ir.tracing

        super().__init__()

        tracing = typecast(IRTracing, config.ir.tracing)

        name = tracing["driver"]

        if not name.startswith("envoy."):
            name = "envoy.%s" % (name.lower())

        driver_config = tracing["driver_config"]

        # We check for the full 'envoy.tracers.datadog' below because that's how it's set in the
        # IR code. The other tracers are configured by their short name and then 'envoy.' is
        # appended above.
        if name.lower() == "envoy.zipkin":
            driver_config["@type"] = "type.googleapis.com/envoy.config.trace.v3.ZipkinConfig"
            # In xDS v3 the old Zipkin-v1 API can only be specified as the implicit default; it
            # cannot be specified explicitly.
            # https://www.envoyproxy.io/docs/envoy/latest/version_history/v1.12.0.html?highlight=http_json_v1
            # https://github.com/envoyproxy/envoy/blob/ae1ed1fa74f096dabe8dd5b19fc70333621b0309/api/envoy/config/trace/v3/zipkin.proto#L27
            if driver_config["collector_endpoint_version"] == "HTTP_JSON_V1":
                del driver_config["collector_endpoint_version"]
        elif name.lower() == "envoy.tracers.datadog":
            driver_config["@type"] = "type.googleapis.com/envoy.config.trace.v3.DatadogConfig"
            if not driver_config.get("service_name"):
                driver_config["service_name"] = "ambassador"
        elif name.lower() == "envoy.lightstep":
            driver_config["@type"] = "type.googleapis.com/envoy.config.trace.v3.LightstepConfig"
        else:
            # This should be impossible, because we ought to have validated the input driver
            # in ambassador/pkg/api/getambassador.io/v2/tracingservice_types.go:47
            raise Exception('Unsupported tracing driver "%s"' % name.lower())

        self["http"] = {"name": name, "typed_config": driver_config}

    @classmethod
    def generate(cls, config: "V3Config") -> None:
        config.tracing = None

        if config.ir.tracing:
            config.tracing = config.save_element("tracing", config.ir.tracing, V3Tracing(config))
