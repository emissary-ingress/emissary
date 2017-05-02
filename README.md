Ambassador
==========

Ambassador is an API Gateway for microservices built on [Envoy](https://lyft.github.io/envoy/). Key features in Ambassador include:

* Ability to flexibly map public URLs to services running inside a Kubernetes cluster
* Simple setup and configuration
* Integrated monitoring

Ambassador is built around the idea of mapping _resources_ (in the REST sense) to _services_ (in the Kubernetes sense). A `resource` is identified by a URL prefix -- for example, you might declare that any URL beginning with `/user/` identifies a "user" resource. A `service` is code running in Kubernetes that can handle the resource you want to map.

At present, a resource can be mapped to only one service, but the same service can be used behind as many different resources as you want. There's no hard limit to the number of mappings Ambassador can handle (though eventually you'll run out of memory).

CAVEATS
-------

Ambassador is ALPHA SOFTWARE. In particular, in version 0.7.0, there is no authentication mechanism, so anyone who can reach the administrative interface can map or unmap resources -- great for self service, of course, but possibly dangerous. For this reason, the administrative requires a Kubernetes port-forward.

Ambassador is under active development; check frequently for updates, and please file issues for things you'd like to see!

I Don't Read Docs, Just Show Me An Example
==========================================

Let's assume you have a microservice running in your Kubernetes cluster called `usersvc`. There is a Kubernetes service for it already, and you can do a `GET` on its `/health` resource to do a health check.

To get an HTTP-only Ambassador running in the first place, clone this repo, then:

```
kubectl apply -f ambassador-http.yaml
kubectl apply -f ambassador.yaml
```

This spins up Ambassador - configured without inbound TLS **even though we do not recommend this** - in your Kubernetes cluster. Next you need to set up access to Ambassador's admin port:

```
POD=$(kubectl get pod -l service=ambassador -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward "$POD" 8888
```

and then you can check the health of Ambassador:

```curl http://localhost:8888/ambassador/health```

You can fire up a demo service called `usersvc` with

```
kubectl apply -f demo-usersvc.yaml
```

and then you can map the `/user/` resource to your `usersvc` with a POST request:

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/user/", "service": "usersvc" }' \
      http://localhost:8888/ambassador/mapping/user
```

Finally, get the URL for microservice access through Ambassador:

```eval $(sh scripts/geturl)```

and that will allow you to go through Ambassador to do a `usersvc` health check:

```curl $AMBASSADORURL/user/health```

To get rid of the mapping, use a DELETE request:

```
curl -XDELETE http://localhost:8888/ambassador/mapping/user
```

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

Using TLS for Client Auth
-------------------------

If you want to use TLS client-certificate authentication, you'll need to tell Ambassador about the CA certificate chain to use to validate client certificates. This is also best done before starting Ambassador. Get the CA certificate chain - including all necessary intermediate certificates - and use `scripts/push-cacert` to push it into a Kubernetes secret:

```
sh scripts/push-cacert $CACERT_PATH
```

After starting Ambassador, you can tell Ambassador about which certificates are allowed (see below).

**NOTE WELL** that the presence of the CA cert chain makes a valid client certificate **mandatory**. If you don't define some valid certificates, Ambassador won't allow any access.

After the Service
-----------------

The easy way to get Ambassador fully running once its service is created is

```
kubectl apply -f ambassador.yaml
```

Once Running
------------

However you started Ambassador, once it's running you'll see pods and services called `ambassador` and `ambassador-store`. Both of these are necessary, and at present only one replica of each should be run.

*ALSO NOTE*: The very first time you start Ambassador, it can take a very long time - like 15 minutes - to get the images pulled down and running. You can use `kubectl get pods` to see when the pods are actually running.

Administering Ambassador
========================

Ambassador's admin interface is reachable over port 8888. This port is deliberately not exposed with a Kubernetes service; you'll need to use `kubectl port-forward` to reach it:

```
POD=$(kubectl get pod -l service=ambassador -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward "$POD" 8888
````

Once that's done, you can use the admin interface for health checks, statistics, and mappings.

Health Checks and Stats
-----------------------

```
curl http://localhost:8888/ambassador/health
```

will do a health check;

```
curl http://localhost:8888/ambassador/mappings
```

will get a list of all the resources that Ambassador has mapped; and

```
curl http://localhost:8888/ambassador/stats
```

will return a JSON dictionary of statistics about resources that Ambassador presently has mapped. Most notably, the `mappings` dictionary lets you know basic health information about the mappings to which Ambassador is providing access:

- `mappings.$mapping.healthy_members` is the number of healthy back-end systems providing the mapped service;
- `mappings.$mapping.upstream_ok` is the number of requests to the mapped resource that have succeeded; and
- `mappings.$mapping.upstream_bad` is the number of requests to the mapped resource that have failed.

Mappings
--------

You use `POST` requests to the admin interface to map a resource to a service:

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "<url-prefix>", "service": "<service-name>", "rewrite": "<rewrite-as>" }' \
      http://localhost:8888/ambassador/mapping/<mapping-name>
