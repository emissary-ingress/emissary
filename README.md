Emissary-ingress
================

<!-- [![Alt Text][image-url]][link-url] -->
[![Version][badge-version-img]][badge-version-link]
[![Image Size][badge-ghcr-size]][badge-ghcr-link]
[![Image Pulls][badge-ghcr-pulls]][badge-ghcr-link]
[![Join Slack][badge-slack-img]][badge-slack-link]
[![Core Infrastructure Initiative: Best Practices][badge-cii-img]][badge-cii-link]
<!-- [![Artifact HUB][badge-artifacthub-img]][badge-artifacthub-link] -->

[badge-version-img]: https://ghcr-badge.egpl.dev/emissary-ingress/emissary/latest_tag?trim=major&label=version
[badge-version-link]: https://github.com/emissary-ingress/emissary/releases
[badge-ghcr-size]: https://ghcr-badge.egpl.dev/emissary-ingress/emissary/size
[badge-ghcr-pulls]: https://img.shields.io/badge/dynamic/json?url=https://ghcr-badge.elias.eu.org/api/emissary-ingress/emissary/emissary&query=downloadCount&logo=docker&label=ghcr%20pulls&color=2496ed
[badge-ghcr-link]: https://github.com/emissary-ingress/emissary/packages
[badge-slack-img]: https://img.shields.io/badge/slack-join-orange.svg
[badge-slack-link]: https://communityinviter.com/apps/cloud-native/cncf
[badge-cii-img]: https://bestpractices.coreinfrastructure.org/projects/1852/badge
[badge-cii-link]: https://bestpractices.coreinfrastructure.org/projects/1852
<!-- [badge-artifacthub-img]: https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/emissary-ingress -->
<!-- [badge-artifacthub-link]: https://artifacthub.io/packages/helm/datawire/emissary-ingress -->

<!-- Links are (mostly) at the end of this document, for legibility. -->

[Emissary-Ingress](https://www.getambassador.io/docs/open-source) is an open-source Kubernetes-native API Gateway +
Layer 7 load balancer + Kubernetes Ingress built on [Envoy Proxy].
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

See the full list of [features](https://emissary-ingress.dev/docs/4.0/about/features-and-benefits/) here.

Branches
========

(If you are looking at this list on a branch other than `main`, it
is likely to be wrong.)

- [`main`](https://github.com/emissary-ingress/emissary/tree/main): main Emissary-ingress development work (:heavy_check_mark: active development)

- [`master`](https://github.com/emissary-ingress/emissary/tree/master): **frozen** branch for Emissary-ingress 3.10 (:heavy_check_mark: frozen)
- [`release/v2.5`](https://github.com/emissary-ingress/emissary/tree/release/v2.5): **frozen** branch for Emissary-ingress 2.5 (:heavy_check_mark: frozen)

Building Emissary 4
===================

`gmake images` should make all the images you need using
[goreleaser](https://goreleaser.com/). **Note that `.goreleaser.yaml` is a
generated file**; do not edit it directly. It's this way so that we
can use the open-source version of `goreleaser`.

If you want to update the Envoy version, edit the `ENVOY_IMAGE` value in
the `Makefile`.

Architecture
============

Emissary is configured via Kubernetes CRDs, or via annotations on
Kubernetes `Service`s. Internally, it uses the [Envoy Proxy] to actually
handle routing data; externally, it relies on Kubernetes for scaling and
resiliency.

Getting Started
===============

The fastest way to get started with Emissary is to follow the
[Quickstart]. For other common questions, check out the [Emissary FAQ] or
the full [Emissary documentation].

Community
=========

Emissary-ingress is a CNCF Incubating project and welcomes any and all
contributors.

**At this point, Emissary's original parent company has stepped back from
direct involvement in the project**, so the community is now more
important than ever. As we've said at the past few KubeCons, Emissary
needs you.

The best way to join the community is to join the [CNCF Slack] and hop
into the `#emissary-ingress` channel. Checking out [open issues] or
[contributing documentation] are both great ways to get started
contributing.

You can also check out the project's governance documentation:

 - [`CODE_OF_CONDUCT.md`](Community/CODE_OF_CONDUCT.md): the community
   code of conduct
 - [`GOVERNANCE.md`](Community/GOVERNANCE.md): the governance structure
 - [`MAINTAINERS.md`](Community/MAINTAINERS.md): the list of maintainers
 - [`MEETING_SCHEDULE.md`](Community/MEETING_SCHEDULE.md): the schedule
   of community meetings

<!-- Please keep this list sorted. -->
[authentication]: https://www.emissary-ingress.dev/docs/4.0/topics/running/services/auth-service/
[canary releases]: https://www.emissary-ingress.dev/docs/4.0/topics/using/canary/
[circuit breaking]: https://www.emissary-ingress.dev/docs/4.0/topics/using/circuit-breakers/
[CNCF Slack]: https://communityinviter.com/apps/cloud-native/cncf
[Consul]: https://www.emissary-ingress.dev/docs/4.0/howtos/consul/
[CRDs]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
[Datadog]: https://www.emissary-ingress.dev/docs/4.0/topics/running/statistics/#datadog
[Emissary documentation]: https://www.emissary-ingress.dev/docs/4.0/
[Emissary FAQ]: https://www.emissary-ingress.dev/docs/4.0/about/faq/
[Envoy Proxy]: https://www.envoyproxy.io
[Grafana]: https://www.emissary-ingress.dev/docs/4.0/topics/running/statistics/#grafana
[gRPC and HTTP/2]: https://www.emissary-ingress.dev/docs/4.0/howtos/grpc/
[Istio]: https://www.emissary-ingress.dev/docs/4.0/howtos/istio/
[Knative serverless integration]: https://www.emissary-ingress.dev/docs/4.0/howtos/knative/
[Linkerd]: https://www.emissary-ingress.dev/docs/4.0/howtos/linkerd2/
[load balancing]: https://www.emissary-ingress.dev/docs/4.0/topics/running/load-balancer/
[metrics]: https://www.emissary-ingress.dev/docs/4.0/topics/running/statistics/
[Prometheus]: https://www.emissary-ingress.dev/docs/4.0/topics/running/statistics/#prometheus
[Quickstart]: https://emissary-ingress.dev/docs/4.0/quick-start/
[rate limiting]: https://www.emissary-ingress.dev/docs/4.0/topics/running/services/rate-limit-service/
[self-service configuration]: https://www.emissary-ingress.dev/docs/4.0/topics/using/mappings/
[sticky sessions]: https://www.emissary-ingress.dev/docs/4.0/topics/running/load-balancer/#sticky-sessions--session-affinity
[TCP]: https://www.emissary-ingress.dev/docs/4.0/topics/using/tcpmappings/
[TLS]: https://www.emissary-ingress.dev/docs/4.0/howtos/tls-termination/
[web sockets]: https://www.emissary-ingress.dev/docs/4.0/topics/using/tcpmappings/
