# Ambassador Early Access Releases

From time to time, Ambassador may ship early access releases to test major changes. **Early access releases are not supported for production use**, but are intended to gain early feedback from our community prior to shipping a release.

Early access releases will always have names that include the string "-ea" followed by a build number, for example `0.50.0-ea1` is the first early access build of Ambassador 0.50.0.

*Datawire Note*: This document currently refers to "0.50.0-tt5". This is a test build before the first EA build. Don't panic. (If you don't work at Datawire, *this document is _too early_ for you*. Come back tomorrow.)

## Ambassador 0.50.0 Early Access Releases

### New Features

Ambassador 0.50.0 is a major rewrite meant to support Envoy V2 configuration, and to make it substantially easier to contribute to Ambassador development.

- Envoy configuration V2 with ADS: future extensibility and rapid configuration updates
   - Switching to V2 configuration opens the door to Ambassador for many new Envoy features that are only accessible with V2, such as SNI.
   - Switching to ADS allows Ambassador to respond to configuration changes nearly instantaneously.

- Major internal reorganization: faster, easier development
   - There is now a clear separation between Ambassador configuration, the internal representation where Ambassador's logic operates, and Envoy configurations.
   - The code now makes extensive use of type hinting to reduce errors and assist comprehension.
   - The automated test framework has been reworked to be dramatically faster.

### Breaking Changes

While Ambassador 0.50.0 is nearly 100% backward compatible, there are some breaking changes:

- You can no longer configure Ambassador with a `ConfigMap`.

- The authentication `Module` is no longer supported: use the `AuthService` resource instead.

- External authentication with `AuthService` now uses the Envoy core `ext_authz` filter instead of Datawire's custom filter. 
   - If you are using a custom external authentication service, `ext_authz` speaks the same HTTP protocol, and your service will continue to work.
   - However, where the custom filter sent _all_ HTTP headers to the external authentication service, `ext_authz` does not. You will need to configure the `AuthService` with the set of headers you wish to send to your external authentication service. 
      - See "Limitations of 0.50.0-tt5" before for more.
- Circuit breakers and outlier detection are no longer supported.
   - These features will be reintroduced in a later version of Ambassador.
- Ambassador requires a TLS `Module` to enable TLS termination.
   - Previous versions of Ambassador would automatically enable TLS termination if a Kubernetes secret named `ambassador-certs` was present. 0.50.0 will not.
   - To get the same behavior as previous versions of Ambassador, use the following TLS `Module`:
        ```
        ---
        kind:  Module
        name:  tls
        config: 
          server:
            secret: ambassador-certs
        ```

### Limitations of Ambassador 0.50.0-tt5:

- TLS termination is not currently supported. It will be resupported in 0.50.0-ea1.
- Helm installation has not been tested.
- `RateLimitService` and `TracingService` resources are not currently supported.
- `statsd` integration has not been tested.
- The diagnostics service cannot correctly drill down by source file, though it can drill down by route or other resources.
- `AuthService` does not have full support for configuring the request headers to hand off to the service. The following headers are always sent:

       Authorization
       Cookie
       Forwarded
       From
       Host
       Proxy-Authenticate
       Proxy-Authorization
       Set-Cookie
       User-Agent
       X-Forwarded-For
       X-Forwarded-Host
       X-Forwarded
       X-Gateway-Proto
       WWW-Authenticate

   To send other headers, include them in `allowed_headers`. This will also cause them to be forwarded from the external auth service on to the upstream service on successful auth. 

### Installing Ambassador 0.50.0-tt5:

We do not recommend Helm for early access releases. Instead, use a Kubernetes deployment as usual, but use image `quay.io/datawire/ambassador:0-50-0-tt5`.

We recommend testing with shadowing, as documented below, before switching to any new Ambassador release. We *strongly* recommend testing with shadowing for all early access releases.
 
## Testing with shadowing

One strategy for testing early access releases involves using Ambassador ID and traffic shadowing. You can do the following:

1. Install Ambassador Early Access on your cluster with a unique Ambassador ID.
2. Shadow traffic from your production Ambassador instance to the Ambassador Early Access release.
3. Monitor the Early Access release to determine if there are any problems.
