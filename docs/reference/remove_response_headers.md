# Remove Response Headers

Ambassador Edge Stack can remove a list of HTTP headers that would be sent to the client in the response (eg. default `x-envoy-upstream-service-time`).

## The `remove_response_headers` attribute

The `remove_response_headers` attribute takes a list of keys used to match to the header.

## A basic example

```yaml
---
apiVersion: getambassador.io/v1
kind:  Mapping
metadata:
  name:  quote-backend
spec:
  prefix: /backend/
  remove_response_headers:
  - x-envoy-upstream-service-time
  service: quote
```

will drop header with key `x-envoy-upstream-service-time`.
