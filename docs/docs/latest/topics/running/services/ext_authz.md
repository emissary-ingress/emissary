## The `ext_authz` Protocol

By design, the `ext_authz` protocol used by [the `AuthService`](../auth-service) and by [`External` `Filters`](../../using/filters/) is highly flexible. The authentication service is the first external service invoked on an incoming request (e.g., it runs before the rate limit filter). Because the logic of authentication is encapsulated in an external service, you can use this to support a wide variety of use cases. For example:

* Supporting traditional SSO authentication protocols, e.g., OAuth, OpenID Connect, etc.
* Supporting HTTP basic authentication (sample implementation available [here](https://github.com/datawire/ambassador-auth-httpbasic)).
* Only authenticating requests that are under a rate limit, and rejecting authentication requests above the rate limit.
* Authenticating specific services (URLs), and not others.

For each request, the external auth service may either
 1. return a direct HTTP *response*, intended to be sent back to the requesting HTTP client (normally *denying* the request from being forwarded to the upstream backend service); or
 2. return a modification to make to the HTTP *request* before sending it to the upstream backend service (normally *allowing* the request to be forwarded to the upstream backend service with modifications).

The external auth service receives information about every request through Ambassador and must indicate whether the request is to be allowed, or not.  If not, the external auth service provides the HTTP response which is to be handed back to the client.  A potential control flow for Authentication is shown in the image below.

Giving the external auth service the ability to control the response allows many different types of auth mechanisms, for example:

- The external auth service can simply return an error page with an HTTP 401 response.
- The external auth service can choose to include a `WWW-Authenticate` header in the 401 response, to ask the client to perform HTTP Basic Auth.
- The external auth service can issue a 301 `Redirect` to divert the client into an OAuth or OIDC authentication sequence.  The control flow of this is shown below.  ![Authentication flow](../../../images/auth-flow.png)

There are two variants of the `ext_authz`: gRPC, and plain HTTP.

### The `proto: grpc` Protocol

When `proto: grpc`, the external auth service must implement the `Authorization` gRPC interface, defined in [Envoy's `external_auth.proto`][external_auth.proto].

[external_auth.proto]: https://github.com/datawire/ambassador/blob/master/api/envoy/service/auth/v2/external_auth.proto

### The `proto: http` Protocol

External services for `proto: http` are often easier to implement, but have several limitations, compared to `proto: grpc`:
 - The list of headers that the external auth service is interested in reading must be known ahead of time, in order to set `allow_request_headers`.  Inspecting headers that are not known ahead of time requires instead using `proto: grpc`.
 - The list of headers that the external auth service would like to set or modify must be known ahead of time, in order to set `allow_authorization_headers`.  Setting headers that are not known ahead of time requires instead using `proto: grpc`.
 - When returning a direct HTTP response, the HTTP status code cannot be 200 or in the 5XX range.  Intercepting with a 200 of 5XX response requires instead using `proto: grpc`.

#### The Request From Ambassador to the External Auth Service

For every incoming request, a similar request is made to the external auth service that mimics the:
 - HTTP request method
 - HTTP request path, potentially modified by `path_prefix`
 - HTTP request headers that are either named in `allowed_request_headers` or in the fixed list of headers that are always included
 - first `include_body.max_bytes` of the HTTP request body.

The `Content-Length` HTTP header is set to the number of bytes in the body of the request sent to the external auth service (`0` if `include_body` is not set).

**ALL** request methods will be proxied, which implies that the external auth service must be able to handle any request that any client could make.

So, for example, if the incoming request is

```
PUT /path/to/service HTTP/1.1
Host: myservice.example.com:8080
User-Agent: curl/7.54.0
Accept: */*
Content-Type: application/json
Content-Length: 27

{ "greeting": "hello world!", "spiders": "OMG no" }
```

then the request Ambassador will make of the auth service is:

```
PUT /path/to/service HTTP/1.1
Host: extauth.example.com:8080
User-Agent: curl/7.54.0
Accept: */*
Content-Type: application/json
Content-Length: 0
```

#### The Response Returned From the External Auth Service to Ambassador

 - If the HTTP response returned from the external auth service to Ambassador has an HTTP status code of 200, then the request is allowed through to the upstream backend service.  **Note well** that **only** 200 indicates this; other 2XX status codes will prevent the request from being allowed through.

   The 200 response should not contain anything in the body, but may contain arbitrary headers.  Any header present in the external auth service' response that is also either listed in the `allow_authorization_headers` attribute of the `AuthService` resource or in the fixed list of headers that are always included will be copied from the external auth service' response into the request going to the upstream backend service.  This allows the external auth service to inject tokens or other information into the request, or to modify headers coming from the client.

   The big limitation here is that the list of headers to be set must be known ahead of time, in order to set `allow_request_headers`.  Setting headers that are not known ahead of time requires instead using `proto: grpc`.

 - If Ambassador cannot reach the external auth service at all, if the external auth service does not return a valid HTTP response, or if the HTTP response has an HTTP status code in the 5XX range, then the communication with the external auth service is considered to have failed, and the `status_on_error` or `failure_mode_allow` behavior is triggered.

 - Any HTTP status code other than 200 or 5XX from the external auth service tells Ambassador to **not** allow the request to continue to the upstream backend service, but that the external auth service is instead intercepting the request.  The entire HTTP response from the external auth service--including the status code, the headers, and the body--is handed back to the client verbatim. This gives the external auth service **complete** control over the entire response presented to the client.

   The big limitation here is that you cannot directly return a 200 or 5XX response.  Intercepting with a 200 of 5XX response requires instead using `proto: grpc`.
