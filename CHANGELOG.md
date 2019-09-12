## Ambassador Pro CHANGELOG

## 0.7.1 (TBD)

Behavior:

 * The `External` Filter no longer erronously follows redirects.
 * Fixed a case-folding bug in the `JWT` Filter

## 0.7.0 (2019-08-29)

Configuration:

 * `amb-sidecar`: The default value of `USE_STATSD` has changed from `true` to `false`.
 * Bump license key schema v0 → v1.  The developer portal requires a
   v1 license with the "devportal" feature enabled.  Some future
   version of the other functionality will drop support for v0 license
   keys.
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
 * The meaning of `REDIS_POOL_SIZE` has changed slightly; there are no longer separate connection pools for ratelimit and filtering; the maxiumum number of connections is now `REDIS_POOL_SIZE` instead of 2×`REDIS_POOL_SIZE`.
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

 * Open Source dependency licence compliance is now automated as part of the release machinery.  Source releases for the Docker images are now present in the images themselves at `/*.opensource.tar.gz`.

## 0.4.3 (2019-05-15)

 * Add the Developer Portal
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
 * No longer nescessary to set `allowed_request_headers` or `allowed_authorization_headers` for `Plugin` Filters
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
