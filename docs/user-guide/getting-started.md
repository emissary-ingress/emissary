# Getting Started with Ambassador

Ambassador is a microservices API Gateway. We'll do a quick tour of Ambassador with a demo configuration, before walking through how to deploy Ambassador in Kubernetes with a custom configuration.

## 1. Running the demo configuration

By default, Ambassador uses a demo configuration to show some of its basic features. Get it running with Docker, and expose Ambassador on port 8080:

```shell
docker run -it -p 8080:80 --name=ambassador --rm datawire/ambassador:{VERSION} --demo
```

## 2. Ambassador's Diagnostics

Ambassador provides live diagnostics viewable with a web browser. While this would normally not be exposed to the public network, the Docker demo publishes the diagnostics service at the following URL:

`http://localhost:8080/ambassador/v0/diag/`

Some of the most important information - your Ambassador version, how recently Ambassador's configuration was updated, and how recently Envoy last reported status to Ambassador - is right at the top. The diagnostics overview can show you what it sees in your configuration map, and which Envoy objects were created based on your configuration.

## 3. The Quote of the Moment service

Since Ambassador is an API gateway, its primary purpose is to provide access to microservices. The demo is preconfigured with a mapping that connects the `/qotm/` resource to the "Quote of the Moment" service -- a demo service that supplies quotations. You can try it out here:

```shell
curl http://localhost:8080/qotm/
```

This request will route to the `qotm` service at `demo.getambassador.io`, and return a quote in a JSON object.

You can also see the mapping by clicking the `mapping-qotm.yaml` link from the diagnostic overview, or by opening

`http://localhost:8080/ambassador/v0/diag/mapping-qotm.yaml`

## 4. Authentication

On the diagnostic overview, you can also see that Ambassador is configured to do authentication -- click the `auth.yaml` link, or open

`http://localhost:8080/ambassador/v0/diag/auth.yaml`

for more here. Ambassador uses a demo authentication service at `demo.getambassador.io` to mediate access to the Quote of the Moment: simply getting a random quote is allowed without authentication, but to get a specific quote, you'll have to authenticate:

```shell
curl -v http://localhost:8080/qotm/quote/5
```

will return a 401, but

```shell
curl -v -u username:password http://localhost:8080/qotm/quote/5
```

will succeed. (Note that that's literally "username" and "password" -- the demo auth service is deliberately not very secure!)

Note that it's up to the auth service to decide what needs authentication -- teaming Ambassador with an authentication service can be as flexible or strict as you need it to be.

## 5. Ambassador in Kubernetes

So far, we've used a demo configuration, and run everything in our local Docker instance. We'll now switch to Kubernetes, using service annotations to configure Ambassador to map `/httpbin/` to `httpbin.org`.

### 5.1 Defining the Ambassador Service

Ambassador is deployed as a Kubernetes service. Create the following YAML and put it in a file called `ambassador-service.yaml`.

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

* It creates a Kubernetes service for Ambassador, of type `LoadBalancer`. Note that if you're not deploying in an environment where `LoadBalancer` is a supported type, you'll need to change this to a different type of service, e.g., `NodePort`.
* It creates a test route that will route traffic from `/httpbin/` to the public `httpbin.org` service. In Ambassador, Kubernetes annotations (as shown above) are used for configuration. More commonly, you'll want to configure routes as part of your service deployment process, as shown in [this more advanced example](https://www.datawire.io/faster/canary-workflow/).

Also, note that we are using the `host_rewrite` attribute for the `httpbin_mapping` -- this forces the HTTP `Host` header, and is often a good idea when mapping to external services. Ambassador supports [many different configuration options](/reference/configuration).

### 5.2 Deploying Ambassador

Once that's done, we need to get Ambassador actually running. It's simplest to use the YAML files we have online for this (though of course you can download them and use them locally if you prefer!). If you're using a cluster with RBAC enabled, you'll need to use

```shell
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-rbac.yaml
```

Without RBAC, you can use

```shell
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-no-rbac.yaml
```

When Ambassador starts, it will notice the `getambassador.io/config` annotation on its own service, and use the `Mapping` contained in it to configure itself. (There's no restriction on what kinds of Ambassador configuration can go into the annotation, but it's important to note that Ambassador only looks at annotations on Kubernetes `service`s.)

### 5.3 Testing the Mapping

To test things out, we'll need the external IP for Ambassador (it might take some time for this to be available):

```shell
kubectl get svc ambassador
```

Eventually, this should give you something like:

```
NAME         CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
ambassador   10.11.12.13     35.36.37.38     80:31656/TCP   1m
```

You should now be able to use `curl` to `httpbin` (don't forget the trailing `/`):

```shell
$ curl 35.36.37.38/httpbin/
```

## 6. Adding a Service

You can add a service just by deploying it with an appropriate annotation. For example, we can deploy the QoTM service locally in this cluster and automatically map it through Ambassador by creating `qotm.yaml` with the following:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: qotm
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind:  Mapping
      name:  qotm_mapping
      prefix: /qotm/
      service: qotm
spec:
  type: ClusterIP
  selector:
    app: qotm
  ports:
  - port: 80
    name: http-qotm
    targetPort: http-api
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: qotm
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: qotm
    spec:
      containers:
      - name: qotm
        image: datawire/qotm:1.1
        ports:
        - name: http-api
          containerPort: 5000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
```

and then applying it with

```
kubectl apply -f qotm.yaml
```

A few seconds after the QoTM service is running, Ambassador should be configured for it. Try it with

```shell
$ curl 35.36.37.38/qotm/
```

## 7. The Diagnostics Service in Kubernetes

Note that we did not expose the diagnostics port for Ambassador, since we don't want to expose it on the Internet. To view it, we'll need to get the name of one of the ambassador pods:

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

## 8. Next

We've just done a quick tour of some of the core features of Ambassador: diagnostics, routing, configuration, and authentication.

- Join us on [Gitter](https://gitter.im/datawire/ambassador);
- Learn how to [add authentication](auth-tutorial.md) to existing services; or
- Learn how to [use gRPC with Ambassador](/how-to/grpc.md); or
- Read about [configuring Ambassador](/reference/configuration.md).
