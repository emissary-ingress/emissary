## RateLimitService

A `RateLimitService` manifest configures Ambassador to use an external service to check and enforce rate limits for incoming requests:

```yaml
---
apiVersion: ambassador/v0
kind: RateLimitService
name: ratelimit
service: "example-rate-limit:5000"
```

- `service` gives the URL of the rate limit service. See [Rate Limiting with an External Rate Limit Service](rate-limit-external.md)

You may only use a single `RateLimitService` manifest.
