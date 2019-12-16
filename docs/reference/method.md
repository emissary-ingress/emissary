# Method-based routing

Ambassador Edge Stack supports routing based on HTTP method and regular expression.

## Using `method`

The `method` annotation specifies the specific HTTP method for a mapping. The value of the `method` annotation must be in all upper case.

For example:

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: get
spec:
  prefix: /backend/get_only/
  method: GET
  service: quote
```

## Using `method_regex`

When `method_regex` is set to `true`, the value of the `method` annotation will be interpreted as a regular expression.
