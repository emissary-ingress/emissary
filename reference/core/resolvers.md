# Service discovery configuration

Service discovery is how applications and (micro)services are located on the network. In a cloud environment, services are ephemeral, so a real-time service discovery mechanism is critical. Ambassador uses information from service discovery to determine where to route incoming requests.

## Ambassador support for service discovery

Ambassador supports different mechanisms for service discovery. These mechanisms are:

* Kubernetes service-level discovery (default).
* Kubernetes endpoint-level discovery.
* Consul endpoint-level discovery.

### Kubernetes service-level discovery

By default, Ambassador uses Kubernetes DNS and service-level discovery. In a `Mapping` resource, specifying `service: foo` will prompt Ambassador to look up the DNS address of the `foo` Kubernetes service. Traffic will be routed to the `foo` service. Kubernetes will then load balance that traffic between multiple pods. For more details on Kubernetes networking and how this works, see https://blog.getambassador.io/session-affinity-load-balancing-controls-grpc-web-and-ambassador-0-52-2b916b396d0c.

### Kubernetes endpoint-level discovery

Ambassador can also watch Kubernetes endpoints. This bypasses the Kubernetes service routing layer, and enables the use of advanced load balancing controls such as session affinity and maglev. For more details, see the [load balancing reference](/reference/core/load-balancer).

### Consul endpoint-level discovery

Ambassador natively integrates with [Consul](https://www.consul.io) for endpoint-level service discovery. In this mode, Ambassador obtains endpoint information from Consul. One of the primary use cases for this architecture is in hybrid cloud environments that run a mixture of Kubernetes services as well as VMs, as Consul can serve as the single global registry for all services.

## The `Resolver` resource

The `Resolver` resource is used to configure Ambassador's service discovery strategy.

### The Kubernetes Service Resolver

The Kubernetes Service Resolver configures Ambassador to use Kubernetes services. If no resolver is specified, this behavior is the default.

```yaml
---
apiVersion: getambassador.io/v2
kind: KubernetesServiceResolver
name: kubernetes-service
```

### The Kubernetes Endpoint Resolver

The Kubernetes Endpoint Resolver configures Ambassador to resolve Kubernetes endpoints. This enables the use of more [advanced load balancing configuration](/reference/core/load-balancer).

```yaml
---
apiVersion: getambassador.io/v2
kind: KubernetesEndpointResolver
name: endpoint
```

### The Consul Resolver

The Consul Resolver configures Ambassador to use Consul for service discovery.

```yaml
---
apiVersion: ambassador/v2
kind: ConsulResolver
name: consul-dc1
address: consul-server:8500
datacenter: dc1
```
- `address`: The address of your Consul server
- `datacenter`: The Consul data center your services are registered to

## Using Resolvers

Once a resolver is defined, you can use them in a given `Mapping`:


```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: qotm
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind:  Mapping
      name:  qotm_mapping
      prefix: /qotm/
      service: qotm
      load_balancer:
        policy: round_robin
      ---
      apiVersion: ambassador/v1
      kind:  Mapping
      name:  bar_mapping
      prefix: /bar/
      service: https://bar:9000
      tls: client-context
      resolver: consul-dc1
      load_balancer:
        policy: round_robin
spec:
  selector:
    service: qotm
  ports:
    - port: 80
      targetPort: http-api
  type: NodePort
```

The YAML configuration above will configure Ambassador to use Kubernetes Service Discovery  to route to the qotm Kubernetes service on requests with `prefix: /qotm/` and use Consul Service Discovery to route to the `bar` service on requests with `prefix: /bar/`.