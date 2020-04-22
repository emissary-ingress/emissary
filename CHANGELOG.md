Information in this file will be used to build the merged release notes for GA
releases. Please keep it up to date.
------------------------------------------------------------------------------

# Ambassador Edge Stack CHANGELOG

## 1.4.2 (TBD)

- Bugfix: The `OAuth2` Filter redirection-endpoint now handles various XSRF errors more consistently (the way we meant it to in 1.2.1)
- Feature (alpha): Route-to-code
- Bugfix: The ACME client now obeys `AMBASSADOR_ID`

## 1.4.1 (2020-04-15)

(no proprietary changes)

## 1.4.0 (2020-04-08)

- Bugfix: The "Filters" tab in the webui no longer renders the value of OAuth client secrets that are stored in Kubernetes secrets.
- Bugfix: The ACME client of of one Ambassador install will no longer interfere with the ACME client of another Ambassador install in the same namespace with a different AMBASSADOR_ID.

## 1.3.2 (2020-04-01)

(no proprietary changes)

## 1.3.1 (2020-03-24)

- Bugfix: `OAuth2` Filter: Correctly ask the user to re-authenticate when the Refresh Token expires, rather than emitting an internal server error.
- Bugfix: The `Password` grant type properly uses a session per user, rather than client-wide
- Bugfix: The `Password` grant type also separates sessions by `Filter`, so multiple `Filter`s can work together correctly.

## 1.3.0 (2020-03-17)

- Feature: Support username and password as headers for OAuth2 authentication (`grantType: Password`)
- Bugfix: The Edge Policy Console now honors the `diagnostics.enabled` setting in the `ambassador` Module
- Bugfix: If the `DEVPORTAL_CONTENT_URL` is not accessible, log a warning but don't crash.
- Change: There is no longer a separate traffic-proxy image; that functionality is now part of the main AES image. Set `command: ["traffic-manager"]` to use it.
- Bugfix: The `Plugin` Filter now correctly sets `request.TLS` to nil/non-nil based on if the original request was encrypted or not.
- Feature: `aes-plugin-runner` now allows passing in `docker run` flags after the main argument list.

## 1.2.2 (2020-03-04)

- Internal: Track maximum usage for 24-hour periods, not just instantaneous whenever we phone home.

## 1.2.1 (2020-03-03)

- Bugfix: The `aes-plugin-runner` binary for GNU/Linux is now statically linked (instead of being linked against musl libc), so it should now work on either musl libc or GNU libc systems
- Feature: An `aes-plugin-runner` binary for Windows is now produced.  (It is un-tested as of yet.)
- Bugfix: The `OAuth2` Filter redirection-endpoint now handles various XSRF errors more consistently
- Change: The `OAuth2` Filter redirection-endpoint now handles XSRF errors by redirecting back to the identity provider

## 1.2.0 (2020-02-24)

- Change: The `ambassador` service now uses the default `externalTrafficPolicy` of `Cluster` rather than explicitly setting it to `Local`. This is a safer setting for GKE where the `Local` policy can cause outages when ambassador is updated. See https://stackoverflow.com/questions/60121956/are-hitless-rolling-updates-possible-on-gke-with-externaltrafficpolicy-local for details.
- Bugfix: The RBAC for `Ingress` now supports the `networking.k8s.io` `apiGroup`
- Bugfix: Quiet Dev Portal debug logs
- Change: There is no longer a separate app-sidecar image; it is now combined in to the main aes image; set `command: ["app-sidecar"]` to use that functionality.
- Change: The `OAuth2` Filter no longer sets cookies when `insteadOfRedirect` triggers
- Change: The `OAuth2` Filter more frequently adjusts the cookies
- Feature: `ifRequestHeader` can now have `valueRegex` instead of `value`
- Feature: The `OAuth2` Filter now has `useSessionCookies` option to have cookies expire when the browser closes, rather than at a fixed duration
- Feature: `ifRequestHeader` now has `negate: bool` to invert the match

## 1.1.1 (2020-02-12)

- Feature: The Policy Console can now set the log level to "trace" (in addition to "info" or "debug")
- Bugfix: Don't have the Policy Console poll for snapshots when logged out
- Bugfix: Do a better job of noticing when the license key changes
- Bugfix: `aes-plugin-runner --version` now works properly
- Bugfix: Only serve the custom CONGRATULATIONS! 404 page on `/`
- Change: The `OAuth2` Filter `stateTTL` setting is now ignored; the lifetime of state-tokens is now managed automatically

## 1.1.0 (2020-01-28)

