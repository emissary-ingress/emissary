Ambassador [![Build Status](https://travis-ci.org/datawire/ambassador.png?branch=master)](https://travis-ci.org/datawire/ambassador) [![Docker Repository](https://quay.io/repository/datawire/ambassador/status "Docker Repository")](https://quay.io/repository/datawire/ambassador) [![Join Slack](https://img.shields.io/badge/slack-join-orange.svg)](https://d6e.co/slack)
==========

[Ambassador](https://www.getambassador.io) API Gateway is an open-source Kubernetes-native API Gateway + Layer 7 load balancer + Kubernetes Ingress built on [Envoy Proxy](https://www.envoyproxy.io). The Ambassador Edge Stack is a complete superset of the OSS Ambassador API Gateway that offers additional functionality. Ambassador is designed to easily expose, secure, and manage traffic to your Kubernetes microservices of any type. Ambassador was built around the ideas of self-service (enabling GitOps-style management) and comprehensiveness (so it works with your situations and technology solutions). 

The Ambassador API Gateway enables its users to:

* Manage ingress traffic with [load balancing](https://www.getambassador.io/docs/latest/topics/running/load-balancer/#load-balancing-in-ambassador-edge-stack), protocol support([gRPC and HTTP/2](https://www.getambassador.io/docs/latest/howtos/grpc/), [TCP](https://www.getambassador.io/docs/latest/topics/using/tcpmappings/), and [web sockets](https://www.getambassador.io/docs/latest/topics/using/tcpmappings/)), and Kubernetes integration
* Manage changes to routing with an easy to use declarative policy engine and [self-service configuration](https://www.getambassador.io/docs/latest/topics/using/mappings/), via Kubernetes [CRDs](https://www.getambassador.io/docs/latest/topics/using/edge-policy-console/) or annotations 
* Secure microservices with [authentication](https://www.getambassador.io/docs/latest/topics/running/services/auth-service/), rate limiting, [TLS](https://www.getambassador.io/docs/latest/howtos/tls-termination/), [automatic HTTPS](https://www.getambassador.io/docs/latest/topics/running/host-crd/), and [custom request fiters](https://www.getambassador.io/docs/latest/howtos/filter-dev-guide/#developing-custom-filters-for-routing)
* Ensure high availability with [sticky sessions](https://www.getambassador.io/docs/latest/topics/running/load-balancer/#sticky-sessions--session-affinity), [rate limiting](https://www.getambassador.io/docs/latest/topics/running/services/rate-limit-service/), and [circuit breaking](https://www.getambassador.io/docs/latest/topics/using/circuit-breakers/)
* Leverage observability with integrations with [Grafana](https://www.getambassador.io/docs/latest/topics/running/statistics/#grafana), [Prometheus](https://www.getambassador.io/docs/latest/topics/running/statistics/#prometheus), and [Datadog](https://www.getambassador.io/docs/latest/topics/running/statistics/#datadog), and comprehensive [metrics](https://www.getambassador.io/docs/latest/topics/running/statistics/) support
* Set up shared development enviornments with [Service Preview](https://www.getambassador.io/docs/latest/topics/using/edgectl/)
* Onboard developers with a [Developer Portal](https://www.getambassador.io/docs/latest/topics/using/dev-portal/)
* Enable progressive delviery with [canary releases](https://www.getambassador.io/docs/latest/topics/using/canary/)
* Connect service meshes including [Consul](https://www.getambassador.io/docs/latest/howtos/consul/), [Linkerd](https://www.getambassador.io/docs/latest/howtos/linkerd2/), and [Istio](https://www.getambassador.io/docs/latest/howtos/istio/)
* [Knative serverless integration](https://www.getambassador.io/docs/latest/howtos/knative/)

See the full list of [features](https://www.getambassador.io/features/) here. Learn [Why the Ambassador Edge Stack?](https://www.getambassador.io/docs/latest/about/why-ambassador/#why-the-ambassador-edge-stack)


Architecture
============

Ambassador deploys the Envoy Proxy for L7 traffic management. Configuration of Ambassador is via Kubernetes annotations. Ambassador relies on Kubernetes for scaling and resilience. For more on Ambassador's architecture and motivation, read [this blog post](https://blog.getambassador.io/building-ambassador-an-open-source-api-gateway-on-kubernetes-and-envoy-ed01ed520844).

Getting Started
===============

You can get Ambassador up and running in just three steps. Follow the instructions here: https://www.getambassador.io/docs/latest/tutorials/getting-started/.


If you are looking for a Kubernetes ingress controller, Ambassador provides a superset of the functionality of a typical ingress controller. (It does the traditional routing, and layers on a raft of configuration options.) This blog post covers [Kubernetes ingress](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d).

For other common questions, view this [FAQ page](https://www.getambassador.io/docs/latest/about/faq/).

You can also use Helm to install Ambassador. For more information, see the instructions in the [Helm installation documentation](https://www.getambassador.io/user-guide/helm).

Community
=========

Ambassador is an open-source project, and welcomes any and all contributors. To get started:

* Join our [Slack channel](https://d6e.co/slack)
* Check out the [Ambassador documentation](https://www.getambassador.io/docs/latest)
* Read the [Contributor's Guide](https://github.com/datawire/ambassador/blob/master/DEVELOPING.md). 

If you're interested in contributing, here are some ways:

* Write a blog post for [our blog](https://blog.getambassador.io)
* Investigate an [open issue](https://github.com/datawire/ambassador/issues)
* Add [more tests](https://github.com/datawire/ambassador/tree/master/ambassador/tests)

The Ambassador Edge Stack is a superset of the Ambassador API Gateway that provides additional functionality including OAuth/OpenID Connect, advanced rate limiting, Swagger/OpenAPI support, integrated ACME support for automatic TLS certificate management, and a UI. For more information, visit https://www.getambassador.io/editions/.
