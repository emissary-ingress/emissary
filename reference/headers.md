# Headers

Ambassador can route to target services based on HTTP headers with the `headers` and `regex_headers` annotations. Multiple mappings with different annotations can be applied to construct more complex routing rules.

## The `headers` annotation

The `headers` attribute is a dictionary of `header`: `value` pairs. Ambassador will only allow requests that match the specified `header`: `value` pairs to reach the target service.

You can also set the `value` of a header to `true` to test for the existence of a header.

## A basic example

```yaml
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
headers:
  x-qotm-mode: canary
  x-random-header: datawire
service: qotm
```

will allow requests to `/qotm/` to succeed only if the `x-qotm-mode` header has the value `canary` _and_ the `x-random-header` has the value `datawire`.

## A conditional example

```yaml
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_mode_mapping
prefix: /qotm/
headers:
  x-qotm-mode: true
service: qotm-mode
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_regular_mapping
prefix: /qotm/
service: qotm-regular
```

will send requests that contain the `x-qotm-mode` header to the `qotm-mode` target, while routing all other requests to the `qotm-regular` target.

## `regex_headers`

The following mapping will route mobile requests from Android and iPhones to a mobile service:

```yaml
name: mobile-ui
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind:  Mapping
      name:  mobile_ui_mapping
      regex_headers:
        user-agent: "^(?=.*\\bAndroid\\b)(?=.*\\b(m|M)obile\\b).*|(?=.*\\biPhone\\b)(?=.*\\b(m|M)obile\\b).*$"
      prefix: /
      service: mobile-ui
```