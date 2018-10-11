# Rate Limits

A mapping that specifies the `rate_limits` list attribute and at least one `rate_limits` rule, will call the external [RateLimitService](services/rate-limit-service) before proceeding with the request. An example:

```yaml
apiVersion: ambassador/v0
kind: Mapping
name: rate_limits_mapping
prefix: /rate-limit/
service: rate-limit-example
rate_limits:
  - {}
  - descriptor: a rate-limit descriptor
    headers:
    - matching-header
```
            
Rate limit rule settings:

- `descriptor`: if present, specifies a string identifying the triggered rate limit rule. This descriptor will be sent to the `RateLimitService`.
- `headers`: if present, specifies a list of other HTTP headers which **must** appear in the request for the rate limiting rule to apply. These headers will be sent to the `RateLimitService`.

Please note that you must use the internal HTTP/2 request header names in `rate_limits` rules. For example:
- the `host` header should be specified as the `:authority` header; and
- the `method` header should be specified as the `:method` header.