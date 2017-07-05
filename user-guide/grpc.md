---
layout: doc
weight: 3.2
title: "GRPC and Ambassador"
categories: user-guide
---
Ambassador makes it easy to access your services from outside your application. This includes GRPC services, although a little bit of additional configuration is required. Why? By default, Envoy connects to upstream services using HTTP/1.x and then upgrades to HTTP/2 whenever possible. However, GRPC is built on HTTP/2 and most GRPC servers do not speak HTTP/1.x at all. Ambassador must tell its underlying Envoy that your GRPC service only wants to speak that HTTP/2. The Ambassador GRPC module makes this possible.

## Example

To demonstrate, let's walk through an example. Start by setting things up as in the [Getting Started](getting-started.md) section up to authentication. Here's the short version; read the full text for the details, particularly for how to set up `$AMBASSADORURL`.

```
# Add Quote of the Moment service
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/demo-qotm.yaml

# Add Ambassador
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/ambassador-http.yaml
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/ambassador.yaml

# Set up port-forwarding
POD=$(kubectl get pod -l service=ambassador -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward "$POD" 8888 &

# Map QotM at /qotm/
curl -XPUT -H"Content-Type: application/json" \
     -d'{ "prefix": "/qotm/", "service": "qotm" }' \
     http://localhost:8888/ambassador/mapping/qotm_map

# Set up $AMBASSADORURL -- no trailing / -- see Getting Started
AMBASSADORURL=...

# Verify things are working
curl $AMBASSADORURL/qotm/
```

Also, add a [Hello World GRPC service](https://github.com/grpc/grpc-go/tree/master/examples/helloworld) to your cluster:

```
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/demo-grpc.yaml
```

To create an Ambassador mapping for this service, you need the URL prefix, which is the full service name (including package path) as described in the [proto definition file](https://github.com/grpc/grpc-go/blob/master/examples/helloworld/helloworld/helloworld.proto) for the service. In this example, the service prefix is `helloworld.Greeter`. Create the mapping with the GRPC module included:

```
curl -XPUT -H"Content-Type: application/json" \
     -d'{ "prefix": "/helloworld.Greeter/", "service": "grpc-greet", "rewrite": "/helloworld.Greeter/", "modules": {"grpc": true} }' \
     http://localhost:8888/ambassador/mapping/greeter_map
```

Now you should be able to access your service. In this example, `$AMBASSADORHOST` is the hostname or IP address contained in `$AMBASSADORURL`.

```
docker run -e ADDRESS=${AMBASSADORHOST}:80 enm10k/grpc-hello-world greeter_client
```

#### Note

Some [Kubernetes ingress controllers](https://kubernetes.io/docs/concepts/services-networking/ingress/) do not support HTTP/2 fully. As a result, if you are running Ambassador with an ingress controller in front, e.g., when using [Istio](with-istio.md), you may find that GRPC requests fail even with correct Ambassador configuration.
