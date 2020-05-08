# Istio Integration

Ambassador Edge Stack and Istio: Edge Proxy and Service Mesh together in one. The Edge Stack is deployed at the edge of your network and routes incoming traffic to your internal services (aka "north-south" traffic). [Istio](https://istio.io/) is a service mesh for microservices, and is designed to add application-level Layer (L7) observability, routing, and resilience to service-to-service traffic (aka "east-west" traffic). Both Istio and the Ambassador Edge Stack are built using [Envoy](https://www.envoyproxy.io).

Ambassador Edge Stack and Istio can be deployed together on Kubernetes. In this configuration, incoming traffic from outside the cluster is first routed through the Ambassador Edge Stack, which then routes the traffic to Istio-powered services. The Ambassador Edge Stack handles authentication, edge routing, TLS termination, and other traditional edge functions.

This allows the operator to have the best of both worlds: a high performance, modern edge service (Ambassador Edge Stack) combined with a state-of-the-art service mesh (Istio). While Istio has introduced a [Gateway](https://istio.io/docs/tasks/traffic-management/ingress/#configuring-ingress-using-an-istio-gateway) abstraction, the Ambassador Edge Stack still has a much broader feature set for edge routing than Istio. For more on this topic, see our blog post on [API Gateway vs Service Mesh](https://blog.getambassador.io/api-gateway-vs-service-mesh-104c01fa4784).

## Getting Ambassador Edge Stack Working With Istio

Getting the Ambassador Edge Stack working with Istio is straightforward. In this example, we'll use the `bookinfo` sample application from Istio.

1. Install Istio on Kubernetes, following [the default instructions](https://istio.io/docs/setup/platform-setup/gke/) (without using mutual TLS auth between sidecars)
2. Next, install the Bookinfo sample application, following the [instructions](https://istio.io/docs/examples/bookinfo/#if-you-are-running-on-kubernetes).
3. Verify that the sample application is working as expected.

By default, the Bookinfo application uses the Istio ingress. To use the Ambassador Edge Stack, we need to:

1. [Install the Ambassador Edge Stack](../../topics/install).\
2. Install a sample `Mapping` in the Ambassador Edge Stack by creating a YAML file named `httpbin.yaml` and paste in the following contents:

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata: 
  name: httpbin
spec:     
  prefix: /httpbin/
  service: httpbin.org
  host_rewrite: httpbin.org
```

Then, apply it to the Kubernetes with `kubectl`:

```shell
kubectl apply -f httpbin.yaml
```

The steps above do several things:

* It creates a Kubernetes service for the Ambassador Edge Stack, of type `LoadBalancer`. Note that if you're not deploying in an environment where `LoadBalancer` is a supported type (i.e. MiniKube), you'll need to change this to a different type of service, e.g., `NodePort`.
* It creates a test route that will route traffic from `/httpbin/` to the public `httpbin.org` HTTP Request and Response service (which provides a useful endpoint that can be used for diagnostic purposes). In the Ambassador Edge Stack, Kubernetes annotations (as shown above) are used for configuration. More commonly, you'll want to configure routes as part of your service deployment process, as shown in [this more advanced example](https://www.datawire.io/faster/canary-workflow/).

You can see if the two Ambassador Edge Stack services are running correctly (and also obtain the LoadBalancer IP address when this is assigned after a few minutes) by executing the following commands:

```shell
$ kubectl get services
NAME               TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)          AGE
ambassador         LoadBalancer   10.63.247.1     35.224.41.XX     8080:32171/TCP     11m
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

Above we see that external IP assigned to our LoadBalancer is 35.224.41.XX (XX is used to mask the actual value), and that all ambassador pods are running (Ambassador Edge Stack relies on Kubernetes to provide high availability, and so there should be two small pods running on each node within the cluster).

You can test if the Ambassador Edge Stack has been installed correctly by using the test route to `httpbin.org` to get the external cluster [Origin IP](https://httpbin.org/ip) from which the request was made:

```shell
$ curl -L 35.224.41.XX/httpbin/ip
{
  "origin": "35.192.109.XX"
}
```

If you're seeing a similar response, then everything is working great!

(Bonus: If you want to use a little bit of awk magic to export the LoadBalancer IP to a variable AMBASSADOR_IP, then you can type `export AMBASSADOR_IP=$(kubectl get services ambassador | tail -1 | awk '{ print $4 }')` and use `curl -L $AMBASSADOR_IP/httpbin/ip`

2. Now you are going to modify the `bookinfo` demo `bookinfo.yaml` manifest to include the necessary Ambassador annotations. See below.

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata: 
  name: productpage
spec:     
  prefix: /productpage/
  rewrite: /productpage
  service: productpage:9080
---
apiVersion: v1
kind: Service
metadata:
  name: productpage
  labels:
    app: productpage
spec:
  ports:
  - port: 9080
    name: http
  selector:
    app: productpage
```

The annotation above implements an Ambassador Edge Stack mapping from the `/productpage/` URI to the Kubernetes productpage service running on port 9080 ('productpage:9080'). The 'prefix' mapping URI is taken from the context of the root of your Ambassador Edge Stack service that is acting as the ingress point (exposed externally via port 80 because it is a LoadBalancer) e.g. '35.224.41.XX/productpage/'.

You can now apply this manifest from the root of the Istio GitHub repo on your local file system (taking care to wrap the apply with `istioctl kube-inject`):

```shell
kubectl apply -f <(istioctl kube-inject -f samples/bookinfo/platform/kube/bookinfo.yaml)
```

3. Optionally, delete the Ingress controller from the `bookinfo.yaml` manifest by typing `kubectl delete ingress gateway`.

4. Test the Ambassador Edge Stack by going to the IP of the Ambassador LoadBalancer you configured above e.g. `35.192.109.XX/productpage/`. You can see the actual IP address again for the Ambassador Edge Stack by typing `kubectl get services ambassador`.

## Automatic Sidecar Injection

Newer versions of Istio support Kubernetes initializers to [automatically inject the Istio sidecar](https://istio.io/docs/setup/kubernetes/additional-setup/sidecar-injection/#automatic-sidecar-injection). You don't need to inject the Istio sidecar into the pods of the Ambassador Edge Stack -- Ambassador's Envoy instance will automatically route to the appropriate service(s). Ambassador Edge Stack's pods are configured to skip sidecar injection, using an annotation as [explained in the documentation](https://istio.io/docs/setup/kubernetes/additional-setup/sidecar-injection/#policy).

## Istio Mutual TLS

Istio versions prior to 1.5 store its TLS certificates as Kubernetes secrets by default, so accessing them is a matter of YAML configuration changes. Istio 1.5 changes how secrets are handled; please contact us on [Slack](https://d6e.co/slack) for more details.

1. Load Istio's TLS certificates

Istio creates and stores its TLS certificates in Kubernetes secrets. In order to use those secrets you can set up a `TLSContext` to read directly from Kubernetes:

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: TLSContext
   metadata:
     name: istio-upstream
   spec:
     secret: istio.default
     secret_namespacing: False
   ```

Please note that if you are using RBAC you may need to reference the `istio` secret for your service account, e.g. if your service account is `ambassador` then your target secret should be `istio.ambassador`. See the [Ambassador Edge Stack with Istio](../../../user-guide/with-istio#istio-mutual-tls) documentation for an example with more information.

2. Configure Ambassador Edge Stack to use this `TLSContext` when making connections to upstream services

   The `tls` attribute in a `Mapping` configuration tells Ambassador Edge Stack to use the `TLSContext` we created above when making connections to upstream services:

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: Mapping
   metadata:
     name: productpage
   spec:
     prefix: /productpage/
     rewrite: /productpage
     service: https://productpage:9080
     tls: istio-upstream
   ```
Note the `tls: istio-upstream`, which lets the Ambassador Edge Stack know which certificate to use when communicating with that service.

Ambassador Edge Stack will now use the certificate stored in the secret to originate TLS to Istio-powered services.

In the definition above we also have TLS termination enabled; please see [the TLS termination tutorial](../../howtos/tls-termination) or the [Host CRD](../../topics/running/host-crd) for more details.

### PERMISSIVE mTLS

Istio can be configured in either [PERMISSIVE](https://istio.io/docs/concepts/security/#permissive-mode) or STRICT mode for mTLS. `PERMISSIVE` mode allows for services to opt-in to mTLS to make the transition easier.

For service-to-service calls via the Istio proxy, Istio will automatically handle this mTLS opt-in when you configure a [DestinationRule](https://istio.io/docs/concepts/traffic-management/#destination-rules). However, since there is no Istio proxy running sidecar to the Ambassador Edge Stack, to do mTLS between Ambassador Edge Stack and an Istio service in `PERMISSIVE` mode, we need to tell the service to listen for mTLS traffic by setting `alpn_protocols: "istio"` in the `TLSContext`:

```yaml
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: istio-upstream
spec:
  secret: istio.default
  secret_namespacing: False
  alpn_protocols: "istio"
```

### Istio RBAC Authorization

While using `istio.default` secret works for mutual TLS only, to be able to interop with [Istio RBAC Authorization](https://istio.io/docs/concepts/security/#authorization) the Ambassador Edge Stack needs to have Istio certificate that matches service account that the Ambassador Edge Stack deployment is using (by default the service account is `ambassador`).

The `istio.default` secret is for `default` service account, as can be seen in the certificate Subject Alternative Name: `spiffe://cluster.local/ns/default/sa/default`.
So when the Ambassador Edge Stack is using this certificate but running under `ambassador` service account the Istio RBAC will not work as expected.

Fortunately, Istio automatically creates a secret for each service account, including `ambassador` service account.
These secrets are named as `istio.{service account name}`.

So if your Ambassador Edge Stack deployment uses `ambassador` service account, the solution is simply to use `istio.ambassador` secret instead of `istio.default` secret.

## Tracing Integration

Istio provides a tracing mechanism based on Zipkin, which is one of the drivers supported by the Ambassador Edge Stack. In order to achieve an end-to-end tracing, it is possible to integrate the Ambassador Edge Stack with Istio's Zipkin.

First, confirm that Istio's Zipkin is up and running in the `istio-system` Namespace:

```shell
$ kubectl get service zipkin -n istio-system
NAME      TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)    AGE
zipkin    ClusterIP   10.102.146.104   <none>        9411/TCP   7m
```

If Istio's Zipkin is up & running on `istio-system` Namespace, add the `TracingService` annotation pointing to it:
```yaml
---
apiVersion: getambassador.io/v2
kind: TracingService
metadata:
  name: tracing
spec:
  service: "zipkin.istio-system:9411"
  driver: zipkin
  config: {}
```

*Note:* We are using the DNS entry `zipkin.istio-system` as well as the port that our service is running, in this case, `9411`. Please see [Distributed Tracing](../../topics/running/services/tracing-service) for more details on Tracing configuration.

## Monitoring/Statistics Integration

Istio also provides a Prometheus service that is an open-source monitoring and alerting system which is supported by the Ambassador Edge Stack as well. It is possible to integrate the Ambassador Edge Stack into Istio's Prometheus to have all statistics and monitoring in a single place.

First, we need to change our Ambassador Edge Stack Deployment to use the [Prometheus StatsD Exporter](https://github.com/prometheus/statsd_exporter) as its sidecar. Do this by applying the [ambassador-rbac-prometheus.yaml](../../../../yaml/ambassador/ambassador-rbac-prometheus.yaml):

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

Now we need to add a `scrape` configuration to Istio's Prometheus so that it can pool data from our Ambassador Edge Stack. This is done by applying the new ConfigMap:

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

*Note:* Assuming ambassador-monitor service is running in the default namespace.

*Note:* You can also add the scrape by hand by using `kubectl` edit, or the dashboard.

After adding the `scrape`, Istio's Prometheus POD needs to be restarted:

```sh
$ export PROMETHEUS_POD=`kubectl get pods -n istio-system | grep prometheus | awk '{print $1}'`
$ kubectl delete pod $PROMETHEUS_POD -n istio-system
```

## Grafana Dashboard

Istio provides a Grafana dashboard service as well, and it is possible to import an Ambassador Edge Stack Dashboard into it, to monitor the Statistics provided by Prometheus. We're going to use [Alex Gervais'](https://twitter.com/alex_gervais) template available on [Grafana's](https://grafana.com/) website under entry [4689](https://grafana.com/dashboards/4698) as a starting point.

First, let's start the port-forwarding for Istio's Grafana service:

```sh
$ kubectl -n istio-system port-forward $(kubectl -n istio-system get pod -l app=grafana -o jsonpath='{.items[0].metadata.name}') 3000:3000 &
```

Now, open Grafana tool by accessing: `http://localhost:3000/`

To install the Ambassador Edge Stack Dashboard:

* Click on Create
* Select Import
* Enter number 4698

Now we need to adjust the Dashboard Port to reflect our Ambassador Edge Stack configuration:

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

Now let's save the changes:

* Click on Save Dashboard in the Top Right corner
