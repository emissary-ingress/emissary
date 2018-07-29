## Ambassador on AWS

The following is a sample configuration for deploying Ambassador in AWS (this configuration is templated using [Forge](https://forge.sh)):

```
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador-main-{{ build.profile.name }}
  namespace: {{ service.namespace }}
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-ssl-cert: "{{ service.ambassador.tlsCertificateArn }}"
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
        use_proxy_proto: {{ service.ambassador.useProxyProtocol | lower }}
        use_remote_address: true
spec:
  type: LoadBalancer
  ports:
  - name: ambassador
    port: 443
    targetPort: 80
  selector:
    service: ambassador-{{ build.profile.name }}
```

In this configuration, an ELB is deployed with a multi-domain AWS Certificate Manager certificate. The ELB is configured to route TCP to support both WebSockets and HTTP. Ambassador is configured with `use_remote_address` and `use_proxy_proto` to ensure that remote IP addresses are passed through properly. TLS termination then occurs at the ELB.

### Ambassador and AWS load balancer notes

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

In Kubernetes, when using the AWS integration and a service of type `LoadBalancer`, the only types of load balancers that can be created are ELBs and NLBs (in Kubernetes 1.9 and later).

If you are running an ELB in L4 mode, you need to:

* Run the ELB with the following listener configuration:
  * `:443` -> `:8443` (the Envoy port doesn't matter)
  * `:80` -> `:8080` (the Envoy port doesn't matter)
  * Configure `redirect_cleartext_from` to redirect traffic on `8080` to the secure port

For this setup of an L4 load balancer, here is an example Ambassador configuration:

```yaml
apiVersion: ambassador/v0
kind:  Module
name:  tls
config:
  server:
    enabled: True
    redirect_cleartext_from: 8080
```

If you are running the load balancer in L7 mode, then you will want to redirect all the incoming HTTP requests with `X-FORWARDED-PROTO: http` header to HTTPS. Here is an example Ambassador configuration for this scenario:

```yaml
apiVersion: ambassador/v0
kind: Module
name: ambassador
config:
  x_forwarded_proto_redirect: true
```
