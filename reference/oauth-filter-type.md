# Filter Type: OAuth2

The `OAuth2` filter type performs OAuth2 authorization against an identity provider implementing [OIDC Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html).

## `OAuth2` Global Arguments

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
    accessTokenValidation: "enum-string"     # optional; default is "auto"

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
     UserInfo Endpoint.  This means that Ambassador Edge Stack must
     initiate an HTTP request to the identity provider for each
     authorized request to a protected resource.  This performs
     poorly, but functions properly with a wider range of identity
     providers.
   * `"auto"` attempts has it do `"jwt"` validation if the Access
     Token parses as a JWT and the signature is valid, and otherwise
     falls back to `"userinfo"` validation.

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
 - `stateTTL`: (You decide this) How long Ambassador Edge Stack will
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
   string if `value` is unset.  `ifRequestHeader` does nothing when
   `grantType: "ClientCredentials"`, because Ambassador will never
   redirect the User-Agent to the identity provider for the client
   credentials grant type.
