# kat-backend

   ```console
   $ make help
   # Target          ## ## Description
   ################# ## ## #########################
   help              :  ## Display this help text
   ################# ## ## #########################
   build             :  ## Build everything
   push              :  ## Push docker images
   clean             :: ## Remove everything
   ################# ## ## #########################
   sandbox.http-auth :  ## In docker-compose: run Ambassador, an HTTP AuthService, an HTTP backend service, and a TracingService
   sandbox.grpc-auth :  ## In docker-compose: run Ambassador, a gRPC AuthService, an HTTP backend service, and a TracingService
   sandbox.web       :  ## In docker-compose: run Ambassador with gRPC-web enabled, and a gRPC backend service
   ```

`make build` builds the following artifacts:
 - backend Docker image: `quay.io/datawire/kat-backend:${TAG}`
 - client binary executable: `./client/bin/client_darwin_amd64`
 - client binary executable: `./client/bin/client_linux_amd64`

`make push` publishes the Docker image; the executable are published
by committing them to the Git repo.

## What's in those built artifacts?

KAT defines its own custom "echo" gRPC service.  The definition of
this service is at [`./echo/echo.proto`][].

The backend Docker image runs in 1 of 3 modes:
 1. `KAT_BACKEND_TYPE=grpc_echo`, which is a gRPC backend service
    implementing the above "echo" gRPC service.
 2. `KAT_BACKEND_TYPE=gprc_auth`, which is a service implementing a
    gRPC ext_authz service.
 3. `KAT_BACKEND_TYPE=http`, which is an HTTP backend service.

The client binaries talk to the KAT back-end services.

[`./echo/echo.proto`]: ./echo/echo.proto

## Publishing changes

### Client

Running `make build` will build the client binaries and place them in
the `./client/bin` folder.  These binaries are consumed by Ambassador
KAT tests.  A GH release should be made avery time that they change
and the ambassador.git [`Makefile:KAT_BACKEND_RELEASE`][] variable
should be updated with the new release version.

[`Makefile:KAT_BACKEND_RELEASE`]: https://github.com/datawire/ambassador/blob/master/Makefile

### Backend

Adjust `TAG=` in the `Makefile`, then run `make push`.  You'll need to
adjust something in the ambassador.git `Makefile`.

## docker-compose sandbox

It has all simple docker-compose files that allows building and serve
Envoy proxy and the gRPC-Web, gRPC-Auth as well as the HTTP backend
services. This let the developer try different configuration and
client/server implementations without the need of spinning up a
cluster, Ambassador and Kat tests. How does it work:

1. For running the server, e.g. HTTP:

   ```console
   $ make sandbox.http-auth
   ```

   This will start up the docker all in interactive mode, so you can
   see the logs from both proxy and backend.

2. From another shell:

   ```console
   $ curl -v -H "Requested-Cookie: foo, bar" -H "Requested-Status: 307" http://localhost:61892/get
   ```

   This will pass through Envoy filter, hit the HTTP-Echo service and
   return a HTTP 307 response that should contain two Set-Cookie
   headers: foo=foo and bar=bar.

3. Alternatively, you could also run the `kat-client` by consuming the
   a `url.json` file which is provided as an example for each sandbox:

   ```console
   ./client/bin/client_DIST_amd64 -input sandbox/http_auth/urls.json
   ```

This will be exactly the same mechanics used by the Kat tests in
Ambassador. It will produce and output with the json response for each
call. For running the other sandboxes:

- gRPC-bridge:

  ```console
  $ make sandbox.bridge
  ```

- gRPC-web:

  ```console
  $ make sandbox.web
  ```

- gRPC-auth:

  ```console
  $ make sandbox.grpc_auth
  ```
