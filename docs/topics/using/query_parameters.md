# Query Parameter based Routing

Ambassador Edge Stack can route to target services based on HTTP query parameters with the `query_parameters` and `regex_query_parameters` specifications. Multiple mappings with different annotations can be applied to construct more complex routing rules.

## The `query_parameters` Annotation

The `query_parameters` attribute is a dictionary of `query_parameter`: `value` pairs. Ambassador Edge Stack will only allow requests that match the specified `query_parameter`: `value` pairs to reach the target service.

You can also set the `value` of a query parameter to `true` to test for the existence of a query parameter.

### A Basic Example

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  quote-backend
spec:
  prefix: /backend/
  service: quote
  query_parameters:
    quote-mode: backend
    random-query-parameter: datawire
```

will allow requests to /backend/ to succeed only if the `quote-mode` query paraameter has the value `backend` and the `random-query-parameter` has the value `datawire`.

### A Conditional Example

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  quote-mode
spec:
  prefix: /backend/
  service: quote-mode
  query_parameters:
    quote-mode: true

---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  quote-regular
spec:
  prefix: /backend/
  service: quote-regular
```

will send requests that contain the `quote-mode` query parameter to the `quote-mode` target, while routing all other requests to the `quote-regular` target.

## `regex_query_parameters`

The following mapping will route mobile requests from Android and iPhones to a mobile service:

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  quote-backend
spec:
  regex_query_parameters:
    user-agent: "^(?=.*\\bAndroid\\b)(?=.*\\b(m|M)obile\\b).*|(?=.*\\biPhone\\b)(?=.*\\b(m|M)obile\\b).*$"
  prefix: /backend/
  service: quote
```
