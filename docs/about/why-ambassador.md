# Why Ambassador?

Ambassador is an open source, Kubernetes-native [microservices API gateway](/about/microservices-api-gateways) built on the [Envoy Proxy](https://www.envoyproxy.io). Ambassador is built from the ground up to support multiple, independent teams that need to rapidly publish, monitor, and update services for end users. Ambassador can also be used to handle the functions of a Kubernetes ingress controller and load balancer (for more, see [this blog post](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d)).

Ambassador is:

* Self-service. Ambassador is designed so that developers can manage services directly. This requires a system that is not only easy for developers to use, but provides safety and protection against inadvertent operational issues.
* Operations friendly. Ambassador has virtually no moving parts, and delegates all routing and resilience to [Envoy Proxy](https://www.envoyproxy.io) and Kubernetes, respectively. Ambassador stores all state in Kubernetes (no database!). Multiple Ambassadors can be run in the same cluster, making upgrades easy and seamless.
* Designed for microservices. Ambassador integrates the features teams need for microservices, including authentication, rate limiting, observability, routing, TLS termination, and more.
* Open Source. Ambassador is an open source API Gateway. Install it now for free and join the community [Slack Channel](http://d6e.co/slack).

For more background on the motivations of Ambassador, read [this blog post](https://blog.getambassador.io/building-ambassador-an-open-source-api-gateway-on-kubernetes-and-envoy-ed01ed520844).



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
