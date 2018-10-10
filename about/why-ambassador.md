# Why Ambassador?

Ambassador is an open source, Kubernetes-native [microservices API gateway](microservices-api-gateways) built on the [Envoy Proxy](https://www.envoyproxy.io). Ambassador is built from the ground up to support multiple, independent teams that need to rapidly publish, monitor, and update services for end users. Ambassador can also be used to handle the functions of a Kubernetes ingress controller and load balancer (for more, see [this blog post](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d)).

Ambassador is:

* Self-service. Ambassador is designed so that developers can manage services directly. This requires a system that is not only easy for developers to use, but provides safety and protection against inadvertent operational issues.
* Operations friendly. Ambassador has virtually no moving parts, and delegates all routing and resilience to [Envoy Proxy](https://www.envoyproxy.io) and Kubernetes, respectively. Ambassador stores all state in Kubernetes (no database!). Multiple Ambassadors can be run in the same cluster, making upgrades easy and seamless.
* Designed for microservices. Ambassador integrates the features teams need for microservices, including authentication, rate limiting, observability, routing, TLS termination, and more.
* Open Source. Ambassador is an open source API Gateway. Install it now for free and join the community [Slack Channel](http://d6e.co/slack). 

For more background on the motivations of Ambassador, read [this blog post](https://blog.getambassador.io/building-ambassador-an-open-source-api-gateway-on-kubernetes-and-envoy-ed01ed520844).

## Alternatives to Ambassador

Alternatives to Ambassador fall in three basic categories.

* Hosted API gateways, such as the [Amazon API gateway](https://aws.amazon.com/api-gateway/).
* Traditional API gateways, such as [Kong](https://getkong.org/).
* L7 proxies, such as [Traefik](https://traefik.io/), [NGINX](http://nginx.org/), [HAProxy](http://www.haproxy.org/), or [Envoy](https://www.envoyproxy.io), or Ingress controllers built on these proxies.

Both hosted API gateways and traditional API gateways are:

* Not self-service. The management interfaces on traditional API gateways are not designed for developer self-service, and provide limited safety and usability for developers.
* Not Kubernetes-native. They're typically configured using REST APIs, making it challenging to adopt cloud-native patterns such as GitOps and declarative configuration.
* [Designed for API management, versus microservices](microservices-api-gateways).

A Layer 7 proxy can be used as an API gateway, but typically requires additional bespoke development to support microservices use cases. In fact, many API gateways package the additional features needed for an API gateway on top of a L7 proxy. Ambassador uses Envoy, while Kong uses NGINX. If you're interested in deploying Envoy directly, we've written an [introductory tutorial](https://www.datawire.io/guide/traffic/getting-started-lyft-envoy-microservices-resilience/).

### Istio

[Istio](https://istio.io) is an open source service mesh, built on Envoy. A service mesh is designed to manage east/west traffic, while an API gateway manages north/south traffic. Documentation on how to deploy Ambassador with Istio is [here](../user-guide/with-istio). In general, we've found that north/south traffic is quite different from east/west traffic (i.e., you don't control the client in the North/South use case).

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
