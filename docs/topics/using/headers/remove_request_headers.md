# Remove Request Headers

Ambassador Edge Stack can remove a list of HTTP headers that would be sent to the upstream from the request.

## The `remove_request_headers` Attribute

The `remove_request_headers` attribute takes a list of keys used to match to the header.

`remove_request_headers` can be set either in a `Mapping` or using [`ambassador Module defaults`](../../defaults).

## Mapping Example

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

will drop the header with key `authorization`.

```yaml
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    defaults:
      httpmapping:
        remove_request_headers:
        - authorization
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  quote-backend1
spec:
  prefix: /backend1/
  service: quote
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  quote-backend2
spec:
  prefix: /backend2/
  service: quote
```

This is the same as the mapping example, but the headers will be removed for both mappings.