- Feature: `External` Filter now exactly matches the current OSS `AuthService` definiton.  It has gained `include_body` (deprecating `allow_request_body`), `status_on_error`, and `failure_mode_allow`.
- Docs: add instructions for what to do after downloading `edgectl`
- Bugfix: make it much faster to apply the Edge Stack License
- Bugfix: make sure the ACME terms-of-service link is always shown
- Bugfix: make the Edge Policy Console more performant

## 1.0.0 (2020-01-15)

Behavior:

 * Developer portal no longer requires the /openapi Mapping
 * Renamed environment variable APRO_DEVPORTAL_CONTENT_URL to DEVPORTAL_CONTENT_URL
 * Feature: Developer portal can check out a non-default branch. Control with DEVPORTAL_CONTENT_BRANCH env var
 * Feature: Developer portal can use a subdir of a checkout. Control with DEVPORTAL_CONTENT_DIR env var
 * `apictl traffic initialize` no longer waits for the traffic-proxy to become ready before exiting.
 * Feature: Developer portal will show swagger documentation for up to five services (or more with appropriate license)
 * Feature: local-devportal is now a standalone go binary with no external dependencies
 * `v1` license keys were not being used so augment them to include emails
 * The OAuth2 redirection endpoint has moved from `/callback` to `/.ambassador/oauth2/redirection-endpoint`.  Migrating Pro users will need to notify thier IDP of the change.

Other:

 * `amb-core` and `amb-sidecar` have been merged in to a combined `aes` which is based on Ambassador OSS [version TBD].
 * `login-gate-js`content has been updated for a clearer first time experience. 

## 0.11.0 (2019-12-10)

Configuration:

 * `JWT` Filter now has a `realm` setting to configure the realm mentioned in `WWW-Authenticate` of error responses.
 * Feature: `JWT` Filter now has a FilterPolicy argument `scope` to preform `draft-ietf-oauth-token-exchange`-compatible Scope validation.
 * Feature: `OAuth2` Filter now has a `.insteadOfRedirect.filters` FilterPolicy argument that lets you provide a list of filters to run; as if you were listing them directly in a FilterPolicy.
 * Feature: `OAuth2` Filter now has a `extraAuthorizationParameters` setting to manually pass extra parameters to the IDP's authorization endpoint.
 * Feature: `OAuth2` Filter now has a `accessTokenJWTFilter` setting to use a `JWT` filter for access token validation when `accessTokenValidation: jwt` or `accessTokenValidation: auto`.

Behavior:

 * Feature: `JWT` Filter now generates RFC 6750-compliant responses with the `WWW-Authenticate` header set.

Other:

 * Update Ambassador Core from Ambassador 0.85.0 (Envoy 1.11+half-way-to-1.12) to 0.86.0 (Envoy 1.12.2)

## 0.10.0 (2019-11-11)

Configuration:

 * Feature: `FilterPolicy` may now set `ifRequestHeader` to only apply a `Filter` to requests with appropriate headers.
 * Feature: `FilterPolicy` may now set `onDeny` and `onAllow` to modify how `Filter`s chain together.
 * Feature: `JWT` Filter `injectRequestHeaderse` templates can now read the incoming HTTP request headers.
 * Feature: `JWT` Filter `errorResponse` can now set HTTP headers of the error response.
 * Beta feature: `OAuth2` Filter can now be configured to receive OAuth client credentials in the HTTP request header, and use them to obtain a client credentials grant.  This is only currently tested with Okta.

Behavior:

 * The `OAuth2` filter's XSRF protection now works differently.  You should use the `ambassador_xsrf.{name}.{namespace}` cookie instead of the `ambassador_session.{name}.{namespace}` cookie for XSRF-protection purposes.

## 0.9.1 (2019-10-22)

Configuration:

 * The `JWT` and `OAuth2` Filter types support `renegotiateTLS`
 * The `JWT` Filter now has an `errorResponse` argument that allows templating the filter's error response.

Other:

 * Update Ambassador Core from Ambassador 0.83.0 to 0.85.0

## 0.9.0 (2019-10-08)

Configuration

 * The `OAuth2` filter now has a FilterPolicy argument `insteadOfRedirect` that can specify a different action to perform than redirecting to the IDP.

Behavior:

 * Feature: Developer portal URL can be changed by the user. Adjust the `ambassador-pro-devportal` `Mapping` CRD (or annotation) by changing the `prefix` to desired prefix and changing the `rewrite` to `/docs/`. The `ambassador-pro-devportal-api` can not be adjusted yet. 
 * Feature: The `OAuth2` filter can now perform OIDC-session RP-initiated logout when used with an identity provider that supports it.
 * Bugfix: Properly return a 404 for unknown paths in the amb-sidecar; instead of serving the index page; this could happen if the devportal Mapping is misconfigured.
 * Bugfix: Fix the "loaded filter" log info message.
 * Bugfix: Don't publish the "dev-portal-server" Docker image; it was obviated by "amb-sidecar" in 0.8.0.
 * Bugfix: The `JWT` Filter is no longer case-sensitive with the auth-scheme (`Bearer` vs `bearer`)
 * Bugfix: The `JWT` Filter no longer accepts authorizations that are missing an auth-scheme

Other:

 * Update Ambassador Core from Ambassador 0.75.0 to 0.83.0
 * Incorporate the Envoy 1.11.2 security patches in Ambassador Core
 * Fast iteration on Developer Portal styling and content using a docker image inside a local checkout of Developer Portal content repo (see reference doc for usage guide)

## 0.8.0 (2019-09-16)

Configuration:

 * `amb-sidecar` now takes additional configuration related to the developer portal.

Behavior:

 * Feature: The developer portal is now in "beta", and incorporated into amb-sidecar.
 * Bugfix: The `External` Filter no longer erroneously follows redirects.
 * Bugfix: Fixed a case-folding bug causing the `JWT` Filter to be inoperable.
 * Enhancement: Errors in `Filter` resource definitions are now recorded and included in error messages.

## 0.7.0 (2019-08-29)

Configuration:

 * `amb-sidecar`: The default value of `USE_STATSD` has changed from `true` to `false`.
 * Bump license key schema v0 → v1.  The developer portal requires a v1 license with the "devportal" feature enabled.  Some future version of the other functionality will drop support for v0 license keys.
 * The `JWT` Filter can now inject HTTP request headers; configured with the `injectRequestHeaders` field.

Behavior:

 * Fixed a resource leak in dev-portal-server

Other:

 * There is now a build of Ambassador with Certified Envoy named "amb-core".

## 0.6.0 (2019-08-05)

Configuration:

 * The CRD field `ambassador_id` may now be a single string instead of a list of strings (this should have always been the case, but there was a bug in the parser).
 * Everything is now on one port: `APRO_HTTP_PORT`, which defaults to `8500`.
 * `LOG_LEVEL` no longer exists; everything obeys `APP_LOG_LEVEL`.
 * The meaning of `REDIS_POOL_SIZE` has changed slightly; there are no longer separate connection pools for ratelimit and filtering; the maximum number of connections is now `REDIS_POOL_SIZE` instead of 2×`REDIS_POOL_SIZE`.
 * The `amb-sidecar` RateLimitService can now report to statsd, and attempts to do so by default (`USE_STATSD`, `STATSD_HOST`, `STATSD_PORT`, `GOSTATS_FLUSH_INTERVAL_SECONDS`).

Behavior:

 * Now also handles gRPC requests for `envoy.service.auth.v2`, in addition to `envoy.service.auth.v2alpha`.
 * Log a stacktrace at log-level "debug" whenever the HTTP client encounters an error.
 * Fix bug where the wrong key was selected from a JWKS.
 * Everything in amb-sidecar now runs as a single process.

## 0.5.0 (2019-06-21)

Configuration:

 * Redis is now always required to be configured.
 * The `amb-sidecar` environment variables `$APRO_PRIVATE_KEY_PATH` and `$APRO_PUBLIC_KEY_PATH` are replaced by a Kubernetes secret and the `$APRO_KEYPAIR_SECRET_NAME` and `$APRO_KEYPAIR_SECRET_NAMESPACE` environment variables.
 * If the `$APRO_KEYPAIR_SECRET_NAME` Kubernetes secret (above) does not exist, `amb-sidecar` now needs the "create" permission for secrets in its ClusterRole.
 * The `OAuth2` Filter now ignores the `audience` field setting.  I expect it to make a come-back in 0.5.1 though.
 * The `OAuth2` Filter now acts as if the `openid` scope value is always included in the FilterPolicy's `scopes` argument.
 * The `OAuth2` Filter can verify Access Tokens with several different methods; configured with the `accessTokenValidation` field.

Behavior:

 * The `OAuth2` Filter is now strictly compliant with OAuth 2.0.  It is verified to work properly with:
   - Auth0
   - Azure AD
   - Google
   - Keycloak
   - Okta
   - UAA
 * The `OAuth2` Filter browser cookie has changed:
   - It is now named `ambassador_session.{{filter_name}}.{{filter_namespace}}` instead of `access_token`.
   - It is now an opaque string instead of a JWT Access Token.  The Access Token is still available in the injected `Authorization` header.
 * The `OAuth2` Filter will no longer consider a user-agent-provided `Authorization` header, it will only consider the cookie.
 * The `OAuth2` Filter now supports Refresh Tokens; they must be requested by listing `offline_access` in the `scopes` argument in the FilterPolicy.
 * The `OAuth2` Filter's `/callback` endpoint is no longer vulnerable to XSRF attacks
 * The Developer Portal file descriptor leak is fixed.

Other:

 * Open Source dependency license compliance is now automated as part of the release machinery.  Source releases for the Docker images are now present in the images themselves at `/*.opensource.tar.gz`.

## 0.4.3 (2019-05-15)

 * Add the Developer Portal (experimental; no documentation available yet)
 * `apictl traffic initialize`: Correctly handle non-`default` namespaces
 * `app-sidecar`: Respect the `APP_LOG_LEVEL` environment variable, same as `amb-sidecar`

## 0.4.2 (2019-05-03)

 * Turn down liveness and readiness probe logging from "info" to "debug"

## 0.4.1 (2019-04-23)

 * Add liveness and readiness probes

## 0.4.0 (2019-04-18)

 * Moved all of the default sidecar ports around; YAML will need to be adjusted (hence 0.4.0 instead of 0.3.2).  Additionally, all of the ports are now configurable via environment variables

   | Purpose          | Variable       | Old  | New  |
   | -------          | --------       | ---  | ---  |
   | Auth gRPC        | APRO_AUTH_PORT | 8082 | 8500 |
   | RLS gRPC         | GRPC_PORT      | 8081 | 8501 |
   | RLS debug (HTTP) | DEBUG_PORT     | 6070 | 8502 |
   | RLS HTTP ???     | PORT           | 7000 | 8503 |

 * `apictl` no longer sets an imagePullSecret when deploying Pro things to the cluster (since the repo is now public)

## 0.3.1 (2019-04-05)

 * Support running the Ambassador sidecar as a non-root user

## 0.3.0 (2019-04-03)

 * New Filter type `External`
 * Request IDs in the Pro logs are the same as the Request IDs in the Ambassador logs
 * `OAuth2` Filter type supports `secretName` and `secretNamespace`
 * Switch to using Ambassador OSS gRPC API
 * No longer necessary to set `allowed_request_headers` or `allowed_authorization_headers` for `Plugin` Filters
 * RLS logs requests as `info` instead of `warn`
 * Officially support Okta as an IDP

## 0.2.5 (2019-04-02)

(0.3.0 was initially tagged as 0.2.5)

## 0.2.4 (2019-03-19)

 * `JWT` and `OAuth2` Filter types support `insecureTLS`
 * `OAuth2` now handles JWTs with a `scope` claim that is a JSON list of scopes, instead of a JSON string containing a whitespace-separated list of scopes (such as those generated by UAA)

## 0.2.3 (2019-03-13)

 * Consul Connect integration no longer requires a license key

## 0.2.2 (2019-03-11)

 * Fix Consul certificate rotation

## 0.2.1 (2019-03-08)

 * Move the AuthService from port 8080 to 8082, and make it configurable with `APRO_AUTH_PORT`

## 0.2.0 (2019-03-04)

 * Have everything require license keys
 * Differentiate between components when phoning-home to Scout
 * Phone-home to kubernaut.io/scout, not metriton.datawire.io/scout
 * Fix bug where `apictl traffic inject` wiped existing `imagePullSecrets`
 * Support `AMBASSADOR_ID`, `AMBASSADOR_SINGLE_NAMESPACE`, and `AMBASSADOR_NAMESPACE`
 * Log format changed
 * OIDC support
 * Replace `Tenant` and `Policy` CRDs with `Filter` and `FilterPolicy` CRDs
 * Add JWT validation filter
 * Add `apro-plugin-runner` (previously was in a separate OSS git repo)

## 0.1.2 (2019-01-24)

 * More readable logs in the event of a crash
 * `apictl traffic` sets `imagePullSecret`
 * Have `apictl` also look for the license key in `~/.config/` as a fallback on macOS.  The paths it now looks in, from highest to lowest precedence, are:
    - `$HOME/Library/Application Support/ambassador/license-key` (macOS only)
    - `${XDG_CONFIG_HOME:-$HOME/.config}/ambassador/license-key`
    - `$HOME/.ambassador.key`

## 0.1.1 (2019-01-23)

 - First release with combined rate-limiting and authentication.
