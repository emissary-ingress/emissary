# Tracing

Ambassador can enable distributed traces, one of the "3 pillars of observability", allowing developers to visualize request flows in service-oriented architectures. In this tutorial, we'll configure Ambassador to initiate a trace on some sample requests and use an external tracing service to visualize them.

## Before You Get Started

This tutorial assumes you have already followed the [Ambassador Getting Started](/user-guide/getting-started.html) guide. If you haven't done that already, you should do that now.

After completing [Getting Started](/user-guide/getting-started.html), you'll have a Kubernetes cluster running Ambassador and the Quote of the Moment service. Let's walk through adding tracing to this setup.

## 1. Deploy Zipkin

In this tutorial we will use a simple in-memory deployment of [Zipkin](https://zipkin.io/) to store and visualize the Ambassador-generated traces.

> Zipkin is a distributed tracing system. It helps gather timing data needed to troubleshoot latency problems in microservice architectures. It manages both the collection and lookup of this data.

Here's the YAML we'll start with:

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

This configuration tells Ambassador about the tracing service, notably that Zipkin API is listening on `zipkin:9411`.

Ambassador will see the annotations and reconfigure itself within a few seconds.

## 2. Generate some requests

Use `curl` to generate a few requests to an existing Ambassador mapping. You may need to perform many requests since only a subset of random requests are sampled and instrumented with traces.

```shell
$ curl http://192.168.99.107:31893/httpbin/
```

## 3. Test traces

To test things out, we'll need the external IP for Zipkin (it might take some time for this to be available):

```shell
kubectl get svc -o wide zipkin
```

Eventually, this should give you something like:

```
NAME         CLUSTER-IP      EXTERNAL-IP     PORT(S)          AGE
zipkin       10.11.12.13     35.36.37.38     9411:31043/TCP   1m
```

or on minikube: 

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

Open your web browser to the Zipkin dashboard http://192.168.99.107:31043/zipkin/ and click the "Find Traces" button to get a listing instrumented traces.

## More

For more details about configuring the external tracing service, read the documentation on [external tracing](/reference/services/tracing-service.md).
