# Ambassador Open Source Software (OSS)

In this tutorial, we'll walk through the process of deploying Ambassador Open Source in Kubernetes for ingress routing. Ambassador OSS provides all the functionality of a traditional ingress controller (i.e., path-based routing) while exposing many additional capabilities such as [authentication](/user-guide/auth-tutorial), URL rewriting, CORS, rate limiting, and automatic metrics collection (the [mappings reference](/reference/mappings) contains a full list of supported options). Note that Ambassador Edge Stack can be used as an [Ingress Controller](/user-guide/ingress-controller).

For more background on Kubernetes ingress, [read this blog post](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d).

Ambassador Open Source is designed to allow service authors to control how their service is published to the Internet. We accomplish this by permitting a wide range of annotations on the *service*, which Ambassador OSS reads to configure its Envoy Proxy. Below, we'll use service annotations to configure Ambassador OSS to map `/httpbin/` to `httpbin.org`.

## 1. Deploying Ambassador Open Source

To deploy Ambassador Open Source in your **default** namespace, first you need to check if Kubernetes has RBAC enabled:

```shell
kubectl cluster-info dump --namespace kube-system | grep authorization-mode
```

If you see something like `--authorization-mode=Node,RBAC` in the output, then RBAC is enabled. The majority of current hosted Kubernetes providers (such as GKE) create
clusters with RBAC enabled by default, and unfortunately the above command may not return any information indicating this.

**Note:** If you're using Google Kubernetes Engine with RBAC, you'll need to grant permissions to the account that will be setting up Ambassador OSS. To do this, get your official GKE username, and then grant `cluster-admin` role privileges to that username:

```
$ kubectl create clusterrolebinding my-cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud info --format="value(config.account)")
```

If RBAC is enabled:

```shell
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-rbac.yaml
```

Without RBAC, you can use:

```shell
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-no-rbac.yaml
```

We recommend downloading the YAML files and exploring the content. You will see
that an `ambassador-admin` NodePort Service is created (which provides an
Ambassador ODD Diagnostic web UI), along with an ambassador ClusterRole, ServiceAccount and ClusterRoleBinding (if RBAC is enabled). An Ambassador Open Source Deployment is also created.

When not installing Ambassador Open Source into the default namespace you must update the namespace used in the `ClusterRoleBinding` when RBAC is enabled.

For production configurations, we recommend you download these YAML files as your starting point, and customize them accordingly.


## 2. Defining the Ambassador Open Source Service

Ambassador Open Source is deployed as a Kubernetes Service that references the ambassador Deployment you deployed previously. Create the following YAML and put it in a file called`ambassador-service.yaml`.

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local
  ports:
   - port: 80
     targetPort: 8080
  selector:
    service: ambassador
```

Deploy this service with `kubectl`:

```shell
$ kubectl apply -f ambassador-service.yaml
```

The YAML above creates a Kubernetes service for Ambassador Open Source of type `LoadBalancer`, and configures the `externalTrafficPolicy` to propagate [the original source IP](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip) of the client. All HTTP traffic will be evaluated against the routing rules you create. Note that if you're not deploying in an environment where `LoadBalancer` is a supported type (such as minikube), you'll need to change this to a different type of service, e.g., `NodePort`.

If you have a static IP provided by your cloud provider you can set as `loadBalancerIP`.

## 3. Creating your first service

Create the following YAML and put it in a file called `tour.yaml`.

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: tour
spec:
  ports:
  - name: ui
    port: 5000
    targetPort: 5000
  - name: backend
    port: 8080
    targetPort: 8080
  selector:
    app: tour
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tour
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tour
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: tour
    spec:
      containers:
      - name: tour-ui
        image: quay.io/datawire/tour:ui-$tourVersion$
        ports:
        - name: http
          containerPort: 5000
      - name: quote
        image: quay.io/datawire/tour:backend-$tourVersion$
        ports:
        - name: http
          containerPort: 8080
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: tour-ui
spec:
  prefix: /
  service: tour:5000
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: tour-backend
spec:
  prefix: /backend/
  service: tour:8080
  labels:
    ambassador:
      - request_label:
        - backend
```

