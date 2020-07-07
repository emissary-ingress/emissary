# Filter Type: `JWT`

The `JWT` filter type performs JWT validation on a [Bearer token] present in the HTTP header.  If the Bearer token JWT doesn't validate, or has insufficient scope, an RFC 6750-complaint error response with a `WWW-Authenticate` header is returned.  The list of acceptable signing keys is loaded from a JWK Set that is loaded over HTTP, as specified in `jwksURI`.  Only RSA and `none` algorithms are supported.

[Bearer token]: https://tools.ietf.org/html/rfc6750

## `JWT` Global Arguments

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: "example-jwt-filter"
  namespace: "example-namespace"
spec:
  JWT:
    jwksURI:            "url-string"  # required, unless the only validAlgorithm is "none"
    insecureTLS:        bool          # optional; default is false
    renegotiateTLS:     "enum-string" # optional; default is "never"
    validAlgorithms:                  # optional; default is "all supported algos except for 'none'"
    - "RS256"
    - "RS384"
    - "RS512"
    - "none"

    audience:           "string"      # optional, unless `requireAudience: true`
    requireAudience:    bool          # optional; default is false

    issuer:             "url-string"  # optional, unless `requireIssuer: true`
    requireIssuer:      bool          # optional; default is false

    requireExpiresAt:   bool          # optional; default is false
    leewayForExpiresAt: "duration"    # optional; default is "0"

    requireNotBefore:   bool          # optional; default is false
    leewayForNotBefore: "duration"    # optional; default is "0"

    requireIssuedAt:    bool          # optional; default is false
    leewayForIssuedAt:  "duration"    # optional; default is "0"

    injectRequestHeaders:             # optional; default is []
    - name:   "header-name-string"      # required
      value:  "go-template-string"      # required
       
    errorResponse:                    # optional
      contentType: "string"             # deprecated; use 'headers' instead
      realm: "string"                   # optional; default is "{{.metadata.name}}.{{.metadata.namespace}}"
      headers:                          # optional; default is [{name: "Content-Type", value: "application/json"}]
      - name: "header-name-string"        # required
        value: "go-template-string"       # required
      bodyTemplate: "string"            # optional; default is `{{ . | json "" }}`
```

 - `insecureTLS` disables TLS verification for the cases when `jwksURI` begins with `https://`.  This is discouraged in favor of either using plain `http://` or [installing a self-signed certificate](#installing-self-signed-certificates).
 - `renegotiateTLS` allows a remote server to request TLS renegotiation. Accepted values are "never", "onceAsClient", and "freelyAsClient".
 - `leewayForExpiresAt` allows tokens expired by this much to be used;
   to account for clock skew and network latency between the HTTP
   client and the Ambassador Edge Stack.
 - `leewayForNotBefore` allows tokens that shouldn't be used until
   this much in the future to be used; to account for clock skew
   between the HTTP client and the Ambassador Edge Stack.
 - `leewayForIssuedAt` allows tokens issued this much in the future to
   be used; to account for clock skew between the HTTP client and
   the Ambassador Edge Stack.
 - `injectRequestHeaders` injects HTTP header fields in to the request before sending it to the upstream service; where the header value can be set based on the JWT value.  The value is specified as a [Go `text/template`][] string, with the following data made available to it:

    * `.token.Raw` → `string` the raw JWT
    * `.token.Header` → `map[string]interface{}` the JWT header, as parsed JSON
    * `.token.Claims` → `map[string]interface{}` the JWT claims, as parsed JSON
    * `.token.Signature` → `string` the token signature
    * `.httpRequestHeader` → [`http.Header`][] a copy of the header of the incoming HTTP request.  Any changes to `.httpRequestHeader` (such as by using using `.httpRequestHeader.Set`) have no effect.  It is recommended to use `.httpRequestHeader.Get` instead of treating it as a map, in order to handle capitalization correctly.

   Also available to the template are the [standard functions available
   to Go `text/template`s][Go `text/template` functions], as well as:

    * a `hasKey` function that takes the a string-indexed map as arg1,
      and returns whether it contains the key arg2.  (This is the same
      as the [Sprig function of the same name][Sprig `hasKey`].)

    * a `doNotSet` function that causes the result of the template to
      be discarded, and the header field to not be adjusted.  This is
      useful for only conditionally setting a header field; rather
      than setting it to an empty string or `"<no value>"`.  Note that
      this does _not_ unset an existing header field of the same name;
      in order to prevent the untrusted client from being able to
      spoof these headers, use a [Lua script][Lua Scripts] to remove
      the client-supplied value before the Filter runs.  See below for
      an example.  Not sanitizing the headers first is a potential
      security vulnerability.

   Any headers listed will override (not append to) the original request header with that name.
 - `errorResponse` allows templating the error response, overriding the default json error format.  Make sure you validate and test your template, not to generate server-side errors on top of client errors.
    * `contentType` is deprecated, and is equivalent to including a
      `name: "Content-Type"` item in `headers`.
    * `realm` allows specifying the realm to report in the `WWW-Authenticate` response header.
    * `headers` sets extra HTTP header fields in the error response. The value is specified as a [Go `text/template`][] string, with the same data made available to it as `bodyTemplate` (below). It does not have access to the `json` function.
    * `bodyTemplate` specifies body of the error; specified as a [Go `text/template`][] string, with the following data made available to it:

       * `.status_code` → `integer` the HTTP status code to be returned
       * `.httpStatus` → `integer` an alias for `.status_code` (hidden from `{{ . | json "" }}`)
       * `.message` → `string` the error message string
       * `.error` → `error` the raw Go `error` object that generated `.message` (hidden from `{{ . | json "" }}`)
       * `.error.ValidationError` → [`jwt.ValidationError`][] the JWT validation error, will be `nil` if the error is not purely JWT validation (insufficient scope, malformed or missing `Authorization` header)
       * `.request_id` → `string` the Envoy request ID, for correlation (hidden from `{{ . | json "" }}` unless `.status_code` is in the 5XX range)
       * `.requestId` → `string` an alias for `.request_id` (hidden from `{{ . | json "" }}`)

      Also availabe to the template are the [standard functions
      available to Go `text/template`s][Go `text/template` functions],
      as well as:

       * a `json` function that formats arg2 as JSON, using the arg1
         string as the starting indentation.  For example, the
         template `{{ json "indent>" "value" }}` would yield the
         string `indent>"value"`.

