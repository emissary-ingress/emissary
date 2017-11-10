# gRPC and Ambassador

---

Ambassador makes it easy to access your services from outside your application. This includes gRPC services, although a little bit of additional configuration is required: by default, Envoy connects to upstream services using HTTP/1.x and then upgrades to HTTP/2 whenever possible. However, gRPC is built on HTTP/2 and most gRPC servers do not speak HTTP/1.x at all. Ambassador must tell its underlying Envoy that your gRPC service only wants to speak that HTTP/2. The Ambassador gRPC module makes this possible.

## Example

This tutorial assumes you have already followed the [Ambassador Getting Started](/user-guide/getting-started.html) guide. If you haven't done that already, you should do that now.

After completing [Getting Started](/user-guide/getting-started.html), you'll have a Kubernetes cluster running Ambassador and the Quote of the Moment service. Let's walk through adding the [Hello World gRPC service](https://github.com/grpc/grpc-go/tree/master/examples/helloworld) for this tutorial. 

## Mapping gRPC Services

Ambassador `Mapping`s are based on URL prefixes; for gRPC, the URL prefix is the full service name, including the package path. 

For `Hello World`, in its [proto definition file](https://github.com/grpc/grpc-go/blob/master/examples/helloworld/helloworld/helloworld.proto), we see

```
package helloworld;

service Greeter { ... }
```

so its URL prefix is `helloworld.Greeter`, and a reasonable `Mapping` would be:

```yaml
---
apiVersion: ambassador/v0
kind: Mapping
name: grpc_mapping
grpc: true
prefix: /helloworld.Greeter/
service: grpc-greet
```

Note the `grpc: true` line -- this is necessary when mapping a gRPC service.

## Deploying `Hello World`

To deploy and map `Hello World`, we can use the following YAML:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: grpc-greet
  name: grpc-greet
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind: Mapping
      name: grpc_mapping
      grpc: true
      prefix: /helloworld.Greeter/
      service: grpc-greet
spec:
  type: ClusterIP
  ports:
  - name: grpc-greet
    port: 443
  selector:
    service: grpc-greet
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: grpc-greet
spec:
  replicas: 1
  template:
    metadata:
      labels:
        service: grpc-greet
    spec:
      containers:
      - name: grpc-greet
        image: enm10k/grpc-hello-world
        env:
          - name: PORT
            value: "443"
        command:
          - greeter_server
      restartPolicy: Always
```

This is available from getambassador.io, so you can simply

```shell
kubectl apply -f http://getambassador.io/yaml/demo/demo-grpc.yaml
```

or, as always, you can use a local file instead.

## Testing `Hello World`

Now you should be able to access your service. We'll need the hostname for the Ambassador service, which you can get with

```shell
kubectl get svc ambassador
```

This should give you something like

```
NAME         CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
ambassador   10.11.12.13     35.36.37.38     80:31656/TCP   1m
```

and the `EXTERNAL-IP` element is what we want. We'll call that `$AMBASSADORHOST`. 

To test `Hello World`, we can use the Docker image `enm10k/grpc-hello-world`:

```shell
docker run -e ADDRESS=${AMBASSADORHOST}:80 enm10k/grpc-hello-world greeter_client
```

## Note

Some [Kubernetes ingress controllers](https://kubernetes.io/docs/concepts/services-networking/ingress/) do not support HTTP/2 fully. As a result, if you are running Ambassador with an ingress controller in front, e.g., when using [Istio](../user-guide/with-istio.md), you may find that gRPC requests fail even with correct Ambassador configuration.
