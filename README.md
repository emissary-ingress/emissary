# Ambassador Pro example middleware

This is an example middleware for Ambassador Pro that injects an
`X-Wikipedia` header with the URL for a random Wikipedia article
before passing the request off to the backend service.  It uses
https://en.wikipedia.org/wiki/Special:Random to obtain the URL; as an
example of a middleware needing to talk to an external service.

## Compiling

Just run

	$ make DOCKER_REGISTRY=...

It will generate a Docker image named

	DOCKER_REGISTRY/amb-sidecar-plugin:VERSION
	
where:

 - VERSION is `git describe --tags --always --dirty`.
 - DOCKER_REGISTRY is the `$DOCKER_REGISTRY` environment variable, or
   `localhost:31000` if the variable is not set.

## Deploying

Push that `amb-sidecar-plugin` Docker image to a registry that your
cluster has access to.  You can do this by running `make push
DOCKER_REGISTRY=...`.

Use that image you just pushed instead of
`quay.io/datawire/ambassador_pro:amb-sidecar` when deploying
Ambassador Pro.

Tell Ambassador Pro about the plugin:

```yaml
---
apiVersion: getambassador.io/v1beta1
kind: Middleware
metadata:
  name: wikiplugin # how we'll refer to the plugin in our Policy (below)
  namespace: standalone
spec:
  Plugin:
    name: example-plugin # The plugin's `.so` file's base name
```

Tell Ambassador Pro when to use that middleware:

```yaml
---
apiVersion: getambassador.io/v1beta1
kind: Policy
metadata:
  name: httpbin-policy
  namespace: standalone
spec:
  # everything defaults to private; you can create rules to make stuff
  # public, and you can create rules to require additional scopes
  # which will be automatically checked
  rules:
  - host: "*"
    path: /httpbin/ip
    public: true
  - host: "*"
    path: /httpbin/headers
    public: false # must be false if using a middleware
    middleware:
      name: wikiplugin
```

Finally, edit the Ambassador Pro manifest to (1) use the image with
the plugin, and (2) allow the plugin to inject the `X-Wikipedia`
header:

```patch
@ -171,10 +171,11 @@ metadata:
       - "Client-Secret"
       allowed_authorization_headers:
       - "Authorization"
       - "Client-Id"
       - "Client-Secret"
+      - "X-Wikipedia"
       ---
       apiVersion: ambassador/v1
       kind: Mapping
       name: callback_mapping
       prefix: /callback
@@ -210,11 +211,11 @@ spec:
         service: ambassador-pro
     spec:
       serviceAccountName: ambassador-pro
       containers:
       - name: ambassador-pro
-        image: {{env "AMB_SIDECAR_IMAGE"}}
+        image: localhost:31000/amb-sidecar-plugin:3
         ports:
         - name: ratelimit-grpc
           containerPort: 8081
         - name: ratelimit-debug
           containerPort: 6070
```

## How a plugin works

In `package main`, define a functions named `PluginMain` with this
signature:

	func PluginMain(w http.ResponseWriter, r *http.Request) { … }

The `*http.Request` is the incoming HTTP request that you have the
opportunity to mutate or intercept, which you do via the
`http.ResponseWriter`.

You may mutate the headers by calling `w.Header().Set(HEADERNAME,
VALUE)`.  Finalize this by calling `w.WriteHeader(http.StatusOK)`

If you call `w.WriteHeader()` with any value other than 200
(`http.StatusOK`) instead of modifying the request, the middleware has
taken over the request.  You can call `w.Write()` to write the body of
an error page.

This is all very similar to writing an Envoy ext_authz authorization
service.
