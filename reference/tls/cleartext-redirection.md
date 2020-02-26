# HTTP -> HTTPS Redirection

Most modern websites that force HTTPS will also automatically redirect any requests that come into it over HTTP.

```
Client                    Ambassador Edge Stack
|                             |
| http://<hostname>/api       |
| --------------------------> |
| 301: https://<hostname>/api |
| <-------------------------- |
| https://<hostname>/api      |
| --------------------------> |
|                             |
```

Ambassador Edge Stack exposes configuration for this in two ways:

- Redirecting based off the incoming port
- Redirecting based off the incoming protocol (via the `x-forwarded-proto` header)

Typically, port-based redirection is the preferred method since it is simpler to manage and will work with all use cases. Redirecting based off of the `x-forwarded-proto` header requires an L7 load-balancer or proxy in front of Ambassador Edge Stack to set that header.

## Using the `Host` resource

The [`Host`](/reference/host-crd) resource is used to configure how Ambassador handles cleartext on a domain. You can configure Ambassador to [deny, allow, or redirect](/reference/host-crd/#secure-and-insecure-requests) cleartext to HTTPS on a domain using a single resource.

The following `Host` gives an example of how to enable Ambassador to redirect cleartext to HTTPS on the `host.example.com` domain. 

```yaml
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: example-host
spec:
  hostname: host.example.com
  acmeProvider:
    email: julian@example.com
  insecure:
    action: Redirect
```

**Note:** Ambassador defaults to redirecting cleartext to HTTPS for all `Host`s. 

## For Ambassador 0.86.1 and below

In Ambassador 0.86.1 and below, TLS behavior was configured with the `TLSContext`. This included cleartext redirection. 

The follow reference is **deprecated**. For those using old versions of Ambassador, [upgrade to the latest verion](/user-guide/upgrade-to-edge-stack/) of Ambassador to use the `Host` resource to manage TLS termination. 

### Port-based Redirection

Port-based redirection opens up Ambassador Edge Stack to listen on a defined port and issue a `301` redirect to HTTPS for all traffic that comes in on that port.

In the example at the top of the page;

- The client sends a standard HTTP request (port 80) to Ambassador Edge Stack.
- The request hits Ambassador Edge Stack's redirect listener and Ambassador Edge Stack returns a `301` redirect to https.
- The client resends the request as a standard https request (port 443) to Ambassador Edge Stack.

To configure Ambassador Edge Stack to handle this behavior you need set `redirect_cleartext_from: <http_port>` in a `TLSContext`:

1. Create a `TLSContext` to handle TLS termination, and tell it to enforce redirection. This example shows redirecting traffic to Ambassador's default cleartext service port, `8080`:

    ```yaml
    apiVersion: getambassador.io/v2
    kind: TLSContext
    metadata:
      name: tls
    spec:
      hosts: ["*"]
      secret: ambassador-cert
      redirect_cleartext_from: 8080
    ```

2. Verify that the port assignments on the Ambassador Edge Stack service are correct.

    The below service definition uses the default HTTP and HTTPS port assignments

    ```yaml
    apiVersion: v1
    kind: Service
    metadata:
      name: ambassador
    spec:
      ports:
      - name: http
        port: 80
        targetPort: 8080
      - name: https
        port: 443
        targetPort: 8443
      selector:
        service: ambassador
    ```

**Notes:**

- The ability to `redirect_cleartext_from` was added to the `TLSContext` in Ambassador 0.84.0. Earlier versions of Ambassador need to use a [tls `Module`](../../core/tls#tls-module) for cleartext redirection.

- As shown above, Ambassador Edge Stack performs this http -> https redirection by issuing a `301` redirect to `https://<hostname>/`. The `<hostname>` represents the domain name/IP address and port of the incoming request. This means if a port is defined on an incoming request, it will be redirected to https on that port. Because of this, cleartext redirection is not supported when using non-default HTTP and HTTPS ports.

- If you use multiple `TLSContext`s, it doesn't matter which `TLSContext` sets `redirect_cleartext_from`. However, it is an error to attempt to set `redirect_cleartext_from` on multiple distinct ports in multiple distinct `TLSContext`s.

### Protocol-based Redirection

Ambassador Edge Stack can perform HTTP -> HTTPS redirection based on the protocol of the incoming request. This is done by checking the `x-forwarded-proto` header that can be set by an L7 load balancer or proxy sitting in front of Ambassador Edge Stack.

While port-based redirection is preferred for most use cases, using the `x-forwarded-proto` header to redirect to HTTPS is useful when the front load balancer or proxy is terminating TLS.

Protocol-based redirection is configured in the Ambassador Edge Stack `Module`:

```yaml
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    use_remote_address: false
    x_forwarded_proto_redirect: true
```

**Note**: Ambassador Edge Stack will need to be restarted for this configuration to take effect.

See the [Ambassador on AWS documentation](../../ambassador-with-aws#l7-load-balancer) for an example of protocol-based redirection with TLS termination happening at the load balancer.
