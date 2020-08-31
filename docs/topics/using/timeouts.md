# Timeouts

Ambassador Edge Stack enables you to control timeouts in several different ways.

## Request Timeout: `timeout_ms`

`timeout_ms` is the end-to-end timeout for an entire user-level transaction. It begins after the full incoming request is received up until the full response stream is returned to the client. This timeout includes all retries. By default, this is 3000ms.  It can be disabled by setting the value to 0.

## Idle Timeout: `idle_timeout_ms`

`idle_timeout_ms` controls how long a connection should remain open when no traffic is being sent through the connection. `idle_timeout_ms` is distinct from `timeout_ms`, as the idle timeout applies on either down or upstream request events and is reset every time an encode/decode event occurrs or data is processed for the stream. `idle_timeout_ms` operates on a per-route basis and will overwrite behavior of the `cluster_idle_timeout_ms`.  If not set, Ambassador Edge Stack will default to the value set by `cluster_idle_timeout_ms`. It can be disabled by setting the value to 0.

## Cluster Idle Timeout: `cluster_idle_timeout_ms`

`cluster_idle_timeout_ms` controls how long a connection stream will remain open if there are no active requests. This timeout operates based on outgoing requests to upstream services. By default this is set to 30000ms.  It can be disabled by setting the value to 0.

## Connect Timeout: `connect_timeout_ms`

`connect_timeout_ms` sets the connection-level timeout for Ambassador Edge Stack to an upstream service at the network layer.  This timeout runs until Ambassador can verify that a TCP connection has been established.  This timeout cannot be disabled. The default is 3000ms.

## Module Only

## Listener Idle Timeout: `listener_idle_timeout_ms`

`listener_idle_timeout_ms` controls how long a connection stream will remain open if there are no active requests.  This timeout operates based on incoming requests to the listener.  By default, this is set to 30000ms.  It can be disabled by setting the value to 0.  **Caution** Disabling this timeout increases the likelihood of stream leaks due to missed FINs in the TCP connection.

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
  connect_timeout_ms: 2000
```