# Zipkin Tracing

In this tutorial, we'll configure Ambassador Edge Stack to initiate a trace on some sample requests, and use Zipkin to visualize them.

## Before You Get Started

This tutorial assumes you have already followed the Ambassador Edge Stack [Getting Started](../tutorials/getting-started) guide. If you haven't done that already, you should do that now.

After completing the Getting Started guide you will have a Kubernetes cluster running Ambassador Edge Stack and the Quote of the Moment service. Let's walk through adding tracing to this setup.

## 1. Deploy Zipkin

In this tutorial, you will use a simple deployment of the open-source [Zipkin](https://zipkin.io/) distributed tracing system to store and visualize the Ambassador Edge Stack-generated traces. The trace data will be stored in memory within the Zipkin container, and you will be able to explore the traces via the Zipkin web UI.

First, add the following YAML to a file named `zipkin.yaml`. This configuration will create a Zipkin Deployment that uses the [openzipkin/zipkin](https://hub.docker.com/r/openzipkin/zipkin/) container image and also an associated Service. We will also include a `TracingService` that configures Ambassador Edge Stack to use the Zipkin service (running on the default port of 9411) to provide tracing support.

```yaml
---
apiVersion: getambassador.io/v2
kind: TracingService
metadata:
  name: tracing
spec:
  service: "zipkin:9411"
  driver: zipkin
  config: {}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zipkin
spec:
  replicas: 1
  selector:
    matchLabels:
      app: zipkin
  template:
    metadata:
      labels:
        app: zipkin
    spec:
      containers:
        - name: zipkin
          image: openzipkin/zipkin
          env:
            # note: in-memory storage holds all data in memory, purging older data upon a span limit.
            #       you should use a proper storage in production environments
            - name: STORAGE_TYPE
              value: mem
---
apiVersion: v1
kind: Service
metadata:
  labels:
    name: zipkin
  name: zipkin
spec:
  ports:
    - port: 9411
      targetPort: 9411
  selector:
    app: zipkin
```

You can deploy this configuration into your Kubernetes cluster like so:

```shell
$ kubectl apply -f zipkin.yaml
```

**Important:** the Ambassador Edge Stack will need to be restarted to configure itself to add the tracing header. Delete all Ambassador Edge Stack pods and let Kubernetes restart them.

## 2. Generate Some Requests

Use `curl` to generate a few requests to an existing Ambassador Edge Stack mapping. You may need to perform many requests since only a subset of random requests are sampled and instrumented with traces.

```shell
$ curl -L $AMBASSADOR_IP/httpbin/ip
```

## 3. Test Traces

To test things out, we'll need to access the Zipkin UI. If you're on Kubernetes, get the name of the Zipkin pod:

```shell
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

If you're on `minikube` you can access the `NodePort` directly, and this ports number can be obtained via the `minikube services list` command. If you are using `Docker for Mac/Windows`, you can use the `kubectl get svc` command to get the same information.

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

Open your web browser to the Zipkin dashboard `http://192.168.99.107:31043/zipkin/`.

In the Zipkin UI, click on the "Find Traces" button to get a listing instrumented traces. Each of the traces that are displayed can be clicked on, which provides further information about each span and associated metadata.

## Learn More

For more details about configuring the external tracing service, read the documentation on [external tracing](../topics/running/services/tracing-service).
