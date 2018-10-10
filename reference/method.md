# Method-based routing

Ambassador supports routing based on HTTP method and regular expression.

## Using `method`

The `method` annotation specifies the specific HTTP method for a mapping. The value of the `method` annotation must be in all upper case.

For example:

```yaml
---
aapiVersion: ambassador/v0
kind: Mapping
name: get_mapping
prefix: /get_only/
method: GET
service: qotm
```

## Using `method_regex`

When `method_regex` is set to `true`, the value of the `method` annotation will be interpreted as a regular expression. 