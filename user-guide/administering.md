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

Once that's done, you can use the admin interface for health checks, statistics, [mappings](mappings.md#mappings), [modules](mappings.md#modules), and [consumers](mappings.md#consumers).

### Health Checks and Stats

```
curl http://localhost:8888/ambassador/health
```

will do a health check;

```
curl http://localhost:8888/ambassador/mapping
```

will get a list of all the resources that Ambassador has mapped; and

```
curl http://localhost:8888/ambassador/stats
```

will return a JSON dictionary containing a `stats` dictionary with statistics about resources that Ambassador presently has mapped. Most notably, `stats.mappings` contains basic health information about the mappings to which Ambassador is providing access:

- `stats.mappings.<mapping-name>.healthy_members` is the number of healthy back-end systems providing the mapped service;
- `stats.mappings.<mapping-name>.upstream_ok` is the number of requests to the mapped resource that have succeeded; and
- `stats.mappings.<mapping-name>.upstream_bad` is the number of requests to the mapped resource that have failed.

### Mappings

You use `PUT` requests to the admin interface to map a resource to a service:

```
curl -XPUT -H "Content-Type: application/json" \
      -d <mapping-dict> \
      http://localhost:8888/ambassador/mapping/<mapping-name>
```

where `<mapping-name>` is a unique name that identifies this mapping, and `<mapping-dict>` is a dictionary that defines the mapping:

```
{
    "prefix": <url-prefix>,
    "service": <service-name>,
    "rewrite": <rewrite-as>,
    "modules": <module-dict>
}
```

- `<url-prefix>` is the URL prefix identifying your [resource](#resources)
- `<service-name>` is the name of the [service](#services) handling the resource
- `<rewrite-as>` (optional) is what to [replace](#rewriting) the URL prefix with when talking to the service
- `<module-dict>` (optional) defines any relevant module configuration for this mapping.

The `mapping-name` is used to delete mappings later, and to identify mappings in statistics and such.

The `url-prefix` should probably begin and end with `/` to avoid confusion. An URL prefix of `man` would match the URL `https://getambassador.io/manifold`, which is probably not what you want -- using `/man/` is more clear.

The `service-name` **must** match the name of a service defined in Kubernetes.

The `rewrite-as` part is optional: if not given, it defaults to `/`. Whatever it's set to, the `url-prefix` gets replaced with `rewrite-as` when the request is forwarded:

- If `url-prefix` is `/v1/user/` and `rewrite-as` is `/`, then `/v1/user/foo` will appear to the service as `/foo`.

- If `url-prefix` is `/v1/user/` and `rewrite-as` is `/v2/`, then `/v1/user/foo` will appear to the service as `/v2/foo`.

- If `url-prefix` is `/v1/` and `rewrite-as` is `/v2/`, then `/v1/user/foo` will appear to the service as `/v2/user/foo`.

etc.

Ambassador updates Envoy's configuration five seconds after any mapping change. If another change arrives during that time, the timer is restarted.

#### Listing Mappings

You can list all the extant mappings with

```
curl http://localhost:8888/ambassador/mapping
```

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

### Modules and Consumers

[Modules](mappings.md#modules) let you enable and configure special behaviors for Ambassador, in ways which may apply to Ambassador as a whole or which may apply only to some mappings. 

[Consumers](mappings.md#consumers) represent human end users of Ambassador, and may be required for some modules to function.

At present the only supported module is the [`authentication` module](mappings.md#authentication-module). Its global configuration tells Ambassador which authentication service to use, and it uses per-mapping and per-consumer configuration to tell Ambassador which mappings require authentication and which consumers may authenticate.

To list modules and consumers, use `GET` requests:

```
curl http://localhost:8888/ambassador/module
curl http://localhost:8888/ambassador/consumer
```

You can also directly access per-mapping and per-consumer module configuration:

```
curl http://localhost:8888/ambassador/mapping/<mapping-name>/module
curl http://localhost:8888/ambassador/consumer/<consumer-id>/module
```

See [About Mappings, Modules, and Consumers](mappings.md) for more on administering modules and consumers.

### Ambassador Microservice Access

Access to your microservices through Ambassador is via port 443 (if you configured TLS) or port 80 (if not). This port _is_ exposed with a Kubernetes service; we'll use `$AMBASSADORURL` as shorthand for the base URL through this port.

If you're using TLS, you can set it by hand with something like

```
export AMBASSADORURL=https://your-domain-name
```

where `your-domain-name` is the name you set up when you requested your certs. **Do not include a trailing `/`**, or the examples in this document won't work.

Without TLS, if you have a domain name, great, do the above. If not, look at the `LoadBalancer Ingress` line of `kubectl describe service ambassador` (or use `minikube service --url ambassador` on Minikube) and set `$AMBASSADORURL` based on that. Again, **do not include a trailing `/`**, or the examples in this document won't work.

After that, you can access your microservices by using URLs based on `$AMBASSADORURL` and the URL prefixes defined for your mappings. For example, with first the `user` mapping from above in effect:

```
curl $AMBASSADORURL/v1/user/health
```

would be relayed to the `usersvc` as simply `/health`;
