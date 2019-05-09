# Add Request Headers

Ambassador can add a dictionary of HTTP headers that can be added to each request that is passed to a service.

## The `add_request_headers` annotation

The `add_request_headers` attribute is a dictionary of `header`: `value` pairs. Where the value can be string or another dictionary, which supports advanced configuration of the headers. Envoy dynamic values `%DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT%` and `%PROTOCOL%` are supported, in addition to static values. Please see the examples provided below for more details on configuration. The same configuration is applicable for `add_response_headers`.

Only `append` configurations are supported with additional configrations.

## A basic example

```yaml
---
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
add_request_headers:
  x-test-proto: "%PROTOCOL%"
  x-test-ip: "%DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT%"
  x-test-static: This is a test header
  x-test-append:
    value: TestValue
    append: False
  x-test-append:
    value: TestValue2,
service: qotm
```

will add the protocol, client IP, and a static header to `/qotm/`.
