Ambassador
==========

Ambassador is a tool for easily and flexibly mapping public URLs to services running inside a Kubernetes cluster. Think of it as a simple way to spin up an API gateway for Kubernetes.

Under the hood, Ambassador uses [Envoy](https://lyft.github.io/envoy/) for the heavy lifting. You needn't understand anything about how Envoy works to use Ambassador, however.

CAVEATS
-------

Ambassador is ALPHA SOFTWARE. In particular, in version 0.3.1:

- There is no authentication mechanism, so anyone can bring services up or down.
- There is no SSL support.
- Ambassador updates Envoy's configuration only with a specific request to do so.

Ambassador is under active development; check frequently for updates, and please file issues for things you'd like to see!

Running Ambassador
==================

If you clone this repository, you'll have access to multiple Kubernetes resource files:

- `ambassador-rest.yaml` defines the REST service that is how you'll primarily interact with Ambassador;
- `ambassador-store.yaml` defines the persistent storage that Ambassador uses to remember which services are running;
- `ambassador-sds.yaml` defines the Envoy Service Discovery Service that Ambassador relies on; and finally,
- `ambassador.yaml` wraps up all of the above.

You can get Ambassador running the easy way, or the less easy way.

The Easy Way
------------

The simplest way to get everything running is simply to use `ambassador.yaml` to crank everything up at once:

```
kubectl apply -f ambassador.yaml
```

This is what we recommend.

The Less Easy Way
-----------------

If necessary for some reason, you can instead use the individual resources above and do things by hand. In this case, we recommend the following order:

```
kubectl apply -f ambassador-store.yaml
kubectl apply -f ambassador-sds.yaml
kubectl apply -f ambassador-rest.yaml
```

Once Running
------------

However you started Ambassador, once it's running you'll see pods and services called `ambassador`, `ambassador-sds`, and `ambassador-store`. All of these are necessary, and at present only one replica of each should be run.

*ALSO NOTE*: The very first time you start Ambassador, it can take a very long time - like 15 minutes - to get the images pulled down and running. You can use `kubectl get pods` to see when the pods are actually running.

Using Ambassador
================

Once running, it will take another minute or two for the Ambassador's load balancer to be established. 

As long as you're *not* using Minikube, you can use the following to get the IP address for Ambassador's externally-visible IP address:

```
AMBASSADORURL=http://$(kubectl get service ambassador --output jsonpath='{.status.loadBalancer.ingress[0].ip}') || echo "No IP address yet"
```

If it reports "No IP address yet", wait a minute and try again. 

If you *are* using Minikube, what you want is

```
AMBASSADORURL=$(minikube service --url ambassador)
```

Once `AMBASSADORURL` is assigned, then

```
curl $AMBASSADORURL/ambassador/health
```

will do a health check;

```
curl $AMBASSADORURL/ambassador/services
```

will get a list of all the  Ambassador knows how to map;

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/url/prefix/here/" }' \
      $AMBASSADORURL/ambassador/service/$servicename
```

will create a new service (*note well* that the `$servicename` must match the name of a service defined in Kubernetes!);

```
curl -XDELETE $AMBASSADORURL/ambassador/service/$servicename
```

will delete a service; and

```
curl -XPUT $AMBASSADORURL/ambassador/services
```

will update Envoy's configuration to match the currently-defined set of services.

Finally:

```
curl $AMBASSADOR/ambassador/stats
```

will return a JSON dictionary of statistics about resources that Ambassador presently has mapped. Most notably, the `services` dictionary lets you know basic health information about the services to which Ambassador is providing access:

- `services.$service.healthy_members` is the number of healthy back-end systems providing the service;
- `services.$service.upstream_ok` is the number of requests to the service that have succeeded; and
- `services.$service.upstream_bad` is the number of requests to the service that have failed.



