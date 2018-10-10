# Cross-Origin Resource Sharing

Cross-Origin resource sharing lets users request resources (e.g., images, fonts, videos) from domains outside the original domain.

CORS configuration can be set for all Ambassador mappings in the [ambassador](https://www.getambassador.io/reference/modules#the-ambassador-module) module, or set per [mapping](https://www.getambassador.io/reference/mappings#configuring-mappings).

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
      header:
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
## [AuthService](services/auth-service) and Cross-Origin Resource Sharing

When you use external authorization, each incoming request is authenticated before routing to its destination, including pre-flight `OPTIONS` requests.  
If your `AuthService` implementation wants to deal with CORS itself, by default it will deny these requests, so you have to teach it to accept anything, because you implement CORS on a different level.

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
