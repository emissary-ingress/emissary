# Filters

Filters are used to extend Ambassador to modify or intercept an HTTP
request before sending to the the backend service.  You may use any of
the built-in Filter types, or use the `Plugin` filter type to run
custom code written in Golang.

Filters are created with the `Filter` resource type, which contains
global arguments to that filter.  Which Filter(s) to use for which
HTTP requests is then configured in `FilterPolicy` resources, which
may contain path-specific arguments to the filter.

## `Filter` Definition

Filters are created as `Filter` resources.  The body of the resource
spec depends on the filter type:

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: "string"
spec:
  ambassador_id:           # optional; default is ["default"]
  - "string"
  ambassador_id: "string"  # no need for a list if there's only one value
  FILTER_TYPE:
    GLOBAL_FILTER_ARGUMENTS
```

### Filter Type: `OAuth2`

The `OAuth2` filter type performs OAuth2 authorization against an
identity provider implementing [OIDC Discovery][].

[OIDC Discovery]: https://openid.net/specs/openid-connect-discovery-1_0.html

#### `OAuth2` Global Arguments

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: "example-filter"
spec:
  OAuth2:
    authorizationURL: "url-string"      # required
    clientURL:        "url-string"      # required
    stateTTL:         "duration-string" # optional; default is "5m"
    audience:         "string"
    clientID:         "string"
    secret:           "string"
    MaxStale:         "duration-string" # optional; default is "0"
```

 - `authorizationURL` identifies where to look for the
   `/.well-known/openid-configuration` descriptor to figure out how to
   talk to the OAuth2 provider. (Is this equivalent to what the OIDC
   Discovery spec calls the "Issuer"?  If so, should we rename it to
   Issuer?)
 - `clientURL` identifies a hostname that can appropriate set cookies
   for the application.  Only the scheme (`https://`) and authority
   (`example.com:1234`) parts are used; the path part of the URL is
   ignored.
 - stateTTL: TODO
 - audience: TODO
 - clientID: The client ID you got from your IDP.
 - secret: The client secret you got from your IDP.
 - `maxStale`: How long to keep stale cache OIDC replies for.  This
   sets the `max-stale` Cache-Control directive on requests, and also
   ignores the `no-store` and `no-cache` Cache-Control directives on
   responses.  This is useful for working with IDPs with
   mis-configured Cache-Control.

`"duration-string"` strings are parsed as a sequence of decimal
numbers, each with optional fraction and a unit suffix, such as
"300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or
"µs"), "ms", "s", "m", "h".  See [Go
`time.ParseDuration`][time.ParseDuration].

time.ParseDuration: https://golang.org/pkg/time/#ParseDuration

#### `OAuth2` Path-Specific Arguments

```
---
apiVersion: getambassador.io/v1beta2
kind: FilterPolicy
metadata:
  name: "example-filter-policy"
spec:
  rules:
  - host: "*"
    path: "*"
    filters:
    - name: "example-filter"
      arguments:
        scopes:
        - "scope1"
        - "scope2"
```

You may specify a list of OAuth2 scopes to apply to the authorization
request.

### Filter Type: `Plugin`

The `Plugin` filter type allows you to plug in your own custom code.
This code is compiled to a `.so` file, which you load in to the
Ambassador Pro container at `/etc/ambassador-plugins/${NAME}.so`.

### The Plugin Interface

This code is written in Golang, and must be compiled with the exact
compiler settings as Ambassador Pro.  As of Ambassador Pro v0.2.0-rc1,
that is:

 - `go` compiler `1.11.4`
 - `GOOS=linux`
 - `GOARCH=amd64`
 - `GO111MODULE=on`
 - `CGO_ENABLED=1`

Plugins are compiled with `go build -buildmode=plugin`, and must have
a `main.PluginMain` function with the signature `PluginMain(w
http.ResponseWriter, r *http.Request)`:

```go
package main

import (
	"net/http"
)

func PluginMain(w http.ResponseWriter, r *http.Request) { … }
```

