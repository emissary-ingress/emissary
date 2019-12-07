# Why the Ambassador Edge Stack?

When we talk about Kubernetes microservices, there is the age old discussion of monolith vs microservice architecture, migrating to the cloud, and decentralizing configurations (among countless other things).

So let’s imagine we’re already on the cloud with an application, and we’ve implemented the microservices architecture and enjoy the benefits of all Kubernetes has to offer us. But, there is one problem: the more an application grows, more and more of its microservices are exposed at the “edge,” requiring individual configuration changes for each of those microservices.

For example, when you had “the monolith,” you deployed weekly. With five microservices that can deploy daily, you now have 25x increase in edge configuration changes. How do you scale edge operations as more services are connected to the edge?

Or, think about it this way: what if all of your microservices have different needs? For example, one might need HTTP and rate limiting, while the other requires gRPC and authentication.  How does your edge support the diverse requirements of all your microservices?

Normally, to combat this frenzy, you could deploy an API Gateway and an Ingress Controller- but that’s still one thing too many. Why not make it even easier?

## Enter the Ambassador Edge Stack

The Ambassador Edge Stack is engineered for cloud-native applications andprovides you with a single, comprehensive self-service solution for your Kubernetes cluster. The self-service nature allows app devs to configure the edge, and op erators to set and enforce global policies. It’s decentralized as well, allowing multiple teams to independently configure different parts of Ambassador.

Plus, there’s an interface for you to manage you Ambassador Edge Stack instance if you don’t want to use the command line. The Edge Policy Console supports a fully “round trip” creation of CRDs, such as hosts and mappings, and contains a Developer Portal for you to configure your own API documentation. These features, along with all the rest, are visually displayed so you know exactly what you’re doing.
[Check out the Edge Policy Console](/about/edge-policy-console) and all it has to offer.


The Ambassador Edge Stack is an open source, Kubernetes-native [microservices API gateway](/about/microservices-api-gateways) built on the [Envoy Proxy](https://www.envoyproxy.io). The Ambassador Edge Stack is built from the ground up to support multiple, independent teams that need to rapidly publish, monitor, and update services for end users. Ambassador can also be used to handle the functions of a Kubernetes ingress controller and load balancer (for more, see [this blog post](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d)).

Or, read the documentation on using the Ambassador Edge Stack as an [Ingress Controller](/reference/core/ingress-controller).

## How does Ambassador Work?

The Ambassador Edge Stack is an open source, Kubernetes-native [microservices API gateway](/about/microservices-api-gateways) built on the [Envoy Proxy](https://www.envoyproxy.io). The Ambassador Edge Stack is built from the ground up to support multiple, independent teams that need to rapidly publish, monitor, and update services for end users. Ambassador can also be used to handle the functions of a Kubernetes ingress controller and load balancer (for more, see [this blog post](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d)).

Or, read the documentation on using the Ambassador Edge Stack as an [Ingress Controller](/reference/core/ingress-controller).

## Cloud-native applications today

Traditional cloud applications were built as using a monolithic approach. These applications were designed, coded, and deployed as a single unit. Today's cloud-native applications, by contrast, consist of many individual (micro)services. This results in an architecture that is:

* __Heterogeneous__: Services are implemented using multiple (polyglot) languages, they are designed using multiple architecture styles, and they communicate with each other over multiple protocols.
* __Dynamic__: Services are frequently updated and released (often without coordination), which results in a constantly-changing application.
* __Decentralized__: Services are managed by independent product-focused teams, with different development workflows and release cadences.

### Heterogeneous services

The Ambassador Edge Stack is commonly used to route traffic to a wide variety of services. It supports:

* configuration on a *per-service* basis, enabling fine-grained control of timeouts, rate limiting, authentication policies, and more.
* a wide range of L7 protocols natively, including HTTP, HTTP/2, [gRPC](/user-guide/grpc), [gRPC-Web](https://github.com/grpc/grpc-web), and [WebSockets](/user-guide/websockets-ambassador).
* Can [route raw TCP](/reference/tcpmappings) for services that use protocols not directly supported by The Ambassador Edge Stack 

### Dynamic services

Service updates result in a constantly changing application. The dynamic nature of cloud-native applications introduce new challenges around configuration updates, release, and testing. Ambassador Edge Stack:

* Enables [testing in production](/docs/dev-guide/test-in-prod), with support for [canary routing](/reference/canary) and [traffic shadowing](/reference/shadowing).
* Exposes high resolution observability metrics, providing insight into service behavior.
* Uses a zero downtime configuration architecture, so configuration changes have no end user impact.

### Decentralized workflows

Independent teams can create their own workflows for developing and releasing functionality that are optimized for their specific service(s). With Ambassador Edge Stack, teams can:

* Leverage a [declarative configuration model](/user-guide/cd-declarative-gitops), making it easy to understand the canonical configuration and [implement GitOps-style best practices](/user-guide/gitops-ambassador).
* Independently configure different aspects of Ambassador Edge Stack, eliminating the need to request configuration changes through a centralised operations team.

## Ambassador Edge Stack is engineered for Kubernetes

Ambassador Edge Stack takes full advantage of Kubernetes and Envoy Proxy.

* All of the state required for Ambassador Edge Stack is stored directly in Kubernetes, eliminating the need for an additional database.
* The Ambassador Edge Stack team has added extensive engineering efforts and integration testing to insure optimal performance and scale of Envoy and Kubernetes.

## For more information

[Deploy Ambassador Edge Stack today](/user-guide/install) and join the community [Slack Channel](http://d6e.co/slack).

Interested in learning more?

* [Why did we start building Ambassador Edge Stack?](https://blog.getambassador.io/building-ambassador-an-open-source-api-gateway-on-kubernetes-and-envoy-ed01ed520844)
* [Ambassador Edge Stack Architecture overview](/concepts/architecture)

<GoogleStructuredData/>

