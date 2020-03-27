# Why the Ambassador Edge Stack?

The Ambassador Edge Stack gives platform engineers a comprehensive, self-service edge stack for managing the boundary between end-users and Kubernetes. Built on the [Envoy Proxy](https://www.envoyproxy.io) and fully Kubernetes-native, the Ambassador Edge Stack is made to support multiple, independent teams that need to rapidly publish, monitor, and update services for end-users. A true edge stack, Ambassador can also be used to handle the functions of an API Gateway, a Kubernetes ingress controller and a layer 7 load balancer (for more, see [this blog post](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d)).

## How Does Ambassador Work?

The Ambassador Edge Stack is an open-source, Kubernetes-native [microservices API gateway](../microservices-api-gateways) built on the [Envoy Proxy](https://www.envoyproxy.io). The Ambassador Edge Stack is built from the ground up to support multiple, independent teams that need to rapidly publish, monitor, and update services for end-users. Ambassador can also be used to handle the functions of a Kubernetes ingress controller and load balancer (for more, see [this blog post](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d)).

Or, read the documentation on using the Ambassador Edge Stack as an [Ingress Controller](../../reference/core/ingress-controller).

## Cloud-native Applications Today

Traditional cloud applications were built using a monolithic approach. These applications were designed, coded, and deployed as a single unit. Today's cloud-native applications, by contrast, consist of many individual (micro)services. This results in an architecture that is:

* __Heterogeneous__: Services are implemented using multiple (polyglot) languages, they are designed using multiple architecture styles, and they communicate with each other over multiple protocols.
* __Dynamic__: Services are frequently updated and released (often without coordination), which results in a constantly-changing application.
* __Decentralized__: Services are managed by independent product-focused teams, with different development workflows and release cadences.

### Heterogeneous Services

The Ambassador Edge Stack is commonly used to route traffic to a wide variety of services. It supports:

* configuration on a *per-service* basis, enabling fine-grained control of timeouts, rate limiting, authentication policies, and more.
* a wide range of L7 protocols natively, including HTTP, HTTP/2, [gRPC](../../user-guide/grpc), [gRPC-Web](https://github.com/grpc/grpc-web), and [WebSockets](../../user-guide/websockets-ambassador).
* Can [route raw TCP](../../reference/tcpmappings) for services that use protocols not directly supported by The Ambassador Edge Stack.

### Dynamic Services

Service updates result in a constantly changing application. The dynamic nature of cloud-native applications introduces new challenges around configuration updates, release, and testing. Ambassador Edge Stack:

* Enables [testing in production](../../docs/dev-guide/test-in-prod), with support for [canary routing](../../reference/canary) and [traffic shadowing](../../reference/shadowing).
* Exposes high-resolution observability metrics, providing insight into service behavior.
* Uses a zero downtime configuration architecture, so configuration changes have no end-user impact.

### Decentralized Workflows

Independent teams can create their own workflows for developing and releasing functionality that are optimized for their specific service(s). With Ambassador Edge Stack, teams can:

* Leverage a [declarative configuration model](../../user-guide/cd-declarative-gitops), making it easy to understand the canonical configuration and [implement GitOps-style best practices](../../user-guide/gitops-ambassador).
* Independently configure different aspects of Ambassador Edge Stack, eliminating the need to request configuration changes through a centralized operations team.

## Ambassador Edge Stack is Engineered for Kubernetes

Ambassador Edge Stack takes full advantage of Kubernetes and Envoy Proxy.

* All of the state required for Ambassador Edge Stack is stored directly in Kubernetes, eliminating the need for an additional database.
* The Ambassador Edge Stack team has added extensive engineering efforts and integration testing to ensure optimal performance and scale of Envoy and Kubernetes.

## For More Information

[Deploy Ambassador Edge Stack today](../../user-guide/install) and join the community [Slack Channel](http://d6e.co/slack).

Interested in learning more?

* [Why did we start building Ambassador Edge Stack?](https://blog.getambassador.io/building-ambassador-an-open-source-api-gateway-on-kubernetes-and-envoy-ed01ed520844)
* [Ambassador Edge Stack Architecture overview](../../concepts/architecture)

<GoogleStructuredData/>