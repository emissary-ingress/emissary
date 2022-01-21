<!-- -*- fill-column: 100 -*- -->
# CHANGELOG -- this is a GENERATED FILE, edit docs/releaseNotes.yml and "make generate" to change.

## EMISSARY-INGRESS and AMBASSADOR EDGE STACK

Emissary-ingress is a Kubernatives-native, self-service, open-source API gateway
and ingress controller. It is a CNCF Incubation project, formerly known as the
Ambassador API Gateway.

Ambassador Edge Stack is a comprehensive, self-service solution for exposing,
securing, and managing the boundary between end users and your Kubernetes services.
The core of Ambassador Edge Stack is Emissary-ingress.

**Note well:**

- Ambassador Edge Stack provides all the capabilities of Emissary-ingress,
  as well as additional capabilities including:

  - Security features such as automatic TLS setup via ACME integration, OAuth/OpenID Connect
    integration, rate limiting, and fine-grained access control; and
  - Developer onboarding assistance, including an API catalog, Swagger/OpenAPI documentation
    support, and a fully customizable developer portal.

- Emissary-ingress can do everything that Ambassador Edge Stack can do, but you'll need to
  write your own code to take advantage of the capabilities above.

- Ambassador Edge Stack is free for all users: due to popular demand, Ambassador Edge Stack
  offers a free usage tier of its core features, designed for startups.

In general, references to "Ambassador" in documentation (including this CHANGELOG)
refer both to Emissary-ingress and to the Ambassador Edge Stack.

## UPCOMING BREAKING CHANGES

#### TLS Termination and the `Host` CRD

As of Emissary-ingress v2.0.4, you _must_ supply a `Host` CRD to terminate TLS: it is not
sufficient to define a `TLSContext` (although `TLSContext`s are still the best way to define TLS
configuration information to be shared across multiple `Host`s). The minimal configuration for
TLS termination is now a certificate stored in a Kubernetes `Secret`, and a `Host` referring to
that `Secret`.

For Emissary-ingress v2.0.0 - v2.0.3, you must supply an `AmbassadorHost` CRD.

#### `Ingress` Resources and Namespaces

In a future version of Emissary-ingress, **no sooner than Emissary-ingress v2.1.0**, TLS
secrets in `Ingress` resources will not be able to use `.namespace` suffixes to cross namespaces.

#### Regex Matching

In a future version of Emissary-ingress, **no sooner than Ambassador v2.1.0**, the `regex_type`
and `regex_max_size` fields will be removed from the `ambassador` `Module`, and Ambassador Edge
Stack will support only Envoy `safe_regex` matching. Note that `safe_regex` matching has been
the default for all 1.X releases of Emissary-ingress.

This change is being made the original Envay `regex` matcher was [deprecated in favor of safe_regex]
in Envoy v1.12.0, then removed entirely from the Envoy V3 APIs. Additionally, setting
[max_program_size was deprecated] in Envoy v1.15.0. As such, `regex_type: unsafe` and setting
`regex_max_size` are no longer supported unless `AMBASSADOR_ENVOY_API_VERSION` is set to `V2`.

Please see the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v3/type/matcher/v3/regex.proto.html) for more information.

[deprecated in favor of safe_regex]: https://www.envoyproxy.io/docs/envoy/latest/version_history/v1.12.0.html?highlight=regex#deprecated
[max_program_size was deprecated]: https://www.envoyproxy.io/docs/envoy/latest/version_history/v1.15.0.html?highlight=max_program_size

#### Zipkin Collector Versions

In a future version of Emissary-ingress, **no sooner than Emissary-ingress v2.1.0**, support
for the [HTTP_JSON_V1] Zipkin collector version will be removed.

This change is being made because the HTTP_JSON_V1 collector was deprecated in Envoy v1.12.0, then
removed entirely from the Envoy V3 APIs. As such, the HTTP_JSON_V1 collector is no longer supported
unless `AMBASSADOR_ENVOY_API_VERSION` is set to `V2`. You must migrate to either the HTTP_JSON or
the HTTP_PROTO collector unless `AMBASSADOR_ENVOY_API_VERSION` is set to `V2`.

Please see the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v2/config/trace/v2/zipkin.proto#envoy-api-field-config-trace-v2-zipkinconfig-collector-endpoint-version) for more information.

[HTTP_JSON_V1]: https://www.envoyproxy.io/docs/envoy/latest/api-v2/config/trace/v2/zipkin.proto#envoy-api-field-config-trace-v2-zipkinconfig-collector-endpoint-version

## RELEASE NOTES

## [2.1.2] TBD
[2.1.2]: https://github.com/emissary-ingress/emissary/compare/v2.1.0...v2.1.2

### Emissary-ingress and Ambassador Edge Stack

- Bugfix: Emissary-ingress 2.1.0 generated invalid Envoy configuration for `getambassador.io/v2`
  `Mappings` that set `spec.cors.origins` to a string rather than a list of strings; this has been
  fixed, and these `Mappings` should once again function correctly.

- Bugfix: Changes to the `weight` of `Mapping` in a canary group will now always be correctly
  managed during reconfiguration; such changes could have been missed in earlier releases.

- Bugfix: Using `rewrite: ""` in a `Mapping` is correctly handled to mean "do not rewrite the path
  at all".

- Change: Docker BuildKit is enabled for all Emissary builds. Additionally, the Go build cache is
  fully enabled when building images, speeding up repeated builds.

- Bugfix: If the `ambassador` `Module` sets a global default for `add_request_headers`,
  `add_response_headers`, `remove_request_headers`, or `remove_response_headers`, it is often
  desirable to be able to turn off that setting locally for a specific `Mapping`. For several
  releases this has not been possible for `Mappings` that are native Kubernetes resources (as
  opposed to annotations), as an emtpy value ("mask the global default") was erroneously considered
  to be equivalent to unset ("inherit the global default").  This is now fixed.

## 2.1.1 not issued

*Emissary-ingress 2.1.1 was not issued; Ambassador Edge Stack 2.1.1 uses Emissary-ingress 2.1.0.*

## [2.1.0] December 16, 2021
[2.1.0]: https://github.com/emissary-ingress/emissary/compare/v2.0.5...v2.1.0

*Emissary-ingress 2.1.0 is not recommended; upgrade to 2.1.2 instead.*

### Emissary-ingress and Ambassador Edge Stack

- Feature: It is now possible to use both the `getambassador.io/v2` and `getambassador.io/v3alpha1`
  apiVersions.  This is accomplished with the addition of an `emissary-ingress-apiext` Service and
  Deployment that handles converting between the two versions for the apiserver.  This allows the
  user to semlessly request and author resources in whichever API version is desired, and
  facilitates easier migration from Emissary 1.x to 2.x.  Resources authored in v3alpha1 will not be
  visible to Emissary 1.x; resources authored in v2 will be visible to both Emissary 1.x and 2.x.

- Bugfix: The incremental reconfiguration cache could miss some updates when multiple `Mapping`s had
  the same `prefix` ("canary"ing multiple `Mapping`s together). This has been corrected, so that all
  such updates correctly take effect. ([3945])

- Bugfix: When using Kubernetes Secrets to store ACME private keys (as the Edge Stack ACME client
  does), an error would always be logged about the Secret not being present, even though it was
  present, and everything was working correctly. This error is no longer logged.

- Bugfix: When using gzip compression, upstream services will no longer receive compressed data.
  This bug was introduced in 1.14.0. The fix restores the default behavior of not sending compressed
  data to upstream services. ([3818])

- Security: Update to busybox 1.34.1 to resolve CVE-2021-28831, CVE-2021-42378, CVE-2021-42379,
  CVE-2021-42380, CVE-2021-42381, CVE-2021-42382, CVE-2021-42383, CVE-2021-42384, CVE-2021-42385,
  and CVE-2021-42386.

- Security: Update Python dependencies to resolve CVE-2020-28493 (jinja2), CVE-2021-28363 (urllib3),
  and CVE-2021-33503 (urllib3).

- Security: Previous built images included some Python packages used only for test. These have now
  been removed, resolving CVE-2020-29651.

[3945]: https://github.com/emissary-ingress/emissary/issues/3945
[3818]: https://github.com/emissary-ingress/emissary/issues/3818

## [2.0.5] November 08, 2021
[2.0.5]: https://github.com/emissary-ingress/emissary/compare/v2.0.4...v2.0.5

### Emissary-ingress and Ambassador Edge Stack

- Feature: It is now possible to set the `circuit_breakers` for `AuthServices`, exactly the same as
  for `Mappings` and `TCPMappings`. This makes it possible to configure your `AuthService` to be
  able to handle more than 1024 concurrent requests.

- Bugfix: Any token delimited by '%' is now validated agains a whitelist of valid Envoy command
  operators. Any mapping containing an `error_response_overrides` section with invalid command
  operators will be discarded.

- Bugfix: The `Host` CRD now correctly supports the `mappingSelector` element, as documented. As a
  transition aid, `selector` is a synonym for `mappingSelector`; a future version of
  Emissary-ingress will remove the `selector` element. ([3902])

[3902]: https://github.com/emissary-ingress/emissary/issues/3902

## [2.0.4] October 19, 2021
[2.0.4]: https://github.com/emissary-ingress/emissary/compare/v2.0.3-ea...v2.0.4

We're pleased to introduce Emissary-ingress 2.0.4 for general availability! The 2.X family
introduces a number of changes to allow Emissary-ingress to more gracefully handle larger
installations, reduce global configuration to better handle multitenant or multiorganizational
installations, reduce memory footprint, and improve performance. We welcome feedback!! Join us on <a
href="https://a8r.io/slack">Slack</a> and let us know what you think.

### Emissary-ingress and Ambassador Edge Stack

- Change: The `x.getambassador.io/v3alpha1` API version has become the `getambassador.io/v3alpha1`
  API version. The `Ambassador-` prefixes from `x.getambassador.io/v3alpha1` resources have been
  removed for ease of migration. **Note that `getambassador.io/v3alpha1` is the only supported API
  version for 2.0.4** &mdash; full support for `getambassador.io/v2` will arrive soon in a later 2.X
  version.

- Feature: The `getambassador.io/v3alpha1` API version and the published chart and manifests have
  been updated to support Kubernetes 1.22. Thanks to <a href="https://github.com/imoisharma">Mohit
  Sharma</a> for contributions to this feature!

- Feature: You can now set `dns_type` between `strict_dns` and `logical_dns` in a `Mapping` to
  configure the Service Discovery Type.

- Feature: You can now set `respect_dns_ttl` to `true` to force the DNS refresh rate for a `Mapping`
  to be set to the record's TTL obtained from DNS resolution.

- Feature: You can now set `buffer_limit_bytes` in the `ambassador` `Module` to to change the size
  of the upstream read and write buffers. The default is 1MiB.

