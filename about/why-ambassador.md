# Why Ambassador?

Ambassador is an open source, Kubernetes-native [microservices API gateway](/about/microservices-api-gateways) built on the [Envoy Proxy](https://www.envoyproxy.io). Ambassador is built from the ground up to support multiple, independent teams that need to rapidly publish, monitor, and update services for end users. Ambassador can also be used to handle the functions of a Kubernetes ingress controller and load balancer (for more, see [this blog post](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d)).

## Cloud-native applications today

Traditional cloud applications were built as using a monolithic approach. These applications were designed, coded, and deployed as a single unit. Today's cloud-native applications, by contrast, consist of many individual (micro)services. This results in an architecture that is:

* __Heterogeneous__: Services are implemented using multiple (polyglot) languages, they are designed using multiple architecture styles, and they communicate with each other over multiple protocols.
* __Dynamic__: Services are frequently updated and released (often without coordination), which results in a constantly-changing application.
* __Decentralized__: Services are managed by independent product-focused teams, with different development workflows and release cadences.

## Ambassador and cloud-native applications

Ambassador is engineered for cloud-native applications.

### Heterogeneous services

Ambassador is commonly used to route traffic to a wide variety of services. Ambassador:

* Supports configuration on a *per-service* basis, enabling fine-grained control of timeouts, rate limiting, authentication policies, and more.
* Natively supports a wide range of L7 protocols, including HTTP, HTTP/2, [gRPC](/user-guide/grpc), [gRPC-Web](https://github.com/grpc/grpc-web), and [WebSockets](/user-guide/websockets-ambassador).
* Can [route raw TCP](/reference/tcpmappings) for services that use protocols not directly supported by Ambassador

### Dynamic services

Service updates result in a constantly changing application. The dynamic nature of cloud-native applications introduce new challenges around configuration updates, release, and testing. Ambassador:

* Enables [testing in production](/docs/dev-guide/test-in-prod), with support for [canary routing](/reference/canary) and [traffic shadowing](/reference/shadowing).
* Exposes high resolution observability metrics, providing insight into service behavior.
* Uses a zero downtime configuration architecture, so configuration changes have no end user impact.

### Decentralized workflows

Independent teams can create their own workflows for developing and releasing functionality that are optimized for their specific service(s). With Ambassador, teams can:

* Leverage a [declarative configuration model](/user-guide/cd-declarative-gitops), making it easy to understand the canonical configuration and [implement GitOps-style best practices](/user-guide/gitops-ambassador).
* Independently configure different aspects of Ambassador, eliminating the need to request configuration changes through a centralised operations team.

## Ambassador is engineered for Kubernetes

Ambassador takes full advantage of Kubernetes and Envoy Proxy.

* All of the state required for Ambassador is stored directly in Kubernetes, eliminating the need for an additional database.
* The Ambassador team has added extensive engineering efforts and integration testing to insure optimal performance and scale of Envoy and Kubernetes.

## For more information

[Deploy Ambassador today](https://www.getambassador.io/user-guide/install) and join the community [Slack Channel](http://d6e.co/slack).

If you're interested in commercial support and additional features, see [Ambassador Pro](https://www.getambassador.io/pro).

Interested in learning more?

* [Why did we start building Ambassador?](https://blog.getambassador.io/building-ambassador-an-open-source-api-gateway-on-kubernetes-and-envoy-ed01ed520844).
* [Ambassador Architecture overview](https://www.getambassador.io/concepts/architecture)




<script type="application/ld+json">
  {
    "@context": "http://schema.org/",
    "@type": "SoftwareApplication",
    "name": "Ambassador API Gateway",
    "description": "Ambassador, open source, Kubernetes-native API Gateway for microservices built on the Envoy Proxy.",
    "applicationCategory": "Cloud Software",
    "applicationSubCategory": "API Gateway",
    "operatingSystem": "Kubernetes 1.6 or later"
    "downloadUrl": "https://www.getambassador.io/",
    "author": "Datawire",
    "version": "0.39",
    "offers": {
      "@type": "Offer",
      "priceCurrency": "USD",
      "price": "0.00"
    }
  }
</script>
