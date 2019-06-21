# Filters

Filters are used to extend Ambassador to modify or intercept an HTTP
request before sending to the the backend service.  You may use any of
the built-in Filter types, or use the `Plugin` filter type to run
custom code written in Golang.

Filters are created with the `Filter` resource type, which contains
global arguments to that filter.  Which Filter(s) to use for which
HTTP requests is then configured in `FilterPolicy` resources, which
may contain path-specific arguments to the filter.

For more information about developing filters, see the [Filter Development Guide](/docs/guides/filter-dev-guide).

## `Filter` Definition

Filters are created as `Filter` resources.  The body of the resource
spec depends on the filter type:

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: "string"
spec:
  ambassador_id:           # optional; default is ["default"]
  - "string"
  ambassador_id: "string"  # no need for a list if there's only one value
  FILTER_TYPE:
    GLOBAL_FILTER_ARGUMENTS
```

Currently, Ambassador supports four filter types: `External`, `JWT`, `OAuth2`, and `Plugin`.

### Filter Type: `External`

The `External` filter type exposes the Ambassador `AuthService` interface to external authentication services. This is useful in a number of situations, e.g., if you have already written a custom `AuthService`, but also want to use other filters.

The `External` filter looks very similar to an `AuthService` annotation:

```
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: http-auth-filter
  namespace: standalone
spec:
  External:
    auth_service: "http-auth:4000"
    path_prefix: "/frobnitz"
    proto: http
    allowed_request_headers:
    - "x-allowed-input-header"
    allowed_authorization_headers:
    - "x-input-headers"
    - "x-allowed-output-header"
    allow_request_body: false
