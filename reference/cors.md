# Cross-Origin Resource Sharing

Cross-Origin resource sharing lets users request resources (e.g., images, fonts, videos) from domains outside the original domain.

CORS configuration can be set for all Ambassador Edge Stack mappings in the [`ambassador Module`](../core/ambassador), or set per [`Mapping`](../mappings#configuring-mappings).

When the CORS attribute is set at either the `Mapping` or `Module` level, Ambassador Edge Stack will intercept the pre-flight `OPTIONS` request and respond with the appropriate CORS headers. This means you will not need to implement any logic in your upstreams to handle these CORS `OPTIONS` requests.

The flow of the request will look similar to the following:
```
Client      Ambassador Edge Stack     Upstream
  |      OPTIONS       |               |
  | —————————————————> |               |
  |     CORS_RESP      |               |
  | <————————————————— |               | 
  |      GET /foo/     |               |
  | —————————————————> | ————————————> |
  |                    |      RESP     |
  | <————————————————————————————————— |
```
## The `cors` attribute

The `cors` attribute enables the CORS filter. The following settings are supported:

- `origins`: Specifies a list of allowed domains for the `Access-Control-Allow-Origin` header. To allow all origins, use the wildcard `"*"` value. Format can be either of:
    - comma-separated list, e.g.
      ```yaml
      origins: http://foo.example,http://bar.example
      ```
    - YAML array, e.g.
      ```yaml
      origins:
      - http://foo.example
      - http://bar.example
      ```
- `methods`: if present, specifies a list of allowed methods for the `Access-Control-Allow-Methods` header. Format can be either of:
    - comma-separated list, e.g.
      ```yaml
      methods: POST, GET, OPTIONS
      ```
    - YAML array, e.g.
      ```yaml
      methods:
      - GET
      - POST
      - OPTIONS
      ```
- `headers`: if present, specifies a list of allowed headers for the `Access-Control-Allow-Headers` header. Format can be either of:
    - comma-separated list, e.g.
      ```yaml
      headers: Content-Type
      ```
    - YAML array, e.g.
      ```yaml
      headers:
      - Content-Type
      ```
- `credentials`: if present with a true value (boolean), will send a `true` value for the `Access-Control-Allow-Credentials` header.
- `exposed_headers`: if present, specifies a list of allowed headers for the `Access-Control-Expose-Headers` header. Format can be either of:
    - comma-separated list, e.g.
      ```yaml
      exposed_headers: X-Custom-Header
      ```
    - YAML array, e.g.
      ```yaml
      exposed_headers:
      - X-Custom-Header
      ```
- `max_age`: if present, indicated how long the results of the preflight request can be cached, in seconds. This value must be a string.

## Example

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  cors
spec:
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
## [AuthService](../services/auth-service) and Cross-Origin Resource Sharing

When you use external authorization, each incoming request is authenticated before routing to its destination, including pre-flight `OPTIONS` requests.  

By default, many [`AuthService`](../services/auth-service) implementations will deny these requests. If this is the case, you will need to add some logic to your `AuthService` to accept all CORS headers.

For example, a possible configuration for Spring Boot 2.0.1: 
```java
@EnableWebSecurity
class SecurityConfig extends WebSecurityConfigurerAdapter {

    public void configure(final HttpSecurity http) throws Exception {
        http
            .cors().configurationSource(new PermissiveCorsConfigurationSource()).and()
            .csrf().disable()
            .authorizeRequests()
                .antMatchers("**").permitAll();
    }

    private static class PermissiveCorsConfigurationSource implements CorsConfigurationSource {
        /**
         * Return a {@link CorsConfiguration} based on the incoming request.
         *
         * @param request
         * @return the associated {@link CorsConfiguration}, or {@code null} if none
         */
        @Override
        public CorsConfiguration getCorsConfiguration(final HttpServletRequest request) {
            final CorsConfiguration configuration = new CorsConfiguration();
            configuration.setAllowCredentials(true);
            configuration.setAllowedHeaders(Collections.singletonList("*"));
            configuration.setAllowedMethods(Collections.singletonList("*"));
            configuration.setAllowedOrigins(Collections.singletonList("*"));
            return configuration;
        }
    }
}
```

This is okay since CORS is being handled by Ambassador Edge Stack after authentication.

The flow of this request will look similar to the following:

```
Client     Ambassador Edge Stack       Auth          Upstream
  |      OPTIONS       |               |               |
  | —————————————————> | ————————————> |               |
  |                    | CORS_ACCEPT_* |               |
  |     CORS_RESP      |<——————————————|               |
  | <——————————————————|               |               |
  |      GET /foo/     |               |               |
  | —————————————————> | ————————————> |               |
  |                    | AUTH_RESP     |               |
  |                    | <———————————— |               |
  |                    |   AUTH_ALLOW  |               |
  |                    | ————————————————————————————> |
  |                    |               |     RESP      |
  | <————————————————————————————————————————————————— |
  ```
