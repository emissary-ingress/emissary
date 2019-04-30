# Remove response headers

Ambassador can remove a list of HTTP headers that would be sent to the client in the response (eg. default `x-envoy-upstream-service-time`)

## The `remove_response_headers` annotation

The `remove_response_headers` attribute takes a list of keys used to match to the header

## A basic example

```yaml
---
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
remove_response_headers:
- x-envoy-upstream-service-time
service: qotm
```

will drop header with key `x-envoy-upstream-service-time`.