Then, apply it to the Kubernetes with `kubectl`:

```shell
$ kubectl apply -f tour.yaml
```

This YAML has also been published so you can deploy it remotely:

```
kubectl apply -f https://getambassador.io/yaml/tour/tour.yaml
```

When the `Mapping` CRDs are applied, Ambassador Open Source will use them to configure routing:

- The first `Mapping` causes traffic from the `/` endpoint to be routed to the `tour-ui` React application.
- The second `Mapping` causes traffic from the `/backend/` endpoint to be routed to the `tour-backend` service.

Note also the port numbers in the `service` field of the `Mapping`. This allows us to use a single service to route to both the containers running on the `tour` pod.

<font color=#f9634E>**Important:**</font>

Routing in Ambassador Open Source can be configured with Ambassador OSS resources as shown above, Kubernetes service annotation, and Kubernetes Ingress resources.

Ambassador OSS ustom resources are the recommended config format and will be used throughout the documentation.

See [configuration format](/reference/config-format) for more information on your configuration options.

## 4. Testing the Mapping

To test things out, we'll need the external IP for Ambassador Open Source (it might take some time for this to be available):

```shell
kubectl get svc -o wide ambassador
```

Eventually, this should give you something like:

```
NAME         CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
ambassador   10.11.12.13     35.36.37.38     80:31656/TCP   1m
```


You should now be able to reach the `tour-ui` application from a web browser:

http://35.36.37.38/

or on minikube:

```shell
$ minikube service list
|-------------|----------------------|-----------------------------|
|  NAMESPACE  |         NAME         |             URL             |
|-------------|----------------------|-----------------------------|
| default     | ambassador-admin     | http://192.168.99.107:30319 |
| default     | ambassador           | http://192.168.99.107:31893 |
|-------------|----------------------|-----------------------------|
```
http://192.168.99.107:31893/

or on Docker for Mac/Windows:

```shell
$ kubectl get svc
NAME               TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)          AGE
ambassador         LoadBalancer   10.106.108.64    localhost     80:32324/TCP     13m
ambassador-admin   NodePort       10.107.188.149   <none>        8877:30993/TCP   14m
tour               ClusterIP      10.107.77.153    <none>        80/TCP           13m
kubernetes         ClusterIP      10.96.0.1        <none>        443/TCP          84d
```
http://localhost/

## 5. The Diagnostics Service in Kubernetes

Ambassador Open Source includes an integrated diagnostics service to help with troubleshooting. 

By default, this is exposed to the internet at the URL `http://{{AMBASSADOR_HOST}}/ambassador/v0/diag/`. Go to that URL from a web browser to view the diagnostic UI.

You can change the default so it is not exposed externally by default by setting `diagnostics.enabled: false` in the [ambassador `Module`](/reference/core/ambassador).

```yaml
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    diagnostics:
      enabled: false
```

After applying this `Module`, to view the diagnostics UI, we'll need to get the name of one of the Ambassador Open Source pods:

```
$ kubectl get pods
NAME                          READY     STATUS    RESTARTS   AGE
ambassador-3655608000-43x86   1/1       Running   0          2m
ambassador-3655608000-w63zf   1/1       Running   0          2m
```

Forwarding local port 8877 to one of the pods:

```
kubectl port-forward ambassador-3655608000-43x86 8877
```

will then let us view the diagnostics at http://localhost:8877/ambassador/v0/diag/.

## 6. Enable HTTPS

The versatile HTTPS configuration of Ambassador Open Source lets it support various HTTPS use cases whether simple or complex.

Follow our [enabling HTTPS guide](/user-guide/tls-termination) to quickly enable HTTPS support for your applications.

## Want more?

For more features, check out the latest build of [Ambassador Edge Stack](/user-guide/install).