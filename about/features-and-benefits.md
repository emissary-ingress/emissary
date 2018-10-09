# Features and Benefits

In cloud-native organizations, developers frequently take on responsibility for the full development lifecycle of a service, from development to QA to operations. Ambassador was especially designed for these organizations where developers have operational responsibility for their service(s).

As such, Ambassador is designed to be used by both developers and operators.

## Developers

For developers, Ambassador:

* Enables publishing a service publicly without operational help
* Fine-grained control of routing, with support for regex-based routing, host routing, and more
* Authentication
* Support for gRPC and HTTP/2
* Canary releases
* Shadow traffic
* Transparent monitoring of L7 traffic to given services

## Operators

For operators, Ambassador:

* Is simple to deploy and operate, relying entirely on Envoy and Kubernetes for routing and scaling
* Has extensive support for TLS termination and redirects
* Integrated diagnostics for troubleshooting
* Supports running multiple Ambassadors in a cluster, with different versions, simplifying upgrades and testing
* Integrates with Istio, if you need a service mesh

## Details

More details about some of the features of Ambassador are discussed below.

### Self-Service via Kubernetes Annotations

Ambassador is built from the start to support _self-service_ deployments -- a developer working on a new service doesn't have to go to Operations to get their service added to the mesh, they can do it themselves in a matter of seconds. Likewise, a developer can remove their service from the mesh, or merge services, or separate services, as needed, at their convenience. All of these operations are performed via Kubernetes annotations, so it can easily integrate with your existing development workflow.

### Flexible Canary Deployments

Canary deployments are an essential component of cloud-native development workflows. In a canary deployment, a small percentage of production traffic is routed to a new version of a service to test it under real-world conditions. Ambassador allows developers to easily control and manage the amount of traffic routed to a given service through annotations. [This tutorial](https://www.datawire.io/faster/canary-workflow/) covers a complete canary workflow using Ambassador.

### Kubernetes-Native Architecture

Ambassador relies entirely on Kubernetes for reliability, availability, and scalability. For example, Ambassador persists all state in Kubernetes, instead of requiring a separate database. Scaling Ambassador is as simple as changing the replicas in your deployment, or using a [horizontal pod autoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/).

Ambassador uses [Envoy](https://www.envoyproxy.io) for all traffic routing and proxying. Envoy is a modern L7 proxy that is used in production at companies including Lyft, Apple, Google, and Stripe.

### gRPC and HTTP/2 Support

Ambassador fully supports gRPC and HTTP/2 routing, thanks to Envoy's extensive capabilities in this area. See [gRPC and Ambassador](/user-guide/grpc.md) for more information.

### Istio Integration

Ambassador integrates with the [Istio](https://istio.io) service mesh as the edge proxy. In this configuration, Ambassador routes external traffic to the internal Istio service mesh. See [Istio and Ambassador](/user-guide/with-istio.md) for details.

### Authentication

Ambassador supports authenticating incoming requests. When configured, Ambassador will check with a third party authentication service prior to routing an incoming request. For more information, see the [authentication tutorial](/user-guide/auth-tutorial.md).

### Rate Limiting

Ambassador supports rate limiting incoming requests. When configured, Ambassador will check with a third party rate limit service prior to routing an incoming request. For more information, see the [rate limiting tutorial](/user-guide/rate-limiting-tutorial.md).

### Integrated Diagnostics

Ambassador includes a diagnostics service so that you can quickly debug issues associated with configuring Ambassador. For more information, see [running Ambassador](https://www.getambassador.io/reference/running).
