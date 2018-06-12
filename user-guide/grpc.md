# gRPC and Ambassador

---

Ambassador makes it easy to access your services from outside your application. This includes gRPC services, although a little bit of additional configuration is required: by default, Envoy connects to upstream services using HTTP/1.x and then upgrades to HTTP/2 whenever possible. However, gRPC is built on HTTP/2 and most gRPC servers do not speak HTTP/1.x at all. Ambassador must tell its underlying Envoy that your gRPC service only wants to speak that HTTP/2, using the `grpc` attribute of a `Mapping`.

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
rewrite: /helloworld.Greeter/
service: grpc-greet
```

Note the `grpc: true` line -- this is the necessary magic when mapping a gRPC service. Also note that you'll need `prefix` and `rewrite` the same here, since the gRPC service needs the package and service to be in the request to do the right thing.

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
      rewrite: /helloworld.Greeter/
      service: grpc-greet
spec:
  type: ClusterIP
  ports:
  - port: 80
    name: grpc-greet
    targetPort: grpc-api
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
        ports:
        - name: grpc-api
          containerPort: 9999
        env:
          - name: PORT
            value: "9999"
        command:
          - greeter_server
      restartPolicy: Always
```

(We tell the gRPC service to run on port 9999, then map the container's port 80 inbound to simplify the `Mapping`. There's no magic behind these port numbers: anything will work as long as you're consistent in when mapping everything.)

This is available from getambassador.io, so you can simply

```shell
kubectl apply -f https://getambassador.io/yaml/demo/demo-grpc.yaml
```

or, as always, you can use a local file instead.

## Testing `Hello World`

Now you should be able to access your service. We'll need the hostname for the Ambassador service, which you can get with

```shell
kubectl get svc -o wide ambassador
```

This should give you something like

```
NAME         CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
ambassador   10.11.12.13     35.36.37.38     80:31656/TCP   1m
```

and the `EXTERNAL-IP` element is what we want. We'll call that `$AMBASSADORHOST`. You'll also need the port: if you haven't explicitly configured Ambassador otherwise, this should be 80 for an HTTP Ambassador or 443 for an HTTPS Ambassador. We'll call that `$AMBASSADORPORT`.

To test `Hello World`, we can use the Docker image `enm10k/grpc-hello-world`:

```shell
docker run -e ADDRESS=${AMBASSADORHOST}:${AMBASSADORPORT} enm10k/grpc-hello-world greeter_client
```

Note: If you're trying this out using `NodePort` on minikube and running the docker command above on your host machine, make sure to pass the `--network host` parameter to the docker command.

## Using over TLS

To enable grpc over TLS, ALPN protocol http2 `alpn_protocols: h2` must be added to the TLS module configuration. Refer to [TLS termination guide](/user-guide/tls-termination.html) for more information.

## Note

Some [Kubernetes ingress controllers](https://kubernetes.io/docs/concepts/services-networking/ingress/) do not support HTTP/2 fully. As a result, if you are running Ambassador with an ingress controller in front, you may find that gRPC requests fail even with correct Ambassador configuration.

A simple way around this is to use Ambassador with a `LoadBalancer` service, rather than an Ingress controller.
