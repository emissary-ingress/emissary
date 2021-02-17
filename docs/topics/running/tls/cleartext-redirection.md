# Cleartext Support

While most modern web applications will choose to encrypt all traffic, there
are reasons why you will want to support clients who access your website
without encryption in cleartext.

Ambassador supports both forcing 
[automatic redirection to HTTPS](#http-https-redirection) and 
[serving cleartext](#cleartext-routing) traffic on a `Host`.

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

> **WARNING - Host Configuration:** The `requestPolicy` property of the `Host` `CRD` is applied globally within an Ambassador instance, even if it is applied to only one `Host` when multiple `Host`s are configured. Different `requestPolicy` behaviors cannot be applied to different `Host`s. It is recommended to apply an identical `requestPolicy` to all `Host`s instead of assuming the behavior, to create a more human readable config. 
> 
> If you intend to use more than one type of `requestPolicy`, you will need a separate Ambassador instance for each separate type.
> 
> If multiple `Host`s are applied, the `requestPolicy` from the `Host` with the first alphabetical `metadata.name` is always the one that is applied. Order does not matter.
> If a `requestPolicy` is not defined for a `Host`, it's assumed to be `Redirect`, and so even if a host named `a` does not specify it, the default `requestPolicy` of `Redirect` will be applied to all `Host`s in that Ambassador instance.
> 
> For more information, please refer to the [`Host` documentation](../host-crd#secure-and-insecure-requests).

The `insecure-action` can be one of:

* `Redirect` (the default): redirect to HTTPS
* `Route`: go ahead and route as normal; this will allow handling HTTP requests normally
* `Reject`: reject the request with a 400 response


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

> The `additionalPort` element tells Ambassador to listen on the specified `insecure-port` and treat any request arriving on that port as insecure. **By default, `additionalPort` will be set to 8080 for any `Host` using TLS.** To disable this redirection entirely, set `additionalPort` explicitly to `-1`:

```yaml
requestPolicy:
  insecure:
    additionalPort: -1   # This is how to disable the default redirection from 8080.
```

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
