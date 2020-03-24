# HTTP -> HTTPS Redirection

Most modern websites that force HTTPS will also automatically redirect any 
requests that come into it over HTTP. In the Ambassador Edge Stack, this is
enabled by default but can easily be enabled in any version of Ambassador.

```
Client              Ambassador Edge Stack
|                             |
| http://<hostname>/api       |
| --------------------------> |
| 301: https://<hostname>/api |
| <-------------------------- |
| https://<hostname>/api      |
| --------------------------> |
|                             |
```

In Ambassador, this is configured by setting the 
`insecure.action` in a `Host` to `Redirect`. 

```yaml
requestPolicy:
  insecure:
    action: Redirect
```

Ambassador will then enable cleartext redrection in two ways.

First, Ambassador will listen on the `insecure.additionalPort` and consider any
traffic on this port as `insecure` and redirect it to HTTPS. Ambassador will
default this to port `8080`.

```yaml
requestPolicy:
  insecure:
    action: Redirect
    additionalPort: 8080
```

Additionally, Ambassador will also check the `X-Forwarded-Proto` header of 
the incoming request on the `secure` port (`8443`)and issue a 301 redirect if 
it is set to `HTTP`.

The value of `X-Forwarded-Proto` is dependent on whatever is forwarding traffic
to Ambassador. A couple of options are

- Layer 4 Load Balancer, Proxy, or direct from the client:

   `X-Forwarded-Proto`  is not set or is untrusted. Envoy will set it based 
   off the protocol of the incoming request.

   If Envoy determines the request is encrypted, it will be set to `HTTPS`. If
   not, it will be set to `HTTP`.

- Layer 7 Load Balancer or Proxy:

   `X-Forwarded-Proto` is set by the load balancer or proxy and trusted by
   Envoy. Envoy will trust the value of `X-Forwarded-For` even if the request
   comes in over cleartext.

## tl;dr

The Ambassador Edge Stack will enabled cleartext redirection by default.

To enable cleartext redirection in any version of Ambassador, simply configure
a `Host` to redirect cleartext to HTTPS like below:

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: example-host
spec:
  hostname: host.example.com
  requestPolicy:
    insecure:
      action: Redirect     # Configures Ambassador to redirect cleartext
      additionalPort: 8080 # Optional: The redirect port. Defaults to 8080
```