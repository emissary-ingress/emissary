Custom error responses are configured on the Ambassador module. Here is a basic example:
```
  apiVersion: getambassador.io/v1
  kind: Module
  metadata:
    name: ambassador
    namespace: ambassador
  spec:
    config:
      error_response_overrides:
      - on_status_code: 404
        body:
          text_format: 'file not found'
      - on_status_code: 500
        body:
          json_format:
            error: 'application error'
            status: '%RESPONSE_CODE%'
            cluster: '%UPSTREAM_CLUSTER%'
```

The Ambassador module's error response overrides can be disabled by setting `bypass_external_response_overrides: true` on a Mapping. Example:
```
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: quote-backend
  namespace: ambassador
spec:
  prefix: /api/
  service: quote
  bypass_error_response_overrides: true
```
This is useful if a portion of the domain services an API whose errors should not be rewritten, and the rest of the domain serves a UI that should contain custom errors.

Configuration schema:

`error_response_overrides` -> Array of `ErrorResponseOverride`

ErrorResponseOverride schema:

(required) `on_status_code` -> HTTP status code to match for this rewrite rule. Must be >= 400 and < 600.
(required) `body` {
    (optional) `content_type`: The new content type to set on the response. Must be a string. If not set, the content type is dependent on the format used.
    (oneof) {
        `text_format` -> A string whose value will be used as the new response body. Sets the content type to application/plan, unless overriden by content_type.
        `json_format` -> A config object whose keys and values will be serialized as json and used as the new response body. Sets the content type to application/json.
        `text_format_source` -> {
            (required) `filename`: A path on the Ambassador pod whose contents will be used as the new response body.
        }
    }
 }

Only one of `text_format`, `json_format`, or `text_format_source` may be provided. If `text_format_source` is provided, `filename` must be set and the resulting file must exist on the Ambassador pod. `on_status_code` is required.

The string in text_format, the body of the file in text_format_sourcel.filename, and the values in the json_format object are all subject to Envoy's AccessLog substitution syntax and the variables that come with it. See https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log/usage#config-access-log-format-strings for more.

Known caveats:
* The `text_format` and `text_format_source` strings perform no string escaping on expanded variables. This may break the structural integrity of your response body if, for example, the variable contains HTML data and the response content type is text/html. Be careful when using variables in this way, and consider whether the value may be coming from an untrusted source, like request or response headers.
* The `json_format` field does not support sourcing from a file. For large json responses, use `text_format_source` with a `filename` whose contents are structurally json, and set content_type to application/json. Note that if you use expanded variables, you must consider the fact that no escaping will happen and use caution when expanding variables whose values come from an untrusted source, like request or response headers.

The `text_format` and `json_format` fields can be used to conveniently configure small response bodies. For large custom response bodies, it is recommended to use `text_format_source` with a `filename` to a path on the Ambassador pod where the error response data is stored. Use `content_type` to specify the content type, eg `text/html`, if the content type should be something other than `text/plain`.

Example:
```
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ambassador-errorpages
  namespace: ambassador
data:
  404.html: |
    <html>
      <h1>
        Hey...
      </h1>
      <p>
      It looks like there's no route (details: <b>"%RESPONSE_CODE_DETAILS%"</b>)
      </p>
    </html>
```

Configured using Ambassador Module:
```
  apiVersion: getambassador.io/v1
  kind: Module
  metadata:
    name: ambassador
    namespace: ambassador
  spec:
    config:
      error_response_overrides:
      - on_status_code: 404
        body:
          content_type: 'text/html'
          text_format_source:
            filename: /ambassador/ambassador-errorpages/404.html'
```

With the Ambassador deployment patched to contain a volume for the ConfigMap's data: Please note the follow YAML is in patch format and does not represent the entire Deployment spec. See https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/ for more on kubectl patches.
```
spec:
  template:
    spec:
      containers:
      - name: aes
        volumeMounts:
          - name: ambassador-errorpages
            mountPath: /ambassador/errorpages
      volumes:
      - name: ambassador-errorpages
        configMap:
          name: ambassador-errorpages
```
