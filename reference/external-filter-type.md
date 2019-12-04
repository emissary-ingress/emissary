# Filter Type: `External`

The `External` filter type exposes the Ambassador Edge Stack `AuthService` interface to external authentication services. This is useful in a number of situations, e.g., if you have already written a custom `AuthService`, but also want to use other filters.

The `External` filter looks very similar to an `AuthService` annotation:

```yaml
---
apiVersion: getambassador.io/v2
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
   external filter, in milliseconds.
 - `proto` is either `"http"` or `"grpc"`.

This `spec.External` is mostly identical to an [`AuthService`](/reference/services/auth-service), with the following exceptions:

* It does not contain the `apiVersion` field
* It does not contain the `kind` field
* It does not contain the `name` field
* In an `AuthService`, the `tls` field may either be a Boolean, or a string referring to a TLS context. In an `External`, it may only be a Boolean; referring to a TLS context is not supported.