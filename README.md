Emissary-ingress
================

<!-- [![Alt Text][image-url]][link-url] -->
[![Version][badge-version-img]][badge-version-link]
[![Docker Repository][badge-docker-img]][badge-docker-link]
[![Join Slack][badge-slack-img]][badge-slack-link]
[![Core Infrastructure Initiative: Best Practices][badge-cii-img]][badge-cii-link]

[badge-version-img]: https://img.shields.io/docker/v/emissaryingress/emissary?sort=semver
[badge-version-link]: https://github.com/emissary-ingress/emissary/releases
[badge-docker-img]: https://img.shields.io/docker/pulls/emissaryingress/emissary
[badge-docker-link]: https://hub.docker.com/r/emissaryingress/emissary
[badge-slack-img]: https://img.shields.io/badge/slack-join-orange.svg
[badge-slack-link]: https://a8r.io/slack
[badge-cii-img]: https://bestpractices.coreinfrastructure.org/projects/1852/badge
[badge-cii-link]: https://bestpractices.coreinfrastructure.org/projects/1852

<!-- Links are (mostly) at the end of this document, for legibility. -->

[Emissary-Ingress](https://www.getambassador.io) is an open-source Kubernetes-native API Gateway +
Layer 7 load balancer + Kubernetes Ingress built on [Envoy Proxy](https://www.envoyproxy.io).
Emissary-ingress is a CNCF incubation project (and was formerly known as Ambassador API Gateway).

Emissary-ingress enables its users to:
* Manage ingress traffic with [load balancing], support for multiple protocols ([gRPC and HTTP/2], [TCP], and [web sockets]), and Kubernetes integration
* Manage changes to routing with an easy to use declarative policy engine and [self-service configuration], via Kubernetes [CRDs] or annotations
* Secure microservices with [authentication], [rate limiting], and [TLS]
* Ensure high availability with [sticky sessions], [rate limiting], and [circuit breaking]
* Leverage observability with integrations with [Grafana], [Prometheus], and [Datadog], and comprehensive [metrics] support
* Enable progressive delivery with [canary releases]
* Connect service meshes including [Consul], [Linkerd], and [Istio]
* [Knative serverless integration]

See the full list of [features](https://www.getambassador.io/features/) here.

Branches
========

(If you are looking at this list on a branch other than `master`, it
may be out of date.)

- [`master`](https://github.com/emissary-ingress/emissary/tree/master) - branch for Emissary-ingress dev work (:heavy_check_mark: upcoming release)
- [`release/v3.8`](https://github.com/emissary-ingress/emissary/tree/release/v3.8) - branch for Emissary-ingress 3.8.z work
- [`release/v2.5`](https://github.com/emissary-ingress/emissary/tree/release/v2.5) - branch for Emissary-ingress 2.5.z work (:heavy_check_mark: maintenance)

Architecture
============

Emissary is configured via Kubernetes CRDs, or via annotations on Kubernetes `Service`s. Internally,
it uses the [Envoy Proxy] to actually handle routing data; externally, it relies on Kubernetes for
scaling and resiliency. For more on Emissary's architecture and motivation, read [this blog post](https://blog.getambassador.io/building-ambassador-an-open-source-api-gateway-on-kubernetes-and-envoy-ed01ed520844).

Getting Started
===============

You can get Emissary up and running in just three steps. Follow the instructions here: https://www.getambassador.io/docs/emissary/latest/tutorials/getting-started/

If you are looking for a Kubernetes ingress controller, Emissary provides a superset of the functionality of a typical ingress controller. (It does the traditional routing, and layers on a raft of configuration options.) This blog post covers [Kubernetes ingress](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d).

For other common questions, view this [FAQ page](https://www.getambassador.io/docs/emissary/latest/about/faq/).

You can also use Helm to install Emissary. For more information, see the instructions in the [Helm installation documentation](https://www.getambassador.io/docs/emissary/latest/topics/install/helm/)

Check out full the [Emissary
documentation](https://www.getambassador.io/docs/emissary/) at
www.getambassador.io.

Community
=========

Emissary-ingress is a CNCF Incubating project and welcomes any and all
contributors.

Check out the [`Community/`](Community/) directory for information on
the way the community is run, including:

 - the [`CODE_OF_CONDUCT.md`](Community/CODE_OF_CONDUCT.md)
 - the [`GOVERNANCE.md`](Community/GOVERNANCE.md) structure
 - the list of [`MAINTAINERS.md`](Community/MAINTAINERS.md)
 - the [`MEETING_SCHEDULE.md`](Community/MEETING_SCHEDULE.md) of
   regular trouble-shooting meetings and contributor meetings
 - how to get [`SUPPORT.md`](Community/SUPPORT.md).

The best way to join the community is to join our [Slack
channel](https://a8r.io/slack).

Check out the [`DevDocumentation/`](DevDocumentation/) directory for
information on the technicals of Emissary, most notably the
[`DEVELOPING.md`](DevDocumentation/DEVELOPING.md) contributor's guide.

If you're interested in contributing, here are some ways:
* Write a blog post for [our blog](https://blog.getambassador.io)
* Investigate an [open issue](https://github.com/emissary-ingress/emissary/issues)
* Add [more tests](https://github.com/emissary-ingress/emissary/tree/master/ambassador/tests)

The Ambassador Edge Stack is a superset of Emissary-ingress that provides additional functionality including OAuth/OpenID Connect, advanced rate limiting, Swagger/OpenAPI support, integrated ACME support for automatic TLS certificate management, and a cloud-based UI. For more information, visit https://www.getambassador.io/editions/.

<!-- Please keep this list sorted. -->
[authentication]: https://www.getambassador.io/docs/emissary/latest/topics/running/services/auth-service/
[canary releases]: https://www.getambassador.io/docs/emissary/latest/topics/using/canary/
[circuit breaking]: https://www.getambassador.io/docs/emissary/latest/topics/using/circuit-breakers/
[Consul]: https://www.getambassador.io/docs/emissary/latest/howtos/consul/
[CRDs]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
[Datadog]: https://www.getambassador.io/docs/emissary/latest/topics/running/statistics/#datadog
[Grafana]: https://www.getambassador.io/docs/emissary/latest/topics/running/statistics/#grafana
[gRPC and HTTP/2]: https://www.getambassador.io/docs/emissary/latest/howtos/grpc/
[Istio]: https://www.getambassador.io/docs/emissary/latest/howtos/istio/
[Knative serverless integration]: https://www.getambassador.io/docs/emissary/latest/howtos/knative/
[Linkerd]: https://www.getambassador.io/docs/emissary/latest/howtos/linkerd2/
[load balancing]: https://www.getambassador.io/docs/emissary/latest/topics/running/load-balancer/
[metrics]: https://www.getambassador.io/docs/emissary/latest/topics/running/statistics/
[Prometheus]: https://www.getambassador.io/docs/emissary/latest/topics/running/statistics/#prometheus
[rate limiting]: https://www.getambassador.io/docs/emissary/latest/topics/running/services/rate-limit-service/
[self-service configuration]: https://www.getambassador.io/docs/emissary/latest/topics/using/mappings/
[sticky sessions]: https://www.getambassador.io/docs/emissary/latest/topics/running/load-balancer/#sticky-sessions--session-affinity
[TCP]: https://www.getambassador.io/docs/emissary/latest/topics/using/tcpmappings/
[TLS]: https://www.getambassador.io/docs/emissary/latest/howtos/tls-termination/
[web sockets]: https://www.getambassador.io/docs/emissary/latest/topics/using/tcpmappings/
