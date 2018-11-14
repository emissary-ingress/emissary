# Changelog

## BREAKING NEWS

- In Ambassador 0.36.0, the Envoy dynamic value `%CLIENT_IP%` is no longer supported. Use `%DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT%` instead. (This is due to a change in Envoy 1.7.0.)

- **Ambassador 0.35.0 resupports websockets, with the important caveat that a websocket cannot have multiple upstream services**.
  - This means that you cannot do canary deployments for websockets.
    We're actively working on fixing this.
  - Multiple websocket `Mapping`s are still supported.

- Ambassador version 0.35.0 supports running as a non-root user, to improve security and work on other Kubernetes runtimes (e.g. OpenShift). **Running as non-root will become the default in a future Ambassador release; this will be a breaking change.** We recommend proactively switching to non-root now:
    - Use a `securityContext` in your Ambassador `Deployment` to switch to a non-root user.
    - Set the `service_port` element in the `ambassador` `Module` to a port number greater than 1024. (Ambassador's defaults will change to 8080 for cleartext and 8443 for TLS.)
    - Make sure that incoming traffic to Ambassador routes to the `service_port`. The most likely required change is the `targetPort` in the Kubernetes `Service` resource for Ambassador.
    - If you are using `redirect_cleartext_from`, change the value of this field to match the value you set in `service_port`.
    - If you have modified Ambassador's behavior around TLS certificates using a custom Ambassador build, please contact Datawire for more information.

- Ambassador versions **0.34.2** and **0.34.3** cannot support websockets; see the **WARNING** above. This bug is fixed in Ambassador 0.35.0.

- As of **0.28.0**, Ambassador supports Envoy's `use_remote_address` capability, as described in [the Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http_conn_man/headers.html). Ambassador's default is currently not to include `use_remote_address`, but **this will soon change** to a default value of `true`.

<!---
Add release notes right after this point.

(version number: MAJOR.MINOR.PATCH)

Format:

## [version] <month> <date>, <year>:
[version]: https://github.com/datawire/ambassador/compare/<last released version>...<version>

### Major changes:
- Feature: <insert feature description here>
- Bugfix: <insert bugfix description here>

### Minor changes:
- Feature: <insert feature description here>
- Bugfix: <insert bugfix description here>
--->

<!--- CueAddReleaseNotes --->
## [0.50.0-ea5] November 6, 2018
[0.50.0-ea5]: https://github.com/datawire/ambassador/compare/0.50.0-ea4...0.50.0-ea5

**Ambassador 0.50.0-ea5 is an EARLY ACCESS release! IT IS NOT SUPPORTED FOR PRODUCTION USE.**

### Major changes:

- Ambassador 0.50.0 is a major rearchitecture of Ambassador onto Envoy V2 using the ADS.
- The KAT suite provides dramatically-faster functional testing. See ambassador/tests/kat.
 
### Fixes since 0.50.0-ea4:

- **`use_remote_address` is now set to `true` by default.** If you need the old behavior, you will need to manually set `use_remote_address` to `false` in the `ambassador` `Module`.
- Ambassador 0.50.0-ea5 **supports SNI!**  See the docs for more here.
- Header matching is now supported again, including `host` and `method` headers.

## [0.50.0-ea4] October 31, 2018
[0.50.0-ea4]: https://github.com/datawire/ambassador/compare/0.50.0-ea3...0.50.0-ea4

**Ambassador 0.50.0-ea4 is an EARLY ACCESS release! IT IS NOT SUPPORTED FOR PRODUCTION USE.**

### Major changes:

- Ambassador 0.50.0 is a major rearchitecture of Ambassador onto Envoy V2 using the ADS.
- The KAT suite provides dramatically-faster functional testing. See ambassador/tests/kat.
 
### Fixes since 0.50.0-ea3:

- Ambassador 0.50.0-ea4 uses Envoy 1.8.0.
- `RateLimitService` is now supported. **You will need to restart Ambassador if you change the `RateLimitService` configuration.** We expect to lift this restriction in a later release; for now, the diag service will warn you when a restart is required.
   - The `RateLimitService` also has a new `timeout_ms` attribute, which allows overriding the default request timeout of 20ms.
- GRPC is provisionally supported, but still needs improvements in test coverage.  
- Ambassador will correctly include its EA number when checking for updates.

## [0.50.0-ea3] October 21, 2018
[0.50.0-ea3]: https://github.com/datawire/ambassador/compare/0.50.0-ea2...0.50.0-ea3

**Ambassador 0.50.0-ea3 is an EARLY ACCESS release! IT IS NOT SUPPORTED FOR PRODUCTION USE.**

### Major changes:

- Ambassador 0.50.0 is a major rearchitecture of Ambassador onto Envoy V2 using the ADS.
- The KAT suite provides dramatically-faster functional testing. See ambassador/tests/kat.
 
### Fixes since 0.50.0-ea2:

- `TracingService` is now supported. **You will need to restart Ambassador if you change the `TracingService` configuration.** We expect to lift this restriction in a later release; for now, the diag service will warn you when a restart is required.
- Websockets are now supported, **including** mapping the same websocket prefix to multiple upstream services for canary releases or load balancing.
- KAT supports full debug logs by individual `Test` or `Query`.

**Ambassador 0.50.0 is not yet feature-complete. Read the Limitations and Breaking Changes sections in the 0.50.0-ea1 section below for more information.** 

## [0.50.0-ea2] October 16, 2018
[0.50.0-ea2]: https://github.com/datawire/ambassador/compare/0.50.0-ea1...0.50.0-ea2

**Ambassador 0.50.0-ea2 is an EARLY ACCESS release! IT IS NOT SUPPORTED FOR PRODUCTION USE.**

### Major changes:

- Ambassador 0.50.0 is a major rearchitecture of Ambassador onto Envoy V2 using the ADS.
- The KAT suite provides dramatically-faster functional testing. See ambassador/tests/kat.
 
### Fixes since 0.50.0-ea1:

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

### Major changes:

- Ambassador 0.50.0 is a major rearchitecture of Ambassador onto Envoy V2 using the ADS.
- The KAT suite provides dramatically-faster functional testing. See ambassador/tests/kat.
 
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

- BugFix: The statsd container has been removed by default in order to avoid DoSing Kubernetes DNS. The functionality can be re-enabled by setting the `STATSD_ENABLED` environment variable to `true` in the Ambassador deployment YAML (#568).
- Docs: Added detailed Ambassador + Istio Integration Documentation on monitoring and distributed tracing. - @feitnomore

### Minor Changes:

- Docs: Added instructions for running Ambassador with Docker Compose. - @bcatcho
- BugFix: Fix Ambassador to more aggressively reconnect to Kubernetes (#554). - @nmatsui
- Feature: Diagnostic view displays AuthService, RateLimitService, and TracingService (#730). - @alexgervais
- Feature: Enable Ambassador to tag tracing spans with request headers via `tag_headers`. - @alexgervais

## [0.38.0] August 08, 2018
[0.38.0]: https://github.com/datawire/ambassador/compare/0.37.0...0.38.0

### Major changes:
- Feature: Default CORS configuration can now be set - @KowalczykBartek
- BugFix: Ambassador does not crash with empty YAML config anymore - @rohan47

### Minor changes:
- DevEx: `master` is now latest, `stable` tracks the latest released version
- DevEx: release-prep target added to Makefile to facilitate releasing process
- DevEx: all tests now run in parallel, consuming lesser time
- BugFix: Ambassador SIGCHLD messages are less scary looking now

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

## [0.35.2] July 5, 2018: **READ THE WARNING ABOVE**
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

## [0.34.1] June 4, 2018
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

## [0.32.2] May 2, 2018
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

## [0.28.1] March 5, 2018 (and [0.28.0] on March 2, 2018)
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

## [0.25.0] February 6, 2018
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

## [0.23.0] January 17, 2017
[0.23.0]: https://github.com/datawire/ambassador/compare/v0.22.0...v0.23.0

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
