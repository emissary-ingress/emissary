# Filters

Filters are used to extend Ambassador to run additional custom logic on each request. Filters are writting in Golang.

## Filter Definition

Filters are registered with the `Filter` custom resource definition.

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: param-filter # This is the name used in FilterPolicy
  namespace: standalone
spec:
  Plugin:
    name: param-filter # The plugin's `.so` file's base name
```

## Filter Configuration

The `FilterPolicy` custom resource definition specifies which filters are run, and when. In the example below, the `param-filter` registered above is run on requests to `/test/` and `/httpbin/`. 

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: FilterPolicy
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
    filters: null # make this path public
  - host: "*"
    path: /httpbin/user-agent
    filters:
    - name: param-filter
  - host: "*"
    path: /httpbin/headers
    filters:
    - name: param-filter
    - name: auth0
```

When multiple `Filter`s are specified in a rule:

* The filters are gone through in order
* Later filters have access to _all_ headers inserted by earlier filters.
* The final backend service (i.e., the service where the request will ultimately be routed) will only have access to inserted headers if they are listed in `allowed_authorization_headers` in the Ambassador annotation.
* Filter processing is aborted by the first filter to return a non-200 status.

## The Filter interface

A Filter has one basic interface:

```
func PluginMain(w http.ResponseWriter, r *http.Request) { … }
```

`*http.Request` is the incoming HTTP request that can be mutated or intercepted, which is done by `http.ResponseWriter`.

Headers can be mutated by calling `w.Header().Set(HEADERNAME, VALUE)`.  Finalize changes by calling `w.WriteHeader(http.StatusOK)`.

If you call `w.WriteHeader()` with any value other than 200 (`http.StatusOK`) instead of modifying the request, the middleware has taken over the request.  You can call `w.Write()` to write the body of an error page.

## Creating and Deploying Filters

We've created an example filter that you can customize for your particular use case.

1. Start with the example filter: `git clone https://github.com/datawire/apro-example-plugin/`.

2. Make code changes to `param-plugin.go`. Note: If you're developing a non-trivial filter, see the rapid development section below for a faster way to develop and test your filter.

3. Run `make DOCKER_REGISTRY=...`, setting `DOCKER_REGISTRY` to point to a registry you have access to. This will generate a Docker image named `$DOCKER_REGISTRY/amb-sidecar-plugin:VERSION`.

4. Push the image to your Docker registry: `docker push $DOCKER_REGISTRY/amb-sidecar-plugin:VERSION`.

5. Configure Ambassador Pro to use the plugin by creating a `Filter` and `FilterPolicy` CRD, as per above.

6. If you're adding additional headers, configure the `AuthService` configuration to allow the filter to inject the new header, e.g.,


   ```patch
   allowed_authorization_headers:
   - "Authorization"
   - "Client-Id"
   - "Client-Secret"
+      - "X-Wikipedia"
   ```

7. Update the standard Ambassador Pro manifest to use your Docker image instead of the standard sidecar.

    ```patch
       containers:
       - name: ambassador-pro
    -    image: quay.io/datawire/ambassador_pro:amb-sidecar-0.1.2
    +    image: DOCKER_REGISTRY/amb-sidecar-plugin:VERSION
         ports:
         - name: ratelimit-grpc
           containerPort: 8081
         - name: ratelimit-debug
           containerPort: 6070
    ```

## Rapid development of a custom filter

During development, you may want to sidestep the deployment process for a faster development loop. The [apro-plugin-runner](https://github.com/datawire/apro-plugin-runner) helps you rapidly develop Ambassador filters.

To install the runner, follow these steps:

1. Clone the repository: `git clone https://github.com/datawire/apro-plugin-runner`.
2. Install the repository: `go get github.com/datawire/apro-plugin-runner`.
3. The binary is in placed in `$(go env GOPATH/bin)`. Make sure to update your `$PATH` to include this directory.

Now, you can quickly test and develop your filter.

1. In your filter directory, type: `apro-plugin-runner :8080 ./param-plugin.so`.
2. Test the filter by running `curl`:

    ```
    $ curl -v localhost:8080?db=2
    * Rebuilt URL to: localhost:8080/?db=2
    *   Trying ::1...
    * TCP_NODELAY set
    * Connected to localhost (::1) port 8080 (#0)
    > GET /?db=2 HTTP/1.1
    > Host: localhost:8080
    > User-Agent: curl/7.54.0
    > Accept: */*
    >
    < HTTP/1.1 200 OK
    < X-Dc: Even
    < Date: Mon, 25 Feb 2019 19:58:38 GMT
    < Content-Length: 0
    <
    * Connection #0 to host localhost left intact
    ```

    Note in the example above the `X-Dc` header is added. This lets you inspect the changes the filter is making to your HTTP header.