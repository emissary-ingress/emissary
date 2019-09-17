# Ambassador as an Ingress Controller

Besides CRDs and annotations, Ambassador can also be configured using Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resources which manage external access to the services in a cluster. Ambassador acts an [Ingress Controller](https://kubernetes.io/docs/concepts/services-networking/ingress-controllers/) which reads Ingress resources and configures accordingly.

**Note:** No additional configuration is required from user's side, Ambassador will automatically read Ingress resources with the annotation `kubernetes.io/ingress.class: ambassador`.

**Note:** If you're running multiple Ambassadors in a cluster and wish to scope an ingress resource to a particular Ambassador ID, use the annotation `getambassador.io/ambassador-id: <ambassador id>` in the Ingress resource.


#### Examples

1. An Ingress resource with a [default backend](https://kubernetes.io/docs/concepts/services-networking/ingress/#default-backend):
   ```yaml
   service/networking/ingress.yaml 
   apiVersion: networking.k8s.io/v1beta1
   kind: Ingress
   metadata:
     annotations:
       kubernetes.io/ingress.class: ambassador
     name: test-ingress
   spec:
     backend:
       serviceName: exampleservice
       servicePort: 80
   ```

2. [Name based virtual hosting](https://kubernetes.io/docs/concepts/services-networking/ingress/#name-based-virtual-hosting) with an Ambassador ID:
   ```yaml
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

3. [TLS](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls):
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

**FAQ:**
1. When should I configure Ambassador using Ingress resources instead of annotations or CRDs?
   
   By design, Ingress resources manage basic external access to the services, typically HTTP. You can perform basic load balancing, SSL termination, name-based virtual hosting, etc with Ingress resources, but for more advanced routing and features like authentication, security, rate limiting, etc, it's better to configure Ambassador via annotations/CRDs.
   
2. Can I configure Ambassador only with Ingress resources?

   Yes, you can!
   
3. Can I use Ingress resources with annotations/CRDs?

   Yes, you can! Under the hood, Ambassador simply converts Ingress resources to Ambassador Mappings and TLS Contexts - so yes, different form of inputs can co-exist.