Ambassador
==========

Ambassador is a tool for easily and flexibly mapping public URLs to services running inside a Kubernetes cluster. Under the hood, Ambassador uses [Envoy](https://lyft.github.io/envoy/) for the heavy lifting. You needn't understand anything about how Envoy works to use Ambassador, however.

Ambassador is most effective, at this point, as an API gateway for microservices that's easy to configure and operate. It is under active development; check frequently for updates, and please file issues for things you'd like to see!

CAVEATS
-------

Ambassador is ALPHA SOFTWARE. In particular, in version 0.3.1:

- There is no authentication mechanism, so anyone can map or unmap resources.
- There is no SSL support.

Ambassador is under active development; check frequently for updates, and please file issues for things you'd like to see!

I Don't Read Docs, Just Show Me An Example
==========================================

OK, here we go. Let's assume you have a microservice running in your Kubernetes cluster called `awesomeness-service`. There is a Kubernetes service for it already, and you can do a `GET` on its `/awesome/health` resource to do a health check.

To get Ambassador running in the first place, clone this repo, then:

```
kubectl apply -f ambassador.yaml
```

This spins up the Ambassador in your Kubernetes cluster. Next you need the URL for Ambassador:

```eval $(sh scripts/geturl)```

and then you can check the health of Ambassador:

```curl $AMBASSADORIP/ambassador/health```

You can map the `awesome` resource to your `awesomeness-service` with the `map` script:

```sh scripts/map awesome awesomeness-service```

and then you'll see an awesome health check with

```curl $AMBASSADORIP/awesome/health```

To get rid of the mapping, use

```sh scripts/unmap awesomeness-service```

Read on for more details.

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

As part of its startup, Ambassador will request that the Kubernetes cluster establish a load balancer to connect it to the outside world, which will take another minute or two. To actually talk to Ambassador from outside, you'll need its IP address (or you'll need to associate the load balancer with a DNS name, which is outside the scope of this README).

The Easy Way to Get Ambassador's IP Address
-------------------------------------------

In `scripts/geturl` you'll find a shell script that will do the right thing to get the IP address of Ambassador's load balancer. Just `eval $(sh scripts/geturl).

The Less Easy Way
-----------------

As long as you're *not* using Minikube, you can use the following to get the IP address for Ambassador's externally-visible IP address:

```
AMBASSADORURL=http://$(kubectl get service ambassador --output jsonpath='{.status.loadBalancer.ingress[0].ip}') || echo "No IP address yet"
```

If it reports "No IP address yet", wait a minute and try again. 

If you *are* using Minikube, what you want is

```
AMBASSADORURL=$(minikube service --url ambassador)
```

Health Checks and Stats
-----------------------

Once `AMBASSADORURL` is assigned, then

```
curl $AMBASSADORURL/ambassador/health
```

will do a health check;

```
curl $AMBASSADORURL/ambassador/services
```

will get a list of all the resources that Ambassador has mapped; and

```
curl $AMBASSADOR/ambassador/stats
```

will return a JSON dictionary of statistics about resources that Ambassador presently has mapped. Most notably, the `services` dictionary lets you know basic health information about the services to which Ambassador is providing access:

- `services.$service.healthy_members` is the number of healthy back-end systems providing the service;
- `services.$service.upstream_ok` is the number of requests to the service that have succeeded; and
- `services.$service.upstream_bad` is the number of requests to the service that have failed.

Mappings
--------

You can use `scripts/map` to map a resource to a service:

```
sh scripts/map $prefix $service
```

e.g.

```
sh scripts/map v1/user usersvc
```

to cause requests for any resource with a URL starting with `/v1/user/` to be sent to the `usersvc` Kubernetes service.

*Note well* that `$service` must match the name of a service that is defined in Kubernetes. Also, in this example, the service will receive the entire URL: no rewriting happens (yet).

You can do the same thing with a `POST` request:

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/$prefix/" }' \
      $AMBASSADORURL/ambassador/service/$service
```

To remove a mapping, use `scripts/unmap`:

```
sh scripts/unmap $service
```

e.g., to undo the `usersvc` mapping from above:

```
sh scripts/unmap usersvc
```

(Remember to use the `service` name, not the `prefix`.)

To check whether a mapping exists, you can

```
curl $AMBASSADORURL/ambassador/service/$servicename
```

Ambassador update Envoy's configuration five seconds after a `POST` or `DELETE` changes its mapping. If another change arrives during that time, the timer is restarted.
