# Ambassador as an Ingress Controller

An `Ingress` resource is a popular way to expose Kubernetes services to the Internet. In order to use `Ingress` resources, you need to install an [ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/). The Ambassador Edge Stack can function as a fully-fledged Ingress controller, making it easy to work with other `Ingress`-oriented tools within the Kubernetes ecosystem.

## When and How to Use the `Ingress` Resource

If you're new to the Ambassador Edge Stack and to Kubernetes, we'd recommend you start with our [quickstart](../../../tutorials/getting-started/), instead of using `Ingress`. If you're a power user and need to integrate with other software that leverages the `Ingress` resource, read on. The `Ingress` specification is very basic, and, as such, does not support many of the features of the Ambassador Edge Stack, so you'll be using both `Ingress` resources and `Mapping` resources to manage your Kubernetes services.

### What is Required to Use the `Ingress` Resource?

- Know what version of Kubernetes you are using.

   In Kubernetes 1.13 and below, the `Ingress` was only included in the `extensions` api.

   Starting in Kubernetes 1.14, the `Ingress` was added to the new `networking.k8s.io` api.
   
   Kubernetes 1.18 introduced the `IngressClass` resource to the existing `networking.k8s.io/v1beta1` api.

   **Note:** If you are using 1.14 and above, it is recommended to use `apiVersion: networking.k8s.io/v1beta1` when defining `Ingresses`. Since both are still supported in all 1.14+ versions of Kubernetes, this document will use `extensions/v1beta1` for compatibility reasons.
   If you are using 1.18 and above, sample usage of the `IngressClass` resource and `pathType` field are [available on our blog](https://blog.getambassador.io/new-kubernetes-1-18-extends-ingress-c34abdc2f064).

- You will need RBAC permissions to create `Ingress` resources in either
  the `extensions` `apiGroup` (present in all supported versions of
  Kubernetes) or the `networking.k8s.io` `apiGroup` (introduced in
  Kubernetes 1.14).

- The Ambassador Edge Stack will need RBAC permissions to get, list, watch, and update `Ingress` resources.

  You can see this in the [`aes-crds.yaml`](/yaml/aes.yaml)
  file, but this is the critical rule to add to the Ambassador Edge Stack's `Role` or `ClusterRole`:

      - apiGroups: [ "extensions", "networking.k8s.io" ]
        resources: [ "ingresses", "ingressclasses" ]
        verbs: ["get", "list", "watch"]
      - apiGroups: [ "extensions", "networking.k8s.io" ]
        resources: [ "ingresses/status" ]
        verbs: ["update"]

   **Note:** This is included by default in all recent versions of the Ambassador install YAML

- You must create your `Ingress` resource with the correct `ingress.class`.

  The Ambassador Edge Stack will automatically read Ingress resources with the annotation
  `kubernetes.io/ingress.class: ambassador`.

- You may need to set your `Ingress` resources' `ambassador-id`.

  If you're not using the `default` ID, you'll need to add the `getambassador.io/ambassador-id`
  annotation to your `Ingress`. See the examples below.

- You must create a `Service` resource with the correct `app.kubernetes.io/component` label.

  The Ambassador Edge Stack will automatically load balance Ingress resources using the endpoint exposed 
  from the Service with the annotation `app.kubernetes.io/component: ambassador-service`.
  
  ```yaml
      kind: Service
      apiVersion: v1
      metadata:
        name: ingress-ambassador
        labels:
          app.kubernetes.io/component: ambassador-service
      spec:
        externalTrafficPolicy: Local
        type: LoadBalancer
        selector:
          service: ambassador
        ports:
          - name: http
            port: 80
            targetPort: http
          - name: https
            port: 443
            targetPort: https
            ```

### When Should I Use an `Ingress` Instead of Annotations or CRDs?

As of 0.80.0, Datawire recommends that the Ambassador Edge Stack be configured with CRDs. The `Ingress` resource is available to users who need it for integration with other ecosystem tools, or who feel that it more closely matches their workflows -- however, it is important to  recognize that the `Ingress` resource is rather more limited than the Ambassador Edge Stack `Mapping` is (for example, the `Ingress` spec has no support for rewriting or for TLS origination). **When in doubt, use CRDs.**

## Ambassador `Ingress` Support

The Ambassador Edge Stack supports basic core functionality of the  `Ingress` resource, as
defined by the [`Ingress`](https://kubernetes.io/docs/concepts/services-networking/ingress/)
resource itself:

1. Basic routing, including the `route` specification and the default backend
  functionality, is supported.
    - It's particularly easy to use a minimal `Ingress` to the Ambassador Edge Stack diagnostic UI
2. [TLS termination](../tls/) is supported.
    - you can use multiple `Ingress` resources for SNI
3. Using the `Ingress` resource in concert with the Ambassador Edge Stack CRDs or annotations is supported.
    - this includes the Ambassador Edge Stack annotations on the `Ingress` resource itself

The Ambassador Edge Stack does **not** extend the basic `Ingress` specification except as follows:

- The `getambassador.io/ambassador-id` annotation allows you to set the Ambassador Edge Stack ID for
  the `Ingress` itself; and

- The `getambassador.io/config` annotation can be provided on the `Ingress` resource, just
  as on a `Service`.

Note that if you need to set `getambassador.io/ambassador-id` on the `Ingress`, you will also need to set `ambassador-id` on resources within the annotation.

### `Ingress` Routes and `Mapping`s

The Ambassador Edge Stack actually creates `Mapping` objects from the `Ingress` route rules. These `Mapping` objects interact with `Mapping`s defined in CRDs **exactly** as they would if the `Ingress`route rules had been specified with CRDs originally.

For example, this `Ingress` resource

```yaml
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
  name: test-ingress
spec:
  rules:
  - http:
      paths:
      - path: /foo/
        backend:
          serviceName: service1
          servicePort: 80
```

is **exactly equivalent** to a `Mapping` CRD of 

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: test-ingress-0-0
spec:
  prefix: /foo/
  service: service1:80
```

This means that the following YAML:

```yaml
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
  name: test-ingress
spec:
  rules:
  - http:
      paths:
      - path: /foo/
        backend:
          serviceName: service1
          servicePort: 80
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: my-mapping
spec:
  prefix: /foo/
  service: service2
```

will set up the Ambassador Edge Stack to do canary routing where 50% of the traffic will go to `service1` and 50% will go to `service2`.

### The Minimal `Ingress`

An `Ingress` resource must provide at least some routes or a [default backend](https://kubernetes.io/docs/concepts/services-networking/ingress/#default-backend). The default backend provides for a simple way to direct all traffic to some upstream service:

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
  name: test-ingress
spec:
  backend:
    serviceName: exampleservice
    servicePort: 8080
```

This is equivalent to

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: test-ingress
spec:
  prefix: /
  service: exampleservice:8080
```

### Name based virtual hosting with an Ambassador Edge Stack ID

```yaml
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
    getambassador.io/ambassador-id: externalid
  name: name-virtual-host-ingress
spec:
  rules:
  - host: foo.bar.com
    http:
      paths:
      - backend:
          serviceName: service1
          servicePort: 80
   - host: bar.foo.com
     http:
       paths:
       - backend:
           serviceName: service2
           servicePort: 80
```

This is equivalent to

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: host-foo-mapping
spec:
  ambassador_id: externalid
  prefix: /
  host: foo.bar.com
  service: service1
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: host-bar-mapping
spec:
  ambassador_id: externalid
  prefix: /
  host: bar.foo.com
  service: service2
```

and will result in all requests to `foo.bar.com` going to `service1`, and requests to `bar.foo.com` going to `service2`.

Read more from Kubernetes [here](https://kubernetes.io/docs/concepts/services-networking/ingress/#name-based-virtual-hosting).

### TLS Termination

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    kubernetes.io/ingress.class: ambassador
  name: tls-example-ingress
spec:
  tls:
  - hosts:
    - sslexample.foo.com
    secretName: testsecret-tls
  rules:
    - host: sslexample.foo.com
      http:
        paths:
        - path: /
          backend:
            serviceName: service1
            servicePort: 80
```

This is equivalent to

```yaml
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: sslexample-termination-context
spec:
  hosts:
  - sslexample.foo.com
  secretName: testsecret-tls
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: sslexample-mapping
spec:
  host: sslexample.foo.com
  prefix: /
  service: service1
```

Note that this shows TLS termination, not origination: the `Ingress` spec does not support origination. Read about Kubernetes TLS termination [here](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls).
