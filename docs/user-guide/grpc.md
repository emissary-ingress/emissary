# gRPC and Ambassador

---

Ambassador makes it easy to access your services from outside your application. This includes gRPC services, although a little bit of additional configuration is required: by default, Envoy connects to upstream services using HTTP/1.x and then upgrades to HTTP/2 whenever possible. However, gRPC is built on HTTP/2 and most gRPC servers do not speak HTTP/1.x at all. Ambassador must tell its underlying Envoy that your gRPC service only wants to speak that HTTP/2, using the `grpc` attribute of a `Mapping`.

## Example

This tutorial assumes you have already followed the [Ambassador Getting Started](/user-guide/getting-started.html) guide. If you haven't done that already, you should do that now.

After completing [Getting Started](/user-guide/getting-started.html), you'll have a Kubernetes cluster running Ambassador and the Quote of the Moment service. Let's walk through adding an example [echo gRPC service](https://github.com/datawire/grpc-example) for this tutorial. 

## Mapping gRPC Services

Ambassador `Mapping`s are based on URL prefixes; for gRPC, the URL prefix is the full service name, including the package path. 

For `Hello World`, in its [proto definition file](https://github.com/datawire/grpc-example/blob/master/helloworld/helloworld.proto), we see

```
package helloworld;

service Greeter1 { ... }
```

so its URL prefix is `helloworld.Greeter1`, and a reasonable `Mapping` would be:

```yaml
---
apiVersion: ambassador/v0
kind: Mapping
name: grpc_mapping
grpc: true
prefix: /helloworld.Greeter1/
rewrite: /helloworld.Greeter1/
service: grpc-example
```

Note the `grpc: true` line -- this is the necessary magic when mapping a gRPC service. Also note that you'll need `prefix` and `rewrite` the same here, since the gRPC service needs the package and service to be in the request to do the right thing.

## Deploying `gRPC Example`

To deploy and map `gRPC Example`, we can use the following YAML:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: grpc-example
  name: grpc-example
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind: Mapping
      name: grpc_mapping
      grpc: true
      prefix: /helloworld.Greeter1/
      rewrite: /helloworld.Greeter1/
      service: grpc-example
spec:
  type: ClusterIP
  ports:
  - port: 80
    name: grpc-example
    targetPort: grpc-api
  selector:
    service: grpc-example
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: grpc-example
spec:
  replicas: 1
  template:
    metadata:
      labels:
        service: grpc-example
    spec:
      containers:
      - name: grpc-greet
        image: nkrause/grpc_example:latest
        ports:
        - name: grpc-api
          containerPort: 50052
      restartPolicy: Always
```

(We tell the gRPC service to run on port 50052, then map the container's port 80 inbound to simplify the `Mapping`. There's no magic behind these port numbers: anything will work as long as you're consistent in when mapping everything.)

This is available from getambassador.io, so you can simply

```shell
curl https://raw.githubusercontent.com/datawire/grpc-example/master/grpc_example.yaml | kubectl apply -f -
```

or, as always, you can use a local file instead.

## Testing `gRPC Example`

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

To run the example echo server, clone the [grpc-example repository](https://github.com/datawire/grpc-example).

```shell
git clone https://github.com/datawire/grpc-example.git
```

Once the repository is cloned, change to the `client` directory and run the client application with 
```shell
cd ./grpc-example/client/
python client.py --channel ${AMBASSADORHOST}:${AMBASSADORPORT} --message Hello!
```

The `--channel` option must be set and specifies the ip/host and port to route the gRPC traffic to.

The `--message` option allows you to set your own message. It is optional and will send a default message if left blank.


## Using over TLS

To initiate a gRPC call over a secure channel with TLS you need to do a couple of things:

First, ALPN protocol http2 must be enabled in the TLS module by `alpn_protocols: h2`

Second, the client application needs a root cert to authenticate with you CA. The example grpc service in this document allows you to enable this by setting the`—tls` flag when invoking the client application. Also ensure you set the port to `443` with the `—channel` flag. 

For more information on gRPC and TLS visit: https://grpc.io/docs/guides/auth.html

For more information on the ambassador TLS module visit [TLS termination guide](/user-guide/tls-termination.html)

## Note

Some [Kubernetes ingress controllers](https://kubernetes.io/docs/concepts/services-networking/ingress/) do not support HTTP/2 fully. As a result, if you are running Ambassador with an ingress controller in front, you may find that gRPC requests fail even with correct Ambassador configuration.

A simple way around this is to use Ambassador with a `LoadBalancer` service, rather than an Ingress controller.
