# Operator Guide

Unlike traditional API gateways, the Ambassador Edge Stack has been designed to allow developers and operations to work independently from each other. Now, operations teams can define the global deployment and configuration policy for the application. Developers can then interact directly with the Ambassador Edge Stack without relying on operations, as a result of its self-service model.

For more information, see the [Developer Guide](../../user-guide/developers).

## Why Should Operators or Sysadmins Use Ambassador Edge Stack?

Ambassador Edge Stack allows developers to manage individual service/API deployments and frees time for operations to focus on global, cross-cutting concerns such as authentication, continuous delivery integration, rate limiting and tracing.

In more detail, Ambassador Edge Stack supports operations in the following ways:

* Is simple to [deploy and operate](../../concepts/architecture), relying entirely on Envoy and Kubernetes for routing and scaling
* Has extensive support for [TLS termination](../../user-guide/tls-termination) and [automatic HTTPS](/reference/host-crd) and redirects
* Integrated [diagnostics](../../reference/statistics) and [tracing](../../user-guide/tracing-tutorial) for troubleshooting
* Supports running multiple Ambassador Edge Stacks in a cluster, with different versions, simplifying upgrades and testing
* [Integrates with Istio](../../user-guide/with-istio), if you need a service mesh
