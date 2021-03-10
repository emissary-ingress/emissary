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
    from . import V2Config


class V2Tracing(dict):
    def __init__(self, config: 'V2Config') -> None:
        # We should never be instantiated unless there is, in fact, defined tracing stuff.
        assert config.ir.tracing

        super().__init__()

        tracing = typecast(IRTracing, config.ir.tracing)

        name = tracing['driver']

        if not name.startswith('envoy.'):
            name = 'envoy.%s' % (name.lower())

        driver_config = tracing['driver_config']

        # We check for the full 'envoy.tracers.datadog' below because that's how it's set in the
        # IR code. The other tracers are configured by their short name and then 'envoy.' is
        # appended above.
        if name.lower() == 'envoy.zipkin':
            driver_config['@type'] = 'type.googleapis.com/envoy.config.trace.v2.ZipkinConfig'
            # The collector_endpoint is mandatory now.
            if not driver_config.get('collector_endpoint'):
                driver_config['collector_endpoint'] = '/api/v1/spans'
            # Make 128-bit traceid the default
            if not 'trace_id_128bit' in driver_config:
                driver_config['trace_id_128bit'] = True
        elif name.lower() == 'envoy.tracers.datadog':
            driver_config['@type'] = 'type.googleapis.com/envoy.config.trace.v2.DatadogConfig'
            if not driver_config.get('service_name'):
                driver_config['service_name'] = 'ambassador'
        elif name.lower() == 'envoy.lightstep':
            driver_config['@type'] = 'type.googleapis.com/envoy.config.trace.v2.LightstepConfig'
        else:
            # This should be impossible, because we ought to have validated the input driver
            # in ambassador/pkg/api/getambassador.io/v2/tracingservice_types.go:47
            raise Exception("Unsupported tracing driver \"%s\"" % name.lower())

        self['http'] = {
            "name": name,
            "typed_config": driver_config
        }

    @classmethod
    def generate(cls, config: 'V2Config') -> None:
        config.tracing = None

        if config.ir.tracing:
            config.tracing = config.save_element('tracing', config.ir.tracing, V2Tracing(config))
