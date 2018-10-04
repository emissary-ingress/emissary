# Demo in Detail

Ambassador is an API gateway for microservices, so to get started, it's helpful to actually have a running service to use it with. We'll use Datawire's "Quote of the Moment" service (`qotm`) for this; you can deploy it into Kubernetes with

```shell
kubectl apply -f https://www.getambassador.io/yaml/demo/demo-qotm.yaml
```

This will create a deployment called `qotm` and a corresponding Kubernetes service entry that's also called `qotm`. Quote of the Moment supports a very simple REST API:

* `GET /health` performs a simple health check
* `GET /` returns a random Quote of the Moment
* `GET /quote/:quoteid` returns the Quote of the Moment with a given ID
* `POST /quote` adds a new Quote of the Moment and returns its ID
  * this requires that the POST body carry the new Quote of the Moment

We'll use the health check as our first simple test to make sure that Ambassador is relaying requests, but of course we want all of the above to work through Ambassador -- and we want everything using the `/quote` endpoint to require authentication.

At its heart, Ambassador is controlled by a collection of YAML files that tell it which URLs map to which services. The most straightforward way to run Ambassador is to push these configuration files into a Kubernetes `ConfigMap` named `ambassador-config`, then use Datawire's published Ambassador image to read the published configuration at boot time. When you need to change the configuration, you update the `ConfigMap`, then use Kubernetes' deployment machinery to trigger a rollout.

To set this up, we'll start by creating the `ConfigMap` from a directory of YAML files. In any real scenario, this would be a directory under revision control, but for now, we'll just create an empty `config` directory:

```
mkdir config
```

Our configuration will start with a single mapping: the `/qotm/` resource will be mapped to the Quote of the Moment service. Here's the YAML for that, which we'll put into `config/mapping-qotm.yaml`:

```yaml
---
apiVersion: ambassador/v0
kind: Mapping
name: qotm_mapping
prefix: /qotm/
service: qotm
```

Once that's done, we can create the `ambassador-config` map from our `config` directory:

```shell
kubectl create configmap ambassador-config --from-file config
```

Now we can start Ambassador running in the Kubernetes cluster. We recommend using [TLS](running.md#TLS), but for right now we'll just set up an HTTP-only Ambassador to show you how things work:

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador.yaml
```

This will create the Ambassador service, listening on port 80, and the Ambassador deployment itself. Again, in the real world you'll definitely want TLS, but this is enough for our purposes here.

To actually use the QotM service, we need the URL for microservice access through Ambassador. This is, sadly, a little harder than one might like. If you're using AWS, GKE, or Minikube, you may be able to use the commands below -- **note that these will only work since we already know we're using HTTP**:

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

If all goes well you should get an HTTP 200 response with a JSON body, something like:

```json
{
  "hostname": "qotm-2399866569-9q4pz",
  "msg": "QotM health check OK",
  "ok": true,
  "time": "2017-09-15T04:09:51.897241",
  "version": "1.1"
}
```

Since the `/qotm/` prefix in the path portion of the URL there matches the prefix we used for the `qotm_map` mapping above, Ambassador knows to route the request to the QotM service. In the process it rewrites `/qotm/` to `/` so that `/qotm/health` becomes `/health`, which is what the QotM service expects. (This rewriting is configurable; `/` is just the default.)

Suppose we want a quote for this moment?

```shell
curl $AMBASSADORURL/qotm/
```

(Note that the trailing `/` is mandatory the way we've set things up.) This should return something like

```json
{
  "hostname": "qotm-2399866569-9q4pz",
  "ok": true,
  "quote": "Non-locality is the driver of truth. By summoning, we vibrate.",
  "time": "2017-09-15T04:18:10.371552",
  "version": "1.1"
}
```

and repeating that should yield other (kind of surreal) quotes.

The QotM service also has an endpoint that allows teaching it new quotes, which should be accessible now:

```shell
curl -XPOST -H"Content-Type: application/json" \
     -d'{ "quote": "The grass is never greener anywhere else." }' \
     $AMBASSADORURL/qotm/quote
```

That should return the ID of the new quote:

```json
{
  "hostname": "qotm-2399866569-9q4pz",
  "ok": true,
  "quote": "The grass is never greener anywhere else.",
  "quoteid": 10,
  "time": "2017-09-15T04:19:06.367547",
  "version": "1.1"
}
```

and we should be able to read that back with

```shell
curl $AMBASSADORURL/qotm/quote/10
```

But it's probably not a good idea to allow any random person to update our quotations. Ambassador supports an external authentication service for exactly this reason: let's set it up, using a very very simple demo authentication service:

```shell
kubectl apply -f https://www.getambassador.io/yaml/demo/demo-auth.yaml
```

That will start the demo auth service running. The auth service:

- listens for requests on port 3000;
- expects all URLs to begin with `/extauth/`;
- performs HTTP Basic Auth for all URLs starting with `/qotm/quote/` (other URLs are always permitted);
- accepts only user `username`, password `password`; and
- makes sure that the `x-qotm-session` header is present, generating a new one if needed.

Once the auth service is running, add `config/module-authentication.yaml`:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  authentication
config:
  auth_service: "example-auth:3000"
  path_prefix: "/extauth"
  allowed_headers:
  - "x-qotm-session"
```

which tells Ambassador about the auth service, notably that it needs the `/extauth` prefix, and that it's OK for it to pass back the `x-qotm-session` header. Note that `path_prefix` and `allowed_headers` are optional.

We don't actually need to change any mappings, since the auth service is responsible for knowing about which things need auth and which don't. However, if we try pull quote 10 again:

```shell
curl $AMBASSADORURL/qotm/quote/10
```

...then you'll see that it works, even though it seems like it should need authentication!

The problem is that we only changed the Ambassador config on the local disk -- we need to update the `ConfigMap`, and we need to force a new rollout of Ambassador so that it can reconfigure everything.

We can update the `ConfigMap` by using `kubectl` to rewrite it in place:

```shell
kubectl create configmap ambassador-config --from-file config -o yaml --dry-run | \
    kubectl replace -f -
```

This neat little trick uses `-o yaml --dry-run` option to make `kubectl` write a file that we can feed into `kubectl replace`, since `kubectl replace` doesn't understand the `--from-file` option. The result is that the `ambassador-config` `ConfigMap` gets updated with all the new config at once.

Once that's done, we need to trigger a new rollout of Ambassador using the Kubernetes deployment machinery. There are several ways to do this, but the most painless (unless you happen to need to upgrade to a new Ambassador release!) is to update an annotation on the deployment:

```shell
kubectl patch deployment ambassador -p \
  "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"date\":\"`date +'%s'`\"}}}}}"
```

This simply patches the deployment with an annotation containing a timestamp of when the configuration was last updated. Kubernetes will respond by rolling out new Ambassador pods, which will pick up the new configuration. You can use `kubectl rollout status deployment/ambassador` to keep an eye on this process.

Once that's all done, if we try to read our quote back:

```shell
curl $AMBASSADORURL/qotm/quote/10
```

then we should get a 401, since we haven't authenticated.

```shell
HTTP/1.1 401 Unauthorized
x-powered-by: Express
x-request-id: 9793dec9-323c-4edf-bc30-352141b0a5e5
www-authenticate: Basic realm="Ambassador Realm"
content-type: text/html; charset=utf-8
content-length: 0
etag: W/"0-2jmj7l5rSw0yVb/vlWAYkK/YBwk"
date: Fri, 15 Sep 2017 15:22:09 GMT
x-envoy-upstream-service-time: 2
server: envoy
```

It will work, though, if we authenticate to the QotM service:

```shell
curl -v -u username:password $AMBASSADORURL/qotm/quote/10
```

which should now return something like the following (including the `x-qotm-session` header!):

```
HTTP/1.1 200 OK
content-type: application/json
x-qotm-session: 599969b7-8f00-4663-bb8c-fefd3dfc6174
content-length: 176
server: envoy
date: Fri, 15 Sep 2017 15:39:58 GMT
x-envoy-upstream-service-time: 4

{
  "hostname": "qotm-2399866569-9q4pz",
  "ok": true,
  "quote": "The grass is never greener anywhere else.",
  "time": "2017-09-15T15:39:59.200961",
  "version": "1.1"
}
```
