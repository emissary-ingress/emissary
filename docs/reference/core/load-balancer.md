# Load Balancing in Ambassador

Ambassador lets users control how it load balances between resulting endpoints for a given mapping. This feature ships in Early Access for Ambassador 0.52, and requires setting the environment variable `AMASSADOR_ENABLE_ENDPOINTS` to `true` to enable this feature.

Load balancing configuration can be set for all Ambassador mappings in the [ambassador](/reference/core/ambassador) module, or set per [mapping](https://www.getambassador.io/reference/mappings#configuring-mappings). If nothing is set, simple round robin balancing is used via Kubernetes services.

The `load_balancer` attribute configures the load balancing. The following fields are supported:

```yaml
load_balancer:
  policy: <load balancing policy to use>
```

Supported load balancer policies:
- `round_robin`
- `ring_hash`
- `maglev`

### Round Robin
When policy is set to `round_robin`, Ambassador discovers healthy endpoints for the given mapping, and load balances the incoming requests in a round robin fashion. For example:

```yaml
apiVersion: ambassador/v1
kind:  Module
name:  ambassador
config:
  load_balancer:
    policy: round_robin
```

or, per mapping:

```yaml
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
load_balancer:
  policy: round_robin
```

Note that load balancing may not appear to be "even" due to Envoy's threading model. For more details, see the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/faq/concurrency_lb).

### Sticky Sessions / Session Affinity
Configuring sticky sessions makes Ambassador route requests to the same backend service in a given session. In other words, requests in a session are served by the same Kubernetes pod. Ambassador lets you configure session affinity based on the following parameters in an incoming request:

- Cookie
- Header
- Source IP

**NOTE:** Ambassador supports sticky sessions using 2 load balancing policies, `ring_hash` and `maglev`.


##### Cookie
```yaml
load_balancer:
  policy: ring_hash
  cookie:
    name: <name of the cookie, required>
    ttl: <TTL to set in the generated cookie>
    path: <name of the path for the cookie>
```

If the cookie you wish to set affinity on is already present in incoming requests, then you only need the `cookie.name` field. However, if you want Ambassador to generate and set a cookie in response to the first request, then you need to specify a value for the `cookie.ttl` field which generates a cookie with the given expiration time.

For example, the following configuration asks the client to set a cookie named `sticky-cookie` with expiration of 60 seconds in response to the first request if the cookie is not already present.

```yaml
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
load_balancer:
  policy: ring_hash
  cookie:
    name: sticky-cookie
    ttl: 60s
```

##### Header
```yaml
load_balancer:
  policy: ring_hash
  header: <header name>
```

Ambassador allows header based session affinity if the given header is present on incoming requests.

Example:
```yaml
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
load_balancer:
  policy: ring_hash
  header: STICKY_HEADER
```

##### Source IP
```yaml
load_balancer:
  policy: ring_hash
  source_ip: <boolean>
```

Ambassador allows session affinity based on the source IP of incoming requests. For example:

```yaml
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
load_balancer:
  policy: ring_hash
  source_ip: true
```

Load balancing can be configured both globally, and overridden on a per mapping basis. The following example configures the default load balancing policy to be round robin, while using header-based session affinity for requests to `/qotm/`:

```yaml
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  load_balancer:
    policy: round_robin
---
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
load_balancer:
  policy: ring_hash
  header: STICKY_HEADER
```