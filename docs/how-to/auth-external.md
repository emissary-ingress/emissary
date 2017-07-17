# Auth with an External Auth Service

As an alternative to Ambassador's built-in authentication service, you can direct Ambassador to use an external authentication service. If set up this way, Ambassador will query the auth service on every request. It is up to the service to decide whether to accept the request or how to reject it.

## Ambassador External Auth API

Ambassador sends a POST request to the auth service at the path `/ambassador/auth` with body  containing a JSON mapping of the request headers in HTTP/2 style, e.g., `:authority` instead of `Host`. If Ambassador cannot reach the auth service, it returns 503 to the client. If the auth service response code is 200, then Ambassador allows the client request to resume being processed by the normal Ambassador Envoy flow. This typically means that the client will receive the expected response to its request. If Ambassador receives any response from the auth service other than 200, it returns that full response (header and body) to the client. Ambassador assumes the auth service will return an appropriate response, such as 401.

## Example

We will use an example auth service to demonstrate the feature. You can deploy it into Kubernetes with

```shell
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador-auth-service/master/example-auth.yaml
```

Let's also set things up as in the [Getting Started](../user-guide/getting-started.md) section up to the point where we want to add authentication. Here's the short version; read the full text for the details, particularly for how to set up `$AMBASSADORURL`.

```shell
# Add Quote of the Moment service
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/demo-qotm.yaml

# Add Ambassador
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/ambassador-http.yaml
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador/master/ambassador.yaml

# Set up port-forwarding
POD=$(kubectl get pod -l service=ambassador -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward "$POD" 8888 &

# Map QotM at /qotm/
curl -XPUT -H"Content-Type: application/json" \
     -d'{ "prefix": "/qotm/", "service": "qotm" }' \
     http://localhost:8888/ambassador/mapping/qotm_map

# Set up $AMBASSADORURL -- no trailing / -- see Getting Started
AMBASSADORURL=...

# Verify things are working
curl $AMBASSADORURL/qotm/
```

Now let's turn on authentication using the `example-auth` service we deployed earlier. Instead of using `ambassador: basic` as the authentication configuration, use `auth_service: example-auth:3000`. This tells Ambassador to send authentication requests to our service.

```shell
curl -XPUT -H"Content-Type: application/json" \
     -d'{ "auth_service": "example-auth:3000" }' \
     http://localhost:8888/ambassador/module/authentication
```

The [`example-auth` service (GitHub repo)](https://github.com/datawire/ambassador-auth-service) is a simple Node/Express implementation of the API described above. In short, the service expects a POST to the path `/ambassador/auth` with client request headers as a JSON map in the POST body. If auth is okay (HTTP Basic Auth), it returns a 200 to allow the client request to go through. Otherwise it returns a 401 with an HTTP Basic Auth WWW-Authenticate header. This example auth service only performs auth when the client headers indicate a request under the `/service` path prefix. The only valid credentials are `username:password`.

Let's create a second mapping for QotM at `/service/` so we can demonstrate authentication.

```shell
curl -XPUT -H"Content-Type: application/json" \
     -d'{ "prefix": "/service/", "service": "qotm" }' \
     http://localhost:8888/ambassador/mapping/qotm_service_map
```

You should still be able to access QotM through the old mapping:

```shell
curl -v $AMBASSADORURL/qotm/
```

but you'll get a 401 if you try to use the new mapping:

```shell
curl -v $AMBASSADORURL/service/
```

unless you use the correct credentials:

```shell
curl -v -u username:password $AMBASSADORURL/service/
```
