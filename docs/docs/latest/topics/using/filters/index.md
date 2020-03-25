# Filters and Authentication

Filters are used to extend the Ambassador Edge Stack to modify or intercept a request before sending to your backend service. The most common use case for Filters is authentication, and Edge Stack includes a number of built-in filters for this purpose. Edge Stack also supports developing custom filters.

Filters are managed using a `FilterPolicy` resource. The `FilterPolicy` resource specifies a particular host or URL to match, along with a set of filters to run when an request matches the host/URL.

## Filter Types

Edge Stack supports the following filter types:

* [`JWT`](jwt), which validates JSON Web Tokens
* [`OAuth2`](oauth2), which performs OAuth2 authorization against an identity provider implementing [OIDC Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html).
* [`Plugin`](plugin), which allows users to write custom Filters in Go that run as part of the Edge Stack container
* [`External`](external), which allows users to call out to other services for request processing. This can include both custom services (in any language) or third party services.

## Managing Filters

Filters are created with the `Filter` resource type, which contains global arguments to that filter.  Which Filter(s) to use for which HTTP requests is then configured in `FilterPolicy` resources, which may contain path-specific arguments to the filter.

### `Filter` Definition

Filters are created as `Filter` resources.  The body of the resource spec depends on the filter type:

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

### `FilterPolicy` Definition

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
        name: "string"              # required
        negate: bool                # optional; default is false
        # It is invalid to specify both "value" and "valueRegex".
        value: "string"             # optional; default is any non-empty string
        valueRegex: "regex-string"  # optional; default is any non-empty string
      onDeny: "enum-string"       # optional; default is "break"
      onAllow: "enum-string"      # optional; default is "continue"
      arguments: DEPENDS          # optional
```

Rule configuration values include:

| Value     | Example    | Description |
| -----     | -------    | -----------                  |
| `host`    | `*`, `foo.com` | the Host that a given rule should match |
| `path`    | `/foo/url/`    | the URL path that a given rule should match to |
| `filters`  | `name: keycloak`       | the name of a given filter to be applied|

The wildcard `*` is supported for both `path` and `host`.

The type of the `arguments` property is dependent on which Filter type is being referred to; see the "Path-Specific Arguments" documentation for each Filter type.

When multiple `Filter`s are specified in a rule:

 * The filters are gone through in order
 * Each filter may either
   1. return a direct HTTP *response*, intended to be sent back to the requesting HTTP client (normally *denying* the request from
      being forwarded to the upstream service); or
   2. return a modification to make to the HTTP *request* before sending it to other filters or the upstream service (normally *allowing* the request to be forwarded to the upstream service with modifications).
 * If a filter has an `ifRequestHeader` setting, the filter is skipped
   unless the request (including any modifications made by earlier
   filters) has the HTTP header field `name` (case-insensitive) either
   set to (if `negate: false`) or not set to (if `negate: true`)
    + a non-emtpy string if neither `value` nor `valueRegex` are set
    + the exact string `value` (case-sensitive) (if `value` is set)
    + a string that matches the regular expression `valueRegex` (if
      `valueRegex` is set).  This uses [RE2][] syntax (always, not
      obeying [`regex_type`][] in the Ambassador module) but does not
      support the `\C` escape sequence.
 * `onDeny` identifies what to do when the filter returns an "HTTP response":
   - `"break"`: End processing, and return the response directly to
     the requesting HTTP client.  Later filters are not called.  The request is not forwarded to the upstream service.
   - `"continue"`: Continue processing.  The request is passed to the next filter listed; or if at the end of the list, it is forwarded
   - `"continue"`: Continue processing.  The request is passed to the
     next filter listed; or if at the end of the list, it is forwarded to the upstream service.  The HTTP response returned from the filter is discarded.
 * `onAllow` identifies what to do when the filter returns a
   "modification to the HTTP request":
   - `"break"`: Apply the modification to the request, then end filter processing, and forward the modified request to the upstream service.  Later filters are not called.
   - `"continue"`: Continue processing.  Apply the request modification, then pass the modified request to the next filter
     listed; or if at the end of the list, forward it to the upstream service.
 * Modifications to the request are cumulative; later filters have access to _all_ headers inserted by earlier filters.

#### `FilterPolicy` Example

In the example below, the `param-filter` Filter Plugin is loaded, and configured to run on requests to `/httpbin/`.

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

#### Multiple Domains

In this example, the `foo-keycloak` filter is used for requests to `foo.bar.com`, while the `example-auth0` filter is used for requests to `example.com`. This configuration is useful if you are hosting multiple domains in the same cluster.

```yaml
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
  name: multi-domain-policy
spec:
  rules:
  - host: foo.bar.com
    path: *
    filters:
      - name: foo-keycloak
  - host: example.com
    path: *
    filters:
      - name: example-auth0
```

## Installing self-signed certificates

The `JWT` and `OAuth2` filters speak to other services over HTTP or HTTPS.  If those services are configured to speak HTTPS using a self-signed certificate, attempting to talk to them will result in an error mentioning `ERR x509: certificate signed by unknown authority`. You can fix this by installing that self-signed certificate into the AES container following the standard procedure for Alpine Linux 3.8: Copy the certificate to `/usr/local/share/ca-certificates/` and then run `update-ca-certificates`.  Note that the `aes` image sets `USER 1000`, but that `update-ca-certificates` needs to be run as root.

```Dockerfile
FROM quay.io/datawire/aes:$version$
USER root
COPY ./my-certificate.pem /usr/local/share/ca-certificates/my-certificate.crt
RUN update-ca-certificates
USER 1000
```

When deploying the Ambassador Edge Stack, refer to that custom Docker image, rather than to `quay.io/datawire/aes:$version$`
