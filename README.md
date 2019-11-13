# Ambassador Pro example filters

This is an example filter for Ambassador Pro that injects an
`X-Wikipedia` header with the URL for a random Wikipedia article
before passing the request off to the backend service.  It uses
https://en.wikipedia.org/wiki/Special:Random to obtain the URL; as an
example of a middleware needing to talk to an external service.

## Compiling

### Prerequisites

- A Linux or MacOS host (WSL on Windows is okay)
- Docker
- rsync

The `Plugin` is compiled by building a container with a stable Golang environment and using `rsync` to copy files to and from the container.

---

To compile all `.go` files in the root of this repository just run

	```shell
	$ make DOCKER_REGISTRY=...
	```

It will generate a Docker image named

	```shell
	DOCKER_REGISTRY/amb-sidecar-custom:VERSION
	```
	
where:

 - VERSION is `git describe --tags --always --dirty`.
 - DOCKER_REGISTRY is the `$DOCKER_REGISTRY` environment variable, or
   `localhost:31000` if the variable is not set.

### Options

To generate a Docker image with a custom name run

	```shell
	$ make DOCKER_IMAGE={{REGISTRY}}/{{IMAGE_NAME}}:{{IMAGE_TAG}}
	```

To build `.go` files in a non-root directory run

	```shell
	$ make DOCKER_REGISTRY=... PLUGIN_DIR=...
	```
	where `PLUGIN_DIR` is the directory for your `.go` files.

To compile for a specific version of Ambassador Pro, set `APRO_VERSION`:

	$ make APRO_VERSION=0.2.2-rc2 DOCKER_REGISTRY=...

When switching Ambassador Pro versions, it may be nescessary to edit
the `go.mod` file.

## Deploying

Push that `amb-sidecar-plugin` Docker image to a registry that your
cluster has access to.  You can do this by running `make push
DOCKER_REGISTRY=...`.

Use that image you just pushed instead of
`quay.io/datawire/ambassador_pro:amb-sidecar` when deploying
Ambassador Pro. For more details on deployment, consult the Filter documentation.

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
