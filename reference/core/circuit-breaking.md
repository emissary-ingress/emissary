# Circuit Breaking in Ambassador

Ambassador lets users configure circuit breaking limits at the network level.

Circuit breaking configuration can be set for all Ambassador mappings in the [ambassador](/reference/core/ambassador) module, or set per [mapping](https://www.getambassador.io/reference/mappings#configuring-mappings).

The `circuit_breakers` attribute configures circuit breaking. The following fields are supported:
```yaml
circuit_breakers:
- priority: <string>
  max_connections: <integer>
  max_pending_requests: <integer>
  max_requests: <integer>
  max_retries: <integer>
```

### `priority`
(Default: `default`) Specifies the priority to which the circuit breaker settings apply to; can be set to either `default` or `high`.

### `max_connections`
(Default: `1024`) Specifies the maximum number of connections that Ambassador will make to the services. In practice, this is more applicable to HTTP/1.1 than HTTP/2.

### `max_pending_requests`
(Default: `1024`) Specifies the maximum number of requests that will be queued while waiting for a connection. In practice, this is more applicable to HTTP/1.1 than HTTP/2.

### `max_requests`
(Default: `1024`) Specifies the maximum number of parallel outstanding requests to hosts. In practice, this is more applicable to HTTP/2 than HTTP/1.1.

### `max_retries`
(Default: `3`) Specifies the maximum number of parallel retries allowed to hosts.

### Examples:

- Circuit breakers defined on a single mapping -
```yaml
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
circuit_breakers:
- max_connections: 2048
  max_pending_requests: 2048
```

- Circuit breakers defined globally -
```yaml
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  circuit_breakers:
  - max_connections: 2048
    max_pending_requests: 2048
---
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
```

**Notes:**

- For more insight on how circuit breakers behave, see the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/circuit_breaking).
- The responses from a broken circuit contain the `x-envoy-overloaded` header.
- The following are the default values for circuit breaking if nothing is specified:

```yaml
circuit_breakers:
- priority: default
  max_connections: 1024
  max_pending_requests: 1024
  max_requests: 1024
  max_retries: 3
```

# Automatic Retries

Ambassador does not retry a failed request unless explicitly configured. Automatic retries let you add resilience to your services in case of request failures.

Retry policy can be set for all Ambassador mappings in the [ambassador](/reference/core/ambassador) module, or set per [mapping](https://www.getambassador.io/reference/mappings#configuring-mappings).

The `retry_policy` attribute configures circuit breaking. The following fields are supported:
```yaml
retry_policy:
  retry_on: <string>
  num_retries: <integer>
  per_try_timeout: <string>
```

### `retry_on`
(Required) Specifies the condition under which Ambassador retries a failed request like upstream server errors, gateway errors, etc.

One of `5xx`, `gateway-error`, `connect-failure`, `retriable-4xx`, `refused-stream`, `retriable-status-codes`. Read in detail at [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.5.0/configuration/http_filters/router_filter#x-envoy-retry-on)

### `num_retries`
(Default: 1) Specifies the number of retries to do for a failed request.

### `per_try_timeout`
(Default: global request timeout) Specify the timeout for each retry, e.g. `1s`, `1500ms`.

### Examples:

- Retries defined on a single mapping -
```yaml
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
retry_policy:
  retry_on: "5xx"
  num_retries: 10
```

- Circuit breakers defined globally -
```yaml
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  retry_policy:
    retry_on: "retriable-4xx"
    num_retries: 4
---
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
```