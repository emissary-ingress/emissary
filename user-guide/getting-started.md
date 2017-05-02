---
layout: doc
weight: 1
title: "Getting Started"
categories: user-guide
---

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

```
curl http://localhost:8888/ambassador/health
```

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

```
eval $(sh scripts/geturl)
```

and that will allow you to go through Ambassador to do a `usersvc` health check:

```
curl $AMBASSADORURL/user/health
```

To get rid of the mapping, use a DELETE request:

```
curl -XDELETE http://localhost:8888/ambassador/mapping/user
```
