# Alternatives to Ambassador

Alternatives to Ambassador fall in three basic categories.

* Hosted API gateways, such as the [Amazon API gateway](https://aws.amazon.com/api-gateway/).
* Traditional API gateways, such as [Kong](https://konghq.org/).
* L7 proxies, such as [Traefik](https://traefik.io/), [NGINX](http://nginx.org/), [HAProxy](http://www.haproxy.org/), or [Envoy](https://www.envoyproxy.io), or Ingress controllers built on these proxies.

Both hosted API gateways and traditional API gateways are:

* Not self-service. The management interfaces on traditional API gateways are not designed for developer self-service, and provide limited safety and usability for developers.
* Not Kubernetes-native. They're typically configured using REST APIs, making it challenging to adopt cloud-native patterns such as GitOps and declarative configuration.
* [Designed for API management, versus microservices](/about/microservices-api-gateways).

A Layer 7 proxy can be used as an API gateway, but typically requires additional bespoke development to support microservices use cases. In fact, many API gateways package the additional features needed for an API gateway on top of a L7 proxy. Ambassador uses Envoy, while Kong uses NGINX. If you're interested in deploying Envoy directly, we've written an [introductory tutorial](https://www.datawire.io/guide/traffic/getting-started-lyft-envoy-microservices-resilience/).

## Istio

[Istio](https://istio.io) is an open source service mesh, built on Envoy. A service mesh is designed to manage east/west traffic, while an API gateway manages north/south traffic. Documentation on how to deploy Ambassador with Istio is [here](/user-guide/with-istio). In general, we've found that north/south traffic is quite different from east/west traffic (i.e., you don't control the client in the North/South use case).
