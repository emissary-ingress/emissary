---
layout: doc
weight: 1
title: "Getting Started"
categories: user-guide
---

Let's assume you have a microservice running in your Kubernetes cluster called `usersvc`, with a corresponding Kubernetes service entry that's also called `usersvc`. Let's further assume that you can do a `GET` on its `/health` resource to do a health check. (You can use

```
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/demo-usersvc.yaml
```

to actually start such a service running for this demo.)

How can we set up Ambassador as an API gateway for this service?

First we need to get Ambassador running in the Kubernetes cluster. We recommend using *TLS*, but for right now we'll just set up an HTTP-only Ambassador to show you how things work:

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
  "version": "0.8.0"
}
```

Mapping the `/user/` resource to your `usersvc` needs a POST request:

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/user/", "service": "usersvc" }' \
      http://localhost:8888/ambassador/mapping/user
```

and after that, you can read back and see that the mapping is there:

```
curl http://localhost:8888/ambassador/mappings
```

To actually _use_ the `usersvc`, we need the URL for microservice access through Ambassador. You can use `kubectl describe service ambassador` to work this out, but it's much easier to use our `geturl` script:

```
curl -O https://raw.githubusercontent.com/datawire/ambassador/master/scripts/geturl
eval $(sh geturl)
```

That will set `$AMBASSADORURL` for you, and you'll be able to use that for a basic health check on the `usersvc`:

```
curl $AMBASSADORURL/user/health
```

Since the `/user/` prefix in the path portion of the URL there matches the prefix we used for the `user` mapping above, Ambassador knows to route the request to the `usersvc`. In the process it rewrites `/user/` to `/` so that `/user/health` becomes `/health`, which is what the `usersvc` expects. (This rewriting is configurable; `/` is just the default.)

That's all there is to it. Ambassador will faithfully proxy any HTTP request matching the mapping to your service, transparently.

Finally, to get rid of the mapping, use a DELETE request:

```
curl -XDELETE http://localhost:8888/ambassador/mapping/user
```

and you're done!
