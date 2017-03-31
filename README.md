Ambassador
==========

Ambassador is an overseer for Envoy in Kubernetes deployments. You can tell it about new services you're creating and deleting, and it will update Envoy's configuration to match.

Ambassador is ALPHA SOFTWARE. In particular, at present it does not include an authentication mechanism, and it updates Envoy's configuration only with a specific request to do so -- be aware!

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

Once running, it will take another minute or two for the Ambassador's load balancer to be established. Use the following to get its externally-visible IP address:

```
AMBASSADORIP=$(kubectl get service ambassador --output jsonpath='{.status.loadBalancer.ingress[0].ip}') || echo "No IP address yet"
```

If it reports "No IP address yet", wait a minute and try again. Once `AMBASSADORIP` is assigned, then

```
curl http://$AMBASSADORIP/ambassador/health
```

will do a health check;

```
curl http://$AMBASSADORIP/ambassador/services
```

will get a list of all user-defined services Ambassador knows about;

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/url/prefix/here", "port": 80 }' \
      http://$AMBASSADORIP/ambassador/service/$servicename
```

will create a new service (*note well* that the `$servicename` must match the name of a service defined in Kubernetes!);

```
curl -XDELETE http://$AMBASSADORIP/ambassador/service/$servicename
```

will delete a service; and

```
curl -XPUT http://$AMBASSADORIP/ambassador/services
```

will update Envoy's configuration to match the currently-defined set of services.

