## Cross-Origin Resource Sharing

Cross-Origin resource sharing lets users request resources (e.g., images, fonts, videos) from domains outside the original domain. 

## The `cors` attribute

The `cors` attribute enables the CORS filter. The following settings are supported:

- `origins`: Specifies a comma-separated list of allowed domains for the `Access-Control-Allow-Origin` header. To allow all origins, use the wildcard `"*"` value.
- `methods`: if present, specifies a comma-separated list of allowed methods for the `Access-Control-Allow-Methods` header.
- `headers`: if present, specifies a comma-separated list of allowed headers for the `Access-Control-Allow-Headers` header.
- `credentials`: if present with a true value (boolean), will send a `true` value for the `Access-Control-Allow-Credentials` header.
- `exposed_headers`: if present, specifies a comma-separated list of allowed headers for the `Access-Control-Expose-Headers` header.
- `max_age`: if present, indicated how long the results of the preflight request can be cached, in seconds. This value must be a string.

## Example

```yaml
apiVersion: ambassador/v0
kind:  Mapping
name:  cors_mapping
prefix: /cors/
service: cors-example
cors:
  origins: http://foo.example,http://bar.example
  methods: POST, GET, OPTIONS
  headers: Content-Type
  credentials: true
  exposed_headers: X-Custom-Header
  max_age: "86400"
```
