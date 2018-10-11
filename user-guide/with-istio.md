# Ambassador and Istio: Edge Proxy and Service Mesh

Ambassador is a Kubernetes-native API gateway for microservices. Ambassador is deployed at the edge of your network, and routes incoming traffic to your internal services (aka "north-south" traffic).  [Istio](https://istio.io/) is a service mesh for microservices, and is designed to add application-level Layer (L7) observability, routing, and resilience to service-to-service traffic (aka "east-west" traffic). Both Istio and Ambassador are built using [Envoy](https://www.envoyproxy.io).

Ambassador and Istio can be deployed together on Kubernetes. In this configuration, incoming traffic from outside the cluster is first routed through Ambassador, which then routes the traffic to Istio-powered services. Ambassador handles authentication, edge routing, TLS termination, and other traditional edge functions.

This allows the operator to have the best of both worlds: a high performance, modern edge service (Ambassador) combined with a state-of-the-art service mesh (Istio). Istio's basic [ingress controller](https://istio.io/docs/tasks/traffic-management/ingress.html) is very limited, and has no support for authentication or many of the other features of Ambassador.

## Getting Ambassador Working With Istio

Getting Ambassador working with Istio is straightforward. In this example, we'll use the `bookinfo` sample application from Istio.

1. Install Istio on Kubernetes, following [the default instructions](https://istio.io/docs/setup/kubernetes/quick-start.html) (without using mutual TLS auth between sidecars)
2. Next, install the Bookinfo sample application, following the [instructions](https://istio.io/docs/guides/bookinfo.html).
3. Verify that the sample application is working as expected.

By default, the Bookinfo application uses the Istio ingress. To use Ambassador, we need to:

1. Install Ambassador.

First you will need to deploy the Ambassador ambassador-admin service to your cluster:

It's simplest to use the YAML files we have online for this (though of course you can download them and use them locally if you prefer!).

First, you need to check if Kubernetes has RBAC enabled:

```shell
kubectl cluster-info dump --namespace kube-system | grep authorization-mode
```
If you see something like `--authorization-mode=Node,RBAC` in the output, then RBAC is enabled.

If RBAC is enabled, you'll need to use:


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
$ kubectl get services
NAME               TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)          AGE
ambassador         LoadBalancer   10.63.247.1     35.224.41.XX     80:32171/TCP     11m
ambassador-admin   NodePort       10.63.250.17    <none>           8877:32107/TCP   12m
details            ClusterIP      10.63.241.224   <none>           9080/TCP         16m
kubernetes         ClusterIP      10.63.240.1     <none>           443/TCP          24m
productpage        ClusterIP      10.63.248.184   <none>           9080/TCP         16m
ratings            ClusterIP      10.63.255.72    <none>           9080/TCP         16m
reviews            ClusterIP      10.63.252.192   <none>           9080/TCP         16m

$ kubectl get pods
NAME                             READY     STATUS    RESTARTS   AGE
ambassador-2680035017-092rk      2/2       Running   0          13m
ambassador-2680035017-9mr97      2/2       Running   0          13m
ambassador-2680035017-thcpr      2/2       Running   0          13m
details-v1-3842766915-3bjwx      2/2       Running   0          17m
productpage-v1-449428215-dwf44   2/2       Running   0          16m
ratings-v1-555398331-80zts       2/2       Running   0          17m
reviews-v1-217127373-s3d91       2/2       Running   0          17m
reviews-v2-2104781143-2nxqf      2/2       Running   0          16m
reviews-v3-3240307257-xl1l6      2/2       Running   0          16m
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

You can now apply this manifest from the root of the Istio GitHub repo on your local file system (taking care to wrap the apply with istioctl kube-inject):

```shell
kubectl apply -f <(istioctl kube-inject -f samples/bookinfo/kube/bookinfo.yaml)
```

3. Optionally, delete the Ingress controller from the `bookinfo.yaml` manifest by typing `kubectl delete ingress gateway`.

4. Test Ambassador by going to the IP of the Ambassador LoadBalancer you configured above e.g. `35.192.109.XX/productpage/`. You can see the actual IP address again for Ambassador by typing `kubectl get services ambassador`.

## Automatic Sidecar Injection

Newer versions of Istio support Kubernetes initializers to [automatically inject the Istio sidecar](https://istio.io/docs/setup/kubernetes/sidecar-injection.html#automatic-sidecar-injection). You don't need to inject the Istio sidecar into Ambassador's pods -- Ambassador's Envoy instance will automatically route to the appropriate service(s). Ambassador's pods are configured to skip sidecar injection, using an annotation as [explained in the documentation](https://istio.io/docs/setup/kubernetes/sidecar-injection.html#policy).

## Istio Mutual TLS

In case Istio mutual TLS is enabled on the cluster, the mapping outlined above will not function correctly as the Istio sidecar will intercept the connections and the service will only be reachable via `https` using the Istio managed certificates, which are available in each namespace via the `istio.default` secret. To get the proxy working we need to tell Ambassador to use those certificates when communicating with Istio enabled service. To do this we need to modify the Ambassador deployment installed above.

In case of RBAC:

``` yaml
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: ambassador
spec:
  replicas: 3
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
      labels:
        service: ambassador
    spec:
      serviceAccountName: ambassador
      containers:
      - name: ambassador
        image: quay.io/datawire/ambassador:0.33.1
        resources:
          limits:
            cpu: 1
            memory: 400Mi
          requests:
            cpu: 200m
            memory: 100Mi
        env:
        - name: AMBASSADOR_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace          
        livenessProbe:
          httpGet:
            path: /ambassador/v0/check_alive
            port: 8877
          initialDelaySeconds: 30
          periodSeconds: 3
        readinessProbe:
          httpGet:
            path: /ambassador/v0/check_ready
            port: 8877
          initialDelaySeconds: 30
          periodSeconds: 3
        volumeMounts:
          - mountPath: /etc/istiocerts/
            name: istio-certs
            readOnly: true
      - name: statsd
        image: quay.io/datawire/statsd:0.33.1
      restartPolicy: Always
      volumes:
      - name: istio-certs
        secret:
          optional: true
          secretName: istio.default
```

Specifically note the mounting of the Istio secrets. For non RBAC cluster modify accordingly. Next we need to modify the Ambassador configuration to tell it use the new certificates for Istio enabled services:

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
      ---
      apiVersion: ambassador/v0
      kind:  Module
      name: tls
      config:
        server:
          enabled: True
          redirect_cleartext_from: 80
        client:
          enabled: False
        upstream:
          cert_chain_file: /etc/istiocerts/cert-chain.pem
          private_key_file: /etc/istiocerts/key.pem
spec:
  type: LoadBalancer
  ports:
  - name: ambassador
    port: 80
    targetPort: 80
  selector:
    service: ambassador
```

This will define an `upstream` that uses the Istio certificates. We can now reuse the `upstream` in all Ambassador mappings to enable communication with Istio pods.

``` yaml
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
      tls: upstream
      service: https://productpage:9080
spec:
  ports:
  - port: 9080
    name: http
    protocol: TCP
  selector:
    app: productpage
```
Note the `tls: upstream`, which lets Ambassador know which certificate to use when communicating with that service.

In the definition above we also have TLS termination enabled; please see [the TLS termination tutorial](https://www.getambassador.io/user-guide/tls-termination) for more details.

## Tracing Integration

Istio provides a tracing mechanism based on Zipkin, which is one of the drivers supported by Ambassador. In order to achieve an end-to-end tracing, it is possible to integrate Ambassador with Istio's Zipkin.  
First confirm that Istio's Zipkin is up and running in the `istio-system` Namespace:

```shell
$ kubectl get service zipkin -n istio-system
NAME      TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)    AGE
zipkin    ClusterIP   10.102.146.104   <none>        9411/TCP   7m
```

If Istio's Zipkin is up & running on `istio-system` Namespace, add the `TracingService` annotation pointing to it:
```yaml
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind: TracingService
      name: tracing
      service: "zipkin.istio-system:9411"
      driver: zipkin
      config: {}
```

*Note:* We are using the DNS entry `zipkin.istio-system` as well as the port that our service is running, in this case `9411`.  
Please see [Distributed Tracing](https://www.getambassador.io/reference/services/tracing-service) for more details on Tracing configuration.

## Monitoring/Statistics Integration

Istio also provides a Prometheus service that is an open-source monitoring and alerting system which is supported by Ambassador as well. It is possible to integrate Ambassador into Istio's Prometheus to have all statistics and monitoring in a single place.  

First we need to change our Ambassador Deployment to use the [Prometheus StatsD Exporter](https://github.com/prometheus/statsd_exporter) as its sidecar. Do this by applying the [ambassador-rbac-prometheus.yaml](https://www.getambassador.io/yaml/ambassador/ambassador-rbac-prometheus.yaml):
```sh
$ kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-rbac-prometheus.yaml
```

This YAML is changing the StatsD container definition on our Deployment to use the Prometheus StatsD Exporter as a sidecar:
```yaml
      - name: statsd-sink
        image: datawire/prom-statsd-exporter:0.6.0
      restartPolicy: Always
```

Next, a Service needs to be created pointing to our `Prometheus StatsD Exporter` sidecar:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: ambassador-monitor
  labels:
    app: ambassador
    service: ambassador-monitor
spec:
  type: ClusterIP
  ports:
   - port: 9102
     name: prometheus-metrics
  selector:
    service: ambassador
```

Now we need to add a `scrape` configuration to Istio's Prometheus so that it can pool data from our Ambassador. This is done by applying the new ConfigMap:
```sh
$ kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-istio-configmap.yaml
```

This ConfigMap YAML changes the `prometheus` ConfigMap that is on `istio-system` Namespace and adds the following:
```yaml
    - job_name: 'ambassador'
      static_configs:
      - targets: ['ambassador-monitor.default:9102']
        labels:  {'application': 'ambassador'}
```

*Note:* Assuming ambassador-monitor service is runnning in default namespace.  

*Note:* You can also add the scrape by hand by using kubectl edit or dashboard.

Afer adding the `scrape`, Istio's Prometheus POD needs to be restarted:
```sh
$ export PROMETHEUS_POD=`kubectl get pods -n istio-system | grep prometheus | awk '{print $1}'`
$ kubectl delete pod $PROMETHEUS_POD -n istio-system
```

More details can be found in [Statistics and Monitoring](https://www.getambassador.io/reference/statistics).


## Grafana Dashboard

Istio provides a Grafana dashboad service as well, and it is possible to import an Ambassador Dashboard into it, to monitor the Statistics provided by Prometheus. We're going to use [Alex Gervais'](https://twitter.com/alex_gervais) template available on [Grafana's](https://grafana.com/) website under entry [4689](https://grafana.com/dashboards/4698) as a starting point.  

First let's start the port-forwarding for Istio's Grafana service:
```sh
$ kubectl -n istio-system port-forward $(kubectl -n istio-system get pod -l app=grafana -o jsonpath='{.items[0].metadata.name}') 3000:3000 &
```

Now, open Grafana tool by acessing: `http://localhost:3000/`

To install Ambassador Dashboard:

* Click on Create  
* Select Import  
* Enter number 4698  

Now we need to adjust the Dashboard Port to reflect our Ambassador configuration:

* Open the Imported Dashboard
* Click on Settings in the Top Right corner
* Click on Variables
* Change the port to 80 (according to the ambassador service port)

Next, adjust the Dashboard Registered Services metric:

* Open the Imported Dashboard
* Find Registered Services
* Click on the down arrow and select Edit
* Change the Metric to:
```yaml
envoy_cluster_manager_active_clusters{job="ambassador"}
```

Now lets save the changes:
* Click on Save Dashboard in the Top Right corner

## Roadmap

There are a number of roadmap items that we'd like to tackle in improving Istio integration. This includes supporting Istio routing rules in Ambassador and full propagation of request headers (e.g., Zipkin tracing) between Ambassador and Istio. If you're interested in contributing, we'd welcome the help!
