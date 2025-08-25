Emissary-ingress
================

<!-- [![Alt Text][image-url]][link-url] -->
[![Version][badge-version-img]][badge-version-link]
[![Docker Repository][badge-docker-img]][badge-docker-link]
[![GHCR Repository][badge-ghcr-img]][badge-ghcr-link]
[![Join Slack][badge-slack-img]][badge-slack-link]
[![Core Infrastructure Initiative: Best Practices][badge-cii-img]][badge-cii-link]
[![Artifact HUB][badge-artifacthub-img]][badge-artifacthub-link]

[badge-version-img]: https://img.shields.io/docker/v/emissaryingress/emissary?sort=semver
[badge-version-link]: https://github.com/emissary-ingress/emissary/releases
[badge-docker-img]: https://img.shields.io/docker/pulls/emissaryingress/emissary
[badge-docker-link]: https://hub.docker.com/r/emissaryingress/emissary
[badge-ghcr-img]: https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fraw.githubusercontent.com%2Fipitio%2Fghcr-pulls%2Fmaster%2Findex.json&query=%24%5B%3F(%40.owner%3D%3D%22emissary-ingress%22%20%26%26%20%40.repo%3D%3D%22emissary%22%20%26%26%20%40.image%3D%3D%22emissary%22)%5D.pulls&logo=github&label=pulls
[badge-ghcr-link]: https://github.com/emissary-ingress/emissary/pkgs/container/emissary
[badge-slack-img]: https://img.shields.io/badge/slack-join-orange.svg
[badge-slack-link]: https://communityinviter.com/apps/cloud-native/cncf
[badge-cii-img]: https://bestpractices.coreinfrastructure.org/projects/1852/badge
[badge-cii-link]: https://bestpractices.coreinfrastructure.org/projects/1852
[badge-artifacthub-img]: https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/emissary-ingress
[badge-artifacthub-link]: https://artifacthub.io/packages/helm/datawire/emissary-ingress

<!-- Links are (mostly) at the end of this document, for legibility. -->

---

## QUICKSTART

Looking to get started as quickly as possible? Check out [the
QUICKSTART](https://emissary-ingress.dev/docs/3.10/quick-start/)!

### Latest Release

The latest production version of Emissary is **3.10.0**.

**Note well** that there is also an Ambassador Edge Stack 3.10.0, but
**Emissary 3.10 and Edge Stack 3.10 are not equivalent**. Their codebases have
diverged and will continue to do so.

---

Emissary-ingress
================

[Emissary-ingress](https://www.getambassador.io/docs/open-source) is an
open-source, developer-centric, Kubernetes-native API gateway built on [Envoy
Proxy]. Emissary-ingress is a CNCF incubating project (and was formerly known
as Ambassador API Gateway).

### Design Goals

The first problem faced by any organization trying to develop cloud-native
applications is the _ingress problem_: allowing users outside the cluster to
access the application running inside the cluster. Emissary is built around
the idea that the application developers should be able to solve the ingress
problem themselves, without needing to become Kubernetes experts and without
needing dedicated operations staff: a self-service, developer-centric workflow
is necessary to develop at scale.

Emissary is open-source, developer-centric, role-oriented, opinionated, and
Kubernatives-native.

- open-source: Emissary is licensed under the Apache 2 license, permitting use
  or modification by anyone.
- developer-centric: Emissary is designed taking the application developer
  into account first.
- role-oriented: Emissary's configuration deliberately tries to separate
  elements to allow separation of concerns between developers and operations.
- opinionated: Emissary deliberately tries to make easy things easy, even if
  that comes of the cost of not allowing some uncommon features.

### Features

Emissary supports all the table-stakes features needed for a modern API
gateway:

* Per-request [load balancing]
* Support for routing [gRPC], [HTTP/2], [TCP], and [web sockets]
* Declarative configuration via Kubernetes [custom resources]
* Fine-grained [authentication] and [authorization]
* Advanced routing features like [canary releases], [A/B testing], [dynamic routing], and [sticky sessions]
* Resilience features like [retries], [rate limiting], and [circuit breaking]
* Observability features including comprehensive [metrics] support using the [Prometheus] stack
* Easy service mesh integration with [Linkerd], [Istio], [Consul], etc.
* [Knative serverless integration]

See the full list of [features](https://www.getambassador.io/docs/emissary) here.

### Branches

(If you are looking at this list on a branch other than `master`, it
may be out of date.)

- [`main`](https://github.com/emissary-ingress/emissary/tree/main): Emissary 4 development work

**No further development is planned on any branches listed below.**

- [`master`](https://github.com/emissary-ingress/emissary/tree/master) - **Frozen** at Emissary 3.10.0
- [`release/v3.10`](https://github.com/emissary-ingress/emissary/tree/release/v3.10) - Emissary-ingress 3.10.0 release branch
- [`release/v3.9`](https://github.com/emissary-ingress/emissary/tree/release/v3.9)
  - Emissary-ingress 3.9.1 release branch
- [`release/v2.5`](https://github.com/emissary-ingress/emissary/tree/release/v2.5) - Emissary-ingress 2.5.1 release branch

**Note well** that there is also an Ambassador Edge Stack 3.10.0, but
**Emissary 3.10 and Edge Stack 3.10 are not equivalent**. Their codebases have
diverged and will continue to do so.

#### Community

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

The best way to join the community is to join the `#emissary-ingress` channel
in the [CNCF Slack]. This is also the best place for technical information
about Emissary's architecture or development.

If you're interested in contributing, here are some ways:
* Write a blog post for [our blog](https://blog.getambassador.io)
* Investigate an [open issue](https://github.com/emissary-ingress/emissary/issues)
* Add [more tests](https://github.com/emissary-ingress/emissary/tree/main/ambassador/tests)

<!-- Please keep this list sorted. -->
[CNCF Slack]: https://communityinviter.com/apps/cloud-native/cncf
[Envoy Proxy]: https://www.envoyproxy.io

<!-- Legacy: clean up these links! -->

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
