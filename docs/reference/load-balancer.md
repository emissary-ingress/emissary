# Load Balancing in Ambassador

Ambassador lets users control how Ambassador load balances between resulting endpoints for a given mapping.

Load balancing configuration can be set for all Ambassador mappings in the [ambassador](https://www.getambassador.io/reference/modules#the-ambassador-module) module, or set per [mapping](https://www.getambassador.io/reference/mappings#configuring-mappings).
If nothing is set, simple round robin balancing is used via Kubernetes services.

## The `load_balancer` attribute

The `load_balancer` attribute configures the load balancing. The following fields are supported:

- `type`: Specifies the type of load balancer to use.
- `policy`: Specifies the load balancing policy to use.

### `type: kubernetes`
The `kubernetes` type delegates load balancing to Kubernetes. Ambassador will route traffic directly to a Kubernetes service and [Kubernetes service networking](https://kubernetes.io/docs/concepts/services-networking/) is then responsible for load balancing traffic between pods. Supported policies:
- `round_robin` (default)

### `type: envoy`
The `envoy` type delegates load balancing to Envoy. Supported policies:
- `round_robin`

## Examples

- Configuring a mapping to use round robin policy via Envoy
```yaml
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
load_balancer:
  type: envoy
  policy: round_robin
```

- Configuring global load balancing to Envoy's load balancing and mapping's load balancing to Kubernetes' load balancing
```yaml
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  load_balancer:
    type: envoy
    policy: round_robin
---
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
load_balancer:
  type: kubernetes
  policy: round_robin
```