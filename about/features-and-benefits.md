# Features and Benefits

In cloud-native organizations, developers frequently take on responsibility for the full development lifecycle of a service, from development to QA to operations. Ambassador Edge Stack was especially designed for these organizations where developers have operational responsibility for their service(s).

As such, Ambassador Edge Stack is designed to be used by both developers and operators.

## Self-Service via Kubernetes Annotations

Ambassador Edge Stack is built from the start to support _self-service_ deployments -- a developer working on a new service doesn't have to go to Operations to get their service added to the mesh, they can do it themselves in a matter of seconds. Likewise, a developer can remove their service from the mesh, or merge services, or separate services, as needed, at their convenience. All of these operations are performed via Kubernetes annotations, so it can easily integrate with your existing development workflow.

## Flexible Canary Deployments

Canary deployments are an essential component of cloud-native development workflows. In a canary deployment, a small percentage of production traffic is routed to a new version of a service to test it under real-world conditions. Ambassador Edge Stack allows developers to easily control and manage the amount of traffic routed to a given service through annotations. [This tutorial](https://www.datawire.io/faster/canary-workflow/) covers a complete canary workflow using Ambassador Edge Stack.

## Kubernetes-Native Architecture

Ambassador Edge Stack relies entirely on Kubernetes for reliability, availability, and scalability. For example, Ambassador persists all state in Kubernetes, instead of requiring a separate database. Scaling Ambassador Edge Stack is as simple as changing the replicas in your deployment, or using a [horizontal pod autoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/).

Ambassador Edge Stack uses [Envoy](https://www.envoyproxy.io) for all traffic routing and proxying. Envoy is a modern L7 proxy that is used in production at companies including Lyft, Apple, Google, and Stripe.

## gRPC and HTTP/2 Support

Ambassador Edge Stack fully supports gRPC and HTTP/2 routing, thanks to Envoy's extensive capabilities in this area. See [gRPC and Ambassador Edge Stack](/user-guide/grpc) for more information.

## Istio Integration

Ambassador Edge Stack integrates with the [Istio](https://istio.io) service mesh as the edge proxy. In this configuration, Ambassador Edge Stack routes external traffic to the internal Istio service mesh. See [Istio and Ambassador Edge Stack](/user-guide/with-istio) for details.

## Authentication

Ambassador Edge Stack supports authenticating incoming requests. When configured, Ambassador Edge Stack will check with a third party authentication service prior to routing an incoming request. For more information, see the [authentication tutorial](/user-guide/auth-tutorial).

## Rate Limiting

Ambassador Edge Stack supports rate limiting incoming requests. When configured, Ambassador Edge Stack will check with a third party rate limit service prior to routing an incoming request. For more information, see the [rate limiting tutorial](/user-guide/rate-limiting-tutorial).

## Integrated Diagnostics

Ambassador Edge Stack includes a diagnostics service so that you can quickly debug issues associated with configuring Ambassador Edge Stack. For more information, see [running Ambassador Edge Stack](/reference/running).