- Bugfix: The release now shows its actual released version number, rather than the internal
  development version number. ([#3854])

- Bugfix: Large configurations no longer cause Emissary-ingress to be unable to communicate with
  Ambassador Cloud. ([#3593])

- Bugfix: The `l7Depth` element of the `Listener` CRD is properly supported.

[#3854]: https://github.com/emissary-ingress/emissary/issues/3854
[#3593]: https://github.com/emissary-ingress/emissary/issues/3593

## [2.0.3-ea] September 16, 2021
[2.0.3-ea]: https://github.com/emissary-ingress/emissary/compare/v2.0.2-ea...v2.0.3-ea

We're pleased to introduce Emissary-ingress 2.0.3 as a **developer preview**. The 2.X family
introduces a number of changes to allow Emissary-ingress to more gracefully handle larger
installations, reduce global configuration to better handle multitenant or multiorganizational
installations, reduce memory footprint, and improve performance. We welcome feedback!! Join us on <a
href="https://a8r.io/slack">Slack</a> and let us know what you think.

### Emissary-ingress and Ambassador Edge Stack

- Feature: The environment variable `AES_LOG_LEVEL` now also sets the log level for the `diagd`
  logger. ([#3686]) ([#3666])

- Feature: You can now set `dns_type` in the `AmbassadorMapping` to configure how Envoy will use the
  DNS for the service.

- Bugfix: It is no longer necessary to set `DOCKER_BUILDKIT=0` when building Emissary. A future
  change will fully support BuildKit. ([#3707])

[#3686]: https://github.com/emissary-ingress/emissary/issues/3686
[#3666]: https://github.com/emissary-ingress/emissary/issues/3666
[#3707]: https://github.com/emissary-ingress/emissary/issues/3707

## [2.0.2-ea] August 24, 2021
[2.0.2-ea]: https://github.com/emissary-ingress/emissary/compare/v2.0.1-ea...v2.0.2-ea

We're pleased to introduce Emissary-ingress 2.0.2 as a **developer preview**. The 2.X family
introduces a number of changes to allow Emissary-ingress to more gracefully handle larger
installations, reduce global configuration to better handle multitenant or multiorganizational
installations, reduce memory footprint, and improve performance. We welcome feedback!! Join us on <a
href="https://a8r.io/slack">Slack</a> and let us know what you think.

### Emissary-ingress and Ambassador Edge Stack

- Bugfix: Upgraded envoy to 1.17.4 to address security vulnerabilities CVE-2021-32777,
  CVE-2021-32778, CVE-2021-32779, and CVE-2021-32781.

- Feature: You can now set `allow_chunked_length` in the Ambassador Module to configure the same
  value in Envoy.

- Change: Envoy-configuration snapshots get saved (as `ambex-#.json`) in `/ambassador/snapshots`.
  The number of snapshots is controlled by the `AMBASSADOR_AMBEX_SNAPSHOT_COUNT` environment
  variable; set it to 0 to disable. The default is 30.

## [2.0.1-ea] August 12, 2021
[2.0.1-ea]: https://github.com/emissary-ingress/emissary/compare/v2.0.0-ea...v2.0.1-ea

We're pleased to introduce Emissary-ingress 2.0.1 as a **developer preview**. The 2.X family
introduces a number of changes to allow Emissary-ingress to more gracefully handle larger
installations, reduce global configuration to better handle multitenant or multiorganizational
installations, reduce memory footprint, and improve performance. We welcome feedback!! Join us on <a
href="https://a8r.io/slack">Slack</a> and let us know what you think.

### Emissary-ingress and Ambassador Edge Stack

- Feature: Ambassador Agent reports sidecar process information and `AmbassadorMapping` OpenAPI
  documentation to Ambassador Cloud to provide more visibility into services and clusters.

- Feature: The optional `stats_prefix` element of the `AmbassadorListener` CRD now determines the
  prefix of HTTP statistics emitted for a specific `AmbassadorListener`.

- Feature: The optional `stats_name` element of `AmbassadorMapping`, `AmbassadorTCPMapping`,
  `AuthService`, `LogService`, `RateLimitService`, and `TracingService` now sets the name under
  which cluster statistics will be logged. The default is the `service`, with non-alphanumeric
  characters replaced by underscores.

- Bugfix: We have updated to `k8s.io/klog/v2` to track upstream and to quiet unnecessary log output.

- Change: Logs now include subsecond time resolutions, rather than just seconds.

- Change: Set `AMBASSADOR_AMBEX_NO_RATELIMIT` to `true` to completely disable ratelimiting Envoy
  reconfiguration under memory pressure. This can help performance with the endpoint or Consul
  resolvers, but could make OOMkills more likely with large configurations. The default is `false`,
  meaning that the rate limiter is active.

## [2.0.0-ea] June 24, 2021
[2.0.0-ea]: https://github.com/emissary-ingress/emissary/compare/v1.14.2...v2.0.0-ea

We're pleased to introduce Emissary-ingress 2.0.0 as a **developer preview**. The 2.X family
introduces a number of changes to allow Emissary-ingress to more gracefully handle larger
installations, reduce global configuration to better handle multitenant or multiorganizational
installations, reduce memory footprint, and improve performance. We welcome feedback!! Join us on <a
href="https://a8r.io/slack">Slack</a> and let us know what you think.

### Emissary-ingress and Ambassador Edge Stack

- Feature: Emissary-ingress 2.0.0 introduces API version `x.getambassador.io/v3alpha1` for
  configuration changes that are not backwards compatible with the 1.X family.  API versions
  `getambassador.io/v0`, `getambassador.io/v1`, and `getambassador.io/v2` are deprecated.  Further
  details are available in the <a
  href="../about/changes-2.0.0/#1-configuration-api-version-xgetambassadoriov3alpha1">2.0.0
  Changes</a> document.

- Feature: The new `AmbassadorListener` CRD defines where and how to listen for requests from the
  network, and which `AmbassadorHost` definitions should be used to process those requests. Note
  that the `AmbassadorListener` CRD is **mandatory** and consolidates *all* port configuration; see
  the <a href="../topics/running/listener">`AmbassadorListener` documentation</a> for more details.

- Feature: Where `AmbassadorMapping`'s `host` field is either an exact match or (with `host_regex`
  set) a regex, the new `hostname` element is always a DNS glob. Use `hostname` instead of `host`
  for best results.

- Feature: The behavior of the Ambassador module `prune_unreachable_routes` field is now automatic,
  which should reduce Envoy memory requirements for installations with many `AmbassadorHost`s

- Bugfix: Each `AmbassadorHost` can specify its `requestPolicy.insecure.action` independently of any
  other `AmbassadorHost`, allowing for HTTP routing as flexible as HTTPS routing. ([#2888])

- Bugfix: Emissary-ingress 2.0.0 fixes a regression in detecting the Ambassador Kubernetes service
  that could cause the wrong IP or hostname to be used in Ingress statuses -- thanks, <a
  href="https://github.com/impl">Noah Fontes</a>!

- Bugfix: Emissary-ingress 2.0.0 fixes a bug where mTLS could use the wrong configuration when SNI
  and the `:authority` header didn't match

- Change: The `TLSContext` `redirect_cleartext_from` and `AmbassadorHost`
  `requestPolicy.insecure.additionalPort` elements are no longer supported. Use a
  `AmbassadorListener` for this functionality instead.

- Change: The `use_proxy_protocol` element of the Ambassador `Module` is no longer supported, as it
  is now part of the `AmbassadorListener` resource (and can be set per-`AmbassadorListener` rather
  than globally).

- Change: An `AmbassadorMapping` will only be matched with an `AmbassadorHost` if the
  `AmbassadorMapping`'s `host` or the `AmbassadorHost`'s `selector` (or both) are explicitly set,
  and match. This change can significantly improve Emissary-ingress's memory footprint when many
  `AmbassadorHost`s are involved. Further details are available in the <a
  href="../about/changes-2.0.0/#host-and-mapping-association">2.0.0 Changes</a> document.

- Change: An `AmbassadorHost` or `Ingress` resource is now required when terminating TLS -- simply
  creating a `TLSContext` is not sufficient. Further details are available in the <a
  href="../about/changes-2.0.0/#host-tlscontext-and-tls-termination">`AmbassadorHost` CRD
  documentation.</a>

- Change: By default, Emissary-ingress will configure Envoy using the V3 Envoy API. This change is
  mostly transparent to users, but note that Envoy V3 does not support unsafe regular expressions
  or, e.g., Zipkin's V1 collector protocol. Further details are available in the <a
  href="../about/changes-2.0.0">2.0.0 Changes</a> document.

- Change: The `tls` module and the `tls` field in the Ambassador module are no longer supported.
  Please use `TLSContext` resources instead.

- Change: The environment variable `AMBASSADOR_FAST_RECONFIGURE` is now set by default, enabling the
  higher-performance implementation of the code that Emissary-ingress uses to generate and validate
  Envoy configurations.

- Change: Service Preview and the `AGENT_SERVICE` environment variable are no longer supported. The
  Telepresence product replaces this functionality.

- Change: The `edgectl` CLI tool has been deprecated; please use the `emissary-ingress` helm chart
  instead.

[#2888]: https://github.com/datawire/ambassador/issues/2888

## [1.14.2] September 29, 2021
[1.14.2]: https://github.com/emissary-ingress/emissary/compare/v1.14.1...v1.14.2

### Emissary-ingress and Ambassador Edge Stack

- Feature: You can now set `respect_dns_ttl` in Ambassador Mappings. When true it configures that
  upstream's refresh rate to be set to resource record’s TTL

- Feature: You can now set `dns_type` in Ambassador Mappings to use Envoy's `logical_dns` resolution
  instead of the default `strict_dns`.

- Feature: You can now set `buffer_limit_bytes` in the `ambassador` `Module` to to change the size
  of the upstream read and write buffers. The default is 1MiB.

### Ambassador Edge Stack only

- Feature: You can now set `preserve_servers` in Ambassador Edge Stack's `DevPortal` resource to
  configure the DevPortal to use server definitions from the OpenAPI document when displaying
  connection information for services in the DevPortal.

## [1.14.1] August 24, 2021
[1.14.1]: https://github.com/emissary-ingress/emissary/compare/v1.14.0...v1.14.1

### Emissary-ingress and Ambassador Edge Stack

- Change: Upgraded Envoy to 1.17.4 to address security vulnerabilities CVE-2021-32777,
  CVE-2021-32778, CVE-2021-32779, and CVE-2021-32781.

## [1.14.0] August 19, 2021
[1.14.0]: https://github.com/emissary-ingress/emissary/compare/v1.13.10...v1.14.0

### Emissary-ingress and Ambassador Edge Stack

- Change: Update from Envoy 1.15 to 1.17.3

- Feature: You can now set `allow_chunked_length` in the Ambassador Module to configure the same
  value in Envoy.

- Change: `AMBASSADOR_ENVOY_API_VERSION` now defaults to `V3`

- Change: Logs now include subsecond time resolutions, rather than just seconds.

## [1.13.10] July 28, 2021
[1.13.10]: https://github.com/emissary-ingress/emissary/compare/v1.13.9...v1.13.10

### Emissary-ingress and Ambassador Edge Stack

- Bugfix: Fixed a regression when specifying a comma separated string for `cors.origins` on the
  `Mapping` resource. ([#3609](https://github.com/emissary-ingress/emissary/issues/3609))

- Change: Envoy-configuration snapshots get saved (as `ambex-#.json`) in `/ambassador/snapshots`.
  The number of snapshots is controlled by the `AMBASSADOR_AMBEX_SNAPSHOT_COUNT` environment
  variable; set it to 0 to disable. The default is 30.

- Change: Set `AMBASSADOR_AMBEX_NO_RATELIMIT` to `true` to completely disable ratelimiting Envoy
  reconfiguration under memory pressure. This can help performance with the endpoint or Consul
  resolvers, but could make OOMkills more likely with large configurations. The default is `false`,
  meaning that the rate limiter is active.

### Ambassador Edge Stack only

- Bugfix: The `Mapping` resource can now specify `docs.timeout_ms` to set the timeout when the Dev
  Portal is fetching API specifications.

- Bugfix: The Dev Portal will now strip HTML tags when displaying search results, showing just the
  actual content of the search result.

- Change: Consul certificate-rotation logging now includes the fingerprints and validity timestamps
  of certificates being rotated.

## [1.13.9] June 30, 2021
[1.13.9]: https://github.com/emissary-ingress/emissary/compare/v1.13.8...v1.13.9

### Emissary-ingress and Ambassador Edge Stack

- Bugfix: Configuring multiple TCPMappings with the same ports (but different hosts) no longer
  generates invalid Envoy configuration.

## [1.13.8] June 08, 2021
[1.13.8]: https://github.com/emissary-ingress/emissary/compare/v1.13.7...v1.13.8

### Emissary-ingress and Ambassador Edge Stack

- Bugfix: Ambassador Agent now accurately reports up-to-date Endpoint information to Ambassador
  Cloud

- Feature: Ambassador Agent reports ConfigMaps and Deployments to Ambassador Cloud to provide a
  better Argo Rollouts experience. See [Argo+Ambassador
  documentation](https://www.getambassador.io/docs/argo) for more info.

## [1.13.7] June 03, 2021
[1.13.7]: https://github.com/datawire/ambassador/compare/v1.13.6...v1.13.7

### Emissary-ingress and Ambassador Edge Stack

- Feature: Add AMBASSADOR_JSON_LOGGING to enable JSON for most of the Ambassador control plane. Some
  (but few) logs from gunicorn and the Kubernetes client-go package still log text.

- Bugfix: Fixed a bug where the Consul resolver would not actually use Consul endpoints with
  TCPMappings.

- Change: Ambassador now calculates its own memory usage in a way that is more similar to how the
  kernel OOMKiller tracks memory.

## [1.13.6] May 24, 2021
[1.13.6]: https://github.com/datawire/ambassador/compare/v1.13.5...v1.13.6

### Emissary-ingress and Ambassador Edge Stack

- Bugfix: Fixed a regression where Ambassador snapshot data was logged at the INFO label when using
  `AMBASSADOR_LEGACY_MODE=true`.

## [1.13.5] May 13, 2021
[1.13.5]: https://github.com/datawire/ambassador/compare/v1.13.4...v1.13.5

### Emissary-ingress and Ambassador Edge Stack

- Bugfix: Fix a regression from 1.8.0 that prevented `ambassador` `Module` config keys `proper_case`
  and `preserve_external_request_id` from working correctly.

- Bugfix: Fixed a regression in detecting the Ambassador Kubernetes service that could cause the
  wrong IP or hostname to be used in Ingress statuses (thanks, [Noah
  Fontes](https://github.com/impl)!

## [1.13.4] May 11, 2021
[1.13.4]: https://github.com/datawire/ambassador/compare/v1.13.3...v1.13.4

### Emissary-ingress and Ambassador Edge Stack

- Security: Incorporate the Envoy 1.15.5 security update by adding the
  `reject_requests_with_escaped_slashes` option to the Ambassador module.

## [1.13.3] May 03, 2021
[1.13.3]: https://github.com/datawire/ambassador/compare/v1.13.2...v1.13.3

### Emissary Ingress and Ambassador Edge Stack

- Bugfix: Fixed a regression that caused Ambassador to crash when loading the Edge Policy Console when any RateLimit resources exist ([#3348])

## [1.13.2] April 29, 2021
[1.13.2]: https://github.com/datawire/ambassador/compare/v1.13.1...v1.13.2

### Emissary Ingress and Ambassador Edge Stack

- Bugfix: Fixed a regression that caused endpoint routing to not work when defining mappings in service annotations ([#3369])

[#3369]: https://github.com/datawire/ambassador/issues/3369

## [1.13.1] April 22, 2021
[1.13.1]: https://github.com/datawire/ambassador/compare/v1.13.0...v1.13.1

### Emissary Ingress and Ambassador Edge Stack

- Bugfix: Potentially increased CPU Usage for deployments with large numbers of Hosts ([#3358])

[#3358]: https://github.com/datawire/ambassador/issues/3358

## [1.13.0] April 20, 2021
[1.13.0]: https://github.com/datawire/ambassador/compare/v1.12.4...v1.13.0

### Emissary Ingress and Ambassador Edge Stack

**Note**: Support for the deprecated `v2alpha` `protocol_version` has been removed from the `AuthService` and `RateLimitService`.

- Feature: Added support for the [Mapping AuthService setting] `auth_context_extensions`, allowing supplying custom per-mapping information to external auth services (thanks, [Giridhar Pathak](https://github.com/gpathak)!).
- Feature: Added support in ambassador-agent for reporting [Argo Rollouts] and [Argo Applications] to Ambassador Cloud
- Feature: The [Ambassador Module configuration] now supports the `diagnostics.allow_non_local` flag to expose admin UI internally only ([#3074] -- thanks, [Fabrice](https://github.com/jfrabaute)!)
- Feature: Ambassador will now use the Envoy v3 API internally when the AMBASSADOR_ENVOY_API_VERSION environment variable is set to "V3". By default, Ambassador will continue to use the v2 API.
- Feature: The [Ambassador Agent] is now available (and deployed by default) for the API Gateway (https://app.getambassador.io).
- Feature: The [Ambassador Module configuration] now supports `merge_slashes` which tells Ambassador to merge adjacent slashes when performing route matching. For example, when true, a request with URL '//foo/' would match a Mapping with prefix '/foo/'.
- Feature: Basic support for a subset of the [Kubernetes Gateway API] has been added.
- Feature: Ambassador now supports the `DD_ENTITY_ID` environment variable to set the `dd.internal.entity_id` statistics tag on metrics generated when using DogStatsD.
- Bugfix: Make Knative paths match on prefix instead of the entire path to better align to the Knative specification ([#3224]).
- Bugfix: The endpoint routing resolver will now properly watch services that include a scheme.
- Bugfix: Environment variable interpolation works again for `ConsulResolver.Spec.Address` without setting `AMBASSADOR_LEGACY_MODE` ([#3182], [#3317])
- Bugfix: Endpoint routing will now detect endpoint changes when your service field includes `.svc.cluster.local`. ([#3324])
- Bugfix: Upgrade PyYAML to 5.4.1 ([#3349])
- Change: The Helm chart has been moved into this repo, in the `charts/ambassador` directory.
- Change: The `Mapping` CRD has been modified so that `kubectl get mappings` now has a column for not just the source path-prefix (`.spec.prefix`), but the source host (`.spec.host`) too.
- Change: The yaml in yaml/docs is now generated from the contents of the helm chart in the `charts/ambassador` directory.
- Change: Support for the deprecated `v2alpha` `protocol_version` has been removed from the `AuthService` and `RateLimitService`.

[Ambassador Agent]: https://www.getambassador.io/docs/cloud/latest/service-catalog/quick-start/
[Ambassador Module configuration]: https://getambassador.io/docs/edge-stack/latest/topics/running/ambassador/
[Argo Applications]: https://www.getambassador.io/docs/argo/latest/quick-start/
[Argo Rollouts]: https://www.getambassador.io/docs/argo/latest/quick-start/
[Kubernetes Gateway API]: https://getambassador.io/docs/edge-stack/latest/topics/using/gateway-api/
[Mapping AuthService setting]: https://getambassador.io/docs/edge-stack/latest/topics/using/authservice

[#3074]: https://github.com/datawire/ambassador/issues/3074
[#3182]: https://github.com/datawire/ambassador/issues/3182
[#3224]: https://github.com/datawire/ambassador/issues/3224
[#3317]: https://github.com/datawire/ambassador/issues/3317
[#3324]: https://github.com/datawire/ambassador/issues/3324
[#3349]: https://github.com/datawire/ambassador/issues/3349

### Ambassador Edge Stack only

- Feature: DevPortal: Added doc.display_name attribute to the Mapping CRD. This value allows for a custom name and documentation URL path of the service in the DevPortal.
- Feature: DevPortal: Added `naming_scheme` enum to the DevPortal CRD. This enum controls the way services are displayed in the DevPortal. Supported values are `namespace.name` (current behavior) and `name.prefix`, which will use the Mapping name and Mapping prefix to display the services.
- Feature: DevPortal: `DEVPORTAL_DOCS_BASE_PATH` environment variable makes the base path of service API documentation configurable.
- Feature: DevPortal: DevPortal will now reload content on changes to Mapping and DevPortal resources.
- Feature: DevPortal: DevPortal now supports a search endpoint at `/docs/api/search`
- Feature: DevPortal search can be configured to only search over titles (with search.type=`title-only`in the DevPortal CRD) or to search over all content (search.type=`all-content`)
- Feature: DevPortal search supports deep linking to openapi spec entries (must set `search.type=all-content` and `search.enabled=true` on the DevPortal CRD)
- Feature: DevPortal: Trigger content refresh by hitting `/docs/api/refreshContent`
- Feature: The AES ratelimit preview service now supports [burst ratelimiting] (aka token bucket ratelimiting).
- Bugfix: The AES ratelimit preview no longer ignores LOCAL_CACHE_SIZE_IN_BYTES.
- Bugfix: The AES ratelimit preview no longer ignores NEAR_LIMIT_RATIO.
- Bugfix: The AES ratelimit preview no longer ignores EXPIRATION_JITTER_MAX_SECONDS.
- Change: Silence DevPortal warnings when DevPortal cannot parse a hostname from a Mapping. (#3341)

[burst ratelimiting]: https://getambassador.io/docs/edge-stack/latest/topics/using/rate-limits/rate-limits/

[#3341]: https://github.com/datawire/ambassador/issues/3341

## [1.12.4] April 19, 2021
[1.12.4]: https://github.com/datawire/ambassador/compare/v1.12.3...v1.12.4

Bugfix: Fix the Envoy base image build step and, as a result, correctly ship the Envoy 1.15.4 security updates.

## [1.12.3] April 15, 2021
[1.12.3]: https://github.com/datawire/ambassador/compare/v1.12.2...v1.12.3

Bugfix: Incorporate the Envoy 1.15.4 security update.

## [1.12.2] March 29, 2021
[1.12.2]: https://github.com/datawire/ambassador/compare/v1.12.1...v1.12.2

- Bugfix: Update OpenSSL to 1.1.1k to address CVE-2021-23840), CVE-2021-3450), CVE-2021-23841), CVE-2021-3449), CVE-2021-23839), CVE-2021-23840), CVE-2021-3450), CVE-2021-23841), CVE-2021-3449), and CVE-2021-23839)

## [1.12.1] March 12, 2021
[1.12.1]: https://github.com/datawire/ambassador/compare/v1.12.0...v1.12.1

- Bugfix: The endpoint routing resolver will now properly watch services with mappings that define the service field with an explicit port.
- Bugfix: Correctly manage cluster load assignments with very long cluster names and `AMBASSADOR_FAST_RECONFIGURE`

## [1.12.0] March 08, 2021
[1.12.0]: https://github.com/datawire/ambassador/compare/v1.11.2...v1.12.0

### Ambasssador API Gateway + Ambassador Edge Stack

- Feature: Endpoint routing is now much more performant, especially in situations where reconfigurations are frequent.
- Feature: A scrubbed ambassador snapshot is now accessible outside the pod at `:8005/snapshot-external`. This port is exposed on the ambassador-admin Kubernetes service.
- Feature: Ambassador now supports configuring the maximum lifetime of an upstream connection using `cluster_max_connection_lifetime_ms`. After the configured time, upstream connections are drained and closed, allowing an operator to set an upper bound on how long any upstream connection will remain open. This is useful when using Kubernetes Service resolvers (the default) and modifying label selectors for traffic shifting.
- Feature: The Ambassador Module configuration now supports `cluster_request_timeout_ms` to set a default request `timeout_ms` for Mappings. This allows an operator to update the default request timeout (currently 3000ms) without needing to update every Mapping.
- Feature: The Ambassador Module configuration now supports `suppress_envoy_headers` to prevent Ambassador from setting additional headers on requests and responses. These headers are typically used for diagnostic purposes and are safe to omit when they are not desired.
- Feature: All Kubernetes services managed by Ambassador are automatically instrumented with service catalog discovery annotations.
- Feature: [`headers_with_underscores_action`](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/core/protocol.proto#enum-core-httpprotocoloptions-headerswithunderscoresaction) is now configurable in the Ambassador `Module`.
- Feature: The Ambassador Module configuration now supports `strip_matching_host_port` to control whether the port should be removed from the host/Authority header before any processing by request filters / routing. This behavior only applies if the port matches the associated Envoy listener port.
- Bugfix: Ambassador now does a better job of cleaning up gRPC connections when shutting down.
- Bugfix: Prevent potential reconcile loop when updating the status of an Ingress.
- Bugfix: Update Python requirements, including addressing CVE-2020-36242 ([#3233])
- Bugfix: Remove unnecessary logs about Kubernetes Secrets ([#3229])

[#3229]: https://github.com/datawire/ambassador/issues/3229
[#3233]: https://github.com/datawire/ambassador/issues/3233

### Ambassador Edge Stack only

- Feature: Added support for ambassador-agent deployment, reporting to Ambassador Cloud Service Catalog (https://app.getambassador.io)
- Feature: `edgectl login` will automatically open your browser, allowing you to login into Service Catalog (https://app.getambassador.io)
- Feature: `edgectl install` command allows you to install a new Ambassador Edge Stack automatically connected to Ambassador Cloud by passing a `--cloud-connect-token` argument.
- Feature: `AES_AUTH_TIMEOUT` now allows you to configure the timeout of the AES authentication service. Defaults to 4s.
- Bugfix: Prevent Dev Portal from sporadically responding with upstream connect timeout when loading content

## [1.11.2] March 01, 2021
[1.11.2]: https://github.com/datawire/ambassador/compare/v1.11.1...v1.11.2

### Ambasssador API Gateway + Ambassador Edge Stack

- Bugfix: Changes to endpoints when endpoint routing is not active will no longer cause reconfiguration
- Bugfix: Correctly differentiate int values of 0 and Boolean values of `false` from non-existent attributes in CRDs ([#3212])
- Bugfix: Correctly support Consul datacenters other than "dc1" without legacy mode.

[#3212]: https://github.com/datawire/ambassador/issues/3212

## [1.11.1] February 04, 2021
[1.11.1]: https://github.com/datawire/ambassador/compare/v1.11.0...v1.11.1

- Bugfix: Fix an issue that caused Dev Portal to sporadically respond with upstream connect timeout when loading content

## [1.11.0] January 26, 2021
[1.11.0]: https://github.com/datawire/ambassador/compare/v1.10.0...v1.11.0

### Ambasssador API Gateway + Ambassador Edge Stack

- Feature: Ambassador now reads the ENVOY_CONCURRENCY environment variable to optionally set the [--concurrency](https://www.envoyproxy.io/docs/envoy/latest/operations/cli#cmdoption-concurrency) command line option when launching Envoy. This controls the number of worker threads used to serve requests and can be used to fine-tune system resource usage.
- Feature: The %DOWNSTREAM_PEER_CERT_V_START% and %DOWNSTREAM_PEER_CERT_V_END% command operators now support custom date formatting, similar to %START_TIME%. This can be used for both header formatting and access log formatting.
- Feature: Eliminate the need to drain and recreate listeners when routing configuration is changed. This reduces both memory usage and disruption of in-flight requests.
- Bugfix: Make sure that `labels` specifying headers with extra attributes are correctly supported again ([#3137]).
- Bugfix: Support Consul services when the `ConsulResolver` and the `Mapping` aren't in the same namespace, and legacy mode is not enabled.
- Bugfix: Fix failure to start when one or more IngressClasses are present in a cluster ([#3142]).
- Bugfix: Properly handle Kubernetes 1.18 and greater when RBAC prohibits access to IngressClass resources.
- Bugfix: Support `TLSContext` CA secrets with fast validation ([#3005]).
- Bugfix: Dev Portal correctly handles transient failures when fetching content
- Bugfix: Dev Portal sidebar pages have a stable order
- Bugfix: Dev Portal pages are now marked cacheable

### Ambassador Edge Stack only

- Feature: RateLimit CRDs now suport specifying an `action` for each limit. Possible values include "Enforce" and "LogOnly", case insensitive. LogOnly may be used to implement dry run rules that do not actually enforce.
- Feature: RateLimit CRDs now support specifying a symbolic `name` for each limit. This name can later be used in the access log to know which RateLimit, if any, applied to a request.
- Feature: RateLimit metadata is now available using the `DYNAMIC_METADATA(envoy.http.filters.ratelimit: ... )` command operator in the Envoy access logs. See [Envoy Documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log/usage) for more on using dynamic metadata in the access log.
- Feature: OAuth2 Filter: The SameSite cookie attribute is now configurable.

[#3005]: https://github.com/datawire/ambassador/issues/3005
[#3137]: https://github.com/datawire/ambassador/issues/3137
[#3142]: https://github.com/datawire/ambassador/issues/3142

## [1.10.0] January 04, 2021
[1.10.0]: https://github.com/datawire/ambassador/compare/v1.9.1...v1.10.0

### Ambasssador API Gateway + Ambassador Edge Stack

- Feature: The redirect response code returned by Ambassador is now configurable using `redirect_reponse_code` on `Mappings` that use `host_redirect`.
- Feature: The redirect location header returned by Ambassador now supports prefix rewrites using `prefix_redirect` on `Mappings` that use `host_redirect`.
- Feature: The redirect location header returned by Ambassador now supports regex rewrites using `regex_redirect` on `Mappings` that use `host_redirect`.
- Feature: Expose `max_request_headers_kb` in the Ambassador `Module`. This directly exposes the same value in Envoy; see [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v2/config/filter/network/http_connection_manager/v2/http_connection_manager.proto) for more information.
- Feature: Support Istio mTLS certification rotation for Istio 1.5 and higher. See the [howto](https://www.getambassador.io/docs/edge-stack/latest/howtos/istio/) for details.
- Feature: The Ambassador Module's `error_response_overrides` now support configuring an empty response body using `text_format`. Previously, empty response bodies could only be configured by specifying an empty file using `text_format_source`.
- Feature: OAuth2 Filter: Support injecting HTTP header fields in to the request before passing on to the upstream service. Enables passing along `id_token` information to the upstream if it was returned by the IDP.
- Bugfix: Fix the grpc external filter to properly cache grpc clients thereby avoiding initiating a separate connection to the external filter for each filtered request.
- Bugfix: Fix a bug in the Mapping CRD where the `text_format_source` field was incorrectly defined as type `string` instead of an object, as documented.
- Bugfix: The RBAC requirements when `AMBASSADOR_FAST_RECONFIGURE` is enabled now more-closely match the requirements when it's disabled.
- Bugfix: Fix error reporting and required-field checks when fast validation is enabled. Note that fast validation is now the default; see below.
- Change: **Fast validation is now the default**, so the `AMBASSADOR_FAST_VALIDATION` variable has been removed. The Golang boot sequence is also now the default. Set `AMBASSADOR_LEGACY_MODE=true` to disable these two behaviors.
- Change: ambassador-consul-connect resources now get deployed into the `ambassador` namespace instead of the active namespace specified in the user's kubernetes context (usually `default`). Old resource cleanup is documented in the Ambassador Consul integration documentation.

### Ambassador Edge Stack only

- Default-off early access: Ratelimiting now supports redis clustering, local caching of exceeded ratelimits, and an upgraded redis client with improved scalability. Must set AES_RATELIMIT_PREVIEW=true to access these improvements.
- Bugfix: OAuth2 Filter: Fix `insufficient_scope` error when validating Azure access tokens.
- Bugfix: Filters: Fix a capitalization-related bug where sometimes existing headers are appended to when they should be overwritten.

## [1.9.1] November 19, 2020
[1.9.1]: https://github.com/datawire/ambassador/compare/v1.9.0...v1.9.1

### Ambassador Edge Stack only

- Bugfix: DevPortal: fix a crash when the `host` cannot be parsed as a valid hostname.

## [1.9.0] November 12, 2020
[1.9.0]: https://github.com/datawire/ambassador/compare/v1.8.1...v1.9.0

### Ambasssador API Gateway + Ambassador Edge Stack

- Feature: Support configuring the gRPC Statistics Envoy filter to enable telemetry of gRPC calls (see the `grpc_stats` configuration flag -- thanks, [Felipe Roveran](https://github.com/feliperoveran)!)
- Feature: The `RateLimitService` and `AuthService` configs now support switching between gRPC protocol versions `v2` and `v2alpha` (see the `protocol_version` setting)
- Feature: The `TracingService` Zipkin config now supports setting `collector_hostname` to tell Envoy which host header to set when sending spans to the collector
- Feature: Ambassador now supports custom error response mapping
- Bugfix: Ambassador will no longer mistakenly post notices regarding `regex_rewrite` and `rewrite` directive conflicts in `Mapping`s due to the latter's implicit default value of `/` (thanks, [obataku](https://github.com/obataku)!)
- Bugfix: The `/metrics` endpoint will no longer break if invoked before configuration is complete (thanks, [Markus Jevring](https://github.com/markusjevringsesame)!)
- Bugfix: Update Python requirements to address CVE-2020-25659
- Bugfix: Prevent mixing `Mapping`s with `host_redirect` set with `Mapping`s that don't in the same group
- Bugfix: `ConsulResolver` will now fallback to the `Address` of a Consul service if `Service.Address` is not set.
- Docs: Added instructions for building ambassador from source, within a docker container (thanks, [Rahul Kumar Saini](https://github.com/rahul-kumar-saini)!)
- Update: Upgrade Alpine 3.10→3.12, GNU libc 2.30→2.32, and Python 3.7→3.8
- Update: Knative serving tests were bumped from version 0.11.0 to version 0.18.0 (thanks, [Noah Fontes](https://github.com/impl)!)

### Ambassador Edge Stack only

- Change: The DevPortal no longer looks for documentation at `/.ambassador-internal/openapi-docs`.  A new field in `Mappings`, `docs`, must be used for specifying the source for documentation.  This can result in an empty Dev Portal after upgrading if `Mappings` do not include a `docs` attribute.
- Feature: How the `OAuth2` Filter authenticates itself to the identity provider is now configurable with the `clientAuthentication` setting.
- Feature: The `OAuth2` Filter can now use RFC 7523 JWT assertions to authenticate itself to the identity provider; this is usable with all grant types.
- Feature: When validating a JWT's scope, the `JWT` and `OAuth2` Filters now support not just RFC 8693 behavior, but also the behavior of various drafts leading to it, making JWT scope validation usable with more identity providers.
- Feature: The `OAuth2` Filter now has `inheritScopeArgument` and `stripInheritedScope` settings that can further customize the behavior of `accessTokenJWTFilter`.
- Feature: DevPortal: default configuration using the `ambassador` `DevPortal` resource.
- Change: The `OAuth2` Filter argument `scopes` has been renamed to `scope`, for consistency.  The name `scopes` is deprecated, but will continue to work for backward compatibility.
- Bugfix: `OAuth2` Filter: Don't have `accessTokenValidation: auto` fall back to "userinfo" validation for a client_credentials grant; it doesn't make sense there and only serves to obscure a more useful error message.

## [1.8.1] October 16, 2020
[1.8.1]: https://github.com/datawire/ambassador/compare/v1.8.0...v1.8.1

### Ambasssador API Gateway + Ambassador Edge Stack

- Bugfix: Ambassador no longer fails to configure Envoy listeners when a TracingService or LogService has a service name whose underlying cluster name has over 40 charcters.
- Bugfix: The Ambassador diagnostics page no longer returns HTTP 500 when a TracingService or LogService has a service name whose underlying cluster name has over 40 characters.

## [1.8.0] October 08, 2020
[1.8.0]: https://github.com/datawire/ambassador/compare/v1.7.4...v1.8.0

### Ambasssador API Gateway + Ambassador Edge Stack

- Feature: HTTP IP Allow/Deny ranges are supported.
- Bugfix: Ambassador's health checks don't claim that Envoy has failed when reconfiguration taking a long time (thanks, [Fabrice](https://github.com/jfrabaute), for contributions here!).
- Bugfix: The `edgectl connect` command now works properly when using zsh on a Linux platform.
- Bugfix: The container no longer exits "successfully" when the Deployment specifies an invalid `command`.

### Ambassador Edge Stack only

- Feature: `RateLimit` CRDs now support setting a response body, configurable with the `errorResponse` setting.
- Bugfix: `External` `Filter` can now properly proxy the body to the configured `auth_service`
- Bugfix: The RBAC for AES now grants permission to "patch" `Events.v1.core` (previously it granted "create" but not "patch")

## [1.7.4] October 06, 2020
[1.7.4]: https://github.com/datawire/ambassador/compare/v1.7.3...v1.7.4

### Ambasssador API Gateway + Ambassador Edge Stack

- Bugfix: Several regressions in the 1.7.x series are resolved by removing the ability to set `insecure.action` on a per-`Host`-resource basis, which was an ability added in 1.7.0.  This reverts to the pre-1.7.0 behavior of having one `Host`'s insecure action "win" and be used for all `Host`s.
- Bugfix: Ambassador will no longer generate invalid Envoy configuration with duplicate clusters in certain scenarios when `AMBASSADOR_FAST_RECONFIGURE=true`.
- Enhancement: When `AMBASSADOR_FAST_RECONFIGURE=true` is set, Ambassador now logs information about memory usage.

## [1.7.3] September 29, 2020
[1.7.3]: https://github.com/datawire/ambassador/compare/v1.7.2...v1.7.3

### Ambasssador API Gateway + Ambassador Edge Stack

- Incorporate the Envoy 1.15.1 security update.
- Bugfix: A regression introduced in 1.7.2 when `AMBASSADOR_FAST_RECONFIGURE=true` has been fixed where Host resources `tls.ca_secret` didn't work correctly.
- Bugfix: `TLSContext` resources and `spec.tls` in `Host` resources now correctly handle namespaces with `.` in them.
- Bugfix: Fix `spec.requestPolicy.insecure.action` for `Host` resources with a `*` wildcard in the hostname.
- Bugfix: Reduce lock contention while generating diagnostics.

## [1.7.2] September 16, 2020
[1.7.2]: https://github.com/datawire/ambassador/compare/v1.7.1...v1.7.2

### Ambasssador API Gateway + Ambassador Edge Stack

- Bugfix: A regression introduced in 1.7.0 with the various `Host` resource `spec.requestPolicy.insecure.action` behaviors, including handling of X-Forwarded-Proto, has been fixed.
- Bugfix: Host resources no longer perform secret namespacing when the `AMBASSADOR_FAST_RECONFIGURE` flag is enabled.

## [1.7.1] September 08, 2020
[1.7.1]: https://github.com/datawire/ambassador/compare/v1.7.0...v1.7.1

### Ambasssador API Gateway + Ambassador Edge Stack

- Bugfix: Support `envoy_validation_timeout` in the Ambassador Module to set the timeout for validating new Envoy configurations

### Ambassador Edge Stack only

- Bugfix: `consul_connect_integration` is now built correctly.
- Bugfix: The developer portal again supports requests for API documentation

## [1.7.0] August 27, 2020
[1.7.0]: https://github.com/datawire/ambassador/compare/v1.6.2...v1.7.0

### Ambassador API Gateway + Ambassador Edge Stack

- Feature: Upgrade from Envoy 1.14.4 to 1.15.0.
- Bugfix: Correctly handle a `Host` object with incompatible manually-specified `TLSContext`
- Feature: The Ambassador control-plane now publishes Prometheus metrics alongside the existing Envoy data-plane metrics under the `/metrics` endpoint on port 8877.
- Default-off early access: Experimental changes to allow Ambassador to more quickly process configuration changes (especially with larger configurations) have been added. The `AMBASSADOR_FAST_RECONFIGURE` env var must be set to enable this. `AMBASSADOR_FAST_VALIDATION` should also be set for maximum benefit.
- Bugfix: Fixed insecure route action behavior. Host security policies no longer affect other Hosts.

### Ambassador API Gateway only

- Bugfix: Fixes regression in 1.5.1 that caused it to not correctly know its own version number, leading to notifications about an available upgrade despite being on the most recent version.

### Ambassador Edge Stack only

- Feature: DevPortal can now discover openapi documentation from `Mapping`s that set `host` and `headers`
- Feature: `edgectl install` will automatically enable Service Preview with a Preview URL on the Host resource it creates.
- Feature: Service Preview will inject an `x-service-preview-path` header in filtered requests with the original request prefix to allow for context propagation.
- Feature: Service Preview can intercept gRPC requests using the `--grpc` flag on the `edgectl intercept add` command and the `getambassador.io/inject-traffic-agent-grpc: "true"` annotation when using automatic Traffic-Agent injection.
- Feature: The `TracingService` Zipkin config now supports setting `collector_endpoint_version` to tell Envoy to use Zipkin v2.
- Feature: You can now inject request and/or response headers from a `RateLimit`.
- Bugfix: Don't crash during startup if Redis is down.
- Bugfix: Service Preview correctly uses the Host default `Path` value for the `spec.previewUrl.type` field.
- Bugfix: The `JWT`, `OAuth2`, and other Filters are now better about reusing connections for outgoing HTTP requests.
- Bugfix: Fixed a potential deadlock in the HTTP cache used for fetching JWKS and such for `Filters`.
- Bugfix: Internal Ambassador data is no longer exposed to the `/.ambassador-internal/` endpoints used by the DevPortal.
- Bugfix: Problems with license key limits will no longer trigger spurious HTTP 429 errors.  Using the `RateLimit` resource beyond 5rps without any form of license key will still trigger 429 responses, but now with a `X-Ambassador-Message` header indicating that's what happned.
- Bugfix: When multiple `RateLimit`s overlap, it is supposed to enforce the strictest limit; but the strictness comparison didn't correctly handle comparing limits with different units.
- Change: The Redis settings have been adjusted to default to the pre-1.6.0 behavior, and have been adjusted to be easier to understand.
- Feature: `consul_connect_integration` is now part of the AES image.
- Bugfix: `consul_connect_integration` now correctly handles certificates from Hashicorp Vault.

## [1.6.2] July 30, 2020
[1.6.2]: https://github.com/datawire/ambassador/compare/v1.6.1...v1.6.2

### Ambassador API Gateway + Ambassador Edge Stack

- Bugfix: The (new in 1.6.0) `Host.spec.tls` and `Host.spec.tlsContext` fields now work when `AMBASSADOR_FAST_VALIDATION=fast` is not set.
- Bugfix: Setting `use_websocket: true` on a `Mapping` now only affects routes generated from that `Mapping`, instead of affecting all routes on that port.
- Feature: It is now possible to "upgrade" to non-HTTP protocols other than WebSocket; the new `allow_upgrade` is a generalization of `use_websocket`.

### Ambassador Edge Stack only

- Bugfix: The `Host.spec.requestPolicy.insecure.additionalPort` field works again.
- Bugfix: The `Host.spec.ambassadorId` is once again handled in addition to `.ambassador_id`; allowing hosts written by older versions AES prior to 1.6.0 to continue working.
- Bugfix: Fix a redirect loop that could occur when using using multiple `protectedOrigins` in a `Host`.

## [1.6.1] July 23, 2020
[1.6.1]: https://github.com/datawire/ambassador/compare/v1.6.0...v1.6.1

### Ambassador API Gateway + Ambassador Edge Stack

- Bugfix: Mapping with `https` scheme for service are correctly parsed.
- Bugfix: Mapping with both a scheme and a hostname of `localhost` is now handled correctly.
- Bugfix: ConsulResolver now works again for Mappings outside of Ambassador's namespace.

## [1.6.0] July 21, 2020
[1.6.0]: https://github.com/datawire/ambassador/compare/v1.5.5...v1.6.0

### Ambassador API Gateway + Ambassador Edge Stack

- Incorporate the Envoy 1.14.4 security update.
- API CHANGE: Turning off the Diagnostics UI via the Ambassador Module now disables access to the UI from both inside and outside the Ambassador Pod.
- API CHANGE: Default changes updating `Mapping` status from default-on to default-off; see below.
- Feature: Add support for circuit breakers in TCP mapping (thanks, [Pierre Fersing](https://github.com/PierreF)!)
- Feature: Ambassador CRDs now include schema. This enables validation by `kubectl apply`.
- Feature: Advanced TLS configuration can be specified in `Host` resource via `tlsContext` and `tls` fields.
- Feature: Implement sampling percentage in tracing service.
- Performance improvement: Diagnostics are generated on demand rather than on every reconfig.
- Performance improvement: Experimental fast validation of the contents of Ambassador resources has been added. The `AMBASSADOR_FAST_VALIDATION` env var must be set to enable this.
- Internal: Configuration endpoints used internally by Ambassador are no longer accessible from outside the Ambassador Pod.
- Bugfix: `envoy_log_format` can now be set with `envoy_log_type: json`.
- Docs: Fixed OAuth2 documentation spelling errors (thanks, [Travis Byrum](https://github.com/travisbyrum)!)

As previously announced, the default value of `AMBASSADOR_UPDATE_MAPPING_STATUS`
has now changed from `true` to `false`; Ambassador will no longer attempt to
update the `Status` of a `Mapping` unless you explicitly set
`AMBASSADOR_UPDATE_MAPPING_STATUS=true` in the environment.  If you do not have
tooling that relies on `Mapping` status updates, we do not recommend setting
`AMBASSADOR_UPDATE_MAPPING_STATUS`.

**In Ambassador 1.7**, TLS secrets in `Ingress` resources will not be able to use
`.namespace` suffixes to cross namespaces.

### Ambassador Edge Stack only

- Feature: The Edge Policy Console's Debugging page now has a "Log Out" button to terminate all EPC sessions.
- Feature: `X-Content-Type-Options: nosniff` to response headers are now set for the Edge Policy Console, to prevent MIME confusion attacks.
- Feature: The `OAuth2` Filter now has a `allowMalformedAccessToken` setting to enable use with IDPs that generate access tokens that are not compliant with RFC 6750.
- Bugfix: All JWT Filter errors are now formatted per the specified `errorResponse`.
- Feature: Options for making Redis connection pooling configurable.
- Bugfix: User is now directed to the correct URL after clicking in Microsoft Office.
- Feature: The Console's Dashboard page has speedometer gauges to visualize Rate Limited and Authenticated traffic.

## [1.5.5] June 30, 2020
[1.5.5]: https://github.com/datawire/ambassador/compare/v1.5.4...v1.5.5

### Ambassador API Gateway + Ambassador Edge Stack

- Incorporate the Envoy 1.14.3 security update.

## [1.5.4] June 23, 2020
[1.5.4]: https://github.com/datawire/ambassador/compare/v1.5.3...v1.5.4

### Ambassador API Gateway + Ambassador Edge Stack

- Bugfix: Allow disabling `Mapping`-status updates (RECOMMENDED: see below)
- Bugfix: Logging has been made _much_ quieter; the default Envoy log level has been turned down from "warning" to "error"
- Ambassador now logs timing information about reconfigures

We recommend that users set `AMBASSADOR_UPDATE_MAPPING_STATUS=false`
in the environment to tell Ambassador not to update `Mapping` statuses
unless you have some script that relies on `Mapping` status updates.
The default value of `AMBASSADOR_UPDATE_MAPPING_STATUS` will change to
`false` in Ambassador 1.6.

## [1.5.3] June 16, 2020
[1.5.3]: https://github.com/datawire/ambassador/compare/v1.5.2...v1.5.3

### Ambassador API Gateway + Ambassador Edge Stack

- Bugfix: Restore Envoy listener drain time to its pre-Ambassador 1.3.0 default of 10 minutes.
- Bugfix: Read Knative ingress generation from the correct place in the Kubernetes object

### Ambassador Edge Stack only

- Bugfix: Allow deletion of ProjectControllers.
- Bugfix: Fix regression introduced in 1.4.2 where the `OAuth2` AuthorizationCode filter no longer works when behind another gateway that rewrites the request hostname.  The behavior here is now controllable via the `internalOrigin` sub-field.

## [1.5.2] June 10, 2020
[1.5.2]: https://github.com/datawire/ambassador/compare/v1.5.1...v1.5.2

### Ambassador API Gateway + Ambassador Edge Stack

- Incorporate the [Envoy 1.14.2](https://www.envoyproxy.io/docs/envoy/v1.14.2/intro/version_history#june-8-2020) security update.
- Upgrade the base Docker images used by several tests (thanks, [Daniel Sutton](https://github.com/ducksecops)!).

### Ambassador Edge Stack only

- Feature (BETA): Added an in-cluster micro CI/CD system to enable building, staging, and publishing of GitHub projects from source.  This has been included in previous versions as an alpha, but disabled by default. It is now in BETA.
- Bugfix: The `DEVPORTAL_CONTENT_URL` environment variable now properly handles `file:///` URLs to refer to volume-mounted content.
- Bugfix: `acmeProvider.authority: none` is no longer case sensitive
- Bugfix: `edgectl connect` works again on Ubuntu and other Linux setups with old versions of nss-mdns (older than version 0.11)
- Bugfix: `edgectl` works again on Windows
- Bugfix: The Edge Policy Console now correctly creates FilterPolicy resources

## [1.5.1] June 05, 2020
[1.5.1]: https://github.com/datawire/ambassador/compare/v1.5.0...v1.5.1

### Ambassador API Gateway + Ambassador Edge Stack

- Bugfix: Logging has been made _much_ quieter
- Bugfix: A service that somehow has no hostname should no longer cause an exception

## [1.5.0] May 28, 2020
[1.5.0]: https://github.com/datawire/ambassador/compare/v1.4.3...v1.5.0

### Ambassador API Gateway + Ambassador Edge Stack

- Change: Switched from quay.io back to DockerHub as our primary publication point. **If you are using your own Kubernetes manifests, you will have to update them!** Datawire's Helm charts and published YAML have already been updated.
- Feature: switch to Envoy 1.14.1
- Feature: Allow defaults for `add_request_header`, `remove_request_header`, `add_response_header`, and `remove_response_header`
- Feature: Inform Knative of the route to the Ambassador service if available (thanks, [Noah Fontes](https://github.com/impl)!)
- Feature: Support the path and timeout options of the Knative ingress path rules (thanks, [Noah Fontes](https://github.com/impl)!)
- Feature: Allow preserving `X-Request-ID` on requests from external clients (thanks, [Prakhar Joshi](https://github.com/prakharjoshi)!)
- Feature: Mappings now support query parameters (thanks, [Phil Peble](https://github.com/ppeble)!)
- Feature: Allow setting the Envoy shared-memory base ID (thanks, [Phil Peble](https://github.com/ppeble)!)
- Feature: Additional security configurations not set on default YAMLs
- Feature: Let Ambassador configure `regex_rewrite` for advanced forwarding
- Bugfix: Only update Knative ingress CRDs when the generation changes (thanks, [Noah Fontes](https://github.com/impl)!)
- Bugfix: Now behaves properly when `AMBASSADOR_SINGLE_NAMESPACE` is set to an empty string; rather than getting in to a weird in-between state
- Bugfix: The websocket library used by the test suite has been upgraded to incorporate security fixes (thanks, [Andrew Allbright](https://github.com/aallbrig)!)
- Bugfix: Fixed evaluation of label selectors causing the wrong IP to be put in to Ingress resource statuses
- Bugfix: The `watt` (port 8002) and `ambex` (port 8003) components now bind to localhost instead of 0.0.0.0, so they are no longer erroneously available from outside the Pod

### Ambassador Edge Stack only

- Feature: `edgectl upgrade` allows upgrading API Gateway installations to AES
- Feature: `edgectl intercept` can generate preview-urls for Host resources that enabled the feature
- Feature: `edgectl install` will now automatically install the Service Preview components (ambassador-injector, telepresence-proxy) and scoped RBAC
- Feature: Rate-limited 429 responses now include the `Retry-After` header
- Feature: The `JWT` Filter now makes `hasKey` and `doNotSet` functions available to header field templates; in order to facilitate only conditionally setting a header field.
- Feature: The `OAuth2` Filter now has an `expirationSafetyMargin` setting that will cause an access token to be treated as expired sooner, in order to have a safety margin of time to send it to the upstream Resource Server that grants insufficient leeway.
- Feature: The `JWT` Filter now has `leewayFor{ExpiresAt,IssuedAt,NotBefore}` settings for configuring leeway when validating the timestamps of a token.
- Feature: The environment variables `REDIS{,_PERSECOND}_{USERNAME,PASSWORD,TLS_ENABLED,TLS_INSECURE}` may now be used to further configure how the Ambassador Edge Stack communicates with Redis.
- Bugfix: Don't start the dev portal running if `POLL_EVERY_SECS` is 0
- Bugfix: Now no longer needs cluster-wide RBAC when running with `AMBASSADOR_SINGLE_NAMESPACE`.
- Bugfix: The `OAuth2` Filter now validates the reported-to-Client scope of an Access Token even if a separate `accessTokenJWTFilter` is configured.
- Bugfix: The `OAuth2` Filter now sends the user back to the identity provider to upgrade the scope if they request an endpoint that requires broader scope than initially requested; instead of erroring.
- Bugfix: The `OAuth2` Filter will no longer send RFC 7235 challenges back to the user agent if it would not accept RFC 7235 credentials (previously it only avoided sending HTTP 401 challenges, but still sent 400 or 403 challenges).
- Bugfix: The `amb-sidecar` (port 8500) component now binds to localhost instead of 0.0.0.0, so it is no longer erroneously available from outside the Pod

## [1.4.3] May 14, 2020
[1.4.3]: https://github.com/datawire/ambassador/compare/v1.4.2...v1.4.3

### Ambassador Edge Stack only

- Bugfix: Don't generate spurious 403s in the logs when using the Edge Policy Console.

## [1.4.2] April 22, 2020
[1.4.2]: https://github.com/datawire/ambassador/compare/v1.4.1...v1.4.2

### Ambassador Edge Stack only

- Bugfix: The Traffic Agent binds to port 9900 by default. That port can be configured in the Agent's Pod spec.
   - For more about using the Traffic Agent, see the [Service Preview documentation](https://www.getambassador.io/docs/edge-stack/latest/topics/using/edgectl/#configuring-service-preview).
- Bugfix: The `OAuth2` Filter redirection-endpoint now handles various XSRF errors more consistently (the way we meant it to in 1.2.1)
- Bugfix: The `OAuth2` Filter now supports multiple authentication domains that share the same credentials.
   - For more about using multiple domains, see the [OAuth2 `Filter` documentation](https://www.getambassador.io/docs/edge-stack/1.4/topics/using/filters/oauth2/).
- Bugfix: The ACME client now obeys `AMBASSADOR_ID`
- Feature (ALPHA): Added an in-cluster micro CI/CD system to enable building, staging, and publishing of GitHub projects from source.  This is disabled by default.

## [1.4.1] April 15, 2020
[1.4.1]: https://github.com/datawire/ambassador/compare/v1.4.0...v1.4.1

### Ambassador Edge Stack only

- Internal: `edgectl install` uses Helm under the hood

## [1.4.0] April 08, 2020
[1.4.0]: https://github.com/datawire/ambassador/compare/v1.3.2...v1.4.0

### Ambassador API Gateway + Ambassador Edge Stack

- Feature: Support Ingress Path types improvements from networking.k8s.io/v1beta1 on Kubernetes 1.18+
- Feature: Support Ingress hostname wildcards
- Feature: Support for the IngressClass Resource, added to networking.k8s.io/v1beta1 on Kubernetes 1.18+
   - For more about new Ingress support, see the [Ingress Controller documentation](https://getambassador.io/docs/edge-stack/1.4/topics/running/ingress-controller).
- Feature: `Mapping`s support the `cluster_tag` attribute to control the name of the generated Envoy cluster (thanks, [Stefan Sedich](https://github.com/stefansedich)!)
   - See the [Advanced Mapping Configuration documentation](https://getambassador.io/docs/edge-stack/1.4/topics/using/mappings) for more.
- Feature: Support Envoy's ability to force response headers to canonical HTTP case (thanks, [Puneet Loya](https://github.com/puneetloya)!)
   - See the [Ambassador Module documentation](https://getambassador.io/docs/edge-stack/1.4/topics/running/ambassador) for more.
- Bugfix: Correctly ignore Kubernetes services with no metadata (thanks, [Fabrice](https://github.com/jfrabaute)!)

### Ambassador Edge Stack only

- Feature: `edgectl install` output has clearer formatting
- Feature: `edgectl install` offers help when installation does not succeed
- Feature: `edgectl install` uploads installer and AES logs to a private area upon failure so Datawire support can help
- Bugfix: The "Filters" tab in the webui no longer renders the value of OAuth client secrets that are stored in Kubernetes secrets.
- Bugfix: The ACME client of of one Ambassador install will no longer interfere with the ACME client of another Ambassador install in the same namespace with a different AMBASSADOR_ID.
- Bugfix: `edgectl intercept` supports matching headers values against regular expressions once more
- Bugfix: `edgectl install` correctly handles more local and cluster environments
   - For more about `edgectl` improvements, see the [Service Preview and Edge Control documentation](https://getambassador.io/docs/edge-stack/1.4/topics/using/edgectl).

## [1.3.2] April 01, 2020
[1.3.2]: https://github.com/datawire/ambassador/compare/v1.3.1...v1.3.2

### Ambassador Edge Stack only

- Bugfix: `edgectl install` correctly installs on Amazon EKS and other clusters that provide load balancers with fixed DNS names
- Bugfix: `edgectl install` when using Helm once again works as documented
- Bugfix: `edgectl install` console logs are improved and neatened
- Bugfix: `edgectl install --verbose` output is improved
- Bugfix: `edgectl install` automatically opens documentation pages for some errors
- Bugfix: `edgectl install` help text is improved

## [1.3.1] March 24, 2020
[1.3.1]: https://github.com/datawire/ambassador/compare/v1.3.0...v1.3.1

### Ambassador Edge Stack only

- Bugfix: `edgectl install` will not install on top of a running Ambassador
- Bugfix: `edgectl install` can detect and report if `kubectl` is missing
- Bugfix: `edgectl install` can detect and report if it cannot talk to a Kubernetes cluster
- Bugfix: When using the `Authorization Code` grant type for `OAuth2`, expired tokens are correctly handled so that the user will be prompted to renew
- Bugfix: When using the `Password` grant type for `OAuth2`, authentication sessions are properly associated with each user
- Bugfix: When using the `Password` grant type for `OAuth2`, you can set up multiple `Filter`s to allow requesting different scopes for different endpoints

## [1.3.0] March 17, 2020
[1.3.0]: https://github.com/datawire/ambassador/compare/v1.2.2...v1.3.0

### Ambassador Edge Stack only

- Feature: Support username and password as headers for OAuth2 authentication (`grantType: Password`)
- Feature: `edgectl install` provides better feedback for clusters that are unreachable from the public Internet
- Feature: `edgectl install` supports KIND clusters (thanks, [@factorypreset](https://github.com/factorypreset)!)
- Feature: `edgectl intercept` supports HTTPS
- Feature: Ambassador Edge Stack Docker image is ~150MB smaller
- Feature: The Edge Policy Console can be fully disabled with the `diagnostics.enabled` element in the `ambassador` Module
- Feature: `aes-plugin-runner` now allows passing in `docker run` flags after the main argument list.
- Bugfix: Ambassador Edge Stack doesn't crash if the Developer Portal content URL is not accessible
- Bugfix: `edgectl connect` does a better job handling clusters with many services
- Bugfix: The `Plugin` Filter now correctly sets `request.TLS` to nil/non-nil based on if the original request was encrypted or not.
- Change: There is no longer a separate traffic-proxy image; that functionality is now part of the main AES image. Set `command: ["traffic-manager"]` to use it.

## [1.2.2] March 04, 2020
[1.2.2]: https://github.com/datawire/ambassador/compare/v1.2.1...v1.2.2

### Ambassador Edge Stack only

- Internal: Fix an error in Edge Stack update checks

## [1.2.1] March 03, 2020
[1.2.1]: https://github.com/datawire/ambassador/compare/v1.2.0...v1.2.1

Edge Stack users SHOULD NOT use this release, and should instead use 1.2.2.

### Ambassador API Gateway + Ambassador Edge Stack

- Bugfix: re-support PROXY protocol when terminating TLS ([#2348])
- Bugfix: Incorporate the Envoy 1.12.3 security update

### Ambassador Edge Stack only

- Bugfix: The `aes-plugin-runner` binary for GNU/Linux is now statically linked (instead of being linked against musl libc), so it should now work on either musl libc or GNU libc systems
- Feature (ALPHA): An `aes-plugin-runner` binary for Windows is now produced.  (It is un-tested as of yet.)
- Bugfix: The `OAuth2` Filter redirection-endpoint now handles various XSRF errors more consistently
- Change: The `OAuth2` Filter redirection-endpoint now handles XSRF errors by redirecting back to the identity provider

[#2348]: https://github.com/datawire/ambassador/issues/2348

## [1.2.0] February 24, 2020
[1.2.0]: https://github.com/datawire/ambassador/compare/v1.1.1...v1.2.0

### Ambassador API Gateway + Ambassador Edge Stack

- Feature: add idle_timeout_ms support for common HTTP listener (thanks, Jordan Neufeld!) ([#2155])
- Feature: allow override of bind addresses, including for IPv6! (thanks to [Josue Diaz](https://github.com/josuesdiaz)!) ([#2293])
- Bugfix: Support Istio mTLS secrets natively (thanks, [Phil Peble](https://github.com/ppeble)!) ([#1475])
- Bugfix: TLS custom secret with period in name doesn't work (thanks, [Phil Peble](https://github.com/ppeble)!) ([#1255])
- Bugfix: Honor ingress.class when running with Knative
- Internal: Fix CRD-versioning issue in CI tests (thanks, [Ricky Taylor](https://github.com/ricky26)!)
- Bugfix: Stop using deprecated Envoy configuration elements
- Bugfix: Resume building a debuggable Envoy binary

### Ambassador Edge Stack only

- Change: The `ambassador` service now uses the default `externalTrafficPolicy` of `Cluster` rather than explicitly setting it to `Local`. This is a safer setting for GKE where the `Local` policy can cause outages when ambassador is updated. See https://stackoverflow.com/questions/60121956/are-hitless-rolling-updates-possible-on-gke-with-externaltrafficpolicy-local for details.
- Feature: `edgectl install` provides a much cleaner, quicker experience when installing Ambassador Edge Stack
- Feature: Ambassador Edge Stack supports the Ambassador operator for automated management and upgrade
- Feature: `ifRequestHeader` can now have `valueRegex` instead of `value`
- Feature: The `OAuth2` Filter now has `useSessionCookies` option to have cookies expire when the browser closes, rather than at a fixed duration
- Feature: `ifRequestHeader` now has `negate: bool` to invert the match
- Bugfix: The RBAC for `Ingress` now supports the `networking.k8s.io` `apiGroup`
- Bugfix: Quiet Dev Portal debug logs
- Bugfix: The Edge Policy Console is much less chatty when logged out
- Change: The intercept agent is now incorporated into the `aes` image
- Change: The `OAuth2` Filter no longer sets cookies when `insteadOfRedirect` triggers
- Change: The `OAuth2` Filter more frequently adjusts the cookies

[#1475]: https://github.com/datawire/ambassador/issues/1475
[#1255]: https://github.com/datawire/ambassador/issues/1255
[#2155]: https://github.com/datawire/ambassador/issues/2155
[#2293]: https://github.com/datawire/ambassador/issues/2293

## [1.1.1] February 12, 2020
[1.1.1]: https://github.com/datawire/ambassador/compare/v1.1.0...v1.1.1

### Ambassador API Gateway + Ambassador Edge Stack

- Bugfix: Load explicitly referenced secrets in another namespace, even when `AMBASSADOR_SINGLE_NAMESPACE` (thanks, [Thibault Cohen](https://github.com/titilambert)!) ([#2202])
- Bugfix: Fix Host support for choosing cleartext or TLS ([#2279])
- Bugfix: Fix intermittent error when rendering `/ambassador/v0/diag/`
- Internal: Various CLI tooling improvements

[#2202]: https://github.com/datawire/ambassador/issues/2202
[#2279]: https://github.com/datawire/ambassador/pull/2279

### Ambassador Edge Stack only

- Feature: The Policy Console can now set the log level to "trace" (in addition to "info" or "debug")
- Bugfix: Don't have the Policy Console poll for snapshots when logged out
- Bugfix: Do a better job of noticing when the license key changes
- Bugfix: `aes-plugin-runner --version` now works properly
- Bugfix: Only serve the custom CONGRATULATIONS! 404 page on `/`
- Change: The `OAuth2` Filter `stateTTL` setting is now ignored; the lifetime of state-tokens is now managed automatically

## [1.1.0] January 28, 2020
[1.1.0]: https://github.com/datawire/ambassador/compare/v1.0.0...v1.1.0

(Note that Ambassador 1.1.0 is identical to Ambassador 1.1.0-rc.0, from January 24, 2020.
 Also, we're now using "-rc.N" rather than just "-rcN", for better compliance with
 [SemVer](https://www.semver.org/).

### Ambassador API Gateway + Ambassador Edge Stack

- Feature: support resources with the same name but in different namespaces ([#2226], [#2198])
- Feature: support DNS overrides in `edgectl`
- Bugfix: Reduce log noise about "kubestatus" updates
- Bugfix: manage the diagnostics snapshot cache more aggressively to reduce memory footprint
- Bugfix: re-enable Docker demo mode (and improve the test to make sure we don't break it again!) ([#2227])
- Bugfix: correct potential issue with building edgectl on Windows
- Internal: fix an error with an undefined Python type in the TLS test (thanks, [Christian Clauss](https://github.com/cclauss)!)

### Ambassador Edge Stack only

- Feature: make the `External` filter type fully compatible with the `AuthService` type
- Docs: add instructions for what to do after downloading `edgectl`
- Bugfix: make it much faster to apply the Edge Stack License
- Bugfix: make sure the ACME terms-of-service link is always shown
- Bugfix: make the Edge Policy Console more performant

[#2198]: https://github.com/datawire/ambassador/issues/2198
[#2226]: https://github.com/datawire/ambassador/issues/2226
[#2227]: https://github.com/datawire/ambassador/issues/2227

## [1.0.0] January 15, 2020
[1.0.0]: https://github.com/datawire/ambassador/compare/v0.86.1...v1.0.0

### Caution!

All of Ambassador's CRDs have been switched to `apiVersion: getambassador.io/v2`, and
**your resources will be upgraded when you apply the new CRDs**. We recommend that you
follow the [migration instructions](https://getambassador.io/early-access/user-guide/upgrade-to-edge-stack/) and check your installation's
behavior before upgrading your CRDs.

## Ambassador API Gateway + Ambassador Edge Stack

### Breaking changes

- When a resource specifies a service or secret name without a corresponding namespace, Ambassador will now
  look for the service or secret in the namespace of the resource that mentioned it. In the past, Ambassador
  would look in the namespace in which Ambassador was running.

### Features

- The Host CR provides an easy way to tell Ambassador about domains it should expect to handle, and
  how it should handle secure and insecure requests for those domains
- Redirection from HTTP to HTTPS defaults to ON when termination contexts are present
- Mapping and Host CRs, as well as Ingress resources, get Status updates to provide better feedback
- Improve performance of processing events from Kubernetes
- Automatic HTTPS should work with any ACME clients doing the http-01 challenge

### Bugfixes

- CORS now happens before rate limiting
- The reconfiguration engine is better protected from exceptions
- Don’t try to check for upgrades on every UI snapshot update
- Reduced reconfiguration churn
- Don't force SNI routes to be lower-priority than non-SNI routes
- Knative mappings fallback to the Ambassador namespace if no namespace is specified
- Fix `ambassador_id` handling for Knative resources
- Treat `ambassadorId` as a synonym for `ambassador_id` (`ambassadorId` is the Protobuf 3 canonical form of `ambassador_id`)

### Ambassador Edge Stack

Ambassador Edge Stack incorporates the functionality of the old Ambassador Pro product.

- Authentication and ratelimiting are now available under a free community license
- Given a Host CR, Ambassador can manage TLS certificates using ACME (or you can manage them by hand)
- There is now an `edgectl` program that you can use for interacting with Ambassador from the command line
- There is a web user-interface for Ambassador
- BREAKING CHANGE: `APP_LOG_LEVEL` is now `AES_LOG_LEVEL`

See the [`CHANGELOG.old-pro.md`](./CHANGELOG.old-pro.md) file for the changelog of
the old Ambassador Pro product.

## [1.0.0-rc6] January 15, 2020
[1.0.0-rc6]: https://github.com/datawire/ambassador/compare/v1.0.0-rc4...v1.0.0-rc6

 - AES: Bugfix: Fix ACME client with multiple replicas
 - AES: Bugfix: Fix ACME client race conditions with the API server and WATT
 - AES: Bugfix: Don't crash in the ACME client if Redis is unavailable

## [1.0.0-rc4] January 13, 2020
[1.0.0-rc4]: https://github.com/datawire/ambassador/compare/v1.0.0-rc1...v1.0.0-rc4

- Change: Less verbose yet more useful Ambassador pod logs
- Bugfix: Various bugfixes for listeners and route rejection
- Bugfix: Don't append the service namespace for `localhost`
- AES: Bugfix: Fix rendering mapping labels YAML in the webui
- AES: Bugfix: Organize help output from `edgectl` so it is easier to read
- AES: Bugfix: Various bugfixes around ACME support with manually-configured TLSContexts
- AES: Change: Don't disable scout or enable extra-verbose logging when migrating from OSS
- AES: BREAKING CHANGE: `APP_LOG_LEVEL` is now `AES_LOG_LEVEL`

## [1.0.0-rc1] January 11, 2020
[1.0.0-rc1]: https://github.com/datawire/ambassador/compare/v1.0.0-rc0...v1.0.0-rc1

- Internal: Improvements to release machinery
- Internal: Fix the dev shell
- Internal: Adjust KAT tests to work with the Edge Stack

## [1.0.0-rc0] January 10, 2020
[1.0.0-rc0]: https://github.com/datawire/ambassador/compare/v1.0.0-ea13...v1.0.0-rc0

- BREAKING CHANGE: Rename Host CR status field `reason` to `errorReason`
- Feature: Host CRs now default `.spec.hostname` to `.metadata.name`
- Feature: Host CRs now have a `requestPolicy` field to control redirecting from cleartext to TLS
- Feature: Redirecting from cleartext to TLS no longer interferes with ACME http-01 challenges
- Feature: Improved `edgectl` help and informational messages
- Bugfix: Host CR status is now a sub-resource
- Bugfix: Have diagd snapshot JSON not include "serialization" keys (which could potentially leak secrets)
- Bugfix: Fix `ambassador_id` handling for Knative resources
- Bugfix: Use the correct namespace for resources found via annotations
- Bugfix: Treat `ambassadorId` as a synonym for `ambassador_id` (`ambassadorId` is the Protobuf 3 canonical form of `ambassador_id`)
- Internal: Allow passing a `DOCKER_NETWORK` variable to the build-system

## [1.0.0-ea13] January 09, 2020
[1.0.0-ea13]: https://github.com/datawire/ambassador/compare/v1.0.0-ea12...v1.0.0-ea13

- Bugfix: Knative mappings populate and fallback to the Ambassador namespace if unspecified
- Internal: Knative tests for versions 0.7.1 and 0.8.0 were removed
- Internal: Knative tests for version 0.11.0 were added
- Internal: Improved performance with Edge Stack using /ambassador/v0/diag/ with an optional `patch_client` query param to send a partial representation in JSON Patch format, reducing the memory and network traffic for large deployments
- Internal: Silencing warnings from `which` in docs preflight-check

## [1.0.0-ea12] January 08, 2020
[1.0.0-ea12]: https://github.com/datawire/ambassador/compare/v1.0.0-ea9...v1.0.0-ea12

- BREAKING CHANGE: When a resource specifies a service or secret name without a corresponding namespace, Ambassador uses the namespace of the resource. In the past, Ambassador would use its own namespace.
- Bugfix: Add the appropriate label so Ingress works with Edge Stack
- Bugfix: Remove superfluous imagePullSecret
- Bugfix: Fix various admin UI quirks, especially in Firefox
  - Bogus warnings about duplicate resources
  - Drag-and-drop reordering of rate limit configuration
  - Missing icons
- Internal: Drop duplicated resources earlier in the processing chain
- Internal: Streamline code generation from protobufs
- Internal: Automated broken-link checks in the documentation

## [1.0.0-ea9] December 23, 2019
[1.0.0-ea9]: https://github.com/datawire/ambassador/compare/v1.0.0-ea7...v1.0.0-ea9

- Bugfix: Use proper executable name for Windows edgectl
- Bugfix: Don't force SNI routes to be lower-priority than non-SNI routes
- Bugfix: Prevent the self-signed fallback context from conflicting with a manual context

## [1.0.0-ea7] December 19, 2019
[1.0.0-ea7]: https://github.com/datawire/ambassador/compare/v1.0.0-ea6...v1.0.0-ea7

- Bugfix: UI buttons can hide themselves
- Bugfix: Developer Portal API acquisition
- Bugfix: Developer Portal internal routing
- Internal: Better JS console usage
- Internal: Rationalize usage reporting for Edge Stack

## [1.0.0-ea6] December 18, 2019
[1.0.0-ea6]: https://github.com/datawire/ambassador/compare/v1.0.0-ea5...v1.0.0-ea6

- Feature: Improve performance of processing events from Kubernetes
- Feature: Automatic HTTPS should work with any ACME clients doing the http-01 challenge
- Internal: General improvements to test infrastructure
- Internal: Improved the release process

`ambassador-internal-access-control` `Filter` and `FilterPolicy` are now
created internally. Remove them from your cluster if upgrading from a
previous version.

## [1.0.0-ea5] December 17, 2019
[1.0.0-ea5]: https://github.com/datawire/ambassador/compare/v1.0.0-ea3...v1.0.0-ea5

- Internal: Improved the reliability of CI
- Internal: Improved the release process

## [1.0.0-ea3] December 16, 2019
[1.0.0-ea3]: https://github.com/datawire/ambassador/compare/v1.0.0-ea1...v1.0.0-ea3

- Feature: initial edgectl support for Windows!
- UX: be explicit that seeing the license applied can take a few minutes
- Bugfix: don’t try to check for upgrades on every UI snapshot update
- Bugfix: don’t activate the fallback TLSContext if its secret is not available
- Bugfix: first cut at reducing reconfiguration churn

## [1.0.0-ea1] December 10, 2019
[1.0.0-ea1]: https://github.com/datawire/ambassador/compare/v0.85.0...v1.0.0-ea1

### Caution!

All of Ambassador's CRDs have been switched to `apiVersion: getambassador.io/v2`, and
**your resources will be upgraded when you apply the new CRDs**. We recommend that you
follow the [migration instructions](https://getambassador.io/early-access/user-guide/upgrade-to-edge-stack/) and check your installation's
behavior before upgrading your CRDs.

### Features

- Authentication and ratelimiting are now available under a free community license
- The Host CRD provides an easy way to tell Ambassador about domains it should expect to handle
- Given a Host CRD, Ambassador can manage TLS certificates using ACME (or you can manage them by hand)
- Redirection from HTTP to HTTPS defaults to ON when termination contexts are present
- Mapping and Host CRDs, as well as Ingress resources, get Status updates to provide better feedback

### Bugfixes

- CVE-2019–18801, CVE-2019–18802, and CVE-2019–18836 are fixed by including Envoy 1.12.2
- CORS now happens before rate limiting
- The reconfiguration engine is better protected from exceptions

## [0.86.1] December 10, 2019
[0.86.1]: https://github.com/datawire/ambassador/compare/v0.84.1...v0.86.1

- Envoy updated to 1.12.2 for security fixes
- Envoy TCP keepalives are now supported (thanks, [Bartek Kowalczyk](https://github.com/KowalczykBartek)!)
- Envoy remote access logs are now supported
- Correctly handle upgrades when the `LogService` CRD is not present

(Ambassador 0.86.0 was superseded by Ambassador 0.86.1.)

## [0.85.0] October 22, 2019
[0.85.0]: https://github.com/datawire/ambassador/compare/v0.84.1...v0.85.0

### Features

- Support configuring the Envoy access log format (thanks to [John Esmet](https://github.com/esmet)!)

## [0.84.1] October 20, 2019
[0.84.1]: https://github.com/datawire/ambassador/compare/v0.84.0...v0.84.1

### Major changes:
- Bugfix: Fix /ambassador permissions to allow running as non-root - Thanks @dmayle (https://github.com/dmayle) for reporting the bug.

## [0.84.0] October 18, 2019
[0.84.0]: https://github.com/datawire/ambassador/compare/v0.83.0...v0.84.0

### Features:

- Support setting window_bits for the GZip filter (thanks to [Florent Delannoy](https://github.com/Pluies)!)
- Correctly support tuning the regex_max_size, and bump its default to 200 (thanks to [Paul Salaberria](https://github.com/psalaberria002)!)
- Support setting redirect_cleartext_from in a TLSContext

### Bugfixes:

- Correctly update loadbalancer status of Ingress resources
- Don't enable diagd debugging in the test suite unless explicitly requested (thanks to [Jonathan Suever](https://github.com/suever)!)
- Switch to an Envoy release build

### Developer Notes:

- Many many things about the build system have changed under the hood!
   - Start with `make help`, and
   - Join our [Slack channel](https://d6e.co/slack) for more help!

## [0.83.0] October 08, 2019
[0.83.0]: https://github.com/datawire/ambassador/compare/v0.82.0...v0.83.0

### Major changes:
- Update Ambassador to address CVE-2019-15225 and CVE-2019-15226.

NOTE: this switches the default regex engine! See the documentation for the `ambassador` `Module` for more.

## [0.82.0] October 02, 2019
[0.82.0]: https://github.com/datawire/ambassador/compare/v0.81.0...v0.82.0

### Major changes:
- Feature: Arrange for the Prometheus metrics endpoint to also return associated headers (thanks, [Jennifer Wu](https://github.com/jhsiaomei)!)
- Feature: Support setting a TLS origination context when doing TLS to a RateLimitService (thanks, [Phil Peble](https://github.com/ppeble)!)
- Feature: Allow configuring Envoy's access log path (thanks, [Jonathan Suever](https://github.com/suever)!)
- Update: Switch to Python 3.7 and Alpine 3.10

### Developer notes:
- Switch back to the latest mypy (currently 0.730)
- Environment variable KAT_IMAGE_PULL_POLICY can override the imagePullPolicy when running KAT tests
- Updated Generated Envoy Golang APIs

## [0.81.0] September 26, 2019
[0.81.0]: https://github.com/datawire/ambassador/compare/v0.80.0...v0.81.0

### Major changes:
- Feature: ${} environment variable interpolation is supported in all Ambassador configuration resources (thanks, [Stefan Sedich](https://github.com/stefansedich)!)
- Feature: DataDog APM tracing is now supported (thanks again, [Stefan Sedich](https://github.com/stefansedich)!)
- Bugfix: Fix an error in the TLSContext schema (thanks, [@georgekaz](https://github.com/georgekaz)!)

### Developer notes:
- Test services can now be built, deployed, and tested more easily (see BUILDING.md)
- `mypy` is temporarily pinned to version 0.720.

## [0.80.0] September 20, 2019
[0.80.0]: https://github.com/datawire/ambassador/compare/v0.78.0...v0.80.0

### Major changes:
- Feature: Basic support for the Kubernetes Ingress resource
- Feature: Basic reporting for some common configuration errors (lack of Mappings, lack of TLS contexts)
- Bugfix: Update Envoy to prevent crashing when updating AuthService under load

### Developer notes
- Golang components now use Go 1.13
- Ambassador build now _requires_ clean type hinting
- KAT client and server have been pulled back into the Ambassador repo

## [0.78.0] September 11, 2019
[0.78.0]: https://github.com/datawire/ambassador/compare/v0.77.0...v0.78.0

### Major changes:
- Feature: Support setting cipher_suites and ecdh_curves in TLSContext - #1782 (Thanks @teejaded)
- Feature: Make 128-bits traceids the default - #1794 (Thanks @Pluies)
- Feature: Set cap_net_bind_service to allow binding to low ports - #1720 (Thanks @swalberg)

### Minor changes:
- Testing: Add test that ambassador cli does not crash when called with --help - #1806 (Thanks @rokostik)

## [0.77.0] September 05, 2019
[0.77.0]: https://github.com/datawire/ambassador/compare/v0.76.0...v0.77.0

- (Feature) Support the `least_request` load balancer policy (thanks, [Steve Flanders](https://github.com/flands)!)
- (Misc) Many test and release-engineering improvements under the hood

## [0.76.0] August 26, 2019
[0.76.0]: https://github.com/datawire/ambassador/compare/v0.75.0...v0.76.0

- circuit breakers now properly handle overriding a global circuit breaker within a Mapping ([#1767])
- support for Knative 0.8.0 ([#1732])

[#1767]: https://github.com/datawire/ambassador/issues/1767
[#1732]: https://github.com/datawire/ambassador/issues/1732

## [0.75.0] August 13, 2019
[0.75.0]: https://github.com/datawire/ambassador/compare/0.74.1...0.75.0

- (Feature) Update to Envoy 1.11.1, including security fixes
- (Feature) You can use a `TLSContext` without a `secret` to set origination options ([#1708])
- (Feature) Canary deployments can now use multiple `host_rewrite` values ([#1159])
- (Bugfix) Make sure that Ambassador won't mistakenly complain about the number of RateLimit and Tracing services (thanks, [Christian Claus](https://github.com/cclauss)!)

[#1159]: https://github.com/datawire/ambassador/issues/1159
[#1708]: https://github.com/datawire/ambassador/issues/1708

## [0.74.1] August 06, 2019
[0.74.1]: https://github.com/datawire/ambassador/compare/0.74.0...0.74.1

- (bugfix) Make sure that updates properly trigger reconfigures ([#1727])
- (misc) Arrange for startup logging to have timestamps

[#1727]: https://github.com/datawire/ambassador/issues/1727

## [0.74.0] July 30, 2019
[0.74.0]: https://github.com/datawire/ambassador/compare/0.73.0...0.74.0

- Bugfix: Make sure that the pod dies if Envoy dies
- Bugfix: Correctly allow setting `timeout_ms` for `AuthService` (thanks, [John Esmet!](https://www.github.com/esmet)!)
- Feature: Permit configuring `cluster_idle_timeout_ms` for upstream services (thanks, [John Esmet!](https://www.github.com/esmet)!) ([#1542])

[#1542]: https://github.com/datawire/ambassador/issues/1542

## [0.73.0] July 11, 2019
[0.73.0]: https://github.com/datawire/ambassador/compare/0.72.0...0.73.0

- Feature: Experimental native support for Knative! ([#1579])
- Feature: Better Linkerd interoperability! ([#1578], [#1594])

- Feature: Add a legend for the colors of service names on the diagnostic overview (thanks, [Wyatt Pearsall](https://github.com/wpears)!)
- Feature: Allow switching Envoy to output JSON logs (thanks, [Pedro Tavares](https://github.com/ServerlessP)!)
- Feature: Allow setting `AMBASSADOR_LABEL_SELECTOR` and `AMBASSADOR_FIELD_SELECTOR` to let Ambassador use Kubernetes selectors to determine which things to read (thanks, [John Esmet](https://github.com/esmet)!) ([#1292])
- Feature: Allow configuring retries for `AuthService` (thanks, [Kevin Dagostino](https://github.com/TonkWorks)!) ([#1622], [#1461])

- Bugfix: Allow Ambassador to ride through Envoy-validation timeouts (thanks, [John Morrisey](https://github.com/jwm)!)
- Bugfix: Allow Ambassador to ride through parse errors on input resources (thanks, [Andrei Predoiu](https://github.com/Andrei-Predoiu)!) ([#1625])
- Bugfix: Allow '.' in a `secret` name to just be a '.' ([#1255])

- Bugfix: Allow manually defining an Ambassador `Service` resource, same as any other resource
- Bugfix: Prevent spurious duplicate-resource errors when loading config from the filesystem

[#1255]: https://github.com/datawire/ambassador/issues/1255
[#1292]: https://github.com/datawire/ambassador/issuse/1292
[#1461]: https://github.com/datawire/ambassador/issues/1461
[#1578]: https://github.com/datawire/ambassador/issuse/1578
[#1579]: https://github.com/datawire/ambassador/issuse/1579
[#1594]: https://github.com/datawire/ambassador/issuse/1594
[#1622]: https://github.com/datawire/ambassador/issues/1622
[#1625]: https://github.com/datawire/ambassador/issues/1625

## [0.72.0] June 13, 2019
[0.72.0]: https://github.com/datawire/ambassador/compare/0.71.0...0.72.0

- Envoy: Update Envoy to commit 8f57f7d765
- Bugfix: Auth spans are now properly connected to requests ([#1414])
- Bugfix: `include_body` now works correctly ([#1531], [#1595])
- Bugfix: `x_forwarded_proto_redirect` works again (thanks to [Kyle Martin](https://github.com/KyleMartin901)!) ([#1571])
- Bugfix: Ambassador works correctly with read-only filesystems (thanks, [Niko Kurtti](https://github.com/n1koo)!) ([#1614], [#1619])
- Bugfix: Correctly render groups associated with a given resolver in diagnostics JSON output
- Feature: Give the Ambassador CLI a way to specify the directory into which to write secrets.

[#1414]: https://github.com/datawire/ambassador/issues/1414
[#1531]: https://github.com/datawire/ambassador/issues/1531
[#1571]: https://github.com/datawire/ambassador/issues/1571
[#1595]: https://github.com/datawire/ambassador/issues/1595
[#1614]: https://github.com/datawire/ambassador/issues/1614
[#1619]: https://github.com/datawire/ambassador/issues/1619

## [0.71.0] June 06, 2019
[0.71.0]: https://github.com/datawire/ambassador/compare/0.70.1...0.71.0

- Feature: GZIP support [#744]
- Feature: diag UI shows active Resolvers [#1453]
- Feature: CRDs exist for Resolvers [#1563]
- Feature: Resolvers with custom names work, even as CRDs [#1497]
- Feature: The `/metrics` endpoint provides direct access to Prometheus-format stats (thanks to [Rotem Tamir](https://github.com/rotemtam)!)
- Bugfix: `statsd-exporter` now correctly defaults to port 8125 (thanks to [Jonathan Suever](https://github.com/suever)!)
- Bugfix: redirect_cleartext_from no longer strips the URL path [#1463]
- Bugfix: canary weights of 0 and 100 work correctly [#1379]
- Bugfix: `docker run` works again for the Ambassador demo, and is part of our tests now [#1569]
- Bugfix: Scout `DEBUG` messages don’t get leaked into the diag UI [#1573]
- Maintenance: warn of upcoming protocol version changes
- Maintenance: check in with Scout every 24 hours, but no more than twice per day

[#744]: https://github.com/datawire/ambassador/issues/744
[#1379]: https://github.com/datawire/ambassador/issues/1379
[#1453]: https://github.com/datawire/ambassador/issues/1453
[#1463]: https://github.com/datawire/ambassador/issues/1463
[#1497]: https://github.com/datawire/ambassador/issues/1497
[#1563]: https://github.com/datawire/ambassador/issues/1563
[#1569]: https://github.com/datawire/ambassador/issues/1569
[#1573]: https://github.com/datawire/ambassador/issues/1573

## [0.70.1] May 24, 2019
[0.70.1]: https://github.com/datawire/ambassador/compare/0.70.0...0.70.1

### Minor changes:
- Bugfix: Disable CRD support if Ambassador cannot access them
- Upgrade: Upgrade to watt 0.5.1

## [0.70.0] May 20, 2019
[0.70.0]: https://github.com/datawire/ambassador/compare/0.61.0...0.70.0

### Major changes:
- Feature: Support CRDs in the `getambassador.io` API group for configuration ([#482])
- Feature: Update to Envoy 1.10

### Minor changes:
- Feature: Support removing request headers (thanks @ysaakpr!)
- Bugfix: `watt` should better coalesce calls to the watch hook on startup
- Bugfix: Ambassador no longer uses ports 7000 or 18000 ([#1526], [#1527])

[#482]: https://github.com/datawire/ambassador/issues/482
[#1526]: https://github.com/datawire/ambassador/issues/1526
[#1527]: https://github.com/datawire/ambassador/issues/1527

## [0.61.1] May 16, 2019
[0.61.1]: https://github.com/datawire/ambassador/compare/0.61.0...0.61.1

- Bugfix: Make sure that Consul discovery properly handles the datacenter name ([#1533])
- Bugfix: Make sure that the feature-walk code is protected against clusters with no endpoints at all ([#1532])

[#1532]: https://github.com/datawire/ambassador/issues/1532
[#1533]: https://github.com/datawire/ambassador/issues/1533

## [0.61.0] May 08, 2019
[0.61.0]: https://github.com/datawire/ambassador/compare/0.60.3...0.61.0

Ambassador 0.61.0 metadata

### Changes:
- Feature: Support for minimum and maximum TLS versions (#689)
- Feature: Allow choosing whether to append or overwrite when adding request or response headers (#1481) - thanks to @ysaakpr
- Feature: Support for circuit breakers (#360)
- Feature: Support for automatic retries (#1127) - thanks to @l1v3
- Feature: Support for shadow traffic weighting - thanks to @nemo83
- Feature: Support for HTTP/1.0 (#988) - thanks to @cyrus-mc
- Bugfix: Problem with local Consul agent resolver and non-standard HTTP port (#1508)
- Bugfix: Round each mapping's weight to an integer to prevent invalid Envoy configurations when using weights (#1289) - thanks to @esmet
- Bugfix: Fix deadlock on invalid Envoy configuration (#1491) - thanks to @esmet
- Bugfix: Fixed LightStep gRPC TracingService (#1189) - thanks to @sbaum1994
## [0.60.3] May 01, 2019
[0.60.3]: https://github.com/datawire/ambassador/compare/0.60.2...0.60.3

### Changes since 0.60.2

- When scanning its configuration for secrets and endpoints that must be watched, 0.60.2 could fail with certain configurations if TLS termination but not origination was active. Those failures are fixed now.

## [0.60.2] April 29, 2019
[0.60.2]: https://github.com/datawire/ambassador/compare/0.60.1...0.60.2

### Changes since 0.60.1

- Ambassador is now much more careful about which endpoints and secrets it pays attention to. ([#1465] again -- thanks to [@flands](https://github.com/flands) and @seandon for the help here!)

[#1465]: https://github.com/datawire/ambassador/issues/1465

## [0.60.1] April 25, 2019
[0.60.1]: https://github.com/datawire/ambassador/compare/0.60.0...0.60.1

### Changes since 0.60.0

- Speed up initial parsing of WATT snapshots considerably ([#1465])
- Don't look at secrets in the kube-system namespace, or for service-account tokens.
- Make sure that secrets we do look at are correctly associated with their namespaces ([#1467] -- thanks to @flands and @derrickburns for their contributions here!)
- Allow tuning the number of input snapshots retained for debugging
- Include the grab-snapshots.py script to help with debuggability

[#1465]: https://github.com/datawire/ambassador/issues/1465
[#1467]: https://github.com/datawire/ambassador/issues/1467

## [0.60.0] April 23, 2019
[0.60.0]: https://github.com/datawire/ambassador/compare/0.53.1...0.60.0

### Changes since 0.53.1

- BREAKING CHANGE: Ambassador listens on 8080 and 8443 by default so it does not need to run as root
- Ambassador natively supports using Consul for service discovery
- `AMBASSADOR_ENABLE_ENDPOINTS` is no longer needed; configure using the `Resolver` resource instead
- Support for the Maglev load balancing algorithm
- Support `connect_timeout_ms`. Thanks to Pétur Erlingsson.
- Support for `idle_timeout_ms` Thanks to Aaron Triplett.
- Ambassador will properly reload renewed Let's Encrypt certificates (#1416). Thanks to Matthew Ceroni.
- Ambassador will now properly redirect from HTTP to HTTPS based on `x-forwarded-proto` (#1233).
- The `case_sensitive` field now works when `host_redirect` is set to true (#699). Thanks to Peter Choi and Christopher Coté.

## [0.53.1] April 05, 2019
[0.53.1]: https://github.com/datawire/ambassador/compare/0.52.1...0.53.1

(0.53.0 was immediately supplanted by 0.53.1.)

## SECURITY FIXES

Ambassador 0.53.1 addresses two security issues in Envoy Proxy, CVE-2019-9900 and CVE-2019-9901:

- CVE-2019-9900 (Score 8.3/High). When parsing HTTP/1.x header values, Envoy 1.9 and before does not reject embedded zero characters (NUL, ASCII 0x0).

- CVE-2019-9901 (Score 8.3/High). Envoy does not normalize HTTP URL paths in Envoy 1.9 and before.

Since these issues can potentially allow a remote attacker to use maliciously-crafted URLs to bypass
authentication, anyone running an Ambassador prior to 0.53.1 should upgrade.

### UPCOMING CHANGES

Ambassador 0.60 will listen on ports 8080/8443 by default. The diagnostics service in Ambassador 0.52.0
will try to warn you if your configuration will be affected by this change.

## Other changes since 0.52.1

- `AuthService` version `ambassador/v1` can now explicitly configure how much body data is sent
  to the external authentication service.

## [0.52.1] March 26, 2019
[0.52.1]: https://github.com/datawire/ambassador/compare/0.52.0...0.52.1

### Changes since 0.52.0

- You can specify the `AMBASSADOR_NO_SECRETS` environment variable to prevent Ambassador from
  watching Kubernetes secrets at all (thanks [@esmet](https://github.com/esmet)!) ([#1293])
- The services used when you do `docker run ambassador --demo` have been moved into the Docker image,
  to remove external dependencies from the Ambassador quickstart.

[#1293]: https://github.com/datawire/ambassador/issues/1293

## [0.52.0] March 21, 2019
[0.52.0]: https://github.com/datawire/ambassador/compare/0.51.2...0.52.0

### Changes since 0.51.2

- Initial support for endpoint routing, rather than relying on `kube-proxy` ([#1031])
   - set `AMBASSADOR_ENABLE_ENDPOINTS` in the environment to allow this
- Initial support for Envoy ring hashing and session affinity (requires endpoint routing!)
- Support Lua filters (thanks to [@lolletsoc](https://github.com/lolletsoc)!)
- Support gRPC-Web (thanks to [@gertvdijk](https://github.com/gertvdijk)!) ([#456])
- Support for gRPC HTTP 1.1 bridge (thanks to [@rotemtam](https://github.com/rotemtam)!)
- Allow configuring `num-trusted-hosts` for `X-Forwarded-For`
- External auth services using gRPC can now correctly add new headers ([#1313])
- External auth services correctly add trace spans
- Ambassador should respond to changes more quickly now ([#1294], [#1318])
- Ambassador startup should be faster now

[#456]: https://github.com/datawire/ambassador/issues/456
[#1031]: https://github.com/datawire/ambassador/issues/1031
[#1294]: https://github.com/datawire/ambassador/issues/1294
[#1313]: https://github.com/datawire/ambassador/issues/1313
[#1318]: https://github.com/datawire/ambassador/issues/1318

## [0.51.2] March 12, 2019
[0.51.2]: https://github.com/datawire/ambassador/compare/0.51.1...0.51.2

### Changes since 0.51.1

- Cookies are now correctly handled when using external auth services... really. ([#1211])

[#1211]: https://github.com/datawire/ambassador/issues/1211

## [0.51.1] March 11, 2019
[0.51.1]: https://github.com/datawire/ambassador/compare/0.51.0...0.51.1

### Changes since 0.51.0

- Ambassador correctly handles services in namespaces other than the one Ambassador is running in.

## [0.51.0] March 08, 2019
[0.51.0]: https://github.com/datawire/ambassador/compare/0.50.3...0.51.0

**0.51.0 is not recommended: upgrade to 0.51.1.**

### Changes since 0.50.3

- Ambassador can now route any TCP connection, using the new `TCPMapping` resource. ([#420])
- Cookies are now correctly handled when using external auth services ([#1211])
- Lots of work in docs and testing under the hood

[#420]: https://github.com/datawire/ambassador/issues/420
[#1211]: https://github.com/datawire/ambassador/issues/1211

### Limitations in 0.51.0

At present, you cannot mix HTTP and HTTPS upstream `service`s in any Ambassador resource. This restriction will be lifted in a future Ambassador release.

## [0.50.3] February 21, 2019
[0.50.3]: https://github.com/datawire/ambassador/compare/0.50.2...0.50.3

### Fixes since 0.50.2

- Ambassador saves configuration snapshots as it manages configuration changes. 0.50.3 keeps only 5 snapshots,
  to bound its disk usage. The most recent snapshot has no suffix; the `-1` suffix is the next most recent, and
  the `-4` suffix is the oldest.
- Ambassador will not check for available updates more often than once every four hours.

### Limitations in 0.50.3

At present, you cannot mix HTTP and HTTPS upstream `service`s in any Ambassador resource. This restriction will be lifted in a future Ambassador release.

## [0.50.2] February 15, 2019
[0.50.2]: https://github.com/datawire/ambassador/compare/0.50.1...0.50.2

### Important fixes since 0.50.1

- Ambassador no longer requires annotations in order to start -- with no configuration, it will launch with only the diagnostics service available. ([#1203])
- If external auth changes headers, routing will happen based on the changed values. ([#1226])

### Other changes since 0.50.1

- Ambassador will no longer log errors about Envoy statistics being unavaible before startup is complete ([#1216])
- The `tls` attribute is again available to control the client certificate offered by an `AuthService` ([#1202])

### Limitations in 0.50.2

At present, you cannot mix HTTP and HTTPS upstream `service`s in any Ambassador resource. This restriction will be lifted in a future Ambassador release.

[#1202]: https://github.com/datawire/ambassador/issues/1202
[#1203]: https://github.com/datawire/ambassador/issues/1203
[#1216]: https://github.com/datawire/ambassador/issues/1216
[#1226]: https://github.com/datawire/ambassador/issues/1226

## [0.50.1] February 07, 2019
[0.50.1]: https://github.com/datawire/ambassador/compare/0.50.0...0.50.1

**0.50.1 is not recommended: upgrade to 0.52.0.**

### Changes since 0.50.0

- Ambassador defaults to only doing IPv4 DNS lookups. IPv6 can be enabled in the Ambassador module or in a Mapping. ([#944])
- An invalid Envoy configuration should not cause Ambassador to hang.
- Testing using `docker run` and `docker compose` is supported again. ([#1160])
- Configuration from the filesystem is supported again, but see the "Running Ambassador" documentation for more.
- Datawire's default Ambassador YAML no longer asks for any permissions for `ConfigMap`s.

[#944]: https://github.com/datawire/ambassador/issues/944
[#1160]: https://github.com/datawire/ambassador/issues/1160

## [0.50.0] January 29, 2019
[0.50.0]: https://github.com/datawire/ambassador/compare/0.50.0-rc6...0.50.0

**Ambassador 0.50.0 is a major rearchitecture of Ambassador onto Envoy V2 using the ADS. See the "BREAKING NEWS"
section above for more information.**

(Note that Ambassador 0.50.0-rc7 and -rc8 were internal releases.)

### Changes since 0.50.0-rc6

- `AMBASSADOR_SINGLE_NAMESPACE` is finally correctly supported and properly tested ([#1098])
- Ambassador won't throw an exception for name collisions between resources ([#1155])
- A TLS `Module` can now coexist with SNI (the TLS `Module` effectively defines a fallback cert) ([#1156])
- `ambassador dump --diag` no longer requires you to explicitly state `--v1` or `--v2`

### Limitations in 0.50.0 GA

- Configuration from the filesystem is not supported in 0.50.0. It will be resupported in 0.50.1.
- A `TLSContext` referencing a `secret` in another namespace will not function when `AMBASSADOR_SINGLE_NAMESPACE` is set.

[#1098]: https://github.com/datawire/ambassador/issues/1098
[#1155]: https://github.com/datawire/ambassador/issues/1155
[#1156]: https://github.com/datawire/ambassador/issues/1156

## [0.50.0-rc6] January 28, 2019
[0.50.0-rc6]: https://github.com/datawire/ambassador/compare/0.50.0-rc5...0.50.0-rc6

**Ambassador 0.50.0-rc6 is a release candidate**.

### Changes since 0.50.0-rc5

- Ambassador watches certificates and automatically updates TLS on certificate changes ([#474])
- Ambassador no longer saves secrets it hasn't been told to use to disk ([#1093])
- Ambassador correctly honors `AMBASSADOR_SINGLE_NAMESPACE` rather than trying to access all namespaces ([#1098])
- Ambassador correctly honors the `AMBASSADOR_CONFIG_BASE_DIR` setting again ([#1118])
- Configuration changes take effect much more quickly than in RC5 ([#1148])
- `redirect_cleartext_from` works with no configured secret, to support TLS termination at a downstream load balancer ([#1104])
- `redirect_cleartext_from` works with the `PROXY` protocol ([#1115])
- Multiple `AuthService` resources (for canary deployments) work again ([#1106])
- `AuthService` with `allow_request_body` works correctly with an empty body and no `Content-Length` header ([#1140])
- `Mapping` supports the `bypass_auth` attribute to bypass authentication (thanks, @patricksanders! [#174])
- The diagnostic service no longer needs to re-parse the configuration on every page load ([#483])
- Startup is now faster and more stable
- The Makefile should do the right thing if your PATH has spaces in it (thanks, @er1c!)
- Lots of Helm chart, statsd, and doc improvements (thanks, @Flydiverny, @alexgervais, @bartlett, @victortv7, and @zencircle!)

[#174]: https://github.com/datawire/ambassador/issues/174
[#474]: https://github.com/datawire/ambassador/issues/474
[#483]: https://github.com/datawire/ambassador/issues/483
[#1093]: https://github.com/datawire/ambassador/issues/1093
[#1098]: https://github.com/datawire/ambassador/issues/1098
[#1104]: https://github.com/datawire/ambassador/issues/1104
[#1106]: https://github.com/datawire/ambassador/issues/1106
[#1115]: https://github.com/datawire/ambassador/issues/1115
[#1118]: https://github.com/datawire/ambassador/issues/1118
[#1140]: https://github.com/datawire/ambassador/issues/1140
[#1148]: https://github.com/datawire/ambassador/issues/1148

## [0.50.0-rc5] January 14, 2019
[0.50.0-rc5]: https://github.com/datawire/ambassador/compare/0.50.0-rc4...0.50.0-rc5

**Ambassador 0.50.0-rc5 is a release candidate**.

### Changes since 0.50.0-rc4

- Websocket connections will now be authenticated if an AuthService is configured [#1026]
- Client certificate authentication should function whether configured from a TLSContext resource or from the the old-style TLS module (this is the full fix for [#993])
- Ambassador can now switch listening ports without a restart (e.g. switching from cleartext to TLS) [#1100]
- TLS origination certificates (including Istio mTLS) should now function [#1071]
- The diagnostics service should function in all cases. [#1096]
- The Ambassador image is significantly (~500MB) smaller than RC4.

[#933]: https://github.com/datawire/ambassador/issues/993
[#1026]: https://github.com/datawire/ambassador/issues/1026
[#1071]: https://github.com/datawire/ambassador/issues/1071
[#1096]: https://github.com/datawire/ambassador/issues/1096
[#1100]: https://github.com/datawire/ambassador/issues/1100

## [0.50.0-rc4] January 09, 2019
[0.50.0-rc4]: https://github.com/datawire/ambassador/compare/0.50.0-rc3...0.50.0-rc4

**Ambassador 0.50.0-rc4 is a release candidate**, and fully supports running under Microsoft Azure.

### Changes since 0.50.0-rc3

- Ambassador fully supports running under Azure [#1039]
- The `proto` attribute of a v1 `AuthService` is now optional, and defaults to `http`
- Ambassador will warn about the use of v0 configuration resources.

[#1039]: https://github.com/datawire/ambassador/issues/1039

## [0.50.0-rc3] January 03, 2019
[0.50.0-rc3]: https://github.com/datawire/ambassador/compare/0.50.0-rc2...0.50.0-rc3

**Ambassador 0.50.0-rc3 is a release candidate**, but see below for an important warning about Azure.

### Microsoft Azure

There is a known issue with recently-created Microsoft Azure clusters where Ambassador will stop receiving service
updates after running for a short time. This will be fixed in 0.50.0-GA.

### Changes since 0.50.0-rc2

- The `Location` and `Set-Cookie` headers should always be allowed from the auth service when using an `ambassador/v0` config [#1054]
- `add_response_headers` (parallel to `add_request_headers`) is now supported (thanks, @n1koo!)
- `host_redirect` and `shadow` both now work correctly [#1057], [#1069]
- Kat is able to give better information when it cannot parse a YAML specification.

[#1054]: https://github.com/datawire/ambassador/issues/1054
[#1057]: https://github.com/datawire/ambassador/issues/1057
[#1069]: https://github.com/datawire/ambassador/issues/1069

## [0.50.0-rc2] December 24, 2018
[0.50.0-rc2]: https://github.com/datawire/ambassador/compare/0.50.0-rc1...0.50.0-rc2

**Ambassador 0.50.0-rc2 fixes some significant TLS bugs found in RC1.**

### Changes since 0.50.0-rc1:

- TLS client certificate verification should function correctly (including requiring client certs).
- TLS context handling (especially with multiple contexts and origination contexts) has been made more consistent and correct.
    - Ambassador is now much more careful about reporting errors in TLS configuration (especially around missing keys).
    - You can reference a secret in another namespace with `secret: $secret_name.$namespace`.
    - Ambassador will now save certificates loaded from Kubernetes to `$AMBASSADOR_CONFIG_BASE_DIR/$namespace/secrets/$secret_name`.
- `use_proxy_proto` should be correctly supported [#1050].
- `AuthService` v1 will default its `proto` to `http` (thanks @flands!)
- The JSON diagnostics service supports filtering: requesting `/ambassador/v0/diag/?json=true&filter=errors`, for example, will return only the errors element from the diagnostic output.

[#1050]: https://github.com/datawire/ambassador/issues/1050

## [0.50.0-rc1] December 19, 2018
[0.50.0-rc1]: https://github.com/datawire/ambassador/compare/0.50.0-ea7...0.50.0-rc1

**Ambassador 0.50.0-rc1 is a release candidate.**

### Changes since 0.50.0-ea7:

- Websockets should work happily with external authentication [#1026]
- A `TracingService` using a long cluster name works now [#1025]
- TLS origination certificates are no longer offered to clients when Ambassador does TLS termination [#983]
- Ambassador will listen on port 443 only if TLS termination contexts are present; a TLS origination context will not cause the switch
- The diagnostics service is working, and correctly reporting errors, again. [#1019]
- `timeout_ms` in a `Mapping` works correctly again [#990]
- Ambassador sends additional anonymized usage data to help Datawire prioritize bug fixes, etc.
  See `docs/ambassador/running.md` for more information, including how to disable this function.

[#983]: https://github.com/datawire/ambassador/issues/983
[#990]: https://github.com/datawire/ambassador/issues/990
[#1019]: https://github.com/datawire/ambassador/issues/1019
[#1025]: https://github.com/datawire/ambassador/issues/1025
[#1026]: https://github.com/datawire/ambassador/issues/1026

## [0.50.0-ea7] November 19, 2018
[0.50.0-ea7]: https://github.com/datawire/ambassador/compare/0.50.0-ea6...0.50.0-ea7

**Ambassador 0.50.0-ea7 is an EARLY ACCESS release! IT IS NOT SUPPORTED FOR PRODUCTION USE.**

### Upcoming major changes:

- **API version `ambassador/v0` will be officially deprecated in Ambassador 0.50.0.**
  API version `ambassador/v1` will the minimum recommended version for resources in Ambassador 0.50.0.

- Some resources will change between `ambassador/v0` and `ambassador/v1`.
   - For example, the `Mapping` resource will no longer support `rate_limits` as that functionality will
     be subsumed by `labels`.

### Changes since 0.50.0-ea6:

- Ambassador now supports `labels` for all `Mapping`s.
- Configuration of rate limits for a `Mapping` is now handled by providing `labels` in the domain configured
  for the `RateLimitService` (by default, this is "ambassador").
- Ambassador, once again, supports `statsd` for statistics gathering.
- The Envoy `buffer` filter is supported.
- Ambassador can now use GRPC to call the external authentication service, and also include the message body
  in the auth call.
- It's now possible to use environment variables to modify the configuration directory (thanks @n1koo!).
- Setting environment variable `AMBASSADOR_KUBEWATCH_NO_RETRY` will cause the Ambassador pod to exit, and be
  rescheduled, if it loses its connection to the Kubernetes API server.
- Many dependencies have been updated, most notably including switching to kube-client 8.0.0.

## [0.50.0-ea6] November 19, 2018
[0.50.0-ea6]: https://github.com/datawire/ambassador/compare/0.50.0-ea5...0.50.0-ea6

**Ambassador 0.50.0-ea6 is an EARLY ACCESS release! IT IS NOT SUPPORTED FOR PRODUCTION USE.**

### Changes since 0.50.0-ea5:

- `alpn_protocols` is now supported in the `TLS` module and `TLSContext`s
- Using `TLSContext`s to provide TLS termination contexts will correctly switch Ambassador to listening on port 443.
- `redirect_cleartext_from` is now supported with SNI
- Zipkin `TracingService` configuration now supports 128-bit trace IDs and shared span contexts (thanks, @alexgervais!)
- Zipkin should correctly trace calls to external auth services (thanks, @alexgervais!)
- `AuthService` configurations now allow separately configuring headers allowed from the client to the auth service, and from the auth service upstream
- Ambassador won't endlessly append `:annotation` to K8s resources
- The Ambassador CLI no longer requires certificate files to be present when dumping configurations
- `make mypy` will run full type checks on Ambassador to help developers

## [0.50.0-ea5] November 06, 2018
[0.50.0-ea5]: https://github.com/datawire/ambassador/compare/0.50.0-ea4...0.50.0-ea5

**Ambassador 0.50.0-ea5 is an EARLY ACCESS release! IT IS NOT SUPPORTED FOR PRODUCTION USE.**

### Changes since 0.50.0-ea4:

- **`use_remote_address` is now set to `true` by default.** If you need the old behavior, you will need to manually set `use_remote_address` to `false` in the `ambassador` `Module`.
- Ambassador 0.50.0-ea5 **supports SNI!**  See the docs for more here.
- Header matching is now supported again, including `host` and `method` headers.

## [0.50.0-ea4] October 31, 2018
[0.50.0-ea4]: https://github.com/datawire/ambassador/compare/0.50.0-ea3...0.50.0-ea4

**Ambassador 0.50.0-ea4 is an EARLY ACCESS release! IT IS NOT SUPPORTED FOR PRODUCTION USE.**

### Changes since 0.50.0-ea3:

- Ambassador 0.50.0-ea4 uses Envoy 1.8.0.
- `RateLimitService` is now supported. **You will need to restart Ambassador if you change the `RateLimitService` configuration.** We expect to lift this restriction in a later release; for now, the diag service will warn you when a restart is required.
   - The `RateLimitService` also has a new `timeout_ms` attribute, which allows overriding the default request timeout of 20ms.
- GRPC is provisionally supported, but still needs improvements in test coverage.
- Ambassador will correctly include its EA number when checking for updates.

## [0.50.0-ea3] October 21, 2018
[0.50.0-ea3]: https://github.com/datawire/ambassador/compare/0.50.0-ea2...0.50.0-ea3

**Ambassador 0.50.0-ea3 is an EARLY ACCESS release! IT IS NOT SUPPORTED FOR PRODUCTION USE.**

### Changes since 0.50.0-ea2:

- `TracingService` is now supported. **You will need to restart Ambassador if you change the `TracingService` configuration.** We expect to lift this restriction in a later release; for now, the diag service will warn you when a restart is required.
- Websockets are now supported, **including** mapping the same websocket prefix to multiple upstream services for canary releases or load balancing.
- KAT supports full debug logs by individual `Test` or `Query`.

**Ambassador 0.50.0 is not yet feature-complete. Read the Limitations and Breaking Changes sections in the 0.50.0-ea1 section below for more information.**

## [0.50.0-ea2] October 16, 2018
[0.50.0-ea2]: https://github.com/datawire/ambassador/compare/0.50.0-ea1...0.50.0-ea2

**Ambassador 0.50.0-ea2 is an EARLY ACCESS release! IT IS NOT SUPPORTED FOR PRODUCTION USE.**

### Changes since 0.50.0-ea1:

- Attempting to enable TLS termination without supplying a valid cert secret will result in HTTP on port 80, rather than HTTP on port 443. **No error will be displayed in the diagnostic service yet.** This is a bug and will be fixed in `-ea3`.
- CORS is now supported.
- Logs are no longer full of accesses from the diagnostic service.
- KAT supports isolating OptionTests.
- The diagnostics service now shows the V2 config actually in use, not V1.
- `make` will no longer rebuild the Python venv so aggressively.

**Ambassador 0.50.0 is not yet feature-complete. Read the Limitations and Breaking Changes sections in the 0.50.0-ea1 section below for more information.**

## [0.50.0-ea1] October 11, 2018
[0.50.0-ea1]: https://github.com/datawire/ambassador/compare/0.40.0...0.50.0-ea1

**Ambassador 0.50.0-ea1 is an EARLY ACCESS release! IT IS NOT SUPPORTED FOR PRODUCTION USE.**

### Ambassador 0.50.0 is not yet feature-complete. Limitations:

- `RateLimitService` and `TracingService` resources are not currently supported.
- WebSockets are not currently supported.
- CORS is not currently supported.
- GRPC is not currently supported.
- TLS termination is not
- `statsd` integration has not been tested.
- The logs are very cluttered.
- Configuration directly from the filesystem isn’t supported.
- The diagnostics service cannot correctly drill down by source file, though it can drill down by route or other resources.
- Helm installation has not been tested.
- `AuthService` does not currently have full support for configuring headers to be sent to the extauth service. At present it sends all the headers listed in `allowed_headers` plus:
   - `Authorization`
   - `Cookie`
   - `Forwarded`
   - `From`
   - `Host`
   - `Proxy-Authenticate`
   - `Proxy-Authorization`
   - `Set-Cookie`
   - `User-Agent`
   - `X-Forwarded-For`
   - `X-Forwarded-Host`
   - `X-Forwarded`
   - `X-Gateway-Proto`
   - `WWW-Authenticate`

### **BREAKING CHANGES** from 0.40.0

- Configuration from a `ConfigMap` is no longer supported.
- The authentication `Module` is no longer supported; use `AuthService` instead (which you probably already were).
- External authentication now uses the core Envoy `envoy.ext_authz` filter, rather than the custom Datawire auth filter.
   - `ext_authz` speaks the same protocol, and your existing external auth services should work, however:
   - `ext_authz` does _not_ send all the request headers to the external auth service (see above in `Limitations`).
- Circuit breakers and outlier detection are not supported. They will be reintroduced in a later Ambassador release.
- Ambassador now _requires_ a TLS `Module` to enable TLS termination, where previous versions would automatically enable termation if the `ambassador-certs` secret was present. A minimal `Module` for the same behavior is:

        ---
        kind: Module
        name: tls
        config:
          server:
            secret: ambassador-certs

## [0.40.2] November 26, 2018
[0.40.2]: https://github.com/datawire/ambassador/compare/0.40.1...0.40.2

### Minor changes:
- Feature: Support using environment variables to modify the configuration directory (thanks @n1koo!)
- Feature: In Helmfile, support `volumeMounts` (thanks @kyschouv!)
- Bugfix: In Helmfile, correctly quote `.Values.namespace.single` (thanks @bobby!)
- Bugfix: In Helmfile, correctly support `Nodeport` in HTTP and HTTPS (thanks @n1koo!)

## [0.40.1] October 29, 2018
[0.40.1]: https://github.com/datawire/ambassador/compare/0.40.0...0.40.1

### Minor changes:
- Feature: Support running Ambassador as a `Daemonset` via Helm (thanks @DipeshMitthalal!)
- Feature: Switch to Envoy commit 5f795fe2 to fix a crash if attempting to add headers after using an AuthService (#647, #680)

## [0.40.0] September 25, 2018
[0.40.0]: https://github.com/datawire/ambassador/compare/0.39.0...0.40.0

### Minor changes:

- Feature: Allow users to override the `STATSD_HOST` value (#810). Thanks to @rsyvarth.
- Feature: Support LightStep distributed tracing (#796). Thanks to @alexgervais.
- Feature: Add service label in Helm chart (#778). Thanks to @sarce.
- Feature: Add support for load balancer IP in Helm chart (#765). Thanks to @larsha.
- Feature: Support prometheus mapping configurations (#746). Thanks to @bcatcho.
- Feature: Add support for `loadBalancerSourceRanges` to Helm chart (#764). Thanks to @mtbdeano.
- Feature: Support for namespaces and Ambassador ID in Helm chart (#588, #643). Thanks to @MichielDeMey and @jstol.
- Bugfix: Add AMBASSADOR_VERIFY_SSL_FALSE flag (#782, #807). Thanks to @sonrier.
- Bugfix: Fix Ambassador single namespace in Helm chart (#827). Thanks to @sarce.
- Bugfix: Fix Helm templates and default values (#826).
- Bugfix: Add `stats-sink` back to Helm chart (#763).
- Bugfix: Allow setting `timeout_ms` to 0 for gRPC streaming services (#545). Thanks to @lovers36.
- Bugfix: Update Flask to 0.12.3.

## [0.39.0] August 30, 2018
[0.39.0]: https://github.com/datawire/ambassador/compare/0.38.0...0.39.0

### Major Changes:

- Bugfix: The statsd container has been removed by default in order to avoid DoSing Kubernetes DNS. The functionality can be re-enabled by setting the `STATSD_ENABLED` environment variable to `true` in the Ambassador deployment YAML (#568).
- Docs: Added detailed Ambassador + Istio Integration Documentation on monitoring and distributed tracing. - @feitnomore

### Minor Changes:

- Docs: Added instructions for running Ambassador with Docker Compose. - @bcatcho
- Bugfix: Fix Ambassador to more aggressively reconnect to Kubernetes (#554). - @nmatsui
- Feature: Diagnostic view displays AuthService, RateLimitService, and TracingService (#730). - @alexgervais
- Feature: Enable Ambassador to tag tracing spans with request headers via `tag_headers`. - @alexgervais

## [0.38.0] August 08, 2018
[0.38.0]: https://github.com/datawire/ambassador/compare/0.37.0...0.38.0

### Major changes:
- Feature: Default CORS configuration can now be set - @KowalczykBartek
- Bugfix: Ambassador does not crash with empty YAML config anymore - @rohan47

### Minor changes:
- DevEx: `master` is now latest, `stable` tracks the latest released version
- DevEx: release-prep target added to Makefile to facilitate releasing process
- DevEx: all tests now run in parallel, consuming lesser time
- Bugfix: Ambassador SIGCHLD messages are less scary looking now

## [0.37.0] July 31, 2018:
[0.37.0]: https://github.com/datawire/ambassador/compare/0.36.0...0.37.0

### Major changes:
- Feature: Added support for request tracing (by Alex Gervais)

## [0.36.0] July 26, 2018:
[0.36.0]: https://github.com/datawire/ambassador/compare/0.35.3...0.36.0

### Major changes:
- Fix: HEAD requests no longer cause segfaults
- Feature: TLS can now be configured with arbitrary secret names, instead of predefined secrets
- Change: The Envoy dynamic header value `%CLIENT_IP%` is no longer supported. Use `%DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT%` instead. (This is due to a change in Envoy 1.7.0.)

## [0.35.3] July 18, 2018: **READ THE WARNING ABOVE**
[0.35.3]: https://github.com/datawire/ambassador/compare/0.35.2...0.35.3

### Changed

Major changes:
- Ambassador is now based on Envoy v1.7.0
- Support for X-FORWARDED-PROTO based redirection, generally used with Layer 7 load balancers
- Support for port based redirection using `redirect_cleartext_from`, generally used with Layer 4 load balancers
- Specifying HTTP and HTTPS target ports in Helm chart

Other changes:
- End-to-end tests can now be run with `make e2e` command
- Helm release automation has been fixed
- Mutliple end-to-end tests are now executed in parallel, taking lesser time
- Huge revamp to documentation around unit tests
- Documentation changes

## [0.35.2] July 05, 2018: **READ THE WARNING ABOVE**
[0.35.2]: https://github.com/datawire/ambassador/compare/0.35.1...0.35.2

### Changed

- 0.35.2 is almost entirely about updates to Datawire testing infrastructure.
- The only user-visible change is that Ambassador will do a better job of showing which Kubernetes objects define Ambassador configuration objects when using `AMBASSADOR_ID` to run multiple Ambassadors in the same cluster.

## [0.35.1] June 25, 2018: **READ THE WARNING ABOVE**
[0.35.1]: https://github.com/datawire/ambassador/compare/0.35.0...0.35.1

### Changed

- Properly support supplying additional TLS configuration (such as `redirect_cleartext_from`) when using certificates from a Kubernetes `Secret`
- Update Helm chart to allow customizing annotations on the deployed `ambassador` Kubernetes `Service` (thanks @psychopenguin!)

## [0.35.0] June 25, 2018: **READ THE WARNING ABOVE**
[0.35.0]: https://github.com/datawire/ambassador/compare/0.34.3...0.35.0

### Changed

- 0.35.0 re-supports websockets, but see the **BREAKING NEWS** for an important caveat.
- 0.35.0 supports running as non-root. See the **BREAKING NEWS** above for more information.
- Make sure regex matches properly handle backslashes, and properly display in the diagnostics service (thanks @alexgervais!).
- Prevent kubewatch from falling into an endless spinloop (thanks @mechpen!).
- Support YAML array syntax for CORS array elements.

## [0.34.3] June 13, 2018: **READ THE WARNING ABOVE**
[0.34.3]: https://github.com/datawire/ambassador/compare/0.34.2...0.34.3

### Changed

- **0.34.3 cannot support websockets**: see the **WARNING** above.
- Fix a possible crash if no annotations are found at all (#519).
- Improve logging around service watching and such.

## [0.34.2] June 11, 2018: **READ THE WARNING ABOVE**
[0.34.2]: https://github.com/datawire/ambassador/compare/0.34.1...0.34.2

### Changed

- **0.34.2 cannot support websockets**: see the **WARNING** above.
- Ambassador is now based on Envoy 1.6.0!
- Ambassador external auth services can now modify existing headers in place, as well as adding new headers.
- Re-support the `ambassador-cacert` secret for configuring TLS client-certificate authentication. **Note well** that a couple of things have changed in setting this up: you'll use the key `tls.crt`, not `fullchain.pem`. See https://www.getambassador.io/reference/auth-tls-certs for more.

## [0.34.1] June 04, 2018
[0.34.1]: https://github.com/datawire/ambassador/compare/0.34.0...0.34.1

### Bugfixes

- Unbuffer log output for better diagnostics.
- Switch to gunicorn instead of Werkzeug for the diag service.
- Use the YAML we release as the basis for end-to-end testing.

## [0.34.0] May 16, 2018
[0.34.0]: https://github.com/datawire/ambassador/compare/0.33.1...0.34.0

### Changed

- When originating TLS, use the `host_rewrite` value to set outgoing SNI. If no `host_rewrite` is set, do not use SNI.
- Allow disabling external access to the diagnostics service (with thanks to @alexgervais and @dougwilson).

## [0.33.1] May 16, 2018
[0.33.1]: https://github.com/datawire/ambassador/compare/0.33.0...0.33.1

### Changed

- Fix YAML error on statsd pod.

## [0.33.0] May 14, 2018
[0.33.0]: https://github.com/datawire/ambassador/compare/v0.32.2...0.33.0

### Changed

- Fix support for `host_redirect` in a `Mapping`. **See the `Mapping` documentation** for more details: the definition of the `host_redirect` attribute has changed.

## [0.32.2] May 02, 2018
[0.32.2]: https://github.com/datawire/ambassador/compare/v0.32.0...v0.32.2

(Note that 0.32.1 was an internal release.)

### Changed

- Fix a bad bootstrap CSS inclusion that would cause the diagnostic service to render incorrectly.

## [0.32.0] April 27, 2018
[0.32.0]: https://github.com/datawire/ambassador/compare/v0.31.0...v0.32.0

### Changed

- Traffic shadowing is supported using the `shadow` attribute in a `Mapping`
- Multiple Ambassadors can now run more happily in a single cluster
- The diagnostic service will now show you what `AuthService` configuration is active
- The `tls` keyword now works for `AuthService` just like it does for `Mapping` (thanks @dvavili!)

## [0.31.0] April 12, 2018
[0.31.0]: https://github.com/datawire/ambassador/compare/v0.30.2...v0.31.0

### Changed

- Rate limiting is now supported (thanks, @alexgervais!) See the docs for more detail here.
- The `statsd` container has been quieted down yet more (thanks again, @alexgervais!).

## [0.30.2] March 26, 2018
[0.30.2]: https://github.com/datawire/ambassador/compare/v0.30.1...v0.30.2

### Changed

- drop the JavaScript `statsd` for a simple `socat`-based forwarder
- ship an Ambassador Helm chart (thanks @stefanprodan!)
   - Interested in testing Helm? See below!
- disable Istio automatic sidecar injection (thanks @majelbstoat!)
- clean up some doc issues (thanks @lavoiedn and @endrec!)

To test Helm, make sure you have `helm` installed and that you have `tiller` properly set up for your RBAC configuration. Then:

```
helm repo add datawire https://www.getambassador.io

helm upgrade --install --wait my-release datawire/ambassador
```

You can also use `adminService.type=LoadBalancer`.

## [0.30.1] March 26, 2018
[0.30.1]: https://github.com/datawire/ambassador/compare/v0.30.0...v0.30.1

### Fixed

- The `tls` module is now able to override TLS settings probed from the `ambassador-certs` secret

## [0.30.0] March 23, 2018
[0.30.0]: https://github.com/datawire/ambassador/compare/v0.29.0...v0.30.0

### Changed

- Support regex matching for `prefix` (thanks @radu-c!)
- Fix docs around `AuthService` usage

## [0.29.0] March 15, 2018
[0.29.0]: https://github.com/datawire/ambassador/compare/v0.28.2...v0.29.0

### Changed

- Default restart timings have been increased. **This will cause Ambassador to respond to service changes less quickly**; by default, you'll see changes appear within 15 seconds.
- Liveness and readiness checks are now enabled after 30 seconds, rather than 3 seconds, if you use our published YAML.
- The `statsd` container is now based on `mhart/alpine-node:9` rather than `:7`.
- `envoy_override` has been reenabled in `Mapping`s.

## [0.28.1] March 05, 2018 (and [0.28.0] on March 02, 2018)
[0.28.1]: https://github.com/datawire/ambassador/compare/v0.26.0...v0.28.1
[0.28.0]: https://github.com/datawire/ambassador/compare/v0.26.0...v0.28.1

(Note that 0.28.1 is identical to 0.28.0, and 0.27.0 was an internal release. These are related to the way CI generates tags, which we'll be revamping soon.)

### Changed

- Support tuning Envoy restart parameters
- Support `host_regex`, `method_regex`, and `regex_headers` to allow regular expression matches in `Mappings`
- Support `use_proxy_proto` and `use_remote_address` in the `ambassador` module
- Fine-tune the way we sort a `Mapping` based on its constraints
- Support manually setting the `precedence` of a `Mapping`, so that there's an escape hatch when the automagic sorting gets it wrong
- Expose `alpn_protocols` in the `tls` module (thanks @technicianted!)
- Make logs a lot quieter
- Reorganize and update documentation
- Make sure that `ambassador dump --k8s` will work correctly
- Remove a dependency on a `ConfigMap` for upgrade checks

## [0.26.0] February 13, 2018
[0.26.0]: https://github.com/datawire/ambassador/compare/v0.25.0...v0.26.0

### Changed

- The `authentication` module is deprecated in favor of the `AuthService` resource type.
- Support redirecting cleartext connections on port 80 to HTTPS on port 443
- Streamline end-to-end tests and, hopefully, allow them to work well without Kubernaut
- Clean up some documentation (thanks @lavoiedn!)

## [0.25.0] February 06, 2018
[0.25.0]: https://github.com/datawire/ambassador/compare/v0.23.0...v0.25.0

(Note that 0.24.0 was an internal release.)

### Changed

- CORS support (thanks @alexgervais!)
- Updated docs for
  - GKE
  - Ambassador + Istio
  - Ordering of `Mappings`
  - Prometheus with Ambassador
- Support multiple external authentication service instances, so that canarying `extauth` services is possible
- Correctly support `timeout_ms` in a `Mapping`
- Various build tweaks and end-to-end test speedups

## [0.23.0] January 17, 2018
[0.23.0]: https://github.com/datawire/ambassador/compare/v0.22.0...v0.23.0

### Changed

- Clean up build docs (thanks @alexgervais!)
- Support `add_request_headers` for, uh, adding requests headers (thanks @alexgervais!)
- Make end-to-end tests and Travis build process a bit more robust
- Pin to Kubernaut 0.1.39
- Document the use of the `develop` branch
- Don't default to `imagePullAlways`
- Switch to Alpine base with a stripped Envoy image

## [0.22.0] January 17, 2018
[0.22.0]: https://github.com/datawire/ambassador/compare/v0.21.1...v0.22.0

### Changed

- Switched to using `quay.io` rather than DockerHub. **If you are not using Datawire's published Kubernetes manifests, you will have to update your manifests!**
- Switched to building over Alpine rather than Ubuntu. (We're still using an unstripped Envoy; that'll change soon.)
- Switched to a proper production configuration for the `statsd` pod, so that it hopefully chews up less memory.
- Make sure that Ambassador won't generate cluster names that are too long for Envoy.
- Fix a bug where Ambassador could crash if there were too many egregious errors in its configuration.

## [0.21.1] January 11, 2018
[0.21.1]: https://github.com/datawire/ambassador/compare/v0.21.0...v0.21.1

### Changed

- Ambassador will no longer generate cluster names that exceed Envoy's 60-character limit.

## [0.21.0] January 03, 2018
[0.21.0]: https://github.com/datawire/ambassador/compare/v0.20.1...v0.21.0

### Changed

- If `AMBASSADOR_SINGLE_NAMESPACE` is present in the environment, Ambassador will only look for services in its own namespace.
- Ambassador `Mapping` objects now correctly support `host_redirect`, `path_redirect`, `host_rewrite`, `auto_host_rewrite`, `case_sensitive`, `use_websocket`, `timeout_ms`, and `priority`.

## [0.20.1] December 22, 2017
[0.20.1]: https://github.com/datawire/ambassador/compare/v0.20.0...v0.20.1

### Changed

- If Ambassador finds an empty YAML document, it will now ignore it rather than raising an exception.
- Includes the namespace of a service from an annotation in the name of its generated YAML file.
- Always process inputs in the same order from run to run.

## [0.20.0] December 18, 2017
[0.20.0]: https://github.com/datawire/ambassador/compare/v0.19.2...v0.20.0

### Changed

- Switch to Envoy 1.5 under the hood.
- Refocus the diagnostic service to better reflect what's actually visible when you're working at Ambassador's level.
- Allow the diagnostic service to display, and change, the Envoy log level.

## [0.19.2] December 12, 2017
[0.19.2]: https://github.com/datawire/ambassador/compare/v0.19.1...v0.19.2

### Changed

- Arrange for logs from the subsystem that watches for Kubernetes service changes (kubewatch) to have timestamps and such.
- Only do new-version checks every four hours.

## [0.19.1] December 04, 2017
[0.19.1]: https://github.com/datawire/ambassador/compare/v0.19.0...v0.19.1

### Changed

- Allow the diag service to look good (well, OK, not too horrible anyway) when Ambassador is running with TLS termination.
- Show clusters on the overview page again.
- The diag service now shows you the "health" of a cluster by computing it from the number of requests to a given service that didn't involve a 5xx status code, rather than just forwarding Envoy's stat, since we don't configure Envoy's stat in a meaningful way yet.
- Make sure that the tests correctly reported failures (sigh).
- Allow updating out-of-date diagnostic reports without requiring multiple test runs.

## [0.19.0] November 30, 2017
[0.19.0]: https://github.com/datawire/ambassador/compare/v0.18.2...v0.19.0

### Changed

- Ambassador can now use HTTPS upstream services: just use a `service` that starts with `https://` to enable it.
  - By default, Ambassador will not offer a certificate when using HTTPS to connect to a service, but it is possible to configure certificates. Please [contact us on Slack](https://d6e.co/slack) if you need to do this.
- HTTP access logs appear in the normal Kubernetes logs for Ambassador.
- It’s now possible to tell `ambassador config` to read Kubernetes manifests from the filesystem and build a configuration from the annotations in them (use the `--k8s` switch).
- Documentation on using Ambassador with Istio now reflects Ambassador 0.19.0 and Istio 0.2.12.

## [0.18.2] November 28, 2017
[0.18.2]: https://github.com/datawire/ambassador/compare/v0.18.0...v0.18.2

### Changed

- The diagnostics service will now tell you when updates are available.

## [0.18.0] November 20, 2017
[0.18.0]: https://github.com/datawire/ambassador/compare/v0.17.0...v0.18.0

### Changed

- The Host header is no longer overwritten when Ambassador talks to an external auth service. It will now retain whatever value the client passes there.

### Fixed

- Checks for updates weren’t working, and they have been restored. At present you’ll only see them in the Kubernetes logs if you’re using annotations to configure Ambassador — they’ll start showing up in the diagnostics service in the next release or so.

## [0.17.0] November 14, 2017
[0.17.0]: https://github.com/datawire/ambassador/compare/v0.16.0...v0.17.0

### Changed

- Allow Mappings to require matches on HTTP headers and `Host`
- Update tests, docs, and diagnostic service for header matching

### Fixed

- Published YAML resource files will no longer overwrite annotations on the Ambassador `service` when creating the Ambassador `deployment`

## [0.16.0] November 10, 2017
[0.16.0]: https://github.com/datawire/ambassador/compare/v0.15.0...v0.16.0

### Changed

- Support configuring Ambassador via `annotations` on Kubernetes `service`s
- No need for volume mounts! Ambassador can read configuration and TLS-certificate information directly from Kubernetes to simplify your Kubernetes YAML
- Expose more configuration elements for Envoy `route`s: `host_redirect`, `path_redirect`, `host_rewrite`, `auto_host_rewrite`, `case_sensitive`, `use_websocket`, `timeout_ms`, and `priority` get transparently copied

### Fixed

- Reenable support for gRPC

## [0.15.0] October 16, 2017
[0.15.0]: https://github.com/datawire/ambassador/compare/v0.14.2...v0.15.0

### Changed

- Allow `docker run` to start Ambassador with a simple default configuration for testing
- Support `host_rewrite` in mappings to force the HTTP `Host` header value for services that need it
- Support `envoy_override` in mappings for odd situations
- Allow asking the diagnostic service for JSON output rather than HTML

## [0.14.2] October 12, 2017
[0.14.2]: https://github.com/datawire/ambassador/compare/v0.14.0...v0.14.2

### Changed

- Allow the diagnostic service to show configuration errors.

## [0.14.0] October 05, 2017
[0.14.0]: https://github.com/datawire/ambassador/compare/v0.13.0...v0.14.0

### Changed

- Have a diagnostic service!
- Support `cert_required` in TLS config

## [0.13.0] September 25, 2017
[0.13.0]: https://github.com/datawire/ambassador/compare/v0.12.1...v0.13.0

### Changed

- Support using IP addresses for services.
- Check for collisions, so that trying to e.g. map the same prefix twice will report an error.
- Enable liveness and readiness probes, and have Kubernetes perform them by default.
- Document the presence of the template-override escape hatch.

## [0.12.1] September 22, 2017
[0.12.1]: https://github.com/datawire/ambassador/compare/v0.12.0...v0.12.1

### Changed

- Notify (in the logs) if a new version of Ambassador is available.

## [0.12.0] September 21, 2017
[0.12.0]: https://github.com/datawire/ambassador/compare/v0.11.2...v0.12.0

### Changed

- Support for non-default Kubernetes namespaces.
- Infrastructure for checking if a new version of Ambassador is available.

## [0.11.2] September 20, 2017
[0.11.2]: https://github.com/datawire/ambassador/compare/v0.11.1...v0.11.2

### Changed

- Better schema verification.

## [0.11.1] September 18, 2017
[0.11.1]: https://github.com/datawire/ambassador/compare/v0.11.0...v0.11.1

### Changed

- Do schema verification of input YAML files.

## [0.11.0] September 18, 2017
[0.11.0]: https://github.com/datawire/ambassador/compare/v0.10.14...v0.11.0

### Changed

- Declarative Ambassador! Configuration is now via YAML files rather than REST calls
- The `ambassador-store` service is no longer needed.

## [0.10.14] September 15, 2017
[0.10.14]: https://github.com/datawire/ambassador/compare/v0.10.13...v0.10.14

### Fixed

- Update `demo-qotm.yaml` with the correct image tag.

## [0.10.13] September 05, 2017
[0.10.13]: https://github.com/datawire/ambassador/compare/v0.10.12...v0.10.13

### Changed

- Properly support proxying all methods to an external authentication service, with headers intact, rather than moving request headers into the body of an HTTP POST.

## [0.10.12] August 02, 2017
[0.10.12]: https://github.com/datawire/ambassador/compare/v0.10.10...v0.10.12

### Changed

- Make TLS work with standard K8s TLS secrets, and completely ditch push-cert and push-cacert.

### Fixed

- Move Ambassador out from behind Envoy, so that you can use Ambassador to fix things if you completely botch your Envoy config.
- Let Ambassador keep running if Envoy totally chokes and dies, but make sure the pod dies if Ambassador loses access to its storage.

## [0.10.10] August 01, 2017
[0.10.10]: https://github.com/datawire/ambassador/compare/v0.10.7...v0.10.10

### Fixed

- Fix broken doc paths and simplify building as a developer. 0.10.8, 0.10.9, and 0.10.10 were all stops along the way to getting this done; hopefully we'll be able to reduce version churn from here on out.

## [0.10.7] July 25, 2017
[0.10.7]: https://github.com/datawire/ambassador/compare/v0.10.6...v0.10.7

### Changed
- More CI-build tweaks.

## [0.10.6] July 25, 2017
[0.10.6]: https://github.com/datawire/ambassador/compare/v0.10.5...v0.10.6

### Changed
- Fix automagic master build tagging

## [0.10.5] July 25, 2017
[0.10.5]: https://github.com/datawire/ambassador/compare/v0.10.1...v0.10.5

### Changed
- Many changes to the build process and versioning. In particular, CI no longer has to commit files.

## [0.10.1] July 03, 2017
[0.10.1]: https://github.com/datawire/ambassador/compare/v0.10.0...v0.10.1

### Added
- Changelog


## [0.10.0] June 30, 2017
[0.10.0]: https://github.com/datawire/ambassador/compare/v0.9.1...v0.10.0
[grpc-0.10.0]: https://github.com/datawire/ambassador/blob/v0.10.0/docs/user-guide/grpc.md

### Added
- Ambassador supports [GRPC services][grpc-0.10.0] (and other HTTP/2-only services) using the GRPC module

### Fixed
- Minor typo in Ambassador's `Dockerfile` that break some versions of Docker


## [0.9.1] June 28, 2017
[0.9.1]: https://github.com/datawire/ambassador/compare/v0.9.0...v0.9.1
[building-0.9.1]: https://github.com/datawire/ambassador/blob/v0.9.1/BUILDING.md

### Changed
- Made development a little easier by automating dev version numbers so that modified Docker images update in Kubernetes
- Updated [`BUILDING.md`][building-0.9.1]


## [0.9.0] June 23, 2017
[0.9.0]: https://github.com/datawire/ambassador/compare/v0.8.12...v0.9.0
[start-0.9.0]: https://github.com/datawire/ambassador/blob/v0.9.0/docs/user-guide/getting-started.md
[concepts-0.9.0]: https://github.com/datawire/ambassador/blob/v0.9.0/docs/user-guide/mappings.md

### Added
- Ambassador supports HTTP Basic Auth
- Ambassador now has the concept of _modules_ to enable and configure optional features such as auth
- Ambassador now has the concept of _consumers_ to represent end-users of mapped services
- Ambassador supports auth via an external auth server

Basic auth is covered in [Getting Started][start-0.9.0]. Learn about modules and consumers and see an example of external auth in [About Mappings, Modules, and Consumers][concepts-0.9.0].

### Changed
- State management (via Ambassador store) has been refactored
- Switched to [Ambassador-Envoy] for the base Docker image


## [0.8.12] June 07, 2017
[0.8.12]: https://github.com/datawire/ambassador/compare/v0.8.11...v0.8.12

### Added
- Mappings can now be updated


## [0.8.11] May 24, 2017
[0.8.11]: https://github.com/datawire/ambassador/compare/v0.8.10...v0.8.11
[istio-0.8.11]: https://github.com/datawire/ambassador/blob/v0.8.11/docs/user-guide/with-istio.md
[stats-0.8.11]: https://github.com/datawire/ambassador/blob/v0.8.11/docs/user-guide/statistics.md

### Added
- Ambassador interoperates with [Istio] -- see [Ambassador and Istio][istio-0.8.11]
- There is additional documentation for [statistics and monitoring][stats-0.8.11]

### Fixed
- Bug in mapping change detection
- Release machinery issues


## [0.8.6] May 05, 2017
[0.8.6]: https://github.com/datawire/ambassador/compare/v0.8.5...v0.8.6

### Added
- Ambassador releases are now performed by Travis CI


## [0.8.2] May 04, 2017
[0.8.2]: https://github.com/datawire/ambassador/compare/v0.8.1...v0.8.2

### Changed
- Documentation updates


## [0.8.0] May 02, 2017
[0.8.0]: https://github.com/datawire/ambassador/compare/v0.7.0...v0.8.0
[client-tls-0.8.0]: https://github.com/datawire/ambassador/blob/v0.8.0/README.md#using-tls-for-client-auth

### Added
- [Ambassador has a website!][Ambassador]
- Ambassador supports auth via [TLS client certificates][client-tls-0.8.0]
- There are some additional helper scripts in the `scripts` directory

### Changed
- Ambassador's admin interface is now on local port 8888 while mappings are available on port 80/443 depending on whether TLS is enabled
- Multiple instances of Ambassador talking to the same Ambassador Store pod will pick up each other's changes automatically


## [0.7.0] May 01, 2017
[0.7.0]: https://github.com/datawire/ambassador/compare/v0.6.0...v0.7.0
[start-0.7.0]: https://github.com/datawire/ambassador/blob/v0.7.0/README.md#mappings

### Added
- Ambassador can rewrite the request URL path prefix before forwarding the request to your service (covered in [Getting Started][start-0.7.0])
- Ambassador supports additional stats aggregators: Datadog, Grafana

### Changed
- _Services_ are now known as _mappings_
- Minikube is supported again


## [0.6.0] April 28, 2017
[0.6.0]: https://github.com/datawire/ambassador/compare/v0.5.2...v0.6.0

### Removed
- The Ambassador SDS has been removed; Ambassador routes to service names


## [0.5.2] April 26, 2017
[0.5.2]: https://github.com/datawire/ambassador/compare/v0.5.0...v0.5.2

### Added
- Ambassador includes a local `statsd` so that full stats from Envoy can be collected and pushed to a stats aggregator (Prometheus is supported)

### Changed
- It's easier to develop Ambassador thanks to improved build documentation and `Makefile` fixes


## [0.5.0] April 13, 2017
[0.5.0]: https://github.com/datawire/ambassador/compare/v0.4.0...v0.5.0

### Added
- Ambassador supports inbound TLS
- YAML for a demo user service is now included

### Changed
- The `geturl` script supports Minikube and handles AWS better
- Documentation and code cleanup


## [0.4.0] April 07, 2017
[0.4.0]: https://github.com/datawire/ambassador/compare/v0.3.3...v0.4.0

### Changed
- Ambassador now reconfigures Envoy automatically once changes have settled for five seconds
- Envoy stats and Ambassador stats are separate
- Mappings no longer require specifying the port as it is not needed

### Fixed
- SDS does the right thing with unnamed ports


## [0.3.1] April 06, 2017
[0.3.1]: https://github.com/datawire/ambassador/compare/v0.3.0...v0.3.1

### Added
- Envoy stats accessible through Ambassador
- Basic interpretation of cluster stats

### Changed
- Split up `ambassador.py` into multiple files
- Switch to a debug build of Envoy


## [0.1.9] April 03, 2017
[0.1.9]: https://github.com/datawire/ambassador/compare/v0.1.8...v0.1.9

### Changed
- Ambassador configuration on `/ambassador-config/` prefix rather than exposed on port 8001
- Updated to current Envoy and pinned the Envoy version
- Use Bumpversion for version management
- Conditionalized Docker push

### Fixed
- Ambassador keeps running with an empty services list (part 2)


## [0.1.5] March 31, 2017
[0.1.5]: https://github.com/datawire/ambassador/compare/v0.1.4...v0.1.5

### Fixed
- Ambassador SDS correctly handles ports


## [0.1.4] March 31, 2017
[0.1.4]: https://github.com/datawire/ambassador/compare/v0.1.3...v0.1.4

### Changed
- Ambassador keeps running with an empty services list
- Easier to run with [Telepresence]


## [0.1.3] March 31, 2017
[0.1.3]: https://github.com/datawire/ambassador/compare/82ed5e4...v0.1.3

### Added
- Initial Ambassador
- Ambassador service discovery service
- Documentation


Based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/). Ambassador follows [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

[Ambassador]: https://www.getambassador.io/
[Ambassador-Envoy]: https://github.com/datawire/ambassador-envoy
[Telepresence]: http://telepresence.io
[Istio]: https://istio.io/
