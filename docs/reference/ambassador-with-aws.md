# Ambassador on AWS

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

Ambassador can also terminate TLS directly, as discussed elsewhere in the documentation.
