# Headers

Ambassador Edge Stack can route to target services based on HTTP headers with the `headers` and `regex_headers` annotations. Multiple mappings with different annotations can be applied to construct more complex routing rules.

## The `headers` annotation

The `headers` attribute is a dictionary of `header`: `value` pairs. Ambassador Edge Stack will only allow requests that match the specified `header`: `value` pairs to reach the target service.

You can also set the `value` of a header to `true` to test for the existence of a header.

## A basic example

```yaml
---
apiVersion: getambassador.io/v1
kind:  Mapping
metadata:
  name:  tour-backend
spec:
  prefix: /backend/
  service: tour
  headers:
    x-tour-mode: backend
    x-random-header: datawire

```

will allow requests to `/backend/` to succeed only if the `x-tour-mode` header has the value `backend` _and_ the `x-random-header` has the value `datawire`.

## A conditional example

```yaml
---
apiVersion: getambassador.io/v1
kind:  Mapping
metadata:
  name:  tour-mode
spec:
  prefix: /
  service: tour-mode
  headers:
    x-tour-mode: true

---
apiVersion: getambassador.io/v1
kind:  Mapping
metadata:
  name:  tour-regular
spec:
  prefix: /
  service: tour-regular
```

will send requests that contain the `x-tour-mode` header to the `tour-mode` target, while routing all other requests to the `tour-regular` target.

## `regex_headers`

The following mapping will route mobile requests from Android and iPhones to a mobile service:

```yaml
---
apiVersion: getambassador.io/v1
kind:  Mapping
metadata:
  name:  tour-backend
spec:
  regex_headers:
    user-agent: "^(?=.*\\bAndroid\\b)(?=.*\\b(m|M)obile\\b).*|(?=.*\\biPhone\\b)(?=.*\\b(m|M)obile\\b).*$"
  prefix: /
  service: mobile-ui
```
