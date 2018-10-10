# Tracing

Ambassador can support distributed tracing, one of the ["3 pillars of observability"](https://medium.com/@copyconstruct/monitoring-in-the-time-of-cloud-native-c87c7a5bfa3e), which allows developers to visualize request flows in microservice and service-oriented architectures. In this tutorial, we'll configure Ambassador to initiate a trace on some sample requests, and use an external tracing service to visualize them.

## Before You Get Started

This tutorial assumes you have already followed the [Ambassador Getting Started](/user-guide/getting-started.html) guide. If you haven't done that already, you should do that now.

After completing the Getting Started guide you will have a Kubernetes cluster running Ambassador and the Quote of the Moment service. Let's walk through adding tracing to this setup.

## 1. Deploy Zipkin

In this tutorial you will use a simple deployment of the open source [Zipkin](https://zipkin.io/) distributed tracing system to store and visualize the Ambassador-generated traces. The trace data will be stored in-memory within the
Zipkin container, and you will be able to explore the traces via the Zipkin
web UI.

First, add the following YAML to a file named `zipkin.yaml`. This configuration
will create a zipkin Deployment that uses the [`openzipkin/zipkin`](https://hub.docker.com/r/openzipkin/zipkin/) container image
and also an associated Service. You will notice that the Service also has an
annotation on it that configures Ambassador to use the zipkin service (running on the
default port of 9411) to provide tracing support.

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: zipkin
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind: TracingService
      name: tracing
      service: zipkin:9411
      driver: zipkin
spec:
  selector:
    app: zipkin
  ports:
  - port: 9411
    name: http
    targetPort: http
  type: NodePort
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: zipkin
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: zipkin
    spec:
      containers:
      - name: zipkin
        image: openzipkin/zipkin
        imagePullPolicy: Always
        ports:
        - name: http
          containerPort: 9411
```

You can deploy this configuration into your Kubernetes cluster like so:

```shell
$ kubectl apply -f zipkin.yaml
```

The Ambassador Service will detect the annotations and reconfigure itself within a few seconds.

## 2. Generate some requests

Use `curl` to generate a few requests to an existing Ambassador mapping. You may need to perform many requests since only a subset of random requests are sampled and instrumented with traces.

```shell
$ curl $AMBASSADOR_IP/httpbin/ip
```

## 3. Test traces

To test things out, we'll need to access the Zipkin UI. If you're on Kubernetes, get the name of the Zipkin pod:

```shelll
$ kubectl get pods
NAME                                   READY     STATUS    RESTARTS   AGE
ambassador-5ffcfc798-c25dc             2/2       Running   0          1d
prometheus-prometheus-0                2/2       Running   0          113d
zipkin-868b97667c-58v4r                1/1       Running   0          2h
```

And then use `kubectl port-forward` to access the pod:

```shell
$ kubectl port-forward zipkin-868b97667c-58v4r 9411
```

Open your web browser to `http://localhost:9411` for the Zipkin UI.

If you're on `minikube` you can access the `NodePort` directly, and this ports
number can be obtained via the `minikube services list` command.
If you are using `Docker for Mac/Windows`, you can use the
`kubectl get svc` command to get the same information.

```shell
$ minikube service list
|-------------|----------------------|-----------------------------|
|  NAMESPACE  |         NAME         |             URL             |
|-------------|----------------------|-----------------------------|
| default     | ambassador-admin     | http://192.168.99.107:30319 |
| default     | ambassador           | http://192.168.99.107:31893 |
| default     | zipkin               | http://192.168.99.107:31043 |
|-------------|----------------------|-----------------------------|
```

Open your web browser to the Zipkin dashboard http://192.168.99.107:31043/zipkin/.

In the Zipkin UI, click on the "Find Traces" button to get a listing instrumented traces. Each of the traces that are displayed can be clicked on, which provides further information
about each span and associated metadata.

## More

For more details about configuring the external tracing service, read the documentation on [external tracing](/reference/services/tracing-service).
