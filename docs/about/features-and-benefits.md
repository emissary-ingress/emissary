# Features and Benefits

Key features in Ambassador include:

* Self-service mapping of public URLs to services running inside a Kubernetes cluster
* Flexible canary deployments
* Kubernetes-native architecture (also, no need for a dedicated Kubernetes ingress controller)
* First class gRPC and HTTP/2 support
* Istio integration
* Authentication
* Integrated diagnostics
* Robust TLS support, including TLS client-certificate authentication
* Simple setup and configuration
* Integrated monitoring
* Open source


## Self-Service via Kubernetes Annotations

Ambassador is built from the start to support _self-service_ deployments -- a developer working on a new service doesn't have to go to Operations to get their service added to the mesh, they can do it themselves in a matter of seconds. Likewise, a developer can remove their service from the mesh, or merge services, or separate services, as needed, at their convenience. All of these operations are performed via Kubernetes annotations, so it can easily integrate with your existing development workflow.

## Flexible Canary Deployments

Canary deployments are an essential component of cloud-native development workflows. In a canary deployment, a small percentage of production traffic is routed to a new version of a service to test it under real-world conditions. Ambassador allows developers to easily control and manage the amount of traffic routed to a given service through annotations. [This tutorial](https://www.datawire.io/faster/canary-workflow/) covers a complete canary workflow using Ambassador.

## Kubernetes-Native Architecture

Ambassador relies entirely on Kubernetes for reliability, availability, and scalability. For example, Ambassador persists all state in Kubernetes, instead of requiring a separate database. Scaling Ambassador is as simple as changing the replicas in your deployment, or using a [horizontal pod autoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/).

Ambassador uses [Envoy](https://www.envoyproxy.io) for all traffic routing and proxying. Envoy is a modern L7 proxy that is used in production at companies including Lyft, Apple, Google, and Stripe.

## gRPC and HTTP/2 Support

Ambassador fully supports gRPC and HTTP/2 routing, thanks to Envoy's extensive capabilities in this area. See [gRPC and Ambassador](/user-guide/grpc) for more information.

## Istio Integration

Ambassador integrates with the [Istio](https://istio.io) service mesh as the edge proxy. In this configuration, Ambassador routes external traffic to the internal Istio service mesh. See [Istio and Ambassador](/user-guide/with-istio) for details.

## Authentication

Ambassador supports authenticating incoming requests. When configured, Ambassador will check with a third party authentication service prior to routing an incoming request. For more information, see the [authentication tutorial](/user-guide/auth-tutorial).

## Integrated Diagnostics

Ambassador includes a diagnostics service so that you can quickly debug issues associated with configuring Ambassador. For more information, see [running Ambassador](https://www.getambassador.io/reference/running).
