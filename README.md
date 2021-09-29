Emissary-Ingress (fka Ambassador API Gateway) [![Build Status][build-status]][build-pages] [![CII Best Practices][cii-badge]][cii-status]
 [![Docker Repository][docker-latest]][docker-repo] ![Docker Pulls][docker-pulls] [![Join Slack][slack-join]][slack-url]
==========

[build-pages]:   https://travis-ci.org/datawire/ambassador
[build-status]:  https://travis-ci.org/datawire/ambassador.png?branch=master
[cii-badge]:     https://bestpractices.coreinfrastructure.org/projects/1852/badge
[cii-status]:    https://bestpractices.coreinfrastructure.org/projects/1852
[docker-repo]:   https://hub.docker.com/repository/docker/datawire/ambassador
[docker-latest]: https://img.shields.io/docker/v/datawire/ambassador?sort=semver
[docker-pulls]:  https://img.shields.io/docker/pulls/datawire/ambassador
[slack-url]:     https://a8r.io/slack
[slack-join]:    https://img.shields.io/badge/slack-join-orange.svg

Emissary-Ingress (formerly known as the [Ambassador API Gateway](https://www.getambassador.io)) is an open-source Kubernetes-native API Gateway + Layer 7 load balancer + Kubernetes Ingress built on [Envoy Proxy](https://www.envoyproxy.io). Emissary Ingress is an CNCF incubation project.

The Ambassador Edge Stack is a complete superset of the OSS Emissary Ingress project that offers additional functionality. Edge Stack is designed to easily expose, secure, and manage traffic to your Kubernetes microservices of any type. Edge Stack was built around the ideas of self-service (enabling GitOps-style management) and comprehensiveness (so it works with your situations and technology solutions).

Emissary Ingress enables its users to:

* Manage ingress traffic with [load balancing](https://www.getambassador.io/docs/emissary/latest/topics/running/load-balancer/), protocol support([gRPC and HTTP/2](https://www.getambassador.io/docs/emissary/latest/howtos/grpc/), [TCP](https://www.getambassador.io/docs/emissary/latest/topics/using/tcpmappings/), and [web sockets](https://www.getambassador.io/docs/emissary/latest/topics/using/tcpmappings/)), and Kubernetes integration
* Manage changes to routing with an easy to use declarative policy engine and [self-service configuration](https://www.getambassador.io/docs/emissary/latest/topics/using/mappings/), via Kubernetes [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) or annotations
* Secure microservices with [authentication](https://www.getambassador.io/docs/emissary/latest/topics/running/services/auth-service/), rate limiting, [TLS](https://www.getambassador.io/docs/emissary/latest/howtos/tls-termination/), and [automatic HTTPS](https://www.getambassador.io/docs/emissary/latest/topics/running/host-crd/)
* Ensure high availability with [sticky sessions](https://www.getambassador.io/docs/emissary/latest/topics/running/load-balancer/#sticky-sessions--session-affinity), [rate limiting](https://www.getambassador.io/docs/emissary/latest/topics/running/services/rate-limit-service/), and [circuit breaking](https://www.getambassador.io/docs/emissary/latest/topics/using/circuit-breakers/)
* Leverage observability with integrations with [Grafana](https://www.getambassador.io/docs/emissary/latest/topics/running/statistics/#grafana), [Prometheus](https://www.getambassador.io/docs/emissary/latest/topics/running/statistics/#prometheus), and [Datadog](https://www.getambassador.io/docs/emissary/latest/topics/running/statistics/#datadog), and comprehensive [metrics](https://www.getambassador.io/docs/emissary/latest/topics/running/statistics/) support
* Enable progressive delivery with [canary releases](https://www.getambassador.io/docs/emissary/latest/topics/using/canary/)
* Connect service meshes including [Consul](https://www.getambassador.io/docs/emissary/latest/howtos/consul/), [Linkerd](https://www.getambassador.io/docs/emissary/latest/howtos/linkerd2/), and [Istio](https://www.getambassador.io/docs/emissary/latest/howtos/istio/)
* [Knative serverless integration](https://www.getambassador.io/docs/emissary/latest/howtos/knative/)

See the full list of [features](https://www.getambassador.io/features/) here. Learn [Why the Ambassador Edge Stack?](https://www.getambassador.io/docs/emissary/latest/about/why-ambassador/#why-the-ambassador-edge-stack)


Architecture
============

Ambassador deploys the Envoy Proxy for L7 traffic management. Configuration of Ambassador is via Kubernetes annotations. Ambassador relies on Kubernetes for scaling and resilience. For more on Ambassador's architecture and motivation, read [this blog post](https://blog.getambassador.io/building-ambassador-an-open-source-api-gateway-on-kubernetes-and-envoy-ed01ed520844).

Getting Started
===============

You can get Ambassador up and running in just three steps. Follow the instructions here: https://www.getambassador.io/docs/emissary/latest/tutorials/getting-started/


If you are looking for a Kubernetes ingress controller, Ambassador provides a superset of the functionality of a typical ingress controller. (It does the traditional routing, and layers on a raft of configuration options.) This blog post covers [Kubernetes ingress](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d).

For other common questions, view this [FAQ page](https://www.getambassador.io/docs/emissary/latest/about/faq/).

You can also use Helm to install Ambassador. For more information, see the instructions in the [Helm installation documentation](https://www.getambassador.io/docs/emissary/latest/topics/install/helm/)

Community
=========

Ambassador is an open-source project, and welcomes any and all contributors. To get started:

* Join our [Slack channel](https://d6e.co/slack)
* Check out the [Ambassador documentation](https://www.getambassador.io/docs/emissary/)
* Read the [Contributor's Guide](https://github.com/datawire/ambassador/blob/master/DEVELOPING.md).

If you're interested in contributing, here are some ways:

* Write a blog post for [our blog](https://blog.getambassador.io)
* Investigate an [open issue](https://github.com/datawire/ambassador/issues)
* Add [more tests](https://github.com/datawire/ambassador/tree/master/ambassador/tests)

The Ambassador Edge Stack is a superset of the Ambassador API Gateway that provides additional functionality including OAuth/OpenID Connect, advanced rate limiting, Swagger/OpenAPI support, integrated ACME support for automatic TLS certificate management, and a UI. For more information, visit https://www.getambassador.io/editions/.

