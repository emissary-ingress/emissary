# kat-backend
Get-able package that contains kat the back-end server and all packages including generated pb.go files used in the gRPC services.

## Client

It has the clients used to talk to the Kat back-end services. For building it:

```
$ make client
```

This will build the client binaries and place them in the bin folder. These binaries are consumed by Ambassador kat tests. A GH release should be made avery time that they change and Ambassador [KAT_BACKEND_RELEASE](https://github.com/datawire/ambassador/blob/master/Makefile) env should be updated with the new release version.

## Echo

It has the protobuf definition of the Echo back-end service. For build it:

```
$ make echo
```

This will generate the grpc-web and grpc-bridge `.pb` files. They are needed for the grpc-echo service and for the grpc-web client.

## Services

It has the HTTP echo service as well as the gRPC authorization server and the gRPC echo service. For building it:
```
$ make backend.build
```

This will build a kat-backend container. TAG needs to be manually updated. For pushing:
```
$ make backend
```

## Sandbox
It has all simple docker-compose files that allows building and serve Envoy proxy and the gRPC-Web, gRPC-Auth as well as the HTTP backend services. This let the developer try different configuration and client/server implementations without the need of spinning up a cluster, Ambassador and Kat tests. How does it work:

1. For running the server, e.g. HTTP:
```
$ make sandbox.http-auth
```

This will start up the docker all in interactive mode, so you can see the logs from both proxy and backend. 

2. From another shell:
```
$ curl -v -H "Requested-Cookie: foo, bar" -H "Requested-Status: 307" http://localhost:61892/get 
```

This will pass through Envoy filter, hit the HTTP-Echo service and return a HTTP 307 response that should contain two Set-Cookie headers: foo=foo and bar=bar.

3. Alternatively, you could also run the `kat-client` by consuming the a `url.json` file which is provided as an example for each sandbox:
```
./client/bin/client_DIST_amd64 -input sandbox/http_auth/urls.json
```

This will be exactly the same mechanics used by the Kat tests in Ambassador. It will produce and output with the json response for each call. For running the other sandboxes:

gRPC-bridge
```
$ make sandbox.bridge
```

gRPC-web
```
$ make sandbox.web
```

gRPC-auth
```
$ make sandbox.grpc_auth
```
