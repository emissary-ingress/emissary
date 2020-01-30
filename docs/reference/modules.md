# Global Configuration: Modules

Modules let you enable and configure special behaviors for Ambassador Edge Stack, in ways that may apply to Ambassador Edge Stack as a whole or which may apply only to some mappings. The actual configuration possible for a given module depends on the module.

## Module Configuration

Modules can be added as annotations to an existing Kubernetes service, e.g., the Ambassador Edge Stack service. They can also be implemented as independent Kubernetes Custom Resource Definitions (CRDs). Here is a sample configuration of the core `ambassador Module`:

```yaml
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    enable_grpc_web: true
```

Here is the equivalent configuration as an annotation on the `ambassador` Kubernetes `service`:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: getambassador.io/v2
      kind: Module
      name: ambassador
      config:
        enable_grpc_web: True
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

**Note:** Modules are named resources. A `Module` with `name: ambassador` is distinctly different than a `Module` with `name: my-module`.
 
## The `ambassador` Module

The [`ambassador`](../core/ambassador) module covers general configuration options for Ambassador Edge Stack as a whole. These configuration options generally pertain to routing, protocol support, and the like. Most of these options are likely of interest to operations.

## The `tls` Module

The `tls` module is now deprecated. Use the [TLSContext](../core/tls) manifest type instead.

## The `authentication` Module

The `authentication` Module is now deprecated. Use the [AuthService](../services/auth-service) manifest type instead.
