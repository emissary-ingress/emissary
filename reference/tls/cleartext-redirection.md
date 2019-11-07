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

Typically, port-based redirection is the preferred method since it is simpler to manage and will work with all use cases. Redirecting based off the `x-forwarded-proto` header requires an L7 load-balancer or proxy in front of Ambassador Edge Stack to set that header.

## Port-based Redirection

Port-based redirection opens up Ambassador Edge Stack to listen on a defined port and issue a `301` redirect to HTTPS for all traffic that comes in on that port.

In the example at the top of the page;

- The client sends a standard http request (port 80) to Ambassador Edge Stack.
- The request hits Ambassador Edge Stack's redirect listener and Ambassador Edge Stack returns a `301` redirect to https.
- The client resends the request as a standard https request (port 443) to Ambassador Edge Stack.

To configure Ambassador Edge Stack to handle this behavior you need to create a `tls` `Module` that sets `redirect_cleartext_from: <http_port>`.

1. Create a `TLSContext` to handle TLS termination

    ```yaml
    apiVersion: getambassador.io/v2
    kind: TLSContext
    name: tls
    hosts: ["*"]
    secret: ambassador-cert
    ```

2. Configure a `TLS` `Module` to create the redirect listener in Ambassador Edge Stack on its http port. By default, this is port `8080`

    ```yaml
    apiVersion: getambassador.io/v2
    kind: Module
    name: tls
    config:
      server:
        redirect_cleartext_from: 8080
    ```

3. Verify the port assignments on the Ambassador Edge Stack service are correct.

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

As shown above, Ambassador Edge Stack performs this http -> https redirection by issuing a `301` redirect to `https://<hostname>/`. The `<hostname>` represents the domain name/IP address and port of the incoming request. This means if a port is defined on an incoming request, it will be redirected to https on that port. Because of this, cleartext redirection is not supported when using non-default http and https ports.

## Protocol-based Redirection

Ambassador Edge Stack can perform HTTP -> HTTPS redirection based off the protocol of the incoming request. This is done by checking the `x-forwarded-proto` header that can be set by an L7 load balancer or proxy sitting in front of Ambassador Edge Stack.

While port-based redirection is preferred for most use cases, using the `x-forwarded-proto` header to redirect to HTTPS is useful when the front load balancer or proxy is terminating TLS.

Protocol-based redirection is configured in the Ambassador Edge Stack `Module`:

```yaml
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
config:
  use_remote_address: false
  x_forwarded_proto_redirect: true
```

**Note**: Ambassador Edge Stack will need to be restarted for this configuration to take affect.


See the [AWS documentation](/reference/ambassador-with-aws#l7-load-balancer) an example of protocol-based redirection with TLS termination happening at the load balancer.

<div style="border: thick solid gray;padding:0.5em"> 

Ambassador Edge Stack is a community supported product with 
[features](getambassador.io/features) available for free and 
limited use. For unlimited access and commercial use of
Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) 
for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
</p>
