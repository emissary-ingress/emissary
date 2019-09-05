# Deploying Ambassador to Kubernetes

In this tutorial, we'll walk through the process of deploying Ambassador in Kubernetes for ingress routing. Ambassador provides all the functionality of a traditional ingress controller (i.e., path-based routing) while exposing many additional capabilities such as [authentication](/user-guide/auth-tutorial), URL rewriting, CORS, rate limiting, and automatic metrics collection (the [mappings reference](/reference/mappings) contains a full list of supported options). For more background on Kubernetes ingress, [read this blog post](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d).

Ambassador is designed to allow service authors to control how their service is published to the Internet. We accomplish this by permitting a wide range of annotations on the *service*, which Ambassador reads to configure its Envoy Proxy. Below, we'll use service annotations to configure Ambassador to map `/httpbin/` to `httpbin.org`.

## 1. Deploying Ambassador

To deploy Ambassador in your **default** namespace, first you need to check if Kubernetes has RBAC enabled:

```shell
kubectl cluster-info dump --namespace kube-system | grep authorization-mode
```

If you see something like `--authorization-mode=Node,RBAC` in the output, then RBAC is enabled. The majority of current hosted Kubernetes providers (such as GKE) create
clusters with RBAC enabled by default, and unfortunately the above command may not return any information indicating this.

Note: If you're using Google Kubernetes Engine with RBAC, you'll need to grant permissions to the account that will be setting up Ambassador. To do this, get your official GKE username, and then grant `cluster-admin` role privileges to that username:

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
Ambassador Diagnostic web UI), along with an ambassador ClusterRole, ServiceAccount and ClusterRoleBinding (if RBAC is enabled). An Ambassador Deployment is also created.

When not installing Ambassador into the default namespace you must update the namespace used in the `ClusterRoleBinding` when RBAC is enabled.

For production configurations, we recommend you download these YAML files as your starting point, and customize them accordingly.


## 2. Defining the Ambassador Service

Ambassador is deployed as a Kubernetes Service that references the ambassador
Deployment you deployed previously. Create the following YAML and put it in a file called `ambassador-service.yaml`.

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

The YAML above creates a Kubernetes service for Ambassador of type `LoadBalancer`, and configures the `externalTrafficPolicy` to propagate [the original source IP](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip) of the client. All HTTP traffic will be evaluated against the routing rules you create. Note that if you're not deploying in an environment where `LoadBalancer` is a supported type (such as minikube), you'll need to change this to a different type of service, e.g., `NodePort`.

If you have a static IP provided by your cloud provider you can set as `loadBalancerIP`.

## 3. Creating your first service

Create the following YAML and put it in a file called `tour.yaml`.

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: tour
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind: Mapping
      name: tour-ui_mapping
      prefix: /
      service: tour:5000
      ---
      apiVersion: ambassador/v1
      kind: Mapping
      name: tour-backend_mapping
      prefix: /backend/
      service: tour:8080
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
        image: quay.io/datawire/tour:ui-%tourVersion%
        ports:
        - name: http
          containerPort: 5000
      - name: quote
        image: quay.io/datawire/tour:backend-%tourVersion%
        ports:
        - name: http
          containerPort: 8080
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi

```

Then, apply it to the Kubernetes with `kubectl`:

```shell
$ kubectl apply -f tour.yaml
```

This YAML has also been published so you can deploy it remotely:

```
kubectl apply -f https://getambassador.io/yaml/tour/tour.yaml
```

When the service is deployed, Ambassador will notice the `getambassador.io/config` annotation on the service, and use the `Mapping` contained in it to configure the route.  (There's no restriction on what kinds of Ambassador configuration can go into the annotation, but it's important to note that Ambassador only looks at annotations on Kubernetes `Service`s.)

In this case, the mapping creates two routes. One that will route traffic from the `/` endpoint to the `tour-ui` React application and one that will route traffic from the `/backend/` endpoint to the `tour-backend` service. Note the port assignments in the `service` field of the `Mapping`. This allows us to use a single service to route to both the containers running on the `tour` pod.

## 4. Testing the Mapping

To test things out, we'll need the external IP for Ambassador (it might take some time for this to be available):

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

Ambassador includes an integrated diagnostics service to help with troubleshooting. 

By default, this is exposed to the internet at the URL `http://{{AMBASSADOR_HOST}}/ambassador/v0/diag/`. Go to that URL from a web browser to view the diagnostic UI.

You can change the default so it is not exposed externally by default by setting `diagnostics.enabled: false` in the [ambassador `Module`](/reference/core/ambassador). 

```yaml
apiVersion: ambassador/v1
kind: Module
name: ambassador
config:
  diagnostics:
    enabled: false
```

After applying this `Module`, to view the diagnostics UI, we'll need to get the name of one of the Ambassador pods:

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

Ambassador's versatile HTTPS configuration lets it support various HTTPS use cases whether simple or complex. 

Follow our [enabling HTTPS guide](/user-guide/tls-termination) to quickly enable HTTPS support for your applications.


## Next Steps

We've just done a quick tour of some of the core features of Ambassador: diagnostics, routing, configuration, and authentication.

- Join us on [Slack](https://d6e.co/slack);
- Learn how to [add authentication](/user-guide/auth-tutorial) to existing services; or
- Learn how to [add rate limiting](/user-guide/rate-limiting-tutorial) to existing services; or
- Learn how to [add tracing](/user-guide/tracing-tutorial); or
- Learn how to [use gRPC with Ambassador](/user-guide/grpc); or
- Read about [configuring Ambassador](/reference/configuration).
