# Changelog

## BREAKING NEWS

- As of **0.28.0**, Ambassador supports Envoy's `use_remote_address` capability, as described in [the Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http_conn_man/headers.html). Ambassador's default is currently not to include `use_remote_address`, but **this will soon change** to a default value of `true`.

- As of **0.26.0**, the `authentication` module is deprecated in favor of the `AuthService` resource type, as discussed in [the Ambassador reference](https://getambassador.io/docs/reference/module).

- As of **0.22.0**, Ambassador is distributed via `quay.io` rather than DockerHub. If you are not using Datawire's published Kubernetes manifests, you will have to update your manifests!

- The `statsd` container is likely to be dropped from our default published YAML soon. If you rely on the `statsd` container, consider switching now to local YAML.

## [0.30.2] March 26, 2018
[0.30.2]: https://github.com/datawire/ambassador/compare/v0.30.2...v0.30.1

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
[0.30.1]: https://github.com/datawire/ambassador/compare/v0.30.1...v0.30.0

### Fixed

- The `tls` module is now able to override TLS settings probed from the `ambassador-certs` secret

## [0.30.0] March 23, 2018
[0.30.0]: https://github.com/datawire/ambassador/compare/v0.30.0...v0.29.0

### Changed

- Support regex matching for `prefix` (thanks @radu-c!)
- Fix docs around `AuthService` usage

## [0.29.0] March 15, 2018
[0.29.0]: https://github.com/datawire/ambassador/compare/v0.29.0...v0.28.2

### Changed

- Default restart timings have been increased. **This will cause Ambassador to respond to service changes less quickly**; by default, you'll see changes appear within 15 seconds.
- Liveness and readiness checks are now enabled after 30 seconds, rather than 3 seconds, if you use our published YAML.
- The `statsd` container is now based on `mhart/alpine-node:9` rather than `:7`.
- `envoy_override` has been reenabled in `Mapping`s.

## [0.28.1] March 5, 2018 (and [0.28.0] on March 2, 2018)
[0.28.1]: https://github.com/datawire/ambassador/compare/v0.28.1...v0.26.0
[0.28.0]: https://github.com/datawire/ambassador/compare/v0.28.1...v0.26.0

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
[0.26.0]: https://github.com/datawire/ambassador/compare/v0.26.0...v0.25.0

### Changed

- Support redirecting cleartext connections on port 80 to HTTPS on port 443
- Streamline end-to-end tests and, hopefully, allow them to work well without Kubernaut
- Clean up some documentation (thanks @lavoiedn!)

## [0.25.0] February 6, 2018
[0.25.0]: https://github.com/datawire/ambassador/compare/v0.25.0...v0.23.0

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

## [0.23.0] January 17, 2017
[0.23.0]: https://github.com/datawire/ambassador/compare/v0.23.0...v0.22.0

### Changed

- Clean up build docs (thanks @alexgervais!)
- Support `add_request_headers` for, uh, adding requests headers (thanks @alexgervais!)
- Make end-to-end tests and Travis build process a bit more robust
- Pin to Kubernaut 0.1.39
- Document the use of the `develop` branch
- Don't default to `imagePullAlways`
- Switch to Alpine base with a stripped Envoy image

## [0.22.0] January 17, 2017
[0.22.0]: https://github.com/datawire/ambassador/compare/v0.21.1...v0.22.0

### Changed

- Switched to using `quay.io` rather than DockerHub. **If you are not using Datawire's published Kubernetes manifests, you will have to update your manifests!**
- Switched to building over Alpine rather than Ubuntu. (We're still using an unstripped Envoy; that'll change soon.)
- Switched to a proper production configuration for the `statsd` pod, so that it hopefully chews up less memory.
- Make sure that Ambassador won't generate cluster names that are too long for Envoy.
- Fix a bug where Ambassador could crash if there were too many egregious errors in its configuration.

## [0.21.1] January 11, 2017
[0.21.1]: https://github.com/datawire/ambassador/compare/v0.21.0...v0.21.1

### Changed

- Ambassador will no longer generate cluster names that exceed Envoy's 60-character limit.

## [0.21.0] January 3, 2017
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

## [0.19.1] December 4, 2017
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
  - By default, Ambassador will not offer a certificate when using HTTPS to connect to a service, but it is possible to configure certificates. Please [contact us on Gitter](https://gitter.im/datawire/ambassador) if you need to do this.
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

## [0.14.0] October 5, 2017
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

## [0.10.13] September 5, 2017
[0.10.13]: https://github.com/datawire/ambassador/compare/v0.10.12...v0.10.13

### Changed

- Properly support proxying all methods to an external authentication service, with headers intact, rather than moving request headers into the body of an HTTP POST.

## [0.10.12] August 2, 2017
[0.10.12]: https://github.com/datawire/ambassador/compare/v0.10.10...v0.10.12

### Changed

- Make TLS work with standard K8s TLS secrets, and completely ditch push-cert and push-cacert.

### Fixed

- Move Ambassador out from behind Envoy, so that you can use Ambassador to fix things if you completely botch your Envoy config.
- Let Ambassador keep running if Envoy totally chokes and dies, but make sure the pod dies if Ambassador loses access to its storage.

## [0.10.10] August 1, 2017
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

## [0.10.1] July 3, 2017
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
