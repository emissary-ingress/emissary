# Remove response headers

Ambassador Edge Stack can remove a list of HTTP headers that would be sent to the client in the response (eg. default `x-envoy-upstream-service-time`)

## The `remove_response_headers` annotation

The `remove_response_headers` attribute takes a list of keys used to match to the header

## A basic example

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  tour-ui_mapping
prefix: /
remove_response_headers:
- x-envoy-upstream-service-time
service: tour
```

will drop header with key `x-envoy-upstream-service-time`.


