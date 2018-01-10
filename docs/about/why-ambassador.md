# Why Ambassador?

Ambassador is a Kubernetes-native [microservices API Gateway](microservices-api-gateways) built on the [Envoy Proxy](https://envoyproxy.github.io). Ambassador is built from the ground up to support multiple, independent teams that need to rapidly publish, monitor, and update services for end users.

Ambassador is:

* Self-service. Ambassador is designed so that developers can manage services directly. This requires a system that is not only easy for developers to use, but provides safety and protection against inadvertent operational issues.
* Operations friendly. Ambassador operates as a sidecar process to the [Envoy Proxy](https://envoyproxy.github.io), and integrates Envoy directly with Kubernetes. Thus, all routing, failover, health checking are handled by battle-tested, proven systems.
* Designed for microservices. Ambassador integrates the features teams need for microservices, including authentication, observability, routing, TLS termination, and more.

## Alternatives to Ambassador

Alternatives to Ambassador fall in three basic categories.

* Hosted API Gateways, such as the [Amazon API Gateway](https://aws.amazon.com/api-gateway/).
* Traditional API Gateways, such as [Kong](https://getkong.org/).
* L7 proxies, such as [Traefik](https://traefik.io/), [NGINX](http://nginx.org/), [HAProxy](http://www.haproxy.org/), or [Envoy](https://envoyproxy.github.io).

Both hosted API Gateways and traditional API gateways are:

* Not self-service. The management interfaces on traditional API gateways are not designed for developer self-service, and provide limited safety and usability for developers.
* [Designed for API management, versus microservices](microservices-api-gateways).

A Layer 7 proxy can be used as an API Gateway, but typically requires additional bespoke development to support microservices use cases. In fact, many API Gateways package the additional features needed for an API Gateway on top of a L7 proxy. Ambassador uses Envoy, while Kong uses NGINX. If you're interested in deploying Envoy directly, we've written an [introductory tutorial](https://www.datawire.io/guide/traffic/getting-started-lyft-envoy-microservices-resilience/).

### Istio

[Istio](https://istio.io) is an open source service mesh, built on Envoy. A service mesh is designed to manage east/west traffic, while an API gateway manages north/south traffic. Documentation on how to deploy Ambassador with Istio is [here](../user-guide/with-istio.md).

## Roadmap

We have an ambitious roadmap for Ambassador, and would love for your help. Check out the [Ambassador roadmap](roadmap.md) for more.
