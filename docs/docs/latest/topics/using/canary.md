# Canary Releases

Canary releasing is a deployment pattern where a small percentage of traffic is diverted to an early ("canary") release of a particular service. This technique lets you test a release on a small subset of users, mitigating the impact of any given bug. Canary releasing also allows you to quickly roll back to a known good version in the event of an unexpected error. Detailed monitoring of core service metrics is an essential part of canary releasing, as monitoring enables the rapid detection of problems in the canary release.

## Canary releases in Kubernetes

Kubernetes supports a basic canary release workflow using its core objects. In this workflow, a service owner can create a Kubernetes [service](https://kubernetes.io/docs/concepts/services-networking/service/). This service can then be pointed to multiple [deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/). Each deployment can be a different version. By specifying the number of `replicas` in a given deployment, you can control how much traffic goes between different versions. For example, you could set `replicas: 3` for `v1`, and `replicas: 1` for `v2`, to ensure that 25% of traffic goes to `v2`. This approach works but is fairly coarse-grained unless you have lots of replicas. Moreover, auto-scaling doesn't work well with this strategy.

## Canary Releases in Ambassador Edge Stack

Ambassador Edge Stack supports fine-grained canary releases. Ambassador Edge Stack uses a weighted round-robin scheme to route traffic between multiple services. Full metrics are collected for all services, making it easy to compare the relative performance of the canary and production.

### The `weight` Attribute

The `weight` attribute specifies how much traffic for a given resource will be routed using a given mapping. Its value is an integer percentage between 0 and 100. Ambassador Edge Stack will balance weights to make sure that, for every resource, the mappings for that resource will have weights adding to 100%. (In the simplest case, a single mapping is guaranteed to receive 100% of the traffic no matter whether it's assigned a `weight` or not.)

Specifying a weight only makes sense if you have multiple mappings for the same resource, and typically you would _not_ assign a weight to the "default" mapping (the mapping expected to handle most traffic): letting Ambassador Edge Stack assign that mapping all the traffic not otherwise spoken for tends to make life easier when updating weights.

Here's an example, which might appear during a canary deployment:

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  quote-backend
spec:
  prefix: /backend/
  service: quote
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  quote-backend2
spec:
  prefix: /backend/
  service: quotev2
  weight: 10
```

In this case, the quote-backend2 will receive 10% of the requests for `/backend/`, and Ambassador will assign the remaining 90% to the quote-backend.
