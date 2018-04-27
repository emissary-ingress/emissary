## AuthService

An `AuthService` manifest configures Ambassador to use an external service to check authentication and authorization for incoming requests. Each incoming request is authenticated before routing to its destination.

```yaml
---
apiVersion: ambassador/v0
kind:  AuthService
name:  authentication
auth_service: "example-auth:3000"
path_prefix: "/extauth"
allowed_headers:
- "x-qotm-session"
```

- `auth_service` gives the URL of the authentication service
- `path_prefix` (optional) gives a prefix prepended to every request going to the auth service
- `allowed_headers` (optional) gives an array of headers that will be incorporated into the upstream request if the auth service supplies them.

You may use multiple `AuthService` manifests to round-robin authentication requests among multiple services. **Note well that all services must use the same `path_prefix` and `allowed_headers`;** if you try to have different values, you'll see an error in the diagnostics service, telling you which value is being used.

### AuthService and TLS

You can tell Ambassador to use TLS to talk to your service by using an `auth_service` with an `https://` prefix. However, you may also provide a `tls` attribute: if `tls` is present and `true`, Ambassador will originate TLS even if the `service` does not have the `https://` prefix.

If `tls` is present with a value that is not `true`, the value is assumed to be the name of a defined TLS context, which will determine the certificate presented to the upstream service. TLS context handling is a beta feature of Ambassador at present; please [contact us on Gitter](https://gitter.im/datawire/ambassador) if you need to specify TLS origination certificates.

### The External Authentication Service

When using an external auth service, the HTTP `method` and headers of every incoming request are forwarded to the auth service, with two changes:

1. The `Host` header is overwritten with the host information of the external auth service.
2. The body is removed.

So, for example, if the incoming request is 

```
PUT /path/to/service HTTP/1.1
Host: myservice.example.com:80
User-Agent: curl/7.54.0
Accept: */*
Content-Type: application/json
Content-Length: 27

{ "greeting": "hello world!", "spiders": "OMG no" }
```

then the request Ambassador will make of the auth service is:

```
PUT /path/to/service HTTP/1.1
Host: extauth.example.com:80
User-Agent: curl/7.54.0
Accept: */*
Content-Type: application/json
Content-Length: 0
```

**ALL** request methods will be proxied; the auth service should be able to handle any request that any client could make. If desired, Ambassador can add a prefix to the path before forwarding it to the auth service; see the example below.

If Ambassador cannot reach the auth service, it returns 503 to the client. If Ambassador receives any response from the auth service other than 200, it returns that full response (header and body) to the client. Ambassador assumes the auth service will return an appropriate response, such as 401.

If the auth service response code is 200, then Ambassador allows the client request to resume being processed by the normal Ambassador Envoy flow. This typically means that the client will receive the expected response to its request.

Additionally, Ambassador can be configured to allow headers from the auth service to be passed back to the client when the auth service returns 200; see the example below.

## Example

See [the Ambassador Authentication Tutorial](../../user-guide/auth-tutorial.md) for an example.
