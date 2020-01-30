# Ambassador Configuration with Kubernetes Service Annotations

Ambassador configuration can be expressed as an annotation of a Kubernetes Service.

## Benefits

For many people, this is the preferred format for configuring Ambassador because it directly ties the routing configuration in with the Service definition. i.e. Creating the Service creates the route in Ambassador and deleting the service removes the route.

It also allows you to install and configure Ambassador without needing cluster permissions to add CRDs to the cluster.

## Drawbacks

Annotations, however, can be hard to manage and easy to lose track of since you are not able to manage them the same way you do other Kubernetes resources.

For example:

Using CRDs you can find all `Mapping`s deployed in the cluster with

```
kubectl get mappings -A
```

With annotations, you would have to describe and examine the `metadata.annotations` of every service.

## Example

The following is an example of how to configure annotations for the example quote service from the [install documentation](../../../user-guide/getting-started).

### CRD

```yaml
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: quote-backend
spec:
  prefix: /backend/
  service: quote:5000
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: quote-backend
spec:
  prefix: /backend/
  service: quote:8080
  labels:
    ambassador:
      - request_label:
        - backend
```
With CRDs above, we open up routes in Ambassador using two separate Ambassador `Mapping` resources. These objects are managed separately from the service and must be created and deleted independently.

### Annotations

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: quote
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind: Mapping
      name: quote-ui_mapping
      prefix: /
      service: quote:5000
      ---
      apiVersion: ambassador/v1
      kind: Mapping
      name: quote-backend_mapping
      prefix: /backend/
      service: quote:8080
      labels:
        ambassador:
          - request_label:
            - backend
spec:
  ports:
  - name: ui
    port: 5000
    targetPort: 5000
  - name: backend
    port: 8080
    targetPort: 8080
  selector:
    app: quote
```

With the annotation approach above, we can expose the exact same routes my creating the `Mapping` object as part of the Kubernetes Service. There is more overhead for creating and managing the `Mapping`s but they are created and deleted when the service is.

## CRD Translation

All example configuration in this document is given in CRD format. It is easy to translate any CRD to an `annotation` by following the steps below.

### Example CRD

```yaml
---
apiVersion: getambassador.io/v1
kind: Module
metadata:
  name: ambassador
spec:
  config:
    diagnostics:
      enabled: false
```

Starting with the example above:

1. Change the `apiVersion` from `getambassador.io/v1` to `ambassador/v1`

```diff
---
-apiVersion: getambassador.io/v1
+apiVersion: ambassador/v1
kind: Module
metadata:
  name: ambassador
spec:
  config:
    diagnostics:
      enabled: false
```

2. Remove the `metadata` section and resolve the indentation of `name` to be inline with `kind`

```diff
---
apiVersion: ambassador/v1
kind: Module
-metadata:
-  name: ambassador
+name: ambassador
spec:
  config:
    diagnostics:
      enabled: false
```

3. Remove the `spec` section and drop the indentation of everything below it one level

```diff
---
apiVersion: ambassador/v1
kind: Module
name: ambassador
-spec:
-  config:
-    diagnostics:
-      enabled: false
+config:
+  diagnostics:
+    enabled: false
```

After this step, we are left with a `yaml` object that looks like this:

```yaml
---
apiVersion: ambassador/v1
kind: Module
name: ambassador
config:
  diagnostics:
    enabled: false
```

4. Finally, add the object as an annotation of a Kubernetes Service with key `getambassador.io/config`

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: svc
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind: Module
      name: ambassador
      config:
        diagnostics:
          enabled: false
spec:
  ports:
  - name: http
    port: 80
```
