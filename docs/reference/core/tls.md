## Transport Layer Security (TLS)

Ambassador supports both terminating TLS and originating TLS. By default, Ambassador will enable TLS termination whenever it finds valid TLS certificates stored in the `ambassador-certs` Kubernetes secret. The `tls` module defines system-wide configuration for TLS when additional configuration is needed.

The most common case requiring a `tls` module is redirecting cleartext traffic on port 80 to HTTPS on port 443, which can be done with the following configuration:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  tls
config:
  server:
    enabled: True
    redirect_cleartext_from: 80
```

### X-FORWARDED-PROTO Redirect

In cases when TLS is being terminated at an external layer 7 load balancer, then you would want to redirect only the originating HTTP requests to HTTPS, and let the originating HTTPS requests pass through.

This distinction between an originating HTTP request and an originating HTTPS request is done based on the `X-FORWARDED-PROTO` header that the external layer 7 load balancer adds to every request it forwards after TLS termination.

To enable this `X-FORWARDED-PROTO` based HTTP to HTTPS redirection, add a `x_forwarded_proto_redirect: true` field to ambassador module's configuration.

An example configuration is as follows -

```yaml
apiVersion: ambassador/v0
kind: Module
name: ambassador
config:
  x_forwarded_proto_redirect: true
```

Note: Setting `x_forwarded_proto_redirect: true` will impact all your Ambassador mappings. Every HTTP request to Ambassador will only be allowed to pass if it has an `X-FORWARDED-PROTO: https` header.

### More reading

TLS configuration is examined in more detail in the documentation on [TLS termination](/user-guide/tls-termination.md) and [TLS client certificate authentication](/reference/auth-tls-certs).

