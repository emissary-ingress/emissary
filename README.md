Ambassador
==========

Ambassador is an API Gateway for microservices built on [Envoy](https://lyft.github.io/envoy/). Key features in Ambassador include:

* Ability to flexibly map public URLs to services running inside a Kubernetes cluster
* Simple setup and configuration
* Integrated monitoring

CAVEATS
-------

Ambassador is ALPHA SOFTWARE. In particular, in version 0.3.1, there is no authentication mechanism, so anyone can map or unmap resources. (This is great for self service, of course, but we'll be putting a few controls in place later anyway.)

Ambassador is under active development; check frequently for updates, and please file issues for things you'd like to see!

I Don't Read Docs, Just Show Me An Example
==========================================

Let's assume you have a microservice running in your Kubernetes cluster called `usersvc`. There is a Kubernetes service for it already, and you can do a `GET` on its `/user/health` resource to do a health check.

To get an HTTP-only Ambassador running in the first place, clone this repo, then:

```
kubectl apply -f ambassador.yaml
kubectl apply -f ambassador-http.yaml
```

This spins up Ambassador - configured without inbound TLS **even though we do not recommend this** - in your Kubernetes cluster. Next you need the URL for Ambassador:

```eval $(sh scripts/geturl)```

and then you can check the health of Ambassador:

```curl $AMBASSADORURL/ambassador/health```

You can fire up a demo service called `usersvc` with

```
kubectl apply -f demo-usersvc.yaml
```

and then you can map the `/user/` resource to your `usersvc` with the `map` script:

```sh scripts/map user user usersvc```

Once that's done, you can go through Ambassador to do a `usersvc` health check:

```curl $AMBASSADORURL/user/health```

To get rid of the mapping, use

```sh scripts/unmap user```

Read on for more details.

Building Ambassador
===================

If you just want to **use** Ambassador, read on! You don't need to build anything.

If you really want to customize Ambassador, though, check out [BUILDING.md](BUILDING.md) for the lowdown.

Running Ambassador
==================

If you clone this repository, you'll have access to multiple Kubernetes resource files:

- `ambassador-rest.yaml` defines the main Ambassador server itself;
- `ambassador-store.yaml` defines the persistent storage that Ambassador uses to remember which services are running;
- `ambassador-sds.yaml` defines the Envoy Service Discovery Service that Ambassador relies on; and finally,
- `ambassador.yaml` wraps up all of the above.

Additionally, you can choose either

- `ambassador-https.yaml`, which defines an HTTPS-only service for talking to Ambassador and is recommended, or
- `ambassador-http.yaml', which defines an HTTP-only mechanism to access Ambassador.

The Ambassador Service and TLS
------------------------------

You need to choose up front whether you want to use TLS or not. It's possible to switch this later, but you'll likely need to muck about with your DNS and such to do it, so it's a pain.

*We recommend using TLS: speaking to Ambassador only over HTTPS.* To do this, you need a TLS certificate, which means you'll need the DNS set up correctly. So start by creating the Ambassador's kubernetes service:

```
kubectl apply -f ambassador-https.yaml
```

This will create an L4 load balancer that will later be used to talk to Ambassador. Once created, you'll be able to set up your DNS to associate a DNS name with this service, which will let you request the cert. Sadly, setting up your DNS and requesting a cert are a bit outside the scope of this README -- if you don't know how to do this, check with your local DNS administrator! (If you _are_ the domain admin and are just hunting a CA recommendation, check out [Let's Encrypt](https://letsencrypt.org/).)

Once you have the cert, you can run

```
sh scripts/push-cert $FULLCHAIN_PATH $PRIVKEY_PATH
```

where `$FULLCHAIN_PATH` is the path to a single PEM file containing the certificate chain for your cert (including the certificate for your Ambassador and all relevant intermediate certs -- this is what Let's Encrypt calls `fullchain.pem`), and `$PRIVKEY_PATH` is the path to the corresponding private key. `push-cert` will push the cert into Kubernetes secret storage, for Ambassador's later use.

Without TLS
-----------

If you really, really cannot use TLS, you can do

```
kubectl apply -f ambassador-http.yaml
```

for HTTP-only access.

After the Service
-----------------

The easy way to get Ambassador fully running once its service is created is

```
kubectl apply -f ambassador.yaml
```

This is what we recommend, but if you really need to, you can do it piecemeal:

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

We'll use `$AMBASSADORURL` as shorthand for the base URL of Ambassador. If you're using TLS, you can set it by hand with something like

```
export AMBASSADORURL=https://your-domain-name
```

where `your-domain-name` is the name you set up when you requested your certs.

Without TLS, if you have a domain name, great, do the above. If not, the easy way is to use the supplied `geturl` script:

```
eval $(sh scripts/geturl)
```

will set `AMBASSADORURL` for you.

*NOTE WELL* that if you use `geturl` when you have TLS configured, you'll get a URL that will work -- but you'll all but certainly see a lot of complaints about certificate validation, because the DNS name in the URL is not likely to be the name that you requested for the certificate.

 If you don't trust `geturl`, you can use `kubectl describe service ambassador` or, on Minikube, `minikube service --url ambassador` and set things from that information.

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
sh scripts/map mapping-name url-prefix service-name [rewrite]
```

e.g.

```
sh scripts/map user v1/user usersvc
```

to create a mapping named "user" that will cause requests for any resource with a URL starting with `/v1/user/` to be sent to the `usersvc` Kubernetes service. 

*Note well* that `service-name` must match the name of a service that is defined in Kubernetes.

In this example, when the request is forwarded, the `/v1/user/` part of the URL will be rewritten as `/`, so

```
/v1/user/alice
```

will appear to the `usersvc` as simply `/alice`.

To change this, you can append a value other than `/` to the `map` command:

```
sh scripts/map user v1/user usersvc /v2/
```

would cause `/v1/user/alice` to be forwarded as `/v2/alice`.

You can do all of this with a `POST` request:

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/v1/user/", "service": "usersvc", "rewrite": "/v2/" }' \
      $AMBASSADORURL/ambassador/mapping/user
```

To remove a mapping, use `scripts/unmap`:

```
sh scripts/unmap mapping-name
```

e.g., to undo the `user` mapping from above:

```
sh scripts/unmap user
```

(Remember to use the mapping name -- not the prefix or service name.)

You can also use a `DELETE` request to delete the mapping:

```
curl -XDELETE $AMBASSADORURL/ambassador/mapping/user
```

To check whether the `user` mapping exists, you can simply

```
curl $AMBASSADORURL/ambassador/mapping/user
```

Ambassador updates Envoy's configuration five seconds after a `POST` or `DELETE` changes its mapping. If another change arrives during that time, the timer is restarted.