```

The spec is mostly identical to an `AuthService`, with the following exceptions:

* It does not contain the apiVersion field
* It does not contain the kind field
* It does not contain the name field
* In an AuthService, the tls field may either be a Boolean, or a string referring to a TLS context. In an `External`, it may only be a Boolean; referring to a TLS context is not supported.

### Filter Type: `JWT`

The `JWT` filter type performs JWT validation. The list of acceptable signing keys is loaded from a JWK Set that is loaded over HTTP, as specified in `jwksURI`. Only RSA and `none` algorithms are supported.

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: my-jwt-filter
spec:
  JWT:
    jwksURI: "https://ambassador-oauth-e2e.auth0.com/.well-known/jwks.json" # required, unless the only validAlgorithm is "none"
    insecureTLS: true # optional, default is false
    validAlgorithms: # omitting this means "all supported algos except for 'none'"
      - "RS256"
      - "RS384"
      - "RS512"
      - "none"
    audience: "myapp" # optional unless requireAudience: true
    requireAudience: true # optional, default is false
    issuer: "https://ambassador-oauth-e2e.auth0.com/" # optional unless requireIssuer: true
    requireIssuer: true # optional, default is false
    requireIssuedAt: true # optional, default is false
    requireExpiresAt: true # optional, default is false
    requireNotBefore: true # optional, default is false
```

 - `insecureTLS` disables TLS verification for the cases when
   `jwksURI` begins with `https://`.  This is discouraged in favor of
   either using plain `http://` or [installing a self-signed
   certificate](#installing-self-signed-certificates).

### Filter Type: `OAuth2`

The `OAuth2` filter type performs OAuth2 authorization against an
identity provider implementing [OIDC Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html).

#### `OAuth2` Global Arguments

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: "example-filter"
spec:
  OAuth2:
    authorizationURL:      "url-string"      # required
    clientURL:             "url-string"      # required
    stateTTL:              "duration-string" # optional; default is "5m"
    insecureTLS:           bool              # optional; default is false
    clientID:              "string"
    secret:                "string"
    maxStale:              "duration-string" # optional; default is "0"
    accessTokenValidation: "enum-string"     # optional; default is "auto"
```

 - `authorizationURL`: Identifies where to look for the
   `/.well-known/openid-configuration` descriptor to figure out how to
   talk to the OAuth2 provider.
 - `clientURL`: Identifies a hostname that can appropriately set
   cookies for the application.  Only the scheme (`https://`) and
   authority (`example.com:1234`) parts are used; the path part of the
   URL is ignored.
 - `stateTTL`: How long Ambassador will wait for the user to submit
   credentials to the identity provider and receive a response to that
   effect from the identity provider
 - `insecureTLS` disables TLS verification when speaking to an
   `https://` identity provider.  This is discouraged in favor of
   either using plain `http://` or [installing a self-signed
   certificate](#installing-self-signed-certificates).
   <!-- - `audience`: The OIDC audience. -->
 - `clientID`: The Client ID you get from your identity provider.
 - `secret`: The client secret you get from your identity provider.
 - `maxStale`: How long to keep stale cached OIDC replies for.  This
   sets the `max-stale` Cache-Control directive on requests, and also
   ignores the `no-store` and `no-cache` Cache-Control directives on
   responses.  This is useful for working with identity providers with
   mis-configured Cache-Control.
 - `accessTokenValidation`: How to verify the liveness and scope of
   Access Tokens issued by the identity provider.  Valid values are
   either `"auto"`, `"jwt"`, or `"userinfo"`.  Empty or unset is
   equivalent to `"auto"`.
   * `"jwt"`: Validates the Access Token as a JWT.  It accepts the
     RS256, RS384, or RS512 signature algorithms, and validates the
     signature against the JWKS from OIDC Discovery.  It then
     validates the `exp`, `iat`, `nbf`, `iss` (with the Issuer from
     OIDC Discovery), and `scope` claims; if present, none of the
     scopes are required to be present.  This relies on the identity
     provider using non-encrypted signed JWTs as Access Tokens, and
     configuring the signing appropriately.
   * `"userinfo"`: Validates the access token by polling the OIDC
     UserInfo Endpoint.  This means that Ambassador Pro must initiate
     an HTTP request to the identity provider for each authorized request to a
     protected resource.  This performs poorly, but functions properly
     with a wider range of identity providers.
   * `"auto"` attempts has it do `"jwt"` validation if the Access
     Token parses as a JWT and the signature is valid, and otherwise
     falls back to `"userinfo"` validation.

`"duration-string"` strings are parsed as a sequence of decimal
numbers, each with optional fraction and a unit suffix, such as
"300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or
"µs"), "ms", "s", "m", "h".  See [Go
`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration).

#### `OAuth2` Path-Specific Arguments

```
---
apiVersion: getambassador.io/v1beta2
kind: FilterPolicy
metadata:
  name: "example-filter-policy"
spec:
  rules:
  - host: "*"
    path: "*"
    filters:
    - name: "example-filter"
      arguments:
        scopes:
        - "scope1"
        - "scope2"
```

You may specify a list of OAuth scope values to include in the scope
of the authorization request.  If one of the scope values for a path
is not granted, then access to that resource is forbidden; if the
`scopes` argument lists `foo`, but the authorization response from the
provider does not include `foo` in the scope, then it will be taken to
mean that the authorization server forbade access to this path, as the
authenticated user does not have the `foo` resource scope.

The `openid` scope value is always included in the requested scope,
even if it is not listed in the `FilterPolicy` argument.

As a special case, if the `offline_access` scope value is requested,
but not included in the response then access is not forbidden.  With
many identity providers, requesting the `offline_access` scope is
necessary in order to receive a Refresh Token.

### Filter Type: `Plugin`

The `Plugin` filter type allows you to plug in your own custom code.
This code is compiled to a `.so` file, which you load in to the
Ambassador Pro container at `/etc/ambassador-plugins/${NAME}.so`.

#### The Plugin Interface

This code is written in Golang, and must be compiled with the exact
compiler settings as Ambassador Pro.  As of Ambassador Pro v0.2.0-rc1,
that is:

 - `go` compiler `1.11.4`
 - `GOOS=linux`
 - `GOARCH=amd64`
 - `GO111MODULE=on`
 - `CGO_ENABLED=1`

Plugins are compiled with `go build -buildmode=plugin`, and must have
a `main.PluginMain` function with the signature `PluginMain(w
http.ResponseWriter, r *http.Request)`:

```go
package main

import (
	"net/http"
)

func PluginMain(w http.ResponseWriter, r *http.Request) { … }
```

`*http.Request` is the incoming HTTP request that can be mutated or
intercepted, which is done by `http.ResponseWriter`.

Headers can be mutated by calling `w.Header().Set(HEADERNAME, VALUE)`.
Finalize changes by calling `w.WriteHeader(http.StatusOK)`.

If you call `w.WriteHeader()` with any value other than 200
(`http.StatusOK`) instead of modifying the request, the plugin has
taken over the request, and the request will not be sent to the
backend service.  You can call `w.Write()` to write the body of an
error page.

#### `Plugin` Global Arguments

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: "example-filter" # This is how to refer to the Filter in a FilterPolicy
spec:
  Plugin:
    name: "string" # This tells it where to look for the compiled plugin file; "/etc/ambassador-plugins/${NAME}.so"
```

#### `Plugin` Path-Specific Arguments

Path specific arguments are not supported for Plugin filters at this
time.

## `FilterPolicy` Definition

`FilterPolicy` resources specify which filters (if any) to apply to
which HTTP requests.

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: FilterPolicy
metadata:
  name: "example-filter-policy"
spec:
  rules:
  - host: "glob-string"
    path: "glob-string"
    filters:                 # optional; omit or set to `null` to apply no filters to this request
    - name: "exampe-filter"  # required
      namespace: "string"    # optional; default is the same namespace as the FilterPolicy
      arguments: DEPENDS     # optional
```

The type of the `arguments` property is dependent on the which Filter
type is being referred to; see the "Path-Specific Arguments"
documentation for each Filter type.

When multiple `Filter`s are specified in a rule:

 * The filters are gone through in order
 * Later filters have access to _all_ headers inserted by earlier
   filters.
 * The final backend service (i.e., the service where the request will
   ultimately be routed) will only have access to inserted headers if
   they are listed in `allowed_authorization_headers` in the
   Ambassador annotation.
 * Filter processing is aborted by the first filter to return a
   non-200 status.

### Example

In the example below, the `param-filter` Filter Plugin is loaded, and
configured to run on requests to `/httpbin/`.

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: param-filter # This is the name used in FilterPolicy
  namespace: standalone
spec:
  Plugin:
    name: param-filter # The plugin's `.so` file's base name

---
apiVersion: getambassador.io/v1beta2
kind: FilterPolicy
metadata:
  name: httpbin-policy
spec:
  rules:
  # Default to authorizing all requests with auth0
  - host: "*"
    path: "*"
    filters:
    - name: auth0
  # And also apply the param-filter to requests for /httpbin/
  - host: "*"
    path: /httpbin/*
    filters:
    - name: param-filter
    - name: auth0
  # But don't apply any filters to requests for /httpbin/ip
  - host: "*"
    path: /httpbin/ip
    filters: null
```

## Installing self-signed certificates

The JWT and OAuth2 filters speak to other servers over HTTP or HTTPS.
If those servers are configured to speak HTTPS using a self-signed
certificate, attempting to talk to them will result in a error
mentioning `ERR x509: certificate signed by unknown authority`.  You
can fix this by installing that self-signed certificate in to the Pro
container following the standard procedure for Alpine Linux 3.8: Copy
the certificate to `/usr/local/share/ca-certificates/` and then run
`update-ca-certificates`.  Note that the amb-sidecar image set `USER
1000`, but `update-ca-certificates` needs to be run as root.

```Dockerfile
FROM quay.io/datawire/ambassador_pro:amb-sidecar-%aproVersion%
USER root
COPY ./my-certificate.pem /usr/local/share/ca-certificates/my-certificate.crt
RUN update-ca-certificates
USER 1000
```

When deploying Ambassador Pro, refer to that Docker image, rather than
to `quay.io/datawire/ambassador_pro:amb-sidecar-%aproVersion%`.
