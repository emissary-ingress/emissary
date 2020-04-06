# Filter Type: `External`

The `External` filter calls out to an external service speaking the [`ext_authz` protocol](../../../running/services/ext_authz), providing a highly flexible interface to plug in your own authentication, authorization, and filtering logic.

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: "example-external-filter"
  namespace: "example-namespace"
spec:
  External:
    auth_service:       "url-ish-string"    # required
    tls:                bool                # optional; default is true if `auth_service` starts with "https://" (case-insensitive), false otherwise
    proto:              "enum-string"       # optional; default is "http"
    timeout_ms:         integer             # optional; default is 5000
    allow_request_body: bool                # deprecated; use include_body instead
    include_body:                           # optional; default is null
      max_bytes:          integer             # required
      allow_partial:      bool                # required
    status_on_error:                        # optional
      code:               integer             # optional; default is 403
    failure_mode_allow: bool                # optional; default is false

    # the following are used only if `proto: http`; they are ignored if `proto: grpc`

    path_prefix:                   "/path"  # optional; default is ""
    allowed_request_headers:                # optional; default is []
    - "x-allowed-input-header"
    allowed_authorization_headers:          # optional; default is []
    - "x-input-headers"
    - "x-allowed-output-header"
    add_linkerd_headers:           bool     # optional; default is false
```

 - `auth_service` (required) is of the format `[scheme://]host[:port]`, and identifies the external auth service to talk to.  The scheme-part may be `http://` or `https://`, which influences the default value of `tls`, and of the port-part.  If no scheme-part is given, it behaves as if `http://` was given.

 - `tls` (optional) is whether to use TLS or cleartext when speaking to the external auth service.  The default is based on the scheme-part of the `auth_service`.

 - `proto` (optional) specifies which variant of the [`ext_authz` protocol][] to use when communicating with the external auth service.  Valid options are `http` (default) or `grpc`.

 - `timeout_ms` (optional) is the total maximum duration in milliseconds for the request to the external auth service, before triggering `status_on_error` or `failure_mode_allow`.

 - `allow_request_body` (optional, deprecated) controls whether to buffer the request body in order to pass to the external auth service.  Setting `allow_request_body: true` is exactly equivalent to `include_body: { max_bytes: 4096, allow_partial: true }`, and `allow_request_body: false` is exactly equivalent to `include_body: null`.  It is invalid to set both `allow_request_body` and `include_body`.

 - `include_body` (optional) controls how much to buffer the request body to pass to the external auth service, for use cases such as computing an HMAC or request signature.  If `include_body` is `null` or unset, then the request body is not buffered at all, and an empty body is passed to the external auth service.  If `include_body` is not `null`, both of its sub-fields are required:
    * `max_bytes` (required) controls the amount of body data that will be passed to the external auth service
    * `allow_partial` (required) controls what happens to requests with bodies larger than `max_bytes`:
       * if `allow_partial` is `true`, the first `max_bytes` of the body are sent to the external auth service.
       * if `false`, the message is rejected with HTTP 413 ("Payload Too Large").

   Unfortunately, in order for `include_body` to function properly, the `AuthService` in [`aes.yaml`](/yaml/aes.yaml) must be edited to have `include_body` set with `max_bytes` greater than the largest `max_bytes` used by any `External` filter (so if an `External` filter has `max_bytes: 4096`, then the `AuthService` will need `max_bytes: 4097`), and `allow_partial: true`.

 - `status_on_error` (optional) controls the status code returned when unable to communicate with external auth service.  This is ignored if `failure_mode_allow: true`.
    * `code` (optional) defaults to 403.
    
 - `failure_mode_allow` (optional) being set to `true` causes the request to be allowed through to the upstream backend service if there is an error communicating with the external auth service, instead of returning `status_on_error.code` to the client.  Defaults to false.

The following fields are only used if `proto: http`; they are ignored if `proto: grpc`:

 - `path_prefix` (optional) prepends a string to the request path of the request when sending it to the external auth service.  By default this is empty, and nothing is prepended.  For example, if the client makes a request to `/foo`, and `path_prefix: /bar`, then the path in the request made to the external auth service will be `/foo/bar`.

 - `allowed_request_headers` (optional) lists the headers that will be sent copied from the incoming request to the request made to the external auth service (case-insensitive).  In addition to the headers listed in this field, the following headers are always included:
    * `Authorization`
    * `Cookie`
    * `From`
    * `Proxy-Authorization`
    * `User-Agent`
    * `X-Forwarded-For`
    * `X-Forwarded-Host`
    * `X-Forwarded-Proto`

 - `allowed_authorization_headers` (optional) lists the headers that will be copied from the response from the external auth service to the request sent to the upstream backend service (if the external auth service indicates that the request to the upstream backend service should be allowed).  In addition to the headers listed in this field, the following headers are always included:
    * `Authorization`
    * `Location`
    * `Proxy-Authenticate`
    * `Set-cookie`
    * `WWW-Authenticate`

 - `add_linkerd_headers` (optional) when true, in the request to the external auth service, adds an `l5d-dst-override` HTTP header that is set to the hostname and port number of the external auth service.  Defaults to `false`.

This `.spec.External` is mostly identical to an [`AuthService`](../../../running/services/auth-service) `.spec`, with the following exceptions:

* In an `AuthService`, the `tls` field may either be a Boolean, or a string referring to a `TLSContext`. In an `External` filter, it may only be a Boolean; referring to a TLS context is not supported.
* In an `AuthService`, the `add_linkerd_headers` field defaults based on the [`ambassador Module`](../../../running/ambassador). In an `External` filter, it defaults to `false`. This may change in a future release.

[`ext_authz` protocol]: ../../../running/services/ext_authz
