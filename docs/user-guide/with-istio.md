# Ambassador and Istio

---


Ambassador is an API Gateway for microservices. Ambassador is deployed at the edge of your network, and routes incoming traffic to your internal services (aka "north-south" traffic).  [Istio](https://istio.io/) is a service mesh for microservices, and designed to add L7 observability, routing, and resilience to service-to-service traffic (aka "east-west" traffic). Both Istio and Ambassador are built using [Envoy](https://www.envoyproxy.io).

While Istio ships with a basic [ingress controller](https://istio.io/docs/tasks/traffic-management/ingress.html), the ingress controller is very limited. In particular, it provides limited functionality for authentication, edge routing, TLS termination, and so forth. Luckily, Ambassador *does* provide this functionality.

## Getting Ambassador working with Istio

Getting Ambassador working with Istio is straightforward. In this example, we'll use the `bookinfo` example from Istio.

1. Install Istio on Kubernetes, following [the default instructions](https://istio.io/docs/setup/kubernetes/quick-start.html).
2. Next, install the Bookinfo sample application, following the [instructions](https://istio.io/docs/guides/bookinfo.html).
3. Verify that the sample application is working as expected.

By default, the Bookinfo application uses the Istio ingress. To use Ambassador, we need to:

1. Install Ambassador. See the [quickstart](https://www.getambassador.io/user-guide/getting-started) guide.

2. Update the `bookinfo.yaml` manifest to include the necessary Ambassador annotations. See below.

```
apiVersion: v1
kind: Service
metadata:
  name: productpage
  labels:
    app: productpage
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind: Mapping
      name: productpage_mapping
      prefix: /productpage/
      rewrite: /productpage
      service: productpage:9080
spec:
  ports:
  - port: 9080
    name: http
  selector:
    app: productpage
```

3. Optionally, delete the Ingress controller from the `bookinfo.yaml` manifest by typing `kubectl delete ingress gateway`.

4. Test Ambassador out by going to `$AMBASSADOR_IP/productpage/`. You can get the actual IP address for Ambassador by typing `kubectl get services ambassador`.

## Automatic sidecar injection

Newer versions of Istio support Kubernetes initializers to [automatically inject the Istio sidecar](https://istio.io/docs/setup/kubernetes/sidecar-injection.html#automatic-sidecar-injection). With Ambassador, you don't need to inject the Istio sidecar -- Ambassador's Envoy instance will automatically route to the appropriate service(s). If you're using automatic sidecar injection, you'll need to configure Istio to not inject the sidecar automatically for Ambassador pods. There are several approaches to doing this that are [explained in the documentation](https://istio.io/docs/setup/kubernetes/sidecar-injection.html#configuration-options).
