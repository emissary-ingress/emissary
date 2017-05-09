---
layout: doc
weight: 1
title: "Getting Started"
categories: user-guide
---

Ambassador is an API Gateway for microservices, so to get started, it's helpful to actually have a running service to use it with. We'll use the demo `usersvc` from our "Deploying Envoy with a Python Flask webapp and Kubernetes" [article](https://www.datawire.io/guide/traffic/envoy-flask-kubernetes/); you can deploy it into Kubernetes with

```
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/demo-usersvc.yaml
```

This will create a deployment called `usersvc` and a corresponding Kubernetes service entry that's also called `usersvc`. This `usersvc` supports using a simple REST API:

* `GET /health` performs a simple health check
* `POST /user/:userid` creates a new user
   * this one also requires a JSON dictionary with `fullname` and `password` keys as the `POST` body
* `GET /user/:userid` reads back a user

We'll use the health check as our first simple test to make sure that Ambassador is relaying requests, but of course we want all of the above to work through Ambassador.

To set up Ambassador as an API gateway for this service, first we need to get Ambassador running in the Kubernetes cluster. We recommend using [TLS](running.md#TLS), but for right now we'll just set up an HTTP-only Ambassador to show you how things work:

```
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/ambassador-http.yaml
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/ambassador.yaml
```

That's it for getting Ambassador running, though in the real world you'd need TLS. Next you need to be able to talk to Ambassador's administrative interface, which is a private REST service on Ambassador's port 8888. This isn't exposed anywhere outside the cluster, for security reasons, so you need to use Kubernetes port forwarding to reach it (doing this in a separate shell window is a good idea):

```
POD=$(kubectl get pod -l service=ambassador -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward "$POD" 8888
```

Once that's done, `localhost:8888` is where you can talk to the Ambassador's administrative interface. Let's start with a basic health check of Ambassador itself:

```
$ curl http://localhost:8888/ambassador/health
```

which should give something like this if all is well:

```
{
  "hostname": "ambassador-3176426918-13v2v",
  "msg": "ambassador health check OK",
  "ok": true,
  "resolvedname": "109.196.3.8",
  "version": "0.8.6"
}
```

Mapping the `/user/` resource to your `usersvc` needs a POST request:

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/user/", "service": "usersvc" }' \
      http://localhost:8888/ambassador/mapping/user_map
```

and after that, you can read back and see that the mapping is there:

```
curl http://localhost:8888/ambassador/mappings
```

which should show you something like

```
{
  "count": 1,
  "hostname": "ambassador-3176426918-13v2v",
  "mappings": [
    {
      "name": "user_map",
      "prefix": "/user/",
      "rewrite": "/",
      "service": "usersvc"
    }
  ],
  "ok": true,
  "resolvedname": "109.196.3.8",
  "version": "0.8.6"
}
```

To actually _use_ the `usersvc`, we need the URL for microservice access through Ambassador. Look at the `LoadBalancer Ingress` line of `kubectl describe service ambassador` (or use `minikube service --url ambassador` on Minikube) and set `$AMBASSADORURL` based on that. **Do not include a trailing `/`** on it, or our examples below won't work.

Once `$AMBASSADORURL` is set, you'll be able to use that for a basic health check on the `usersvc`:

```
curl $AMBASSADORURL/user/health
```

If all goes well you should get a health response much like Ambassador's:

```
{ 
  "hostname": "usersvc-1786225466-0tb2t",
  "msg": "user health check OK",
  "ok": true,
  "resolvedname": "109.196.4.8"
}
```

Since the `/user/` prefix in the path portion of the URL there matches the prefix we used for the `user` mapping above, Ambassador knows to route the request to the `usersvc`. In the process it rewrites `/user/` to `/` so that `/user/health` becomes `/health`, which is what the `usersvc` expects. (This rewriting is configurable; `/` is just the default.)

Of course, we can access the other `usersvc` endpoints as well. Let's create a user named Alice:

```
curl -X PUT  -H "Content-Type: application/json"  \
     -d '{ "fullname": "Alice", "password": "alicerules" }'  \
     $AMBASSADORURL/user/alice
```

This should show us our new user, sans password, with something like:

```
{
  "fullname": "Alice",
  "hostname": "usersvc-1786225466-0tb2t",
  "ok": true,
  "resolvedname": "109.196.4.8",
  "uuid": "16CE88880C0D4C869C70C7B3829F54DA"
}
```

and finally, we can read Alice back using a `GET` request:

```
curl $AMBASSADORURL/user/alice
```

which should return the same information as above.

That's all there is to it. If there were other endpoints exposed by the `usersvc` we could use Ambassador to proxy any HTTP requests to them, too: any request matching a mapped prefix will be transparently routed to the mapped service.

Finally, to get rid of the mapping, use a DELETE request:

```
curl -XDELETE http://localhost:8888/ambassador/mapping/user_map
```

and you're done!



