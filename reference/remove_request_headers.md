# Remove request headers

Ambassador Edge Stack can remove a list of HTTP headers that would be sent to the upstream from the request.

## The `remove_request_headers` annotation

The `remove_request_headers` attribute takes a list of keys used to match to the header

## A basic example

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  tour-ui_mapping
prefix: /
remove_request_headers:
- authorization
service: tour
```

will drop header with key `authorization`.
