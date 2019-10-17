# Global Configuration: Modules

Modules let you enable and configure special behaviors for Ambassador, in ways that may apply to Ambassador as a whole or which may apply only to some mappings. The actual configuration possible for a given module depends on the module.

## Module configuration

Modules can be added as annotations to an existing Kubernetes service, e.g., the Ambassador service. They can also be implemented as independent Kubernetes Custom Resource Definitions (CRDs). Here is a sample configuration of the Ambassador and tls`Module`s:

```yaml
---
apiVersion: getambassador.io/v1
kind: Module
metadata:
  name: ambassador
spec:
  config:
    enable_grpc_web: true
---
apiVersion: getambassador.io/v1
kind: Module
metadata:
  name: tls
spec:
  config:
    server:
      enabled: true
      secret: ambassador-certs
      redirect_cleartext_from: 8080
```

Here is the equivalent configuration as annotations on the `ambassador` Kubernetes `service`:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: getambassador.io/v1
      kind: Module
      name: ambassador
      config:
        enable_grpc_web: True
      ---
      apiVersion: getambassador.io/v1
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

**Note:** Modules are named resources. A `Module` with `name: ambassador` is distinctly different than a `Module` with `name: tls`. A `Module` with a name other than `ambassador` and `tls` will be ignored. 
 
## The `ambassador` module

The [`ambassador`](/reference/core/ambassador) module covers general configuration options for Ambassador as a whole. These configuration options generally pertain to routing, protocol support, and the like. Most of these options are likely of interest to operations.

## The `tls` module

The ['tls'](/reference/core/tls) module covers TLS configuration.

## The `authentication` Module

The `authentication` module is now deprecated. Use the [AuthService](/reference/services/auth-service) manifest type instead.
