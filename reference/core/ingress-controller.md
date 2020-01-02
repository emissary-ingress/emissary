# Ambassador as an Ingress Controller

An `Ingress` resource is a popular way to expose Kubernetes services to the Internet. In order to use `Ingress` resources, you need to install an [ingress controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/). Ambassador can function as a fully-fledged Ingress controller, making it easy to work with other `Ingress`-oriented tools within the Kubernetes ecosystem.

## When and How to Use the `Ingress` Resource

If you're new to Ambassador and Kubernetes, we'd recommend you start with our [quickstart](/user-guide/getting-started/), instead of using `Ingress`.

If you're a power user and need to integrate with other software that leverages the `Ingress` resource, read on. The `Ingress` specification is very basic, and, as such, does not support many of the features of Ambassador, so you'll be using both `Ingress` resources and `Mapping` resources to manage your Kubernetes services.

## Ambassador `Ingress` Support

Ambassador supports basic core functionality of the  `Ingress` resource, as
defined by the [`Ingress`](https://kubernetes.io/docs/concepts/services-networking/ingress/)
resource itself:

- Basic routing, including the `route` specification and the default backend
  functionality, is supported.
   - it's particularly easy to use a minimal `Ingress` to the Ambassador diagnostic UI
- TLS termination is supported.
   - you can use multiple `Ingress` resources for SNI
- Using the `Ingress` resource in concert with Ambassador CRDs or annotations is supported.
   - this includes Ambassador annotations on the `Ingress` resource itself

Ambassador does **not** extend the basic `Ingress` specification except as follows:

- the `getambassador.io/ambassador-id` annotation allows you to set an Ambassador ID for
  the `Ingress` itself; and

- the `getambassador.io/config` annotation can be provided on the `Ingress` resource, just
  as on a `Service`.
     - note that if you need to set `getambassador.io/ambassador-id` on the `Ingress`, you
       will also need to set `ambassador-id` on resources within the annotation!

### `Ingress` routes and `Mapping`s

Ambassador actually creates `Mapping` objects from the `Ingress` route rules. These `Mapping`
objects interact with `Mapping`s defined in CRDs **exactly** as they would if the `Ingress`
route rules had been specified with CRDs originally.

For example, this `Ingress` resource

```yaml
---
apiVersion: networking.k8s.io/v1beta1
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
apiVersion: getambassador.io/v1
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
apiVersion: networking.k8s.io/v1beta1
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
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: my-mapping
spec:
  prefix: /foo/
  service: service2
```

will set up Ambassador to do canary routing where 50% of the traffic will go to `service1`
and 50% will go to `service2`.

### The Minimal `Ingress`

An `Ingress` resource must provide at least of some routes or a
[default backend](https://kubernetes.io/docs/concepts/services-networking/ingress/#default-backend).
The default backend provides for a simple way to direct all traffic to some upstream
service:

```yaml
apiVersion: networking.k8s.io/v1beta1
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
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: test-ingress
spec:
  prefix: /
  service: exampleservice:8080
```

### [Name based virtual hosting](https://kubernetes.io/docs/concepts/services-networking/ingress/#name-based-virtual-hosting) with an Ambassador ID

```yaml
---
apiVersion: networking.k8s.io/v1beta1
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
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: host-foo-mapping
spec:
  ambassador_id: externalid
  prefix: /
  host: foo.bar.com
  service: service1
---
apiVersion: getambassador.io/v1
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

### [TLS termination](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls)

```yaml
apiVersion: networking.k8s.io/v1beta1
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
apiVersion: getambassador.io/v1
kind: TLSContext
metadata:
  name: sslexample-termination-context
spec:
  hosts:
  - sslexample.foo.com
  secret: testsecret-tls
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: sslexample-mapping
spec:
  host: sslexample.foo.com
  prefix: /
  service: service1
```

Note that this shows TLS termination, not origination: the `Ingress` spec does not
support origination.

### What is required to use the `Ingress` resource?

The following covers the basic requirements to use the `Ingress` resource with Ambassador. 

- You will need RBAC permissions to create `Ingress` resources.

- Ambassador will need RBAC permissions to get, list, watch, and update `Ingress` resources. The
  default Ambassador installation does this for you. If you are using a custom configuration
  of Ambassador, ensure the following rule is added to Ambassador's `Role` or `ClusterRole`:

      - apiGroups: [ "extensions" ]
        resources: [ "ingresses" ]
        verbs: ["get", "list", "watch"]
      - apiGroups: [ "extensions" ]
        resources: [ "ingresses/status" ]
        verbs: ["update"]

- You must create your `Ingress` resource with the correct `ingress.class`.

  Ambassador will automatically read Ingress resources with the annotation
  `kubernetes.io/ingress.class: ambassador`.

- You may need to set your `Ingress` resources' `ambassador-id`.

  If you're not using the `default` ID, you'll need to add the `getambassador.io/ambassador-id`
  annotation to your `Ingress`. See the examples below.

- You must create a `Service` resource with the correct `app.kubernetes.io/component` label.

  Ambassador will automatically load balance Ingress resources using the endpoint exposed 
  from the Service with the annotation `app.kubernetes.io/component: ambassador-service`.
  
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
