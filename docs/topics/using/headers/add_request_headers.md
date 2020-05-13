# Add Request Headers

Ambassador Edge Stack can add a dictionary of HTTP headers that can be added to each request that is passed to a service.

## The `add_request_headers` attribute

The `add_request_headers` attribute is a dictionary of `header`: `value` pairs. The `value` can be a `string`, `bool` or `object`. When it is an `object`, the object should have a `value` property, which is the actual header value, and the remaining attributes are additional envoy properties.

Envoy dynamic values `%DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT%` and `%PROTOCOL%` are supported, in addition to static values.

`add_request_headers` can be set either in a `Mapping` or using [`ambassador Module defaults`](../../using/defaults).

### Mapping Example

```yaml
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: quote-backend
spec:
  prefix: /backend/
  add_request_headers:
    x-test-proto: "%PROTOCOL%"
    x-test-ip: "%DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT%"
    x-test-static: This is a test header
    x-test-static-2:
      value: This the test header    #same as above "x-test-static header"
    x-test-object:
      value: This the value
      append: False                  #True by default
  service: quote
  ```

will add the protocol, client IP, and a static header to `/backend/`.

### Defaults Example

```yaml
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    defaults:
      httpmapping:
        add_request_headers:
          x-test-proto: "%PROTOCOL%"
          x-test-ip: "%DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT%"
          x-test-static: This is a test header
          x-test-static-2:
            value: This the test header    #same as above "x-test-static header"
          x-test-object:
            value: This the value
            append: False                  #True by default
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: quote-backend1
spec:
  prefix: /backend1/
  service: quote
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: quote-backend2
spec:
  prefix: /backend2/
  service: quote
```

This example will add the same headers for both mappings.
