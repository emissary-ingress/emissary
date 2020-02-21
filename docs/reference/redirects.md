# Redirects

## Host Redirect

To effect an HTTP 301 `Redirect`, the `Mapping` **must** set `host_redirect` to `true`, with `service` set to the host to which the client should be redirected:

```yaml
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  redirect
spec:
  prefix: /redirect/
  service: httpbin.org
  host_redirect: true
```

Using this `Mapping`, a request to `http://$AMBASSADOR_URL/redirect/` will result in an HTTP 301 `Redirect` to `http://httpbin.org/redirect/`.

The `Mapping` **may** also set `path_redirect` to change the path portion of the URL during the redirect:

```yaml
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  redirect
spec:
  prefix: /redirect/
  service: httpbin.org
  host_redirect: true
  path_redirect: /ip
```

Here, a request to `http://$AMBASSADOR_URL/redirect/` will result in an HTTP 301 `Redirect` to `http://httpbin.org/ip`. As always with Ambassador Edge Stack, attention paid to the trailing `/` on a URL is helpful!

## X-FORWARDED-PROTO Redirect

In cases when TLS is being terminated at an external layer 7 load balancer, then you would want to redirect only the originating HTTP requests to HTTPS, and let the originating HTTPS requests pass through.

This distinction between an originating HTTP request and an originating HTTPS request is done based on the `X-FORWARDED-PROTO` header that the external layer 7 load balancer adds to every request it forwards after TLS termination.

To enable this `X-FORWARDED-PROTO` based HTTP to HTTPS redirection, add a `x_forwarded_proto_redirect: true` field to `ambassador Module`'s configuration. Note that when this feature is enabled `use_remote_address` MUST be set to false.

An example configuration is as follows -

```yaml
---
apiVersion: getambassador.io/v2
kind:  Module
metadata:
  name:  ambassador
spec:
  config:
    x_forwarded_proto_redirect: true
    use_remote_address: false
```

Note: Setting `x_forwarded_proto_redirect: true` will impact all your Ambassador Edge Stack mappings. Every HTTP request to Ambassador Edge Stack will only be allowed to pass if it has an `X-FORWARDED-PROTO: https` header.
