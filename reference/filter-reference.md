# Filters

Filters are used to extend Ambassador to run additional custom logic on each request. Filters are writting in Golang.

## Filter Definition

Filters are registered with the `Filter` custom resource definition.

```yaml
---
apiVersion: getambassador.io/v1beta1
kind: Filter
metadata:
  name: param-filter
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

## Creating Filters

We've made it easy for you to get started with writing a Filter.

1. Start with the example filter: `git clone https://github.com/datawire/apro-example-plugin/`.

2. Make code changes to `param-plugin.go`.

3. Run `make DOCKER_REGISTRY=...`, setting `DOCKER_REGISTRY` to point to a registry you have access to. This will generate a Docker image named `$DOCKER_REGISTRY/amb-sidecar-plugin:VERSION`.

4. Push the image to your Docker registry.

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