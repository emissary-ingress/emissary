# AuthService Plugin

The Ambassador API Gateway provides a highly flexible mechanism for authentication, via the `AuthService` resource.  An `AuthService` configures Ambassador to use an external service to check authentication and authorization for incoming requests. Each incoming request is authenticated before routing to its destination.

All requests are validated by the `AuthService` (unless the `Mapping` applied to the request sets `bypass_auth`).  It is not possible to combine multiple `AuthService`s.  While it is possible to create multiple `AuthService` resources, they will be load-balanced between each resource in a round-robin fashion. This is useful for canarying an `AuthService` change, but is not useful for deploying multiple distinct `AuthService`s.  In order to combine multiple external services (either having multiple services apply to the same request, or selecting between different services for the different requests), instead of using an `AuthService`, use [Ambassador Edge Stack `External Filter`](../../filter-reference).

Because of the limitations described above, **the Ambassador Edge Stack does not support `AuthService` resources, and you should instead use an [`External` `Filter`](../../filter-reference),** which is mostly a drop-in replacement for an `AuthService`.

## Configure an External AuthService

The currently supported version of the `AuthService` resource is `getambassador.io/v2`. Earlier versions are deprecated.

```yaml
---
apiVersion: getambassador.io/v2
kind: AuthService
metadata:
  name: authentication
spec:
  ambassador_id: string-or-string-list # optional; default is ["default"]

  auth_service: "example-auth:3000" # required
  tls: true                         # optional; default is true if `auth_service` starts with "https://" (case-insensitive), false otherwise
  proto: http                       # optional; default is "http"
  timeout_ms: 5000                  # optional; default is 5000
  #allow_request_body: true         # deprecated; use include_body instead
  include_body:                     # optional; default is null
    max_bytes: 4096                   # required
    allow_partial: true               # required
  status_on_error:                  # optional
    code: 503                         # optional; default is 403
  failure_mode_allow: false         # optional; default is false

  # the following are used only if `proto: http`; they are ignored if `proto: grpc`

  path_prefix: "/path"             # optional; default is ""
  allowed_request_headers:         # optional; default is []
  - "x-example-header"
  allowed_authorization_headers:   # optional; default is []
  - "x-qotm-session"
  add_linkerd_headers: bool        # optional; default is based on the ambassador Module
```

 - `auth_service` (required) is of the format `[scheme://]host[:port]`, and identifies the external auth service to talk to.  The scheme-part may be `http://` or `https://`, which influences the default value of `tls`, and of the port-part.  If no scheme-part is given, it behaves as if `http://` was given.

 - `tls` (optional) is whether to use TLS or cleartext when speaking to the external auth service.  The default is based on the scheme-part of the `auth_service`.  If the value of `tls` is not a Boolean, the value is taken to be the name of a defined [`TLSContext`](/reference/core/tls/#tlscontext), which will determine the certificate presented to the upstream service.

 - `proto` (optional) specifies which variant of the [`ext_authz` protocol][] to use when communicating with the external auth service.  Valid options are `http` (default) or `grpc`.

 - `timeout_ms` (optional) is the total maximum duration in milliseconds for the request to the external auth service, before triggering `status_on_error` or `failure_mode_allow`.

 - `allow_request_body` (optional, deprecated) controls whether to buffer the request body in order to pass to the external auth service.  Setting `allow_request_body: true` is exactly equivalent to `include_body: { max_bytes: 4096, allow_partial: true }`, and `allow_request_body: false` is exactly equivalent to `include_body: null`.  It is invalid to set both `allow_request_body` and `include_body`.

 - `include_body` (optional) controls how much to buffer the request body to pass to the external auth service, for use cases such as computing an HMAC or request signature.  If `include_body` is `null` or unset, then the request body is not buffered at all, and an empty body is passed to the external auth service.  If `include_body` is not `null`, both of its sub-fields are required:
    * `max_bytes` (required) controls the amount of body data that will be passed to the external auth service
    * `allow_partial` (required) controls what happens to requests with bodies larger than `max_bytes`:
       * if `allow_partial` is `true`, the first `max_bytes` of the body are sent to the external auth service.
       * if `false`, the message is rejected with HTTP 413 ("Payload Too Large").

 - `status_on_error` (optional) controls the status code returned when unable to communicate with external auth service.  This is ignored if `failure_mode_allow: true`.
    * `code` (optional) defaults to 403.

 - `failure_mode_allow` (optional) being set to `true` causes the request to be allowed through to the upstream backend service if there is an error communicating with the external auth service, instead of returning `status_on_error.code` to the client Defaults to false.

The following fields are only used if `proto: http`; they are ignored if `proto: grpc`:

 - `path_prefix` (optional) prepends a string to the request path of the request when sending it to the external auth service.  By default this is empty, and nothing is prepended.  For example, if the client makes a request to `/foo`, and `path_prefix: /bar`, then the path in the request made to the external auth service will be `/foo/bar`.

 - `allowed_request_headers` (optional) lists headers that will be sent copied from the incoming request to the request made to the external auth service (case-insensitive).  In addition to the headers listed in this field, the following headers are always included:
    * `Authorization`
    * `Cookie`
    * `From`
    * `Proxy-Authorization`
    * `User-Agent`
    * `X-Forwarded-For`
    * `X-Forwarded-Host`
    * `X-Forwarded-Proto`

 - `allowed_authorization_headers` (optional) lists headers that will be copied from the response from the external auth service to the request sent to the upstream backend service (if the external auth service indicates that the request to the upstream backend service should be allowed).  In addition to the headers listed in this field, the following headers are always included:
    * `Authorization`
    * `Location`
    * `Proxy-Authenticate`
    * `Set-cookie`
    * `WWW-Authenticate`

 - `add_linkerd_headers` (optional) when true, in the request to the external auth service, adds an `l5d-dst-override` HTTP header that is set to the hostname and port number of the external auth service.  Defaults to the value set in the [`ambassador Module`](../../core/ambassador).

[`ext_authz` protocol]: /reference/services/ext_authz

## Canarying Multiple AuthServices

You may create multiple `AuthService` manifests to round-robin authentication requests among multiple services. **Note well that all services must use the same `path_prefix` and header definitions;** if you try to have different values, you'll see an error in the diagnostics service, telling you which value is being used.

## Configuring Public Mappings

An `AuthService` can be disabled for a mapping by setting `bypass_auth` to `true`. This will tell Ambassador to allow all requests for that mapping through without interacting with the external auth service.

## Example

See the [Authentication Tutorial](../../../user-guide/auth-tutorial) for an example.
