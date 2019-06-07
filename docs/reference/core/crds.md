# Using CRDs with Ambassador

As of Ambassador 0.70, any Ambassador resource can be expressed as a CRD in the `getambassador.io` API group:

- use `apiVersion: getambassador.io/v1`
- use the same `kind` as you would in an annotation
- put the resource name in `metadata.name`
- put everything else in `spec`

As an example, you could use the following CRDs for a very simple Lua test:

```yaml
---
apiVersion: getambassador.io/v1
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
    lua_scripts: |
      function envoy_on_response(response_handle)
        response_handle: headers():add("Lua-Scripts-Enabled", "Processed")
      end
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: lua-target-mapping
  namespace: default
spec:
  prefix: /target/
  service: luatest-http
```

(Note that the `namespace` must be declared in the `metadata`, but if needed, `ambassador_id` must be declared in the `spec`.)

## CRDs supported by Ambassador

The full set of CRDs supported by Ambassador in 0.70.0:

| `Kind` | Kubernetes singular | Kubernetes plural |
| :----- | :------------------ | :---------------- |
| `AuthService` | `authservice` | `authservices` |
| `ConsulResolver` | `consulresolver` | `consulresolvers` |
| `KubernetesEndpointResolver` | `kubernetesendpointresolver` | `kubernetesendpointresolvers` |
| `KubernetesServiceResolver` | `kubernetesserviceresolver` | `kubernetesserviceresolvers` |
| `Mapping` | `mapping` | `mappings` |
| `Module` | `module` | `modules` |
| `RateLimitService` | `ratelimitservice` | `ratelimitservices` |
| `TCPMapping` | `tcpmapping` | `tcpmappings` |
| `TLSContext` | `tlscontext` | `tlscontexts` |
| `TracingService` | `tracingservice` | `tracingservices` |

So, for example, if you're using CRDs then 

```kubectl get mappings```

should show all your `Mapping` CRDs.

## CRDs and RBAC

You will need to grant your Kubernetes service appropriate RBAC permissions to use CRDs. The default Ambassador RBAC examples have been updated, but the appropriate rules are

```yaml
rules:
- apiGroups: [""]
  resources: [ "endpoints", "namespaces", "secrets", "services" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "getambassador.io" ]
  resources: [ "*" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "apiextensions.k8s.io" ]
  resources: [ "customresourcedefinitions" ]
  verbs: ["get", "list", "watch"]
```

## Creating the CRD types within Kubernetes

Before using the CRD types, you must add them to the Kubernetes API server. This is most easily done with the following YAML (which you can find in `docs/yaml/ambassador-crds.yaml`)

```yaml
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: authservices.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: authservices
    singular: authservice
    kind: AuthService
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: consulresolvers.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: consulresolvers
    singular: consulresolver
    kind: ConsulResolver
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: kubernetesendpointresolvers.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: kubernetesendpointresolvers
    singular: kubernetesendpointresolver
    kind: KubernetesEndpointResolver
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: kubernetesserviceresolvers.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: kubernetesserviceresolvers
    singular: kubernetesserviceresolver  
    kind: KubernetesServiceResolver
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: mappings.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: mappings
    singular: mapping
    kind: Mapping
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: modules.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: modules
    singular: module
    kind: Module
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: ratelimitservices.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: ratelimitservices
    singular: ratelimitservice
    kind: RateLimitService
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: tcpmappings.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: tcpmappings
    singular: tcpmapping
    kind: TCPMapping
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: tlscontexts.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: tlscontexts
    singular: tlscontext
    kind: TLSContext
---
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: tracingservices.getambassador.io
spec:
  group: getambassador.io
  version: v1
  versions:
  - name: v1
    served: true
    storage: true
  scope: Namespaced
  names:
    plural: tracingservices
    singular: tracingservice
    kind: TracingService
```
