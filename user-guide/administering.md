---
layout: doc
weight: 3
title: "Administering Ambassador"
categories: user-guide
---
Ambassador's admin interface is reachable over port 8888. This port is deliberately not exposed with a Kubernetes service; you'll need to use `kubectl port-forward` to reach it:

```
POD=$(kubectl get pod -l service=ambassador -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward "$POD" 8888
```

Once that's done, you can use the admin interface for health checks, statistics, and mappings.

### Health Checks and Stats

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

### Mappings

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

#### Creating a Mapping

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

#### Deleting a Mapping

To remove a mapping, use a `DELETE` request:

```
curl -XDELETE http://localhost:8888/ambassador/mapping/user
```

will delete the mapping from above.

#### Checking for a Mapping

To check whether the `user` mapping exists, you can simply

```
curl http://localhost:8888/ambassador/mapping/user
```

### Ambassador Microservice Access

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
