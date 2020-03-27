# Cleartext Support

While most modern web applications will choose to encrypt all traffic, there
are reasons why you will want to support clients who access your website
without encryption in cleartext.

Ambassador supports both forcing 
[automatic redirection to HTTPS](#http---https-redirection) and 
[serving cleartext](#cleartext-routing) traffic on a `Host`.

**Note:** Currently you can only configure a single Ambassador to `Route` 
**or** `Redirect` cleartext. Future versions of Ambassador will allow this
to be configured on a per-`Host` basis.

## Cleartext Routing

Ambassador has full support for routing cleartext traffic to upstream services
for a `Host`.

### Only Cleartext

The default for the Open-Source Ambassador API Gateway is to serve cleartext on
port 8080 in the container. See [TLS documentation](../) for information on
how to configure TLS termination.

For the Ambassador Edge Stack, TLS termination is enabled by default with a
self-signed certificate or an ACME `Host`. To disable TLS termination in the 
Ambassador Edge Stack, delete any existing `Host`s and set the 
`requestPolicy.insecure.action` to `Route` in a `Host`:

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: example-host
spec:
  hostname: host.example.com
  acmeProvider:
    authority: none
  requestPolicy:
    insecure:
      action: Route
```

### HTTPS and Cleartext

Ambassador can also support serving both HTTPS and cleartext traffic from a
single Ambassador.

This configuration is the same whether you are running the Open-Source API 
Gateway or the Ambassador Edge Stack. The configuration is very similar to the
`Host` above but with the `Host` configured to terminate TLS.

```yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: example-host
spec:
  hostname: host.example.com
  acmeProvider:
    authority: none
  tlsSecret:
    name: example-cert
  requestPolicy:
    insecure:
      action: Route
      additionalPort: 8080
```

With the above configuration, we are tell Ambassador to terminate TLS with the
certificate in the `example-cert` `Secret` and route cleartext traffic that
comes in over port `8080`.

## HTTP->HTTPS Redirection

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
    additionalPort: 8080
```

Ambassador will then enable cleartext redrection in two ways.

First, Ambassador will listen on the `insecure.additionalPort` and consider any
traffic on this port as `insecure` and redirect it to HTTPS. 

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