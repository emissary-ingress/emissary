# Distributed Tracing

Applications that consist of multiple services can be difficult to debug, as a single request can span multiple services. Distributed tracing tells the story of your request as it is processed through your system. Distributed tracing is a powerful tool to debug and analyze your system in addition to request logging and [metrics](/reference/statistics).

## The TracingService

When enabled, the `TracingService` will instruct Ambassador to initiate a trace on requests by generating and populating an `x-request-id` HTTP header. Services can make use of this `x-request-id` header in logging and forward it in downstream requests for tracing. Ambassador also integrates with external trace visualization services, including [LightStep](https://lightstep.com/) and Zipkin-compatible APIs such as [Zipkin](https://zipkin.io/) and [Jaeger](https://github.com/jaegertracing/) to allow you to store and visualize traces. You can read further on [Envoy's Tracing capabilities](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/tracing).

A `TracingService` manifest configures Ambassador to use an external trace visualization service:

```yaml
---
apiVersion: ambassador/v0
kind: TracingService
name: tracing
service: "example-zipkin:9411"
driver: zipkin
config: {}
tag_headers:
- ":authority"
- ":path"
```

- `service` gives the URL of the external HTTP trace service.
- `driver` provides the driver information that handles communicating with the `service`. Supported values are `lightstep` and `zipkin`.
- `config` provides additional configuration options for the selected `driver`.
- `tag_headers` (optional) if present, specifies a list of other HTTP request headers which will be used as tags in the trace's span.

Please note that you must use the HTTP/2 preudo-header names. For example:
- the `host` header should be specified as the `:authority` header; and
- the `method` header should be specified as the `:method` header.

### `lightstep` driver configurations:
- `access_token_file` provides the location of the file containing the access token to the LightStep API.

### `zipkin` driver configurations:
- `collector_endpoint` gives the API endpoint of the Zipkin service where the spans will be sent. The default value is `/api/v1/spans`

You may only use a single `TracingService` manifest.

## Example

The [Ambassador Tracing Tutorial](../../user-guide/tracing-tutorial) has a simple Zipkin tracing example.
