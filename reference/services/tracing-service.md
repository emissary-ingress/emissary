# TracingService Plugin

Applications that consist of multiple services can be difficult to debug, as a single request can span multiple services. Distributed tracing tells the story of your request as it is processed through your system. Distributed tracing is a powerful tool to debug and analyze your system in addition to request logging and [metrics](../../statistics).

## The TracingService

When enabled, the `TracingService` will instruct Ambassador Edge Stack to initiate a trace on requests by generating and populating an `x-request-id` HTTP header. Services can make use of this `x-request-id` header in logging and forward it in downstream requests for tracing. Ambassador Edge Stack also integrates with external trace visualization services, including [LightStep](https://lightstep.com/) and Zipkin-compatible APIs such as [Zipkin](https://zipkin.io/) and [Jaeger](https://github.com/jaegertracing/) to allow you to store and visualize traces. You can read further on [Envoy's Tracing capabilities](https://www.envoyproxy.io/docs/envoy/v1.10.0/intro/arch_overview/tracing).

A `TracingService` manifest configures Ambassador Edge Stack to use an external trace visualization service:

```yaml
---
apiVersion: getambassador.io/v2
kind:  TracingService
metadata:
  name:  tracing
spec:
  service: "example-zipkin:9411"
  driver: zipkin
  config: {}
  tag_headers:
  - ":authority"
  - ":path"
```

- `service` gives the URL of the external HTTP trace service.
- `driver` provides the driver information that handles communicating with the `service`. Supported values are `lightstep`, `zipkin`, and `datadog`.
- `config` provides additional configuration options for the selected `driver`.
- `tag_headers` (optional) if present, specifies a list of other HTTP request headers which will be used as tags in the trace's span.

Please note that you must use the HTTP/2 pseudo-header names. For example:

- the `host` header should be specified as the `:authority` header; and
- the `method` header should be specified as the `:method` header.

### `lightstep` driver configurations

- `access_token_file` provides the location of the file containing the access token to the LightStep API.

### `zipkin` driver configurations

- `collector_endpoint` gives the API endpoint of the Zipkin service where the spans will be sent. The default value is `/api/v1/spans`
- `trace_id_128bit` whether a 128bit trace id will be used when creating a new trace instance. Defaults to `true`. Setting to `false` will result in a 64 bit trace id being used.
- `shared_span_context` whether client and server spans will share the same span id. The default value is `true`.

### `datadog` driver configurations

- `service_name` the name of the service which is attached to the traces. The default value is `ambassador`.

You may only use a single `TracingService` manifest per Ambassador deployment. Ensure [ambassador_id](../../running#ambassador_id) is set correctly in the `TracingService` manifest.

## Example

The [Ambassador Edge Stack Tracing Tutorial](../../../user-guide/tracing-tutorial) has a simple Zipkin and DataDog tracing tutorials.
