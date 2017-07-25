# Getting Started

---

Are you looking to run Ambassador within Istio? Check out our [Ambassador and Istio](with-istio.md) quickstart!

---

Ambassador is an API Gateway for microservices, so to get started, it's helpful to actually have a running service to use it with. We'll use Datawire's "Quote of the Moment" service (`qotm`) for this; you can deploy it into Kubernetes with

```shell
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/demo-qotm.yaml
```

This will create a deployment called `qotm` and a corresponding Kubernetes service entry that's also called `qotm`. Quote of the Moment supports a very simple REST API:

* `GET /health` performs a simple health check
* `GET /` returns a random Quote of the Moment
* `GET /quote/:quoteid` returns the Quote of the Moment with a given ID
* `POST /quote` adds a new Quote of the Moment and returns its ID
  * this requires that the POST body carry the new Quote of the Moment

We'll use the health check as our first simple test to make sure that Ambassador is relaying requests, but of course we want all of the above to work through Ambassador -- and we want everything using the `/quote` endpoint to require authentication.

To set up Ambassador as an API gateway for this service, first we need to get Ambassador running in the Kubernetes cluster. We recommend using [TLS](running.md#TLS), but for right now we'll just set up an HTTP-only Ambassador to show you how things work:

```shell
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/ambassador-http.yaml
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/ambassador.yaml
```

That's it for getting Ambassador running, though in the real world you'd need TLS. Next you need to be able to talk to Ambassador's administrative interface, which is a private REST service on Ambassador's port 8888. This isn't exposed anywhere outside the cluster, for security reasons, so you need to use Kubernetes port forwarding to reach it (doing this in a separate shell window is a good idea):

```shell
POD=$(kubectl get pod -l service=ambassador -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward "$POD" 8888
```

Once that's done, `localhost:8888` is where you can talk to the Ambassador's administrative interface. Let's start with a basic health check of Ambassador itself:

```shell
curl http://localhost:8888/ambassador/health
```

which should give something like this if all is well:

```json
{
  "hostname": "ambassador-3176426918-13v2v",
  "msg": "ambassador health check OK",
  "ok": true,
  "resolvedname": "109.196.3.8",
  "version": "{VERSION}"
}
```

Mapping the `/qotm/` resource to your QotM service needs a PUT request:

```shell
curl -XPUT -H"Content-Type: application/json" \
     -d'{ "prefix": "/qotm/", "service": "qotm" }' \
     http://localhost:8888/ambassador/mapping/qotm_map
```

and after that, you can read back and see that the mapping is there:

```shell
curl http://localhost:8888/ambassador/mapping
```

which should show you something like

```json
{
  "count": 1,
  "hostname": "ambassador-3176426918-13v2v",
  "mappings": [
    {
      "modules": {},
      "name": "qotm_map",
      "prefix": "/qotm/",
      "rewrite": "/",
      "service": "qotm"
    }
  ],
  "ok": true,
  "resolvedname": "109.196.3.8",
  "version": "{VERSION}"
}
```

To actually _use_ the QotM service, we need the URL for microservice access through Ambassador. This is, sadly, a little harder than one might like. If you're using AWS, GKE, or Minikube, you may be able to use the commands below -- **note that these will only work since we already know we're using HTTP**:

```shell
# AWS (for Ambassador using HTTP)
AMBASSADORURL=http://$(kubectl get service ambassador --output jsonpath='{.status.loadBalancer.ingress[0].hostname}')

# GKE (for Ambassador using HTTP)
AMBASSADORURL=http://$(kubectl get service ambassador --output jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Minikube (for Ambassador using HTTP)
AMBASSADORURL=$(minikube service --url ambassador)
```

If that doesn't work out, look at the `LoadBalancer Ingress` line of `kubectl describe service ambassador` and set `$AMBASSADORURL` based on that. **Do not include a trailing `/`** on it, or our examples below won't work.

Once `$AMBASSADORURL` is set, you'll be able to use that for a basic health check on the QotM service:

```shell
curl -v $AMBASSADORURL/qotm/health
```

If all goes well you should get an empty response with an HTTP 200 response:

```shell
...
HTTP/1.1 200 OK
content-length: 0
content-type: text/html; charset=utf-8
...
```

Since the `/qotm/` prefix in the path portion of the URL there matches the prefix we used for the `qotm_map` mapping above, Ambassador knows to route the request to the QotM service. In the process it rewrites `/qotm/` to `/` so that `/qotm/health` becomes `/health`, which is what the QotM service expects. (This rewriting is configurable; `/` is just the default.)

Suppose we want a quote for this moment?

```shell
curl $AMBASSADORURL/qotm/
```

(Note that the trailing `/` is mandatory the way we've set things up.) This should return something like

```json
{
  "hostname": "qotm-424688516-883pl",
  "quote": "A small mercy is nothing at all?",
  "time": "2017-06-22T03:53:22.074919"
}
```

and repeating that should yield other (kind of surreal) quotes.

The QotM service also has an endpoint to supply new quotes, which should be accessible now:

```shell
curl -XPOST -H"Content-Type: application/json" \
     -d'{ "quote": "The grass is never greener anywhere else." }' \
     $AMBASSADORURL/qotm/quote
```

That should return the ID of the new quote:

```json
{
  "hostname": "qotm-424688516-883pl",
  "quote": "The grass is never greener anywhere else.",
  "quoteid": 10,
  "time": "2017-06-22T03:57:03.339907"
}
```

and we should be able to read that back with

```shell
curl $AMBASSADORURL/qotm/quote/10
```

But it's probably not a good idea to allow any random person to update our quotations. We can use Ambassador's built-in authentication to prevent that. First, we turn it on by enabling Ambassador's basic-auth module:

```shell
curl -XPUT -H"Content-Type: application/json" \
     -d'{ "ambassador": "basic" }' \
     http://localhost:8888/ambassador/module/authentication
```

That activates the `authentication` module, with `ambassador: basic` as configuration information (in this case telling Ambassador to use its built-in "basic" authentication mechanism).

Next, we add a mapping for `/qotm/quote/` that requires auth:

```shell
curl -XPUT -H"Content-Type: application/json" \
     -d'{ "prefix": "/qotm/quote/", "rewrite": "/quote/", "service": "qotm", "modules": { "authentication": { "type": "basic" } } }' \
     http://localhost:8888/ambassador/mapping/qotm_quote_map
```

That last bit configures this mapping to use the `authentication` module, with config info `type: basic`. (Note also that we use `rewrite` to make sure the QotM service sees the base URL it expects.)

Now, if we try to read our quote back:

```shell
curl $AMBASSADORURL/qotm/quote/10
```

then we should get a 401, since we haven't authenticated.

```shell
HTTP/1.1 401 Unauthorized
auth-service: Ambassador BasicAuth {VERSION}
content-length: 25
content-type: text/html; charset=utf-8
date: Thu, 22 Jun 2017 14:37:28 GMT
server: envoy
www-authenticate: Basic realm="Login Required"
x-envoy-upstream-service-time: 5

No authorization provided
```

We need to provide an authorization, but in order to do that, we need to tell Ambassador who can log in. We do this by defining a `consumer` in Ambassador:

```shell
curl -XPOST -H"Content-Type: application/json" \
     -d'{ "username": "alice", "fullname": "Alice Rules", "modules": { "authentication": { "type":"basic", "password":"alice" } } }' \
     http://localhost:8888/ambassador/consumer
```

That will create a new `consumer` for Alice, and return her `consumer_id`:

```json
{
    "consumer_id": "5D86FCDF509B47CCB8CA64EA4561785E",
    "hostname": "ambassador-3176426918-13v2v",
    "ok": true,
    "resolvedname": "109.196.3.8",
    "version": "{VERSION}"
}
```

We can use Alice's `consumer_id` to read back information about Alice:

```shell
curl http://localhost:8888/ambassador/consumer/5D86FCDF509B47CCB8CA64EA4561785E
```

which will return something like:

```json
{
    "consumer_id": "5D86FCDF509B47CCB8CA64EA4561785E",
    "fullname": "Alice Rules",
    "hostname": "ambassador-3176426918-13v2v",
    "modules": {
        "authentication": {
            "password": "alice",
            "type": "basic"
        }
    },
    "ok": true,
    "resolvedname": "109.196.3.8",
    "shortname": "Alice Rules",
    "username": "alice",
    "version": "{VERSION}"
}
```

and we can now authenticate to the QotM service as Alice:

```shell
curl -u alice:alice $AMBASSADORURL/qotm/quote/10
```
