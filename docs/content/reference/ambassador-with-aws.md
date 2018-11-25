# Ambassador on AWS

The following is a sample configuration for deploying Ambassador in AWS:

```
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador
  namespace: {{ ambassador namespace }}
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: "{{ tls certificate ARN }}"
    service.beta.kubernetes.io/aws-load-balancer-ssl-ports: "*"
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: "tcp"
    service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
    service.beta.kubernetes.io/aws-load-balancer-proxy-protocol: "*"
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind:  Module
      name:  ambassador
      config:
        use_proxy_proto: true
        use_remote_address: true
spec:
  type: LoadBalancer
  ports:
  - name: ambassador
    port: 443
    targetPort: 80
  selector:
    service: ambassador
```

In this configuration, an ELB is deployed with a multi-domain AWS Certificate Manager certificate. The ELB is configured to route TCP to support both WebSockets and HTTP. Ambassador is configured with `use_remote_address` and `use_proxy_proto` to ensure that remote IP addresses are passed through properly. TLS termination then occurs at the ELB.

## Ambassador and AWS load balancer notes

AWS provides three types of load balancers:

* "Classic" Load Balancer (abbreviated ELB or CLB, sometimes referred to as ELBv1 or Elastic Load Balancer)
  * Supports L4 (TCP, TCP+SSL) and L7 load balancing (HTTP 1.1, HTTPS 1.1)
  * Does not support WebSockets unless running in L4 mode
  * Does not support HTTP 2 (which is required for GRPC) unless running in L4 mode
  * Can perform SSL/TLS offload
* Application Load Balancer (abbreviated ALB, sometimes referred to as ELBv2)
  * Supports L7 only
  * Supports WebSockets
  * Supports a broken implementation of HTTP2 (trailers are not supported and these are needed for GRPC)
  * Can perform SSL/TLS offload
* Network Load Balancer (abbreviated NLB)
  * Supports L4 only
  * Cannot perform SSL/TLS offload

In Kubernetes, when using the AWS integration and a service of type `LoadBalancer`, the only types of load balancers that can be created are ELBs and NLBs (in Kubernetes 1.9 and later). When `aws-load-balancer-backend-protocol` is set to `tcp`, AWS will create a L4 ELB. When `aws-load-balancer-backend-protocol` is set to `http`, AWS will create a L7 ELB.

### L4 Load Balancer

When running an ELB in L4 mode, you will need to listen on two ports to redirect all incoming HTTP requests to HTTPS. The first port will listen for HTTP traffic to redirect to HTTPS, while the second port will listen for HTTPS traffic.

Let's say,
- port 80 on the load balancer forwards requests to port 8080 on Ambassador
- port 443 on the load balancer forwards requests to port 8443 on Ambassador

First off, configure this forwarding in your load balancer.

Now, we want every request on port 80 (8080 of Ambassador) to be redirected to port 443 (8443 of Ambassador)

To achieve this, you need to use `redirect_cleartext_from` as follows -

```yaml
apiVersion: ambassador/v0
kind:  Module
name:  tls
config:
  server:
    enabled: True
    redirect_cleartext_from: 8080
```

This configuration makes Ambassador start a new listener on 8080 which redirects all cleartext HTTP traffic to HTTPS.

Note: Ambassador only supports standard ports (80 and 443) on the load balancer for L4 redirection, [yet](https://github.com/datawire/ambassador/issues/702)! For instance, if you configure port 8888 for HTTP and 9999 for HTTPS on the load balancer, then an incoming request to `http://<host>:8888` will be redirected to `https://<host>:8888`. This will fail because HTTPS listener is on port 9999.

### L7 Load Balancer

If you are running the load balancer in L7 mode, then you will want to redirect all the incoming HTTP requests without the `X-FORWARDED-PROTO: https` header to HTTPS. Here is an example Ambassador configuration for this scenario:

```yaml
apiVersion: ambassador/v0
kind: Module
name: ambassador
config:
  x_forwarded_proto_redirect: true
```
