# Remove Request Headers

Ambassador Edge Stack can remove a list of HTTP headers that would be sent to the upstream from the request.

## The `remove_request_headers` Attribute

The `remove_request_headers` attribute takes a list of keys used to match to the header.

## A basic example

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  quote-backend
spec:
  prefix: /backend/
  remove_request_headers:
  - authorization
  service: quote
```

will drop header with key `authorization`.
