# CHANGELOG

## EMISSARY-INGRESS v4

[Emissary] is a Kubernetes-native, self-service, open-source API gateway
and ingress controller. It is a CNCF Incubating project, formerly known
as the Ambassador API Gateway.

## Emissary v4 Release Notes

## [4.0.0] TBD
[4.0.0]: https://github.com/emissary-ingress/emissary/compare/v3.10.0...v4.0.0-rc.0

_These release notes describe Emissary v4.0.0-rc.0._

### Quickstart

Emissary v4 supports both AMD64 and ARM64 architectures. To install
Emissary v4 using Helm, follow the instructions in the [Emissary
Quickstart](https://emissary-ingress.dev/docs/4.0/quick-start/).

Emissary provides two Helm charts:

- `ghcr.io/emissary-ingress/emissary-crds-chart` is the chart for
  Emissary's CRDs.

- `ghcr.io/emissary-ingress/emissary-ingress` is the chart for
  Emissary itself.

The Emissary project recommends using Helm to install Emissary. If you
need YAML instead, use `helm template` to generate the YAML manifests
from the Helm charts.

### Breaking Changes

- **BREAKING CHANGE**: Emissary is now built using `distroless` and Envoy
  1.36.2 with **no custom Envoy changes**, and almost all code specific
  to Ambassador Edge Stack has been removed to lessen maintenance burdens
  (with thanks to [Luke Shumaker] for jumping back in to help with
  tooling work!).

  As a result, Ambassador Edge Stack's custom error responses and
  header-case mangling features are completely unavailable in Emissary
  (which should not affect any Emissary users).

- **BREAKING CHANGE**: The Helm chart and Emissary itself now have aligned
  version numbers (e.g., use chart 4.0.0 to install Emissary 4.0.0).

- **BREAKING CHANGE**: Emissary's Helm charts are now available only from
  the GitHub Container Registry (GHCR):

- **BREAKING CHANGE**: By default, the Emissary CRD Helm chart will not
  enable the conversion webhook that supports Emissary's `v1` and `v2`
  CRDs. To enable the webhook:

  - Set `enableLegacyVersions: true` to enable the conversion webhook
    with support for `v2` CRDs.

  - Also set `enableV1: true` to additionally support `v1` CRDs. There is
    no way to support only `v1` CRDs.

- **BREAKING CHANGE**: Support for the extra metrics endpoint (set by
  supplying `--metrics-endpoint` on the command line for `diagd`) has
  been removed. This was a holdover from Ambassador Edge Stack (thanks to
  [Jeremy Dinsel] for the report!)

- **BREAKING CHANGE**: The default `--banner-endpoint` argument has been
  removed, since it was a holdover from Ambassador Edge Stack. If you
  want to use the diagnostics-banner functionality, you can now set
  `AMBASSADOR_DIAGD_BANNER_ENDPOINT` in the environment to the URL from
  which to fetch the banner.

### Changes

- Fix: Correctly distinguish Mappings that differ only by `weight`, even
  when some weights are missing (thanks, [sekar-saravanan]!).

- Fix: Generate correct cache keys for Mappings using `header_regex`
  match conditions (thanks, [sekar-saravanan]!).

- Fix: Document the `loadBalancerSourceRanges` value in the Helm chart
  (thanks, [Abhay Bothra]!).

- Fix: The Helm chart no longer sets a CPU limit on Emissary's
  deployments (thanks, [Frederic Mereu]!).

- Fix: The Helm chart's `initContainers` are now correctly indented
  (thanks, [Catalin Codreanu]!).

- Fix: The Helm chart now correctly restores the `HOST_IP` value for
  tracing providers (thanks, [Tenshin Higashi]!)

- Fix: When the conversion webhook is active and Emissary is instructed
  to wait for the webhooks at boot time, this is now completely internal
  rather than requiring an extra init container.

- Fix: When trying to use environment variables to tell the conversion
  webhook not to manage aspects of its certificate, don't invert the
  meaning of the environment variables.

- Fix: The diagnostics UI refers to Emissary (instead of Ambassador) and
  links to emissary-ingress.dev's docs.

- Feature: Emissary now supports both `arm64` and `amd64` architectures
  using multiarch Docker images.

- Feature: When the `emissary-apiext` deployment is in use, the Helm
  chart correctly sets its `securityContext` (thanks, [Frederic Mereu]!).

- Feature: The Helm chart can now supply a `loadBalancerClass` on
  Emissary's Services (thanks, [Sunny Kumar]!).

- Feature: The Helm chart supports setting the IP address family for
  Emissary's Services, and disabling the creation of a Module entirely
  (thanks, [Pier]!).

[Abhay Bothra]: https://github.com/bothra90
[Catalin Codreanu]: https://github.com/ccodreanu
[Frederic Mereu]: https://github.com/fad3t
[Jeremy Dinsel]: https://github.com/jdinsel-xealth
[Luke Shumaker]: https://github.com/lukeshu
[Pier]: https://github.com/pie-r
[Sunny Kumar]: https://github.com/sunnykrGupta
[Tenshin Higashi]: https://github.com/tenshinhigashi
[sekar-saravanan]: https://github.com/sekar-saravanan

[Emissary] follows [Semantic
Versioning](http://semver.org/spec/v2.0.0.html).

[Emissary]: https://emissary-ingress.dev/
