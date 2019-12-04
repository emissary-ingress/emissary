# `FilterPolicy` Definition

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
    filters:                    # optional; omit or set to `null` to apply no filters to this request
    - name: "string"              # required
      namespace: "string"         # optional; default is the same namespace as the FilterPolicy
      ifRequestHeader:            # optional; default to apply this filter to all requests matching the host & path
        name: "string"              # required
        value: "string"             # optional; default is any non-empty string
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
   2. return a modification to make the the HTTP *request* before
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
     the requesitng HTTP client.  Later filters are not called.  The
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

**Note:** Ambassador Edge Stack will choose the first `FilterPolicy` rule that matches the incoming request. As in the above example, you must list your rules in the order of least to most generic.