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
       
    errorResponse:                  # optional
      contentType: "string"         # deprecated; use 'headers' instead
      realm: "string"               # optional; default is "{{.metadata.name}}.{{.metadata.namespace}}"
      headers:                      # optional; default is [{name: "Content-Type", value: "application/json"}]
      - name: "header-name-string"  # required
        value: "go-template-string" # required
      bodyTemplate: "string"        # optional; default is `{{ . | json "" }}`
```

 - `insecureTLS` disables TLS verification for the cases when `jwksURI` begins with `https://`.  This is discouraged in favor of either using plain `http://` or [installing a self-signed certificate](#installing-self-signed-certificates).
 - `renegotiateTLS` allows a remote server to request TLS renegotiation. Accepted values are "never", "onceAsClient", and "freelyAsClient".
 - `injectRequestHeaders` injects HTTP header fields in to the request before sending it to the upstream service; where the header value can be set based on the JWT value.  The value is specified as a [Go`text/template`][] string, with the following data made available to it:

    * `.token.Raw` → `string` the raw JWT
    * `.token.Header` → `map[string]interface{}` the JWT header, as parsed JSON
    * `.token.Claims` → `map[string]interface{}` the JWT claims, as parsed JSON
    * `.token.Signature` → `string` the token signature
    * `.httpRequestHeader` → [`http.Header`][] a copy of the header of the incoming HTTP request.  Any changes to `.httpRequestHeader` (such as by using using `.httpRequestHeader.Set`) have no effect.  It is recommended to use `.httpRequestHeader.Get` instead of treating it as a map, in order to handle capitalization correctly.

   Any headers listed will override (not append to) the original request header with that name.
 - `errorResponse` allows templating the error response, overriding the default json error format.  Make sure you validate and test your template, not to generate server-side errors on top of client errors.
    * `contentType` is deprecated, and is equivalent to including a
      `name: "Content-Type"` item in `headers`.
    * `realm` allows specifying the realm to report in the `WWW-Authenticate` response header.
    * `headers` sets extra HTTP header fields in the error response. The value is specified as a [Go `text/template`][] string, with the same data made available to it as `bodyTemplate` (below). It does not have access to the `json` function.
    * `bodyTemplate` specifies body of the error; specified as a [Go`text/template`][] string, with the following data made available to it:

       * `.status_code` → `integer` the HTTP status code to be returned
       * `.httpStatus` → `integer` an alias for `.status_code` (hidden from `{{ . | json "" }}`)
       * `.message` → `string` the error message string
       * `.error` → `error` the raw Go `error` object that generated `.message` (hidden from `{{ . | json "" }}`)
       * `.error.ValidationError` → [`jwt.ValidationError`][] the JWT validation error, will be `nil` if the error is not purely JWT validation (insufficient scope, malformed or missing `Authorization` header)
       * `.request_id` → `string` the Envoy request ID, for correlation (hidden from `{{ . | json "" }}` unless `.status_code` is in the 5XX range)
       * `.requestId` → `string` an alias for `.request_id` (hidden from `{{ . | json "" }}`)

      In addition to the [standard functions available to Go `text/template`s][Go `text/template` functions], there is a `json` function that arg2 as JSON, using the arg1 string as the starting indent level.

**Note**: If you are using a templating system for your YAML that also makes use of Go templating, then you will need to escape the template strings meant to be interpreted by the Ambassador Edge Stack.

[Go `text/template`]: https://golang.org/pkg/text/template/
[Go `text/template` functions]: https://golang.org/pkg/text/template/#hdr-Functions
[`http.Header`]: https://golang.org/pkg/net/http/#Header
[`jwt.ValidationError`]: https://godoc.org/github.com/dgrijalva/jwt-go#ValidationError

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
        scopes:                    # optional; default is []
        - "scope-value-1"
        - "scope-value-2"
```

 - `scopes`: A list of OAuth scope values that Ambassador will require to be listed in the [`scope` claim][].  In addition to the normal of the `scope` claim (a JSON string containing a space-separated list of values), the JWT Filter also accepts a JSON array of values.

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
