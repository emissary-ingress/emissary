# Filters

Filters are used to extend Ambassador Edge Stack to modify or intercept an HTTP request before sending to the the backend service.  You may use any of the built-in Filter types, or use the `Plugin` filter type to run custom code written in Golang.

Filters are created with the `Filter` resource type, which contains global arguments to that filter.  Which Filter(s) to use for which HTTP requests is then configured in `FilterPolicy` resources, which may contain path-specific arguments to the filter.

For more information about developing filters, see the [Filter Development Guide](/docs/guides/filter-dev-guide).

## `Filter` Definition

Filters are created as `Filter` resources.  The body of the resource
spec depends on the filter type:

```yaml
---
apiVersion: getambassador.io/v1beta2
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

Currently, Ambassador supports four filter types: `External`, `JWT`, `OAuth2`, and `Plugin`.

### Filter Type: `External`

The `External` filter type exposes the Ambassador Edge Stack `AuthService` interface to external authentication services. This is useful in a number of situations, e.g., if you have already written a custom `AuthService`, but also want to use other filters.

The `External` filter looks very similar to an `AuthService` annotation:

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: "example-external-filter"
  namespace: "example-namespace"
spec:
  External:
    auth_service:                  "url-ish-string" # required
    tls:                           bool             # optional; default is true if `auth_service` starts with "https://" (case-insensitive), false otherwise
    proto:                         "enum-string"    # optional; default is "http"
    timeout:                       integer          # optional; default is 5000
    allow_request_body:            bool             # optional; default is false

    # the following are used only if `proto: http`; they are ignored if `proto: grpc`

    path_prefix:                   "/path"          # optional; default is "/"
    allowed_request_headers:                        # optional; default is []
    - "x-allowed-input-header"
    allowed_authorization_headers:                  # optional; default is []
    - "x-input-headers"
    - "x-allowed-output-header"
```

 - `auth_service` is of the format `[scheme://]host[:port]`.  The
   scheme-part may be `http://` or `https://`, which influences the
   default value of `tls`, and of the port-part.  If no scheme-part is
   given, it behaves as if `http://` was given.
 - `timeout` is the total timeout for the request to the upstream
   external filter, in milleseconds.
 - `proto` is either `"http"` or `"grpc"`.

This `spec.External` is mostly identical to an [`AuthService`](./auth-service), with the following exceptions:

* It does not contain the `apiVersion` field
* It does not contain the `kind` field
* It does not contain the `name` field
* In an `AuthService`, the `tls` field may either be a Boolean, or a string referring to a TLS context. In an `External`, it may only be a Boolean; referring to a TLS context is not supported.

### Filter Type: `JWT`

The `JWT` filter type performs JWT validation. The list of acceptable signing keys is loaded from a JWK Set that is loaded over HTTP, as specified in `jwksURI`. Only RSA and `none` algorithms are supported.

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: "example-jwt-filter"
  namespace: "example-namespace"
spec:
  JWT:
    jwksURI:          "url-string"  # required, unless the only validAlgorithm is "none"
    insecureTLS:      bool          # optional; default is false
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
     - name:   "header-name-string" # required
       value:  "go-template-string" # required
```

 - `insecureTLS` disables TLS verification for the cases when
   `jwksURI` begins with `https://`.  This is discouraged in favor of
   either using plain `http://` or [installing a self-signed
   certificate](#installing-self-signed-certificates).
 - `injectRequestHeaders` injects HTTP header fields in to the request
   before sending it to the upstream service; where the header value
   can be set based on the JWT value.  The value is specified as a [Go
   `text/template`][] string, with the following data made available
   to it:

    * `.token.Raw` → `string` the raw JWT
    * `.token.Header` → `map[string]interface{}` the JWT header, as parsed JSON
    * `.token.Claims` → `map[string]interface{}` the JWT claims, as parsed JSON
    * `.token.Signature` → `string` the token signature

   Any headers listed will override (not append to) the original
   request header with that name.

   **Note**: If you are using a templating system for your YAML that
   also makes use of Go templating (for instance, Helm), then you will
   need to escape the template strings meant to be interpretted by
   Ambassador Pro.

   


[Go `text/template`]: https://golang.org/pkg/text/template/

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
apiVersion: getambassador.io/v1beta2
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
        value: "{{.token.Raw}}"
        # result will be "eyJhbGciOiJub25lIiwidHlwIjoiSldUIiwiZXh0cmEiOiJzbyBtdWNoIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ."
      - name: "X-Token-H-Alg"
        value: "{{.token.Header.alg}}"
        # result will be "none"
      - name: "X-Token-H-Typ"
        value: "{{.token.Header.typ}}"
        # result will be "JWT"
      - name: "X-Token-H-Extra"
        value: "{{.token.Header.extra}}"
        # result will be "so much"
      - name: "X-Token-C-Sub"
        value: "{{.token.Claims.sub}}"
        # result will be "1234567890"
      - name: "X-Token-C-Name"
        value: "{{.token.Claims.name}}"
        # result will be "John Doe"
      - name: "X-Token-C-Iat"
        value: "{{.token.Claims.iat}}"
        # result will be "1.516239022e+09" (don't expect JSON numbers
        # to always be formatted the same as input; if you care about
        # that, specify the formatting; see the next example)
      - name: "X-Token-C-Iat-Decimal"
        value: "{{printf \"%.0f\" .token.Claims.iat}}"
        # result will be "1516239022"
      - name: "X-Token-S"
        value: "{{.token.Signature}}"
        # result will be "" (since "alg: none" was used in this example JWT)
      - name: "X-Authorization"
        value: "Authenticated {{.token.Header.typ}}; sub={{.token.Claims.sub}}; name={{printf \"%q\" .token.Claims.name}}"
        # result will be: "Authenticated JWT; sub=1234567890; name="John Doe""
```

### Filter Type: `OAuth2`

The `OAuth2` filter type performs OAuth2 authorization against an identity provider implementing [OIDC Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html).

#### `OAuth2` Global Arguments

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: "example-oauth2-filter"
  namespace: "example-namespace"
spec:
  OAuth2:
    authorizationURL:      "url-string"      # required
    insecureTLS:           bool              # optional; default is false
    clientID:              "string"          # required
    # A client secret must be specified.
    # This can be done by including the raw secret as a string in "secret",
    # or by referencing Kubernetes secret with "secretName" (and "secretNamespace").
    # It is invalid to specify both "secret" and "secretName".
    secret:                "string"          # required (unless secretName is set)
    secretName:            "string"          # required (unless secret is set)
    secretNamespace:       "string"          # optional; default is the same namespace as the Filter

    # The above is all information that you get directly from you identity provider.

    clientURL:             "url-string"      # required

    # The below is all information that you decide for yourself.

    accessTokenValidation: "enum-string"     # optional; default is "auto"
    maxStale:              "duration-string" # optional; default is "0"
    stateTTL:              "duration-string" # optional; default is "5m"
```

<!-- There is an `adience` field that is currently ignored -->

Information about your identity provider:

 - `authorizationURL`: Identifies where to look for the
   `/.well-known/openid-configuration` descriptor to figure out how to
   talk to the OAuth2 provider.
 - `insecureTLS` disables TLS verification when speaking to an
   identity provider with an `https://` `authorizationURL`.  This is
   discouraged in favor of either using plain `http://` or [installing
   a self-signed certificate](#installing-self-signed-certificates).
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

Information you must decide, and give to your identity provider:

 - `clientURL`: Identifies a hostname that can appropriately set
   cookies for the application.  Only the scheme (`https://`) and
   authority (`example.com:1234`) parts are used; the path part of the
   URL is ignored.  You will also likely need to register
   `${clientURL}/callback` as an authorized callback endpoint with
   your identity provider.

Information that you decide for yourself:

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
 - `maxStale`: How long to keep stale cached OIDC replies for.  This
   sets the `max-stale` Cache-Control directive on requests, and also
   ignores the `no-store` and `no-cache` Cache-Control directives on
   responses.  This is useful for maintaining good performance when
   working with identity providers with mis-configured Cache-Control.
 - `stateTTL`: How long Ambassador Edge Stack will wait for the user to submit
   credentials to the identity provider and receive a response to that
   effect from the identity provider

`"duration-string"` strings are parsed as a sequence of decimal
numbers, each with optional fraction and a unit suffix, such as
"300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or
"µs"), "ms", "s", "m", "h".  See [Go
`time.ParseDuration`](https://golang.org/pkg/time/#ParseDuration).

#### `OAuth2` Path-Specific Arguments

```yaml
---
apiVersion: getambassador.io/v1beta2
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
        scopes:                   # optional; default is ["openid"]
        - "scope1"
        - "scope2"
        insteadOfRedirect:        # optional; default is to do a redirect to the identity provider
          httpStatusCode: integer # optional; default is 403
          ifRequestHeader:        # optional; default is to return httpStatusCode for all requests that would redirect-to-identity-provider
            name: "string"        # required
            value: "string"       # optional; default is any non-empty string
```

 - `scopes`: A list of OAuth scope values to include in the scope of
   the authorization request.  If one of the scope values for a path
   is not granted, then access to that resource is forbidden; if the
   `scopes` argument lists `foo`, but the authorization response from
   the provider does not include `foo` in the scope, then it will be
   taken to mean that the authorization server forbade access to this
   path, as the authenticated user does not have the `foo` resource
   scope.

   The `openid` scope value is always included in the requested scope,
   even if it is not listed in the `FilterPolicy` argument.

   As a special case, if the `offline_access` scope value is
   requested, but not included in the response then access is not
   forbidden.  With many identity providers, requesting the
   `offline_access` scope is necessary in order to receive a Refresh
   Token.

   The ordering of scope values does not matter, and is ignored.

 - `insteadOfRedirect`: An action to perform instead of redirecting
   the User-Agent to the identity provider.  By default, if the
   User-Agent does not have an currently-authenticated session, then
   Ambassador Edge Stack will redirect the User-Agent to the identity provider.
   Setting `insteadOfRedirect` causes it to instead serve an
   authorization-denied error page; by default HTTP 403 ("Forbidden"),
   but this can be configured by the `httpStatusCode` sub-argument.
   By default, `insteadOfRedirect` will apply to all requests that
   would cause the redirect; setting the `ifRequestHeader`
   sub-argument causes it to only apply to requests that have the HTTP
   header field `name` (case-insensitive) set to `value`
   (case-sensitive); or requests that have `name` set to any non-empty
   string if `value` is unset.

### Filter Type: `Plugin`

The `Plugin` filter type allows you to plug in your own custom code. This code is compiled to a `.so` file, which you load in to the Ambassador Pro container at `/etc/ambassador-plugins/${NAME}.so`.



#### The Plugin Interface

This code is written in the Go programming language (golang), and must be compiled with the exact same compiler settings as Ambassador Pro; and any overlapping libraries used must have their versions match exactly.  This information is dockumented in an [apro-abi.txt][] file for each Ambassador Pro release.



[apro-abi.txt]: https://s3.amazonaws.com/datawire-static-files/apro-abi/apro-abi@%aproVersion%.txt

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

`*http.Request` is the incoming HTTP request that can be mutated or intercepted, which is done by `http.ResponseWriter`.

Headers can be mutated by calling `w.Header().Set(HEADERNAME, VALUE)`.
Finalize changes by calling `w.WriteHeader(http.StatusOK)`.

If you call `w.WriteHeader()` with any value other than 200 (`http.StatusOK`) instead of modifying the request, the plugin has
taken over the request, and the request will not be sent to the backend service.  You can call `w.Write()` to write the body of an error page.

#### `Plugin` Global Arguments

```yaml
---
apiVersion: getambassador.io/v1beta2
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
apiVersion: getambassador.io/v1beta2
kind: FilterPolicy
metadata:
  name: "example-filter-policy"
  namespace: "example-namespace"
spec:
  rules:
  - host: "glob-string"
    path: "glob-string"
    filters:                 # optional; omit or set to `null` to apply no filters to this request
    - name:      "string"    # required
      namespace: "string"    # optional; default is the same namespace as the FilterPolicy
      arguments: DEPENDS     # optional
```

The type of the `arguments` property is dependent on the which Filter type is being referred to; see the "Path-Specific Arguments" documentation for each Filter type.

When multiple `Filter`s are specified in a rule:

 * The filters are gone through in order
 * Later filters have access to _all_ headers inserted by earlier
   filters.
 * The final backend service (i.e., the service where the request will
   ultimately be routed) will only have access to inserted headers if
   they are listed in `allowed_authorization_headers` in the
   Ambassador Edge Stack annotation.
 * Filter processing is aborted by the first filter to return a
   non-200 status.

### `FilterPolicy` Example

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

**Note:** Ambassador Edge Stack will choose the first `FilterPolicy` rule that matches the incoming request. As in the above example, you must list your rules in the order of least to most generic.

## Installing self-signed certificates

The `JWT` and `OAuth2` filters speak to other servers over HTTP or HTTPS.  If those servers are configured to speak HTTPS using a
self-signed certificate, attempting to talk to them will result in an error mentioning `ERR x509: certificate signed by unknown authority`. You can fix this by installing that self-signed certificate in to the
Pro container following the standard procedure for Alpine Linux 3.8: Copy the certificate to `/usr/local/share/ca-certificates/` and then run `update-ca-certificates`.  Note that the `amb-sidecar` image sets `USER 1000`, but that `update-ca-certificates` needs to be run as root.

```Dockerfile
FROM quay.io/datawire/ambassador_pro:amb-sidecar-%aproVersion%
USER root
COPY ./my-certificate.pem /usr/local/share/ca-certificates/my-certificate.crt
RUN update-ca-certificates
USER 1000
```

When deploying Ambassador Pro, refer to that custom Docker image,
rather than to `quay.io/datawire/ambassador_pro:amb-sidecar-%aproVersion%`.


