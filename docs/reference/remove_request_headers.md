# Remove request headers

Ambassador can remove a list of HTTP headers that would be sent to the upstream from the request.

## The `remove_request_headers` annotation

The `remove_request_headers` attribute takes a list of keys used to match to the header

## A basic example

```yaml
---
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
remove_request_headers:
- authorization
service: qotm
```

will drop header with key `authorization`.
