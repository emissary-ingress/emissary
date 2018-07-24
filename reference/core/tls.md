## Transport Layer Security (TLS)

Ambassador supports both terminating TLS and originating TLS. By default, Ambassador will enable TLS termination whenever it finds valid TLS certificates stored in the `ambassador-certs` Kubernetes secret. 

### The `tls` module

The `tls` module defines system-wide configuration for TLS when additional configuration is needed.

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  tls
config:
  # The 'server' block configures TLS termination. 'enabled' is the only
  # required element.
  server:
    # If 'enabled' is not True, TLS termination will not happen.
    enabled: True

    # If you set 'redirect_cleartext_from' to a port number, HTTP traffic 
    # to that port will be redirected to HTTPS traffic. Typically you would
    # use port 80, of course.
    # redirect_cleartext_from: 80

    # These are optional. They should not be present unless you are using
    # a custom Docker build to install certificates onto the container
    # filesystem, in which case YOU WILL STILL NEED TO SET enabled: True
    # above.
    #
    # cert_chain_file: /etc/certs/tls.crt   # remember to set enabled!
    # private_key_file: /etc/certs/tls.key  # remember to set enabled!

    # Enable TLS ALPN protocol, typically HTTP2 to negotiate it with 
    # HTTP2 clients over TLS.
    # This must be set to be able to use grpc over TLS.
    # alpn_protocols: h2

  # The 'client' block configures TLS client-certificate authentication.
  # 'enabled' is the only required element.
  client:
    # If 'enabled' is not True, TLS client-certificate authentication will
    # not happen.
    enabled: False

    # If 'cert_required' is True, TLS client certificates will be required
    # for every connection.
    # cert_required: False

    # This is optional. It should not be present unless you are using
    # a custom Docker build to install certificates onto the container
    # filesystem, in which case YOU WILL STILL NEED TO SET enabled: True
    # above.
    # 
    # cacert_chain_file: /etc/cacert/tls.crt  # remember to set enabled!
```

### Redirecting from cleartext to TLS

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

To enable this `X-FORWARDED-PROTO` based HTTP to HTTPS redirection, add a `x_forwarded_proto_redirect: true` field to the Ambassador module configuration, e.g.,

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

