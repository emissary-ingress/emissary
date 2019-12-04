# Developing Filters

Sometimes you may want Ambassador Edge Stack to manipulate an incoming request. Some example use cases:

* Inspect an incoming request, and add a custom header that can then be used for routing
* Add custom Authorization headers
* Validate an incoming request fits an OpenAPI specification before passing the request to a target service

Ambassador Edge Stack supports these use cases by allowing you to execute custom logic in `Filters`. Filters are written in Golang, and managed by Ambassador Edge Stack.

## Prerequisites

`Plugin` `Filter`s are built as [Go plugins](https://golang.org/pkg/plugin/) and loaded directly into the Ambassador Pro container so they can run in-process with the rest of Ambassador Pro.

To build a `Plugin` `Filter` into the Ambassador Pro container you will need
- Linux or MacOS host (Windows Subsystem for Linux is ok)
- [Docker](https://docs.docker.com/v17.09/engine/installation/) 
- [rsync](https://rsync.samba.org/)

The `Plugin` `Filter` is built by `make` which uses Docker to create a stable build environment in a container and `rsync` to copy files between the container and your host machine.

See the [README](https://github.com/datawire/apro-example-plugin) for more information on how the `Plugin` works.

## Creating and Deploying Filters

We've created an example filter that you can customize for your particular use case.

1. Start with the example filter: `git clone
   https://github.com/datawire/apro-example-plugin/`.

2. Make code changes to `param-plugin.go`. Note: If you're developing a non-trivial filter, see the rapid development section below for a faster way to develop and test your filter.

3. Run `make DOCKER_REGISTRY=...`, setting `DOCKER_REGISTRY` to point
   to a registry you have access to. This will generate a Docker image
   named `$DOCKER_REGISTRY/amb-sidecar-plugin:VERSION`.

4. Push the image to your Docker registry: `docker push $DOCKER_REGISTRY/amb-sidecar-plugin:VERSION`.

5. Configure Ambassador Edge Stack to use the plugin by creating a `Filter`
   and `FilterPolicy` CRD, as per the [filter reference](/reference/filter-reference).

6. Update the standard Ambassador Edge Stack manifest to use your Docker
   image instead of the standard sidecar.

   ```patch
      containers:
      - name: ambassador-pro
   -    image: quay.io/datawire/ambassador_pro:amb-sidecar-$aproVersion$
   +    image: DOCKER_REGISTRY/amb-sidecar-plugin:VERSION
        ports:
        - name: ratelimit-grpc
          containerPort: 8081
        - name: ratelimit-debug
          containerPort: 6070
   ```

## Rapid development of a custom filter

During development, you may want to sidestep the deployment process for a faster development loop. The `apro-plugin-runner` helps you rapidly develop Ambassador Edge Stack filters locally.

To install the runner, download the latest version:

<a class="pro-runner-dl" href="https://s3.amazonaws.com/datawire-static-files/apro-plugin-runner/$aproVersion$/darwin/amd64/apro-plugin-runner">Mac 64-bit</a> |
<a class="pro-runner-linux-dl" href="https://s3.amazonaws.com/datawire-static-files/apro-plugin-runner/$aproVersion$/linux/amd64/apro-plugin-runner">Linux 64-bit</a>

Note that the plugin runner must match the version of Ambassador Edge Stack that you are running. Place the binary somewhere in your `$PATH`.

Information about open source code used in `apro-plugin-runner` can be found by running `apro-plugin-runner --version`.

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

## Filter Type: `Plugin`

The `Plugin` filter type allows you to plug in your own custom code. This code is compiled to a `.so` file, which you load in to the Ambassador Edge Stack container at `/etc/ambassador-plugins/${NAME}.so`.

### The Plugin Interface

This code is written in the Go programming language (golang), and must
be compiled with the exact same compiler settings as Ambassador Edge
Stack; and any overlapping libraries used must have their versions
match exactly.  This information is documented in an [apro-abi.txt][]
file for each Ambassador Edge Stack release.

[apro-abi.txt]: https://s3.amazonaws.com/datawire-static-files/apro-abi/apro-abi@$aproVersion$.txt

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

`*http.Request` is the incoming HTTP request that can be mutated or intercepted, which is done by `http.ResponseWriter`.

Headers can be mutated by calling `w.Header().Set(HEADERNAME, VALUE)`.
Finalize changes by calling `w.WriteHeader(http.StatusOK)`.

If you call `w.WriteHeader()` with any value other than 200 (`http.StatusOK`) instead of modifying the request, the plugin has
taken over the request, and the request will not be sent to your backend service.  You can call `w.Write()` to write the body of an error page.

### `Plugin` Global Arguments

```yaml
---
apiVersion: getambassador.io/v2
kind: Filter
metadata:
  name: "example-plugin-filter"
  namespace: "example-namespace"
spec:
  Plugin:
    name: "string" # required; this tells it where to look for the compiled plugin file; "/etc/ambassador-plugins/${NAME}.so"
```

### `Plugin` Path-Specific Arguments

Path specific arguments are not supported for Plugin filters at this
time.


## Further reading

For more details about configuring filters and the `plugin` interface, see the [filter reference](/reference/filter-reference).
