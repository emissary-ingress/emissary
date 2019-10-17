# HTTP -> HTTPS Redirection

Most modern websites that force HTTPS will also automatically redirect any requests that come into it over HTTP.

```
Client                    Ambassador
|                             |
| http://<hostname>/api       |
| --------------------------> |
| 301: https://<hostname>/api |
| <-------------------------- |
| https://<hostname>/api      |
| --------------------------> |
|                             |
```

Ambassador exposes configuration for this in two ways:

- Redirecting based off the incoming port
- Redirecting based off the incoming protocol (via the `x-forwarded-proto` header)

Typically, port-based redirection is the preferred method since it is simpler to manage and will work with all use cases. Redirecting based off the `x-forwarded-proto` header requires an L7 load-balancer or proxy in front of Ambassador to set that header.

## Port-based Redirection

Port-based redirection opens up Ambassador to listen on a defined port and issue a `301` redirect to HTTPS for all traffic that comes in on that port.

In the example at the top of the page;

- The client sends a standard http request (port 80) to Ambassador.
- The request hits Ambassador's redirect listener and Ambassador returns a `301` redirect to https.
- The client resends the request as a standard https request (port 443) to Ambassador.

To configure Ambassador to handle this behavior you need set `redirect_cleartext_from: <http_port>` in a `TLSContext`:

1. Create a `TLSContext` to handle TLS termination, and tell it to enforce redirection. This example shows redirecting traffic to Ambassador's default cleartext service port, `8080`: 

    ```yaml
    apiVersion: getambassador.io/v1
    kind: TLSContext
    metadata:
      name: tls
    spec:
      hosts: ["*"]
      secret: ambassador-cert
      redirect_cleartext_from: 8080
    ```

2. Verify that the port assignments on the Ambassador service are correct.

    The below service definition uses the default http and https port assignments

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

**Note**: 

As shown above, Ambassador performs this http -> https redirection by issuing a `301` redirect to `https://<hostname>/`. The `<hostname>` represents the domain name/IP address and port of the incoming request. This means if a port is defined on an incoming request, it will be redirected to https on that port. Because of this, cleartext redirection is not supported when using non-default http and https ports.

## Protocol-based Redirection

Ambassador can perform HTTP -> HTTPS redirection based off the protocol of the incoming request. This is done by checking the `x-forwarded-proto` header that can be set by an L7 load balancer or proxy sitting in front of Ambassador.

While port-based redirection is preferred for most use cases, using the `x-forwarded-proto` header to redirect to HTTPS is useful when the front load balancer or proxy is terminating TLS.

Protocol-based redirection is configured in the Ambassador `Module`:

```yaml
apiVersion: getambassador.io/v1
kind: Module
metadata:
  name: ambassador
spec:
  config:
    use_remote_address: false
    x_forwarded_proto_redirect: true
```

**Note**: Ambassador will need to be restarted for this configuration to take affect.


See the [AWS documentation](/reference/ambassador-with-aws#l7-load-balancer) an example of protocol-based redirection with TLS termination happening at the load balancer.