`"duration"` strings are parsed as a sequence of decimal numbers, each
with optional fraction and a unit suffix, such as "300ms", "-1.5h" or
"2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m",
"h".  See [Go `time.ParseDuration`][].

**Note**: If you are using a templating system for your YAML that also makes use of Go templating, then you will need to escape the template strings meant to be interpreted by the Ambassador Edge Stack.

[Go `time.ParseDuration`]: https://golang.org/pkg/time/#ParseDuration
[Go `text/template`]: https://golang.org/pkg/text/template/
[Go `text/template` functions]: https://golang.org/pkg/text/template/#hdr-Functions
[`http.Header`]: https://golang.org/pkg/net/http/#Header
[`jwt.ValidationError`]: https://godoc.org/github.com/dgrijalva/jwt-go#ValidationError
[Lua Scripts]: /docs/latest/topics/running/ambassador/#lua-scripts-lua_scripts
[Sprig `hasKey`]: https://masterminds.github.io/sprig/dicts.html#haskey

## `JWT` Path-Specific Arguments

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

 - `scope`: A list of OAuth scope values that Ambassador will require to be listed in the [`scope` claim][].  In addition to the normal of the `scope` claim (a JSON string containing a space-separated list of values), the JWT Filter also accepts a JSON array of values.

[`scope` claim]: https://tools.ietf.org/html/draft-ietf-oauth-token-exchange-19#section-4.2

## Example `JWT` `Filter`

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
      - name: "X-Token-C-Optional-Empty"
        value: "{{ .token.Claims.optional }}"
        # result will be "<no value>"; the header field will be set
        # even if the "optional" claim is not set in the JWT.
      - name: "X-Token-C-Optional-Unset"
        value: "{{ if hasKey .token.Claims \"optional\" | not }}{{ doNotSet }}{{ end }}{{ .token.Claims.optional }}"
        # Similar to "X-Token-C-Optional-Empty" above, but if the
        # "optional" claim is not set in the JWT, then the header
        # field won't be set either.
        #
        # Note that this does NOT remove/overwrite a client-supplied
        # header of the same name.  In order to distrust
        # client-supplied headers, you MUST use a Lua script to
        # remove the field before the Filter runs (see below).
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
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    lua_scripts: |
      function envoy_on_request(request_handle)
        request_handle:headers():remove("x-token-c-optional-unset")
      end
```
