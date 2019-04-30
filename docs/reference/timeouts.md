# Timeouts

Ambassador enables you to control timeouts in several different ways.

## Request timeout: `timeout_ms`

`timeout_ms` is the timeout for an entire user-level transaction. By default, this is 5000ms. This spans the point at which the entire downstream request has been processed (i.e., end of stream) to the point where the upstream response has been processed. This timeout includes all retries. 

## Idle timeout: `idle_timeout_ms`

`idle_timeout_ms` controls how long a connection should remain open when no traffic is being sent through the connection. If not set, Ambassador will wait 5 minutes (300000 milliseconds).

## Connect timeout: `connect_timeout_ms`

`connect_timeout_ms` controls the connection-level timeout for Ambassador to an upstream service.

### Example

The various timeouts are applied onto a `Mapping` resource and can be combined.

```yaml
---
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
timeout_ms: 4000
idle_timeout_ms: 500000
connect_timeout_ms: 4000
```
