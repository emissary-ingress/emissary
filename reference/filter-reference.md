# Filter Reference

Filters are used to extend the Ambassador Edge Stack to modify or
intercept an HTTP request before sending to your backend service.  You
may use any of the built-in Filter types, or use the `Plugin` filter
type to run custom code written in the Go programming language, or use
the `External` filter type to call run out to custom code written in a
programming language of your choice.

Filters are created with the `Filter` resource type, which contains global arguments to that filter.  Which Filter(s) to use for which HTTP requests is then configured in `FilterPolicy` resources, which may contain path-specific arguments to the filter.

For more information about developing filters, see the [Filter Development Guide](../../docs/guides/filter-dev-guide).

## `Filter` Definition

Filters are created as `Filter` resources.  The body of the resource
spec depends on the filter type:

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name:      "string"      # required; this is how to refer to the Filter in a FilterPolicy
  namespace: "string"      # optional; default is the usual `kubectl apply` default namespace
spec:
  ambassador_id:           # optional; default is ["default"]
  - "string"
  ambassador_id: "string"  # no need for a list if there's only one value
  FILTER_TYPE:
    GLOBAL_FILTER_ARGUMENTS
```

Currently, the Ambassador Edge Stack supports four filter types: `External`, `JWT`, `OAuth2`, and `Plugin`.

### Filter Type: `External`

The `External` filter calls out to an external service speaking the [`ext_authz` protocol][], providing a highly flexible interface to plug in your own authentication, authorization, and filtering logic.

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

 - `add_linkerd_headers` (optional) when true, in the request to the external auth service, adds an `l5d-dst-override` HTTP header that is set to the hostname and port number of the external auth service.  Defaults to `false`.

This `.spec.External` is mostly identical to an [`AuthService`](../services/auth-service) `.spec`, with the following exceptions:

* In an `AuthService`, the `tls` field may either be a Boolean, or a string referring to a `TLSContext`. In an `External` filter, it may only be a Boolean; referring to a TLS context is not supported.
* In an `AuthService`, the `add_linkerd_headers` field defaults based on the [`ambassador Module`](../core/ambassador). In an `External` filter, it defaults to `false`. This may change in a future release.

[`ext_authz` protocol]: /reference/services/ext_authz

### Filter Type: `JWT`

The `JWT` filter type performs JWT validation on a [Bearer token]
present in the HTTP header.  If the Bearer token JWT doesn't validate,
or has insufficient scope, an RFC 6750-complaint error response with a
`WWW-Authenticate` header is returned.  The list of acceptable signing
keys is loaded from a JWK Set that is loaded over HTTP, as specified
in `jwksURI`.  Only RSA and `none` algorithms are supported.

[Bearer token]: https://tools.ietf.org/html/rfc6750

#### `JWT` Global Arguments

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: "example-jwt-filter"
  namespace: "example-namespace"
spec:
  JWT:
    jwksURI:          "url-string"  # required, unless the only validAlgorithm is "none"
    insecureTLS:      bool          # optional; default is false
    renegotiateTLS:   "enum-string" # optional; default is "never"
    validAlgorithms:                # optional; default is "all supported algos except for 'none'"
    - "RS256"
    - "RS384"
    - "RS512"
    - "none"

    audience:         "string"      # optional, unless `requireAudience: true`
    requireAudience:  bool          # optional; default is false

    issuer:           "url-string"  # optional, unless `requireIssuer: true`
    requireIssuer:    bool          # optional; default is false

    requireIssuedAt:  bool          # optional; default is false
    requireExpiresAt: bool          # optional; default is false
    requireNotBefore: bool          # optional; default is false

    injectRequestHeaders:           # optional; default is []
    - name:   "header-name-string"    # required
      value:  "go-template-string"    # required

    realm:            "string"      # optional; default is "{{.metadata.name}}.{{.metadata.namespace}}"
       
    errorResponse:                  # optional
      contentType: "string"         # deprecated; use 'headers' instead
      headers:                      # optional; default is [{name: "Content-Type", value: "application/json"}]
      - name: "header-name-string"  # required
        value: "go-template-string" # required
      bodyTemplate: "string"        # optional; default is `{{ . | json "" }}`
```

 - `insecureTLS` disables TLS verification for the cases when
   `jwksURI` begins with `https://`.  This is discouraged in favor of
   either using plain `http://` or [installing a self-signed
   certificate](#installing-self-signed-certificates).
 - `renegotiateTLS` allows a remote server to request TLS renegotiation. 
   Accepted values are "never", "onceAsClient", and "freelyAsClient".
 - `injectRequestHeaders` injects HTTP header fields in to the request
   before sending it to the upstream service; where the header value
   can be set based on the JWT value.  The value is specified as a [Go
   `text/template`][] string, with the following data made available
   to it:

    * `.token.Raw` → `string` the raw JWT
    * `.token.Header` → `map[string]interface{}` the JWT header, as parsed JSON
    * `.token.Claims` → `map[string]interface{}` the JWT claims, as parsed JSON
    * `.token.Signature` → `string` the token signature
    * `.httpRequestHeader` → [`http.Header`][] a copy of the header of
      the incoming HTTP request.  Any changes to `.httpRequestHeader`
      (such as by using using `.httpRequestHeader.Set`) have no
      effect.  It is recommended to use `.httpRequestHeader.Get`
      instead of treating it as a map, in order to handle
      capitalization correctly.

   Any headers listed will override (not append to) the original
   request header with that name.
 - `realm` allows specifying the realm to report in the
   `WWW-Authenticate` response header.
 - `errorResponse` allows templating the error response, overriding
    the default json error format.  Make sure you validate and test
    your template, not to generate server-side errors on top of client
    errors.
    * `contentType` is deprecated, and is equivalent to including a
      `name: "Content-Type"` item in `headers`.
    * `headers` sets extra HTTP header fields in the error response.
      The value is specified as a [Go `text/template`][] string, with
      the same data made available to it as `bodyTemplate` (below).
      It does not have access to the `json` function.
    * `bodyTemplate` specifies body of the error; specified as a [Go
      `text/template`][] string, with the following data made
      available to it:

       * `.status_code` → `integer` the HTTP status code to be returned
       * `.httpStatus` → `integer` an alias for `.status_code` (hidden from `{{ . | json "" }}`)
       * `.message` → `string` the error message string
       * `.error` → `error` the raw Go `error` object that generated `.message` (hidden from `{{ . | json "" }}`)
       * `.error.ValidationError` → [`jwt.ValidationError`][] the JWT validation error, will be `nil` if the error is not purely JWT validation (insufficient scope, malformed or missing `Authorization` header)
       * `.request_id` → `string` the Envoy request ID, for correlation (hidden from `{{ . | json "" }}` unless `.status_code` is in the 5XX range)
       * `.requestId` → `string` an alias for `.request_id` (hidden from `{{ . | json "" }}`)

      In addition to the [standard functions available to Go
      `text/template`s][Go `text/template` functions], there is a
      `json` function that arg2 as JSON, using the arg1 string as the
      starting indent level.

**Note**: If you are using a templating system for your YAML that also
makes use of Go templating, then you will need to
escape the template strings meant to be interpreted by the Ambassador Edge
Stack.

[Go `text/template`]: https://golang.org/pkg/text/template/
[Go `text/template` functions]: https://golang.org/pkg/text/template/#hdr-Functions
[`http.Header`]: https://golang.org/pkg/net/http/#Header
[`jwt.ValidationError`]: https://godoc.org/github.com/dgrijalva/jwt-go#ValidationError

#### `JWT` Path-Specific Arguments

```yaml
---
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
  name: "example-filter-policy"
  namespace: "example-namespace"
spec:
  rules:
  - host: "*"
    path: "*"
    filters:
    - name: "example-jwt-filter"
      arguments:
        scope:                    # optional; default is []
        - "scope-value-1"
        - "scope-value-2"
```

 - `scope`: A list of OAuth scope values to require be listed in the
   [`scope` claim][].  In addition to the normal of the `scope` claim
   (a JSON string containing a space-separated list of values), the
   JWT Filter also accepts a JSON array of values.

[`scope` claim]: https://tools.ietf.org/html/draft-ietf-oauth-token-exchange-19#section-4.2

#### Example `JWT` `Filter`

```yaml
# Example results are for the JWT:
#
#    eyJhbGciOiJub25lIiwidHlwIjoiSldUIiwiZXh0cmEiOiJzbyBtdWNoIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.
#
# To save you some time decoding that JWT:
#
#   header = {
#     "alg": "none",
#     "typ": "JWT",
#     "extra": "so much"
#   }
#   claims = {
#     "sub": "1234567890",
#     "name": "John Doe",
#     "iat": 1516239022
#   }
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: example-jwt-filter
  namespace: example-namespace
spec:
  JWT:
    jwksURI: "https://getambassador-demo.auth0.com/.well-known/jwks.json"
    validAlgorithms:
      - "none"
    audience: "myapp"
    requireAudience: false
    injectRequestHeaders:
      - name: "X-Fixed-String"
        value: "Fixed String"
        # result will be "Fixed String"
      - name: "X-Token-String"
        value: "{{ .token.Raw }}"
        # result will be "eyJhbGciOiJub25lIiwidHlwIjoiSldUIiwiZXh0cmEiOiJzbyBtdWNoIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ."
      - name: "X-Token-H-Alg"
        value: "{{ .token.Header.alg }}"
        # result will be "none"
      - name: "X-Token-H-Typ"
        value: "{{ .token.Header.typ }}"
        # result will be "JWT"
      - name: "X-Token-H-Extra"
        value: "{{ .token.Header.extra }}"
        # result will be "so much"
      - name: "X-Token-C-Sub"
        value: "{{ .token.Claims.sub }}"
        # result will be "1234567890"
      - name: "X-Token-C-Name"
        value: "{{ .token.Claims.name }}"
        # result will be "John Doe"
      - name: "X-Token-C-Iat"
        value: "{{ .token.Claims.iat }}"
        # result will be "1.516239022e+09" (don't expect JSON numbers
        # to always be formatted the same as input; if you care about
        # that, specify the formatting; see the next example)
      - name: "X-Token-C-Iat-Decimal"
        value: "{{ printf \"%.0f\" .token.Claims.iat }}"
        # result will be "1516239022"
      - name: "X-Token-S"
        value: "{{ .token.Signature }}"
        # result will be "" (since "alg: none" was used in this example JWT)
      - name: "X-Authorization"
        value: "Authenticated {{ .token.Header.typ }}; sub={{ .token.Claims.sub }}; name={{ printf \"%q\" .token.Claims.name }}"
        # result will be: "Authenticated JWT; sub=1234567890; name="John Doe""
      - name: "X-UA"
        value: "{{ .httpRequestHeader.Get \"User-Agent\" }}"
        # result will be: "curl/7.66.0" or
        # "Mozilla/5.0 (X11; Linux x86_64; rv:69.0) Gecko/20100101 Firefox/69.0"
        # or whatever the requesting HTTP client is
    errorResponse:
      headers:
      - name: "Content-Type"
        value: "application/json"
      - name: "X-Correlation-ID"
        value: "{{ .httpRequestHeader.Get \"X-Correlation-ID\" }}"
      # Regarding the "altErrorMessage" below:
      #   ValidationErrorExpired = 1<<4 = 16
      # https://godoc.org/github.com/dgrijalva/jwt-go#StandardClaims
      bodyTemplate: |-
        {
            "errorMessage": {{ .message | json "    " }},
            {{- if .error.ValidationError }}
            "altErrorMessage": {{ if eq .error.ValidationError.Errors 16 }}"expired"{{ else }}"invalid"{{ end }},
            "errorCode": {{ .error.ValidationError.Errors | json "    "}},
            {{- end }}
            "httpStatus": "{{ .status_code }}",
            "requestId": {{ .request_id | json "    " }}
        }
```

### Filter Type: `OAuth2`

The `OAuth2` filter type performs OAuth2 authorization against an identity provider implementing [OIDC Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html).

#### `OAuth2` Global Arguments

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: "example-oauth2-filter"
  namespace: "example-namespace"
spec:
  OAuth2:
    authorizationURL:      "url-string"      # required
    grantType              "enum-string"     # optional; default is "AuthorizationCode"
    extraAuthorizationParameters:            # optional; default is {}
      "string": "string"

    accessTokenValidation: "enum-string"     # optional; default is "auto"
    accessTokenJWTFilter:                    # optional; default is null
      name: "string"                           # required
      namespace: "string"                      # optional; default is the same namespace as the Filter
      arguments: JWT-Filter-Arguments          # optional

    # Settings for grantType=="AuthorizationCode"
    clientURL:             "url-string"      # required
    stateTTL:              "duration-string" # optional; default is "5m"
    clientID:              "string"          # required
    # A client secret must be specified.
    # This can be done by including the raw secret as a string in "secret",
    # or by referencing Kubernetes secret with "secretName" (and "secretNamespace").
    # It is invalid to specify both "secret" and "secretName".
    secret:                "string"          # required (unless secretName is set)
    secretName:            "string"          # required (unless secret is set)
    secretNamespace:       "string"          # optional; default is the same namespace as the Filter

    # HTTP client settings for talking with the identity provider
    insecureTLS:           bool              # optional; default is false
    renegotiateTLS:        "enum-string"     # optional; default is "never"
    maxStale:              "duration-string" # optional; default is "0"
```

General settings:

 - `authorizationURL`: Identifies where to look for the
   `/.well-known/openid-configuration` descriptor to figure out how to
   talk to the OAuth2 provider.
 - `grantType`: Which type of OAuth 2.0 authorization grant to request
   from the identity provider.  Currently supported are:
   * `"AuthorizationCode"`: Authenticate by redirecting to a
     login page served by the identity provider.
   * `"ClientCredentials"`: Authenticate by requiring
     `X-Ambassador-Client-ID` and `X-Ambassador-Client-Secret` HTTP
     headers on incoming requests, and using them to authenticate to
     the identity provider.  Support for the `ClientCredentials` is
     currently preliminary, and only goes through limited testing.
 - `extraAuthorizationParameters`: Extra (non-standard or extension)
    OAuth authorization parameters to use.  It is not valid to specify
    a parameter used by OAuth itself ("response_type", "client_id",
    "redirect_uri", "scope", or "state").
 - `accessTokenValidation`: How to verify the liveness and scope of
   Access Tokens issued by the identity provider.  Valid values are
   either `"auto"`, `"jwt"`, or `"userinfo"`.  Empty or unset is
   equivalent to `"auto"`.
   * `"jwt"`: Validates the Access Token as a JWT.
     + By default: It accepts the RS256, RS384, or RS512 signature
       algorithms, and validates the signature against the JWKS from
       OIDC Discovery.  It then validates the `exp`, `iat`, `nbf`,
       `iss` (with the Issuer from OIDC Discovery), and `scope`
       claims; if present, none of the scopes are required to be
       present.  This relies on the identity provider using
       non-encrypted signed JWTs as Access Tokens, and configuring the
       signing appropriately
	 + This behavior can be modified by delegating to [`JWT`
       Filter](#filter-type-jwt) with `accessTokenJWTFilter`.  The
       arguments are the same as the arguments when erferring to a JWT
       Filter from a FilterPolicy.
   * `"userinfo"`: Validates the access token by polling the OIDC
     UserInfo Endpoint.  This means that the Ambassador Edge Stack
     must initiate an HTTP request to the identity provider for each
     authorized request to a protected resource.  This performs
     poorly, but functions properly with a wider range of identity
     providers.  It is not valid to set `accessTokenJWTFilter` if
     `accessTokenValidation: userinfo`.
   * `"auto"` attempts has it do `"jwt"` validation if
     `accessTokenJWTFilter` is set or if the Access Token parses as a
     JWT and the signature is valid, and otherwise falls back to
     `"userinfo"` validation.

Settings that are only valid when `grantType: "AuthorizationCode"`:

 - `clientURL`: (You determine this, and give it to your identity
   provider) Identifies a hostname that can appropriately set cookies
   for the application.  Only the scheme (`https://`) and authority
   (`example.com:1234`) parts are used; the path part of the URL is
   ignored.  You will also likely need to register
   `${clientURL}/callback` as an authorized callback endpoint with
   your identity provider.
 - `clientID`: The Client ID you get from your identity provider.
 - The client secret you get from your identity provider can be
   specified 2 different ways:
   * As a string, in the `secret` field.
   * As a Kubernetes `generic` Secret, named by
     `secretName`/`secretNamespace`.  The Kubernetes secret must of
     the `generic` type, with the value stored under the key
     `oauth2-client-secret`.  If `secretNamespace` is not given, it
     defaults to the namespace of the Filter resource.
   * **Note**: It is invalid to set both `secret` and `secretName`.
 - `stateTTL`: (You decide this) How long the Ambassador Edge Stack will
   wait for the user to submit credentials to the identity provider
   and receive a response to that effect from the identity provider

HTTP client settings for talking to the identity provider:

 - `maxStale`: How long to keep stale cached OIDC replies for.  This
   sets the `max-stale` Cache-Control directive on requests, and also
   ignores the `no-store` and `no-cache` Cache-Control directives on
   responses.  This is useful for maintaining good performance when
   working with identity providers with mis-configured Cache-Control.
 - `insecureTLS` disables TLS verification when speaking to an
   identity provider with an `https://` `authorizationURL`.  This is
   discouraged in favor of either using plain `http://` or [installing
   a self-signed certificate](#installing-self-signed-certificates).
 - `renegotiateTLS` allows a remote server to request TLS
   renegotiation.  Accepted values are "never", "onceAsClient", and
   "freelyAsClient".

`"duration-string"` strings are parsed as a sequence of decimal
numbers, each with optional fraction and a unit suffix, such as
"300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or
"µs"), "ms", "s", "m", "h".  See [Go
`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration).

#### `OAuth2` Path-Specific Arguments

```yaml
---
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
  name: "example-filter-policy"
  namespace: "example-namespace"
spec:
  rules:
  - host: "*"
    path: "*"
    filters:
    - name: "example-oauth2-filter"
      arguments:
        scopes:                   # optional; default is ["openid"] for `grantType=="AuthorizationCode"`; [] for `grantType=="ClientCredentials"`
        - "scope1"
        - "scope2"
        insteadOfRedirect:        # optional; default is to do a redirect to the identity provider
          ifRequestHeader:        # optional; default is to return httpStatusCode for all requests that would redirect-to-identity-provider
            name: "string"        # required
            value: "string"       # optional; default is any non-empty string
          # option 1:
          httpStatusCode: integer # optional; default is 403 (unless `filters` is set)
          # option 2:
          filters:                # optional; default is to use `httpStatusCode` instead
          - name: "string"          # required
            namespace: "string"     # optional; default is the same namespace as the FilterPolicy
            ifRequestHeader:        # optional; default to apply this filter to all requests matching the host & path
              name: "string"          # required
              value: "string"         # optional; default is any non-empty string
            onDeny: "enum-string"   # optional; default is "break"
            onAllow: "enum-string"  # optional; default is "continue"
            arguments: DEPENDS      # optional
```

 - `scopes`: A list of OAuth scope values to include in the scope of
   the authorization request.  If one of the scope values for a path
   is not granted, then access to that resource is forbidden; if the
   `scopes` argument lists `foo`, but the authorization response from
   the provider does not include `foo` in the scope, then it will be
   taken to mean that the authorization server forbade access to this
   path, as the authenticated user does not have the `foo` resource
   scope.

   If `grantType: "AuthorizationCode"`, then the `openid` scope value
   is always included in the requested scope, even if it is not listed
   in the `FilterPolicy` argument.

   If `grantType: "ClientCredentials"`, then the default scope is
   empty.  If your identity provider does not have a default scope,
   then you will need to configure one here.

   As a special case, if the `offline_access` scope value is
   requested, but not included in the response then access is not
   forbidden.  With many identity providers, requesting the
   `offline_access` scope is necessary in order to receive a Refresh
   Token.

   The ordering of scope values does not matter, and is ignored.

 - `insteadOfRedirect`: An action to perform instead of redirecting
   the User-Agent to the identity provider.  By default, if the
   User-Agent does not have a currently-authenticated session, then the
   Ambassador Edge Stack will redirect the User-Agent to the identity provider.
   Setting `insteadOfRedirect` allows you to modify this behavior.
   `ifRequestHeader` does nothing when `grantType:
   "ClientCredentials"`, because the Ambassador Edge Stack will never
   redirect the User-Agent to the identity provider for the client
   credentials grant type.
    * If `insteadOfRedirect` is non-`null`, then by default it will
      apply to all requests that would cause the redirect; setting the
      `ifRequestHeader` sub-argument causes it to only apply to
      requests that have the HTTP header field `name`
      (case-insensitive) set to `value` (case-sensitive); or requests
      that have `name` set to any non-empty string if `value` is
      unset.
    * By default, it serves an authorization-denied error page; by
      default HTTP 403 ("Forbidden"), but this can be configured by
      the `httpStatusCode` sub-argument.
    * Instead of serving that simple error page, it can instead be
      configured to call out to a list of other Filters, by setting
      the `filters` list.  The syntax and semantics of this list are
      the same as `.spec.rules[].filters` in a
      [`FilterPolicy`](#filterpolicy-definition).  Be aware that if
      one of these filters modify the request rather than returning a
      response, then the request will be allowed through to the
      backend service, even though the `OAuth2` Filter denied it.
    * It is invalid to specify both `httpStatusCode` and `filters`.

### Filter Type: `Plugin`

The `Plugin` filter type allows you to plug in your own custom code. This code is compiled to a `.so` file, which you load in to the Ambassador Edge Stack container at `/etc/ambassador-plugins/${NAME}.so`.

#### The Plugin Interface

This code is written in the Go programming language (golang), and must
be compiled with the exact same compiler settings as the Ambassador
Edge Stack; and any overlapping libraries used must have their
versions match exactly. This information is documented in the
`/ambassador/aes-abi.txt` file in the AES docker image.

Plugins are compiled with `go build -buildmode=plugin -trimpath`, and
must have a `main.PluginMain` function with the signature
`PluginMain(w http.ResponseWriter, r *http.Request)`:

```go
package main

import (
	"net/http"
)

func PluginMain(w http.ResponseWriter, r *http.Request) { … }
```

`*http.Request` is the incoming HTTP request that can be mutated or intercepted, which is done by `http.ResponseWriter`.

Headers can be mutated by calling `w.Header().Set(HEADERNAME, VALUE)`.
Finalize changes by calling `w.WriteHeader(http.StatusOK)`.

If you call `w.WriteHeader()` with any value other than 200 (`http.StatusOK`) instead of modifying the request, the plugin has
taken over the request, and the request will not be sent to your backend service.  You can call `w.Write()` to write the body of an error page.

#### `Plugin` Global Arguments

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: "example-plugin-filter"
  namespace: "example-namespace"
spec:
  Plugin:
    name: "string" # required; this tells it where to look for the compiled plugin file; "/etc/ambassador-plugins/${NAME}.so"
```

#### `Plugin` Path-Specific Arguments

Path specific arguments are not supported for Plugin filters at this
time.

## `FilterPolicy` Definition

`FilterPolicy` resources specify which filters (if any) to apply to
which HTTP requests.

```yaml
---
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
  name: "example-filter-policy"
  namespace: "example-namespace"
spec:
  rules:
  - host: "glob-string"
    path: "glob-string"
    filters:                    # optional; omit or set to `null` or `[]` to apply no filters to this request
    - name: "string"              # required
      namespace: "string"         # optional; default is the same namespace as the FilterPolicy
      ifRequestHeader:            # optional; default to apply this filter to all requests matching the host & path
        name: "string"            # required
        value: "string"           # optional; default is any non-empty string
      onDeny: "enum-string"       # optional; default is "break"
      onAllow: "enum-string"      # optional; default is "continue"
      arguments: DEPENDS          # optional
```

The type of the `arguments` property is dependent on the which Filter type is being referred to; see the "Path-Specific Arguments" documentation for each Filter type.

When multiple `Filter`s are specified in a rule:

 * The filters are gone through in order
 * Each filter may either
   1. return a direct HTTP *response*, intended to be sent back to the
      requesting HTTP client (normally *denying* the request from
      being forwarded to the upstream service); or
   2. return a modification to make to the HTTP *request* before
      sending it to other filters or the upstream service (normally
      *allowing* the request to be forwarded to the upstream service
      with modifications).
 * If a filter has a `ifRequestHeader` setting, the filter is skipped
   unless the request (including any modifications made by earlier
   filters) matches the described header; the request must have the
   HTTP header field `name` (case-insensitive) set to `value`
   (case-sensitive); or have `name` set to any non-empty string if
   `value` is unset.
 * `onDeny` identifies what to do when the filter returns an "HTTP
   response":
   - `"break"`: End processing, and return the response directly to
     the requesting HTTP client.  Later filters are not called.  The
     request is not forwarded to the upstream service.
   - `"continue"`: Continue processing.  The request is passed to the
     next filter listed; or if at the end of the list, it is forwarded
     to the upstream service.  The HTTP response returned from the
     filter is discarded.
 * `onAllow` identifies what to do when the filter returns a
   "modification to the HTTP request":
   - `"break"`: Apply the modification to the request, then end filter
     processing, and forward the modified request to the upstream
     service.  Later filters are not called.
   - `"continue"`: Continue processing.  Apply the request
     modification, then pass the modified request to the next filter
     listed; or if at the end of the list, forward it to the upstream
     service.
 * Modifications to the request are cumulative; later filters have
   access to _all_ headers inserted by earlier filters.

### `FilterPolicy` Example

In the example below, the `param-filter` Filter Plugin is loaded, and
configured to run on requests to `/httpbin/`.

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: param-filter # This is the name used in FilterPolicy
  namespace: standalone
spec:
  Plugin:
    name: param-filter # The plugin's `.so` file's base name

---
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
  name: httpbin-policy
spec:
  rules:
  # Don't apply any filters to requests for /httpbin/ip
  - host: "*"
    path: /httpbin/ip
    filters: null
  # Apply param-filter and auth0 to requests for /httpbin/
  - host: "*"
    path: /httpbin/*
    filters:
    - name: param-filter
    - name: auth0
  # Default to authorizing all requests with auth0
  - host: "*"
    path: "*"
    filters:
    - name: auth0
```

**Note:** The Ambassador Edge Stack will choose the first `FilterPolicy` rule that matches the incoming request. As in the above example, you must list your rules in the order of least to most generic.

## Installing self-signed certificates

The `JWT` and `OAuth2` filters speak to other services over HTTP or HTTPS.  If those services are configured to speak HTTPS using a
self-signed certificate, attempting to talk to them will result in an error mentioning `ERR x509: certificate signed by unknown authority`. You can fix this by installing that self-signed certificate in to the
AES container following the standard procedure for Alpine Linux 3.8: Copy the certificate to `/usr/local/share/ca-certificates/` and then run `update-ca-certificates`.  Note that the `aes` image sets `USER 1000`, but that `update-ca-certificates` needs to be run as root.

```Dockerfile
FROM quay.io/datawire/aes:$version$
USER root
COPY ./my-certificate.pem /usr/local/share/ca-certificates/my-certificate.crt
RUN update-ca-certificates
USER 1000
```

When deploying the Ambassador Edge Stack, refer to that custom Docker image,
rather than to `quay.io/datawire/aes:$version$`