`*http.Request` is the incoming HTTP request that can be mutated or
intercepted, which is done by `http.ResponseWriter`.

Headers can be mutated by calling `w.Header().Set(HEADERNAME, VALUE)`.
Finalize changes by calling `w.WriteHeader(http.StatusOK)`.

If you call `w.WriteHeader()` with any value other than 200
(`http.StatusOK`) instead of modifying the request, the plugin has
taken over the request, and the request will not be sent to the
backend service.  You can call `w.Write()` to write the body of an
error page.

## Creating and Deploying Filters

We've created an example filter that you can customize for your particular use case.

1. Start with the example filter: `git clone
   https://github.com/datawire/apro-example-plugin/`.

2. Make code changes to `param-plugin.go`. Note: If you're developing a non-trivial filter, see the rapid development section below for a faster way to develop and test your filter.

3. Run `make DOCKER_REGISTRY=...`, setting `DOCKER_REGISTRY` to point
   to a registry you have access to. This will generate a Docker image
   named `$DOCKER_REGISTRY/amb-sidecar-plugin:VERSION`.

4. Push the image to your Docker registry: `docker push $DOCKER_REGISTRY/amb-sidecar-plugin:VERSION`.

5. Configure Ambassador Pro to use the plugin by creating a `Filter`
   and `FilterPolicy` CRD, as per above.

6. If you're adding additional headers, configure the `AuthService`
   configuration to allow the filter to inject the new header, e.g.,

   ```patch
   allowed_authorization_headers:
   - "Authorization"
   - "Client-Id"
   - "Client-Secret"
   +      - "X-Wikipedia"
   ```

7. Update the standard Ambassador Pro manifest to use your Docker
   image instead of the standard sidecar.

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

#### `Plugin` Global Arguments

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: Filter
metadata:
  name: "example-filter" # This is how to refer to the Filter in a FilterPolicy
spec:
  Plugin:
    name: "string" # This tells it where to look for the compiled plugin file; "/etc/ambassador-plugins/${NAME}.so"
```

#### `Plugin` Path-Specific Arguments

Path specific arguments are not supported for Plugin filters at this
time.

## `FilterPolicy` Definition

`FilterPolicy` resources specify which filters (if any) to apply to
which HTTP requests.

```yaml
---
apiVersion: getambassador.io/v1beta2
kind: FilterPolicy
metadata:
  name: "example-filter-policy"
spec:
  rules:
  - host: "glob-string"
    path: "glob-string
    filters:                 # optional; omit or set to `null` to apply no filters to this request
    - name: "exampe-filter"  # required
      namespace: "string"    # optional; default is the same namespace as the FilterPolicy
      arguments: DEPENDS     # optional
```

The type of the `arguments` property is dependent on the which Filter
type is being referred to; see the "Path-Specific Arguments"
documentation for each Filter type.

When multiple `Filter`s are specified in a rule:

 * The filters are gone through in order
 * Later filters have access to _all_ headers inserted by earlier
   filters.
 * The final backend service (i.e., the service where the request will
   ultimately be routed) will only have access to inserted headers if
   they are listed in `allowed_authorization_headers` in the
   Ambassador annotation.
 * Filter processing is aborted by the first filter to return a
   non-200 status.

## Example

In the example below, the `param-filter` Filter Plugin is loaded, and
configured to run on requests to `/httpbin/`.

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

---
apiVersion: getambassador.io/v1beta2
kind: FilterPolicy
metadata:
  name: httpbin-policy
spec:
  rules:
  # Default to authorizing all requests with auth0
  - host: "*"
    path: "*"
    filters:
    - name: auth0
  # And also apply the param-filter to requests for /httpbin/
  - host: "*"
    path: /httpbin/*
    filters:
    - name: param-filter
    - name: auth0
  # But don't apply any filters to requests for /httpbin/ip
  - host: "*"
    path: /httpbin/ip
    filters: null
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
