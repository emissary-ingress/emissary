# Timeouts

Ambassador Edge Stack enables you to control timeouts in several different ways.

## Request Timeout: `timeout_ms`

`timeout_ms` is the timeout for an entire user-level transaction. By default, this is 3000ms. This spans the point at which the entire downstream request has been processed (i.e., end of stream) to the point where the upstream response has been processed. This timeout includes all retries.

## Idle Timeout: `idle_timeout_ms`

`idle_timeout_ms` controls how long a connection should remain open when no traffic is being sent through the connection. If not set, Ambassador Edge Stack will wait 5 minutes (300000 milliseconds).

## Connect Timeout: `connect_timeout_ms`

`connect_timeout_ms` controls the connection-level timeout for Ambassador Edge Stack to an upstream service. The default is `3000m`.

### Example

The various timeouts are applied to a Mapping resource and can be combined.

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  quote-backend
spec:
  prefix: /backend/
  service: quote
  timeout_ms: 4000
  idle_timeout_ms: 500000
  connect_timeout_ms: 4000
```
