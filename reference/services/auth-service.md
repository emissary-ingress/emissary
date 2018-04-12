## AuthService

An `AuthService` manifest configures Ambassador to use an external service to check authentication and authorization for incoming requests:

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
