# LogService Plugin

By default, Ambassador Edge Stack puts the access logs on stdout; such
that the can be read using `kubectl logs`.  The format of those logs,
and the local destination of them, can be configured using the
[`envoy_log_` settings in the `ambassador
Module`](../../core/ambassador#the-ambassador-module).  However, the
options there only allow for logging local to Ambassador's Pod.  By
configuring a `LogService`, you can configure Ambassador Edge Stack to
report its access logs to a remote service.

The remote access log service (or ALS) must implement the
`AccessLogService` gRPC interface, defined in [Envoy's `als.proto`][als.proto].

[als.proto]: https://github.com/datawire/ambassador/blob/master/api/envoy/service/accesslog/v2/als.proto

```yaml
---
apiVersion: getambassador.io/v2
kind: LogService
metadata:
  name: example-log-service
spec:
  # Common to all Ambassador resources
  ambassador_id:           # optional; default is ["default"]
  - "string"
  ambassador_id: "string"  # no need for a list if there's only one value

  # LogService specific
  service: "string"                 # required
  driver: "enum-string:[tcp, http]" # required
  driver_config:                    # required
    additional_log_headers:           # optional; default is [] (only for `driver: http`)
    - header_name: string               # required
      during_request: boolean           # optional; default is true
      during_response: boolean          # optional; default is true
      during_trailer: boolean           # optional; default is true
  flush_interval_time: int-seconds  # optional; default is 1
  flush_interval_byte_size: integer # optional; default is 16384
  grpc: boolean                     # optional; default is false
```

 - `service` is where to route the access log gRPC requests to

 - `driver` identifies which type of accesses to log; HTTP requests (`"http"`) or
   TLS connections (`"tcp"`).

 - `driver_config` stores the configuration that is specific to the `driver`:

    * `driver: tcp` has no additional configuration; the config must
      be set as `driver_config: {}`.

    * `driver: http`

       - `additional_log_headers` identifies HTTP headers to include in
         the access log, and when in the logged-request's lifecycle to
         include them.

 - `flush_interval_time` is the maximum number of seconds to buffer
   accesses for before sending them to the ALS.  The logs will be
   flushed to the ALS every time this duration is reached, or when the
   buffered data reaches `flush_interval_byte_size`, whichever comes
   first.  See the [Envoy documentation][flush_interval_time] for more
   information.

 - `flush_interval_byte_size` is soft size limit for the access log
   buffer.  The logs will be flushed to the ALS every time the
   buffered data reaches this size, or whenever `flush_interval_time`
   elapses, whichever comes first.  See the [Envoy
   documentation][flush_interval_byte_size] for more information.

 - `grpc` must be `true`.

[flush_interval_time]: https://www.envoyproxy.io/docs/envoy/latest/api-v2/config/accesslog/v2/als.proto#envoy-api-field-config-accesslog-v2-commongrpcaccesslogconfig-flush-interval-time
[flush_interval_byte_size]: https://www.envoyproxy.io/docs/envoy/latest/api-v2/config/accesslog/v2/als.proto#envoy-api-field-config-accesslog-v2-commongrpcaccesslogconfig-flush-interval-byte-size

## Example

```yaml
---
apiVersion: getambassador.io/v2
kind: LogService
metadata:
  name: als
spec:
  service: "als.default:3000"
  driver: http
  driver_config: {}  # NB: driver_config must be set, even if it's empty
  grpc: true         # NB: grpc must be true
```
