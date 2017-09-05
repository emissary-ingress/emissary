# Auth with an External Auth Service

As an alternative to Ambassador's built-in authentication service, you can direct Ambassador to use an external authentication service. If set up this way, Ambassador will query the auth service on every request. It is up to the service to decide whether to accept the request or how to reject it.

## Ambassador External Auth API

When Ambassador is configured to use the external auth service, the method and headers of every incoming request are forwarded to the auth service, with two changes:

1. The `Host` header is overwritten with the host information of the external auth service.
2. The body is removed.

So, for example, if the incoming request is 

```
PUT /path/to/service HTTP/1.1
Host: myservice.example.com:80
User-Agent: curl/7.54.0
Accept: */*
Content-Type: application/json
Content-Length: 27

{ "greeting": "hello world!", "spiders": "OMG no "}
```

then the request Ambassador will make of the auth service is:

```
PUT /path/to/service HTTP/1.1
Host: extauth.example.com:80
User-Agent: curl/7.54.0
Accept: */*
Content-Type: application/json
Content-Length: 0
```

**ALL** request methods will be proxied; the auth service should be able to handle any request that any client could make. If desired, Ambassador can add a prefix to the path before forwarding it to the auth service; see the example below.

If Ambassador cannot reach the auth service, it returns 503 to the client. If Ambassador receives any response from the auth service other than 200, it returns that full response (header and body) to the client. Ambassador assumes the auth service will return an appropriate response, such as 401.

If the auth service response code is 200, then Ambassador allows the client request to resume being processed by the normal Ambassador Envoy flow. This typically means that the client will receive the expected response to its request.

Additionally, Ambassador can be configured to allow headers from the auth service to be passed back to the client when the auth service returns 200; see the example below.

## Example

We will use an example auth service to demonstrate the feature. You can deploy it into Kubernetes with

```shell
kubectl apply -f https://raw.githubusercontent.com/datawire/ambassador-auth-service/master/example-auth.yaml
```

Let's also set things up as in the [Getting Started](../user-guide/getting-started.md) section up to the point where we want to add authentication. Here's the short version; read the full text for the details, particularly for how to set up `$AMBASSADORURL`.

```shell
# Add Quote of the Moment service
kubectl apply -f https://www.getambassador.io/yaml/demo/demo-qotm.yaml

# Add Ambassador
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-http.yaml
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador.yaml

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
     -d'{ "auth_service": "example-auth:3000", "path_prefix": "/extauth", "allowed_headers": [ "x-qotm-session" ] }' \
     http://localhost:8888/ambassador/module/authentication
```

Here, we're telling extauth to use the `example-auth` service on port 3000, to prepend `/extauth` to every path before handing it to the auth service, and to allow the `x-qotm-session` header to be included in responses from the auth service. Note that `path_prefix` and `allowed_headers` are optional; omit them if you don't want them.

The [`example-auth` service (GitHub repo)](https://github.com/datawire/ambassador-auth-service) is a simple Node/Express implementation of the API described above. It implements HTTP Basic Authentication, where the only valid credentials are user `username` and password `password`, and only requires authentication when the original path requested begins with `/service` (which means that the auth service sees a path starting with `/extauth/service`).

If auth is not required for the request, the service returns 200 immediately.

If auth _is_ required and a valid username and password are supplied, the extauth service returns a 200 to allow the client request to go through, and it makes sure that an `x-qotm-session` header is present (if the client provided one, great! otherwise, the extauth service will create a new session value). Otherwise it returns a 401 with an HTTP Basic Auth WWW-Authenticate header.

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

For any successful request made to `/service`, you should see the `x-qotm-session` header in the response.
