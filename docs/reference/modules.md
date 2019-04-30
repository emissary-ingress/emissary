# Global Configuration: Modules

Modules let you enable and configure special behaviors for Ambassador, in ways that may apply to Ambassador as a whole or which may apply only to some mappings. The actual configuration possible for a given module depends on the module.

## Module configuration

A module is added to an existing Kubernetes service, e.g., the Ambassador service. If you expect to make frequent changes to your Ambassador configuration, you may want to put the module on a dummy service. This would allow you to isolate Ambassador configuration changes from your production routing configuration. Here is a sample module configuration of both the `ambassador` and `tls` modules:

```
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind: Module
      name: ambassador
      config:
        enable_grpc_web: True
      ---
      apiVersion: ambassador/v1
      kind: Module
      name: tls
      config:
        server:
          enabled: true
          secret: ambassador-certs
          redirect_cleartext_from: 8080
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local
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

## The `ambassador` module

The [`ambassador`](/reference/core/ambassador) module covers general configuration options for Ambassador as a whole. These configuration options generally pertain to routing, protocol support, and the like. Most of these options are likely of interest to operations.

## The `tls` module

The ['tls'](/reference/core/tls) module covers TLS configuration.

## The `authentication` Module

The `authentication` module is now deprecated. Use the [AuthService](/reference/services/auth-service) manifest type instead.