```

where

- `<mapping-name>` is a unique name that identifies this mapping
- `<url-prefix>` is the URL prefix identifying your resource
- `<service-name>` is the name of the service handling the resource
- `<rewrite-as>` is what to replace the URL prefix with when talking to the service

The `mapping-name` is used to delete mappings later, and to identify mappings in statistics and such.

The `url-prefix` should probably begin and end with `/` to avoid confusion. An URL prefix of `man` would match the URL `https://getambassador.io/manifold`, which is probably not what you want -- using `/man/` is more clear.

The `service-name` **must** match the name of a service defined in Kubernetes.

The `rewrite-as` part is optional: if not given, it defaults to `/`. Whatever it's set to, the `url-prefix` gets replaced with `rewrite-as` when the request is forwarded:

- If `url-prefix` is `/v1/user/` and `rewrite-as` is `/`, then `/v1/user/foo` will appear to the service as `/foo`.

- If `url-prefix` is `/v1/user/` and `rewrite-as` is `/v2/`, then `/v1/user/foo` will appear to the service as `/v2/foo`.

- If `url-prefix` is `/v1/` and `rewrite-as` is `/v2/`, then `/v1/user/foo` will appear to the service as `/v2/user/foo`.

etc.

Ambassador updates Envoy's configuration five seconds after any mapping change. If another change arrives during that time, the timer is restarted.

### Creating a Mapping

An example mapping:

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/v1/user/", "service": "usersvc" }' \
      http://localhost:8888/ambassador/mapping/user
```

will create a mapping named `user` that will cause requests for any resource with a URL starting with `/v1/user/` to be sent to the `usersvc` Kubernetes service, with the `/v1/user/` part replaced with `/` -- `/v1/user/alice` would appear to the service as simply `/alice`.

If instead you did

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/v1/user/", "service": "usersvc", "rewrite": "/v2/" }' \
      http://localhost:8888/ambassador/mapping/user
```

then `/v1/user/alice` would appear to the service as `/v2/alice`.

### Deleting a Mapping

To remove a mapping, use a `DELETE` request:

```
curl -XDELETE http://localhost:8888/ambassador/mapping/user
```

will delete the mapping from above.

### Checking for a Mapping

To check whether the `user` mapping exists, you can simply

```
curl http://localhost:8888/ambassador/mapping/user
```

Ambassador Microservice Access
------------------------------

Access to your microservices through Ambassador is via port 443 (if you configured TLS) or port 80 (if not). This port _is_ exposed with a Kubernetes service; we'll use `$AMBASSADORURL` as shorthand for the base URL through this port.

If you're using TLS, you can set it by hand with something like

```
export AMBASSADORURL=https://your-domain-name
```

where `your-domain-name` is the name you set up when you requested your certs. **Do not include a trailing `/`**, or the examples in this document won't work.

Without TLS, if you have a domain name, great, do the above. If not, the easy way is to use the supplied `geturl` script:

```
eval $(sh scripts/geturl)
```

will set `AMBASSADORURL` for you.

*NOTE WELL* that if you use `geturl` when you have TLS configured, you'll get a URL that will work -- but you'll all but certainly see a lot of complaints about certificate validation, because the DNS name in the URL is not likely to be the name that you requested for the certificate.

If you don't trust `geturl`, you can use `kubectl describe service ambassador` or, on Minikube, `minikube service --url ambassador` and set things from that information. Again, **do not include a trailing `/`**, or the examples in this document won't work.

After that, you can access your microservices by using URLs based on `$AMBASSADORURL` and the URL prefixes defined for your mappings. For example, with first the `user` mapping from above in effect:

```
curl $AMBASSADORURL/v1/user/health
```

would be relayed to the `usersvc` as simply `/health`; 


