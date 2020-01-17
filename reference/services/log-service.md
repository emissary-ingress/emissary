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
  ambassador_id:           # optional; default is ["default"]
  - "string"
  ambassador_id: "string"  # no need for a list if there's only one value

  service: "string"                 # required
  driver: "enum-string:[tcp, http]" # required
  driver_config:                    # required
    additional_log_headers:         # optional; default is [] (only for `driver: http`)
    - header_name: string           # required
      during_request: boolean       # optional; default is true
      during_response: boolean      # optional; default is true
      during_trailer: boolean       # optional; default is true
  flush_interval_time: integer      # optional; default is 1
  flush_interval_byte_size: integer # optional; default is 16384
  grpc: boolean                     # optional; default is false
```
