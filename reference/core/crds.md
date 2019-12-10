# Using Custom Resource Definitions

As of Ambassador 0.70, any Ambassador Edge Stack resource can be expressed as a CRD in the `getambassador.io` API group:

- use `apiVersion: getambassador.io/v2`
- use the same `kind` as you would in an attribute
- put the resource name in `metadata.name`
- put everything else in `spec`

As an example, you could use the following CRDs for a very simple Lua test:

```yaml
---
apiVersion: getambassador.io/v2
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
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: lua-target-mapping
  namespace: default
spec:
  prefix: /target/
  service: luatest-http
```

Note that the `namespace` must be declared in the `metadata`, but if needed, `ambassador_id` must be declared in the `spec`.

## CRDs supported by Ambassador Edge Stack

The full set of CRDs supported by the Abassador Edge Stack:

| `Kind` | Kubernetes singular | Kubernetes plural |
| :----- | :------------------ | :---------------- |
| `AuthService` | `authservice` | `authservices` |
| `ConsulResolver` | `consulresolver` | `consulresolvers` |
| `Filter` | `filter` | `filters` |
| `FilterPolicy` | `filterpolicy` | `filterpolicies`|
| `Host` | `host`| `hosts` |
| `KubernetesEndpointResolver` | `kubernetesendpointresolver` | `kubernetesendpointresolvers` |
| `KubernetesServiceResolver` | `kubernetesserviceresolver` | `kubernetesserviceresolvers` |
| `LogService` | `logservice` | `logservices` |
| `Mapping` | `mapping` | `mappings` |
| `Module` | `module` | `modules` |
| `RateLimit` | `ratelimit` | `ratelimits` |
| `RateLimitService` | `ratelimitservice` | `ratelimitservices` |
| `TCPMapping` | `tcpmapping` | `tcpmappings` |
| `TLSContext` | `tlscontext` | `tlscontexts` |
| `TracingService` | `tracingservice` | `tracingservices` |

So, for example, if you're using CRDs, then

```kubectl get mappings```

should show all your `Mapping` custom resources.

## CRDs and RBAC

You will need to grant your Kubernetes service appropriate RBAC permissions to use CRDs. The default Ambassador Edge Stack RBAC examples have been updated, but the appropriate rules are

```yaml
rules:
- apiGroups: [""]
  resources: [ "endpoints", "namespaces", "services" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "getambassador.io" ]
  resources: [ "*" ]
  verbs: ["get", "list", "watch", "update", "patch", "create", "delete" ]
- apiGroups: [ "apiextensions.k8s.io" ]
  resources: [ "customresourcedefinitions" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "networking.internal.knative.dev" ]
  resources: [ "clusteringresses", "ingresses" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "networking.internal.knative.dev" ]
  resources: [ "ingresses/status", "clusteringresses/status" ]
  verbs: ["update"]
- apiGroups: [ "extensions" ]
  resources: [ "ingresses" ]
  verbs: ["get", "list", "watch"]
- apiGroups: [ "extensions" ]
  resources: [ "ingresses/status" ]
  verbs: ["update"]
- apiGroups: [""]
  resources: [ "secrets" ]
  verbs: ["get", "list", "watch", "create", "update"]
```

## Creating the CRD types within Kubernetes

Before using the CRD types, you must add them to the Kubernetes API server. This is most easily done by applying [`aes-crds.yaml`](../../../yaml/aes-crds.yaml).
