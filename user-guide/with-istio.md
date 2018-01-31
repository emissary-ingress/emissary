# Ambassador and Istio: Edge proxy and service mesh

---

Ambassador is a Kubernetes-native API Gateway for microservices. Ambassador is deployed at the edge of your network, and routes incoming traffic to your internal services (aka "north-south" traffic).  [Istio](https://istio.io/) is a service mesh for microservices, and is designed to add application-level Layer (L7) observability, routing, and resilience to service-to-service traffic (aka "east-west" traffic). Both Istio and Ambassador are built using [Envoy](https://www.envoyproxy.io).

Ambassador and Istio can be deployed together on Kubernetes. In this configuration, incoming traffic from outside the cluster is first routed through Ambassador, which then routes the traffic to Istio. Ambassador handles authentication, edge routing, TLS termination, and other traditional edge functions.

This allows the operator to have the best of both worlds: a high performance, modern edge service (Ambassador) combined with a state-of-the-art service mesh (Istio). Istio's basic [ingress controller](https://istio.io/docs/tasks/traffic-management/ingress.html) is very limited, and has no support for authentication or many of the other features of Ambassador.

## Getting Ambassador working with Istio

Getting Ambassador working with Istio is straightforward. In this example, we'll use the `bookinfo` sample application from Istio.

1. Install Istio on Kubernetes, following [the default instructions](https://istio.io/docs/setup/kubernetes/quick-start.html) (without using mutual TLS auth between sidecars)
2. Next, install the Bookinfo sample application, following the [instructions](https://istio.io/docs/guides/bookinfo.html).
3. Verify that the sample application is working as expected.

By default, the Bookinfo application uses the Istio ingress. To use Ambassador, we need to:

1. Install Ambassador.

First you will need to deploy the Ambassador ambassador-admin service to your cluster:

It's simplest to use the YAML files we have online for this (though of course you can download them and use them locally if you prefer!). If you're using a cluster with RBAC enabled, you'll need to use:

```shell
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-rbac.yaml
```

Without RBAC, you can use:

```shell
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-no-rbac.yaml
```
(Note that if you are planning to use mutual TLS for communication between Ambassador and Istio/services in the future, then the order in which you deploy the ambassador-admin service and the ambassador LoadBalancer service below may need to be swapped)

Next you will deploy an ambassador service that acts as a point of ingress into the cluster via the LoadBalancer type. Create the following YAML and put it in a file called `ambassador-service.yaml`.

```yaml
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: ambassador
  name: ambassador
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind:  Mapping
      name:  httpbin_mapping
      prefix: /httpbin/
      service: httpbin.org:80
      host_rewrite: httpbin.org
spec:
  type: LoadBalancer
  ports:
  - name: ambassador
    port: 80
    targetPort: 80
  selector:
    service: ambassador
```

Then, apply it to the Kubernetes with `kubectl`:

```shell
kubectl apply -f ambassador-service.yaml
```

The YAML above does several things:

* It creates a Kubernetes service for Ambassador, of type `LoadBalancer`. Note that if you're not deploying in an environment where `LoadBalancer` is a supported type (i.e. MiniKube), you'll need to change this to a different type of service, e.g., `NodePort`.
* It creates a test route that will route traffic from `/httpbin/` to the public `httpbin.org` HTTP Request and Response service (which provides useful endpoint that can be used for diagnostic purposes). In Ambassador, Kubernetes annotations (as shown above) are used for configuration. More commonly, you'll want to configure routes as part of your service deployment process, as shown in [this more advanced example](https://www.datawire.io/faster/canary-workflow/).

You can see if the two Ambassador services are running correctly (and also obtain the LoadBalancer IP address when this is assigned after a few minutes) by executing the following commands:

```shell
$ kubectl get svc
NAME               TYPE           CLUSTER-IP      EXTERNAL-IP    PORT(S)          AGE
ambassador         LoadBalancer   10.63.252.13    35.224.41.XX   80:32474/TCP     52s
ambassador-admin   NodePort       10.63.240.197   <none>         8877:32425/TCP   41s
kubernetes         ClusterIP      10.63.240.1     <none>         443/TCP          8m

$ kubectl get pods
NAME                          READY     STATUS    RESTARTS   AGE
ambassador-2680035017-2vzlt   2/2       Running   0          38s
ambassador-2680035017-qx769   2/2       Running   0          38s
ambassador-2680035017-vr2cd   2/2       Running   0          38s
```

Above we see that external IP assigned to our LoadBalancer is 35.224.41.XX (XX is used to mask the actual value), and that all ambassador pods are running (Ambassador relies on Kubernetes to provide high availability, and so there should be two small pods running on each node within the cluster).

You can test if Ambassador has been installed correctly by using the test route to `httpbin.org` to get the external cluster [Origin IP](https://httpbin.org/ip) from which the request was made:

```shell
$ curl 35.224.41.XX/httpbin/ip
{
  "origin": "35.192.109.XX"
}
```

If you're seeing a similar response, then everything is working great!

(Bonus: If you want to use a little bit of awk magic to export the LoadBalancer IP to a variable AMBASSADOR_IP, then you can type `export AMBASSADOR_IP=$(kubectl get services ambassador | tail -1 | awk '{ print $4 }')` and use `curl $AMBASSADOR_IP/httpbin/ip`

2. Now you are going to modify the bookinfo demo `bookinfo.yaml` manifest to include the necessary Ambassador annotations. See below.

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

The annotation above implements an Ambassador mapping from the '/productpage/' URI to the Kubernetes productpage service running on port 9080 ('productpage:9080'). The 'prefix' mapping URI is taken from the context of the root of your Ambassador service that is acting as the ingress point (exposed externally via port 80 because it is a LoadBalancer) e.g. '35.224.41.XX/productpage/'.

You can now apply this manifest from the root of the Istio GitHub repo on your local file system:

```shell
kubectl apply -f samples/bookinfo/kube/bookinfo.yaml
```

3. Optionally, delete the Ingress controller from the `bookinfo.yaml` manifest by typing `kubectl delete ingress gateway`.

4. Test Ambassador by going to the IP of the Ambassador LoadBalancer you configured above e.g. `35.192.109.XX/productpage/`. You can see the actual IP address again for Ambassador by typing `kubectl get services ambassador`.

## Automatic sidecar injection

Newer versions of Istio support Kubernetes initializers to [automatically inject the Istio sidecar](https://istio.io/docs/setup/kubernetes/sidecar-injection.html#automatic-sidecar-injection). With Ambassador, you don't need to inject the Istio sidecar -- Ambassador's Envoy instance will automatically route to the appropriate service(s). If you're using automatic sidecar injection, you'll need to configure Istio to not inject the sidecar automatically for Ambassador pods. There are several approaches to doing this that are [explained in the documentation](https://istio.io/docs/setup/kubernetes/sidecar-injection.html#configuration-options).
