# control-plane

[![CircleCI](https://circleci.com/gh/envoyproxy/go-control-plane.svg?style=svg)](https://circleci.com/gh/envoyproxy/go-control-plane)
[![Go Report Card](https://goreportcard.com/badge/github.com/envoyproxy/go-control-plane)](https://goreportcard.com/report/github.com/envoyproxy/go-control-plane)
[![GoDoc](https://godoc.org/github.com/envoyproxy/go-control-plane?status.svg)](https://godoc.org/github.com/envoyproxy/go-control-plane)

This repository contains a Go-based implementation of an API server that
implements the discovery service APIs defined in
[data-plane-api](https://github.com/envoyproxy/data-plane-api).


## Scope

Due to the variety of platforms out there, there is no single
control plane implementation that can satisfy everyone's needs. Hence this
code base does not attempt to be a full scale control plane for a fleet of
Envoy proxies. Instead, it provides infrastructure that is shared by
multiple different control plane implementations. The components provided
by this library are:

* _API Server:_ A generic gRPC based API server that implements xDS APIs as defined
  in the
  [data-plane-api](https://github.com/envoyproxy/data-plane-api). The API
  server is responsible for pushing configuration updates to
  Envoys. Consumers should be able to import this go library and use the
  API server as is, in production deployments.

* _Configuration Cache:_ The library will cache Envoy configurations in
memory in an attempt to provide fast response to consumer Envoys. It is the
responsibility of the consumer of this library to populate the cache as
well as invalidate it when necessary. The cache will be keyed based on a
pre-defined hash function whose keys are based on the
[Node information](https://github.com/envoyproxy/data-plane-api/blob/d4988844024d0bcff4bcd030552eabe3396203fa/api/base.proto#L26-L36).

At this moment, this repository will not tackle translating platform
specific representation of resources (e.g., services, instances of
services, etc.) into Envoy-style configuration. Based on usage and
feedback, we might decided to revisit this aspect at a later point in time.

## Requirements

1. Go 1.12+

## Quick start

It's recommended to run the command with script `./build/run_docker.sh` as it executes the command
in the same environment as the circle ci. This makes sure to produce a consistent set of generated files.

1. Setup existing build:

    ```sh
    ./build/run_docker.sh make build test
    ```

1. Run [integration test](pkg/test/main/README.md) against the latest Envoy binary:

    ```sh
    ./build/run_docker.sh make integration
    ```

## Usage

Register services on the gRPC server as follows.

```go
import (
	"google.golang.org/grpc"
	"net"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
)

func main() {
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	server := xds.NewServer(snapshotCache, nil)
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":8080")

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	api.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	api.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	api.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			// error handling
		}
	}()
}
```

As mentioned in [Scope](https://github.com/envoyproxy/go-control-plane/blob/master/README.md#scope), you need to cache Envoy configurations.
Generate the key based on the node information as follows and cache the configurations.

```go
import "github.com/envoyproxy/go-control-plane/pkg/cache"

var clusters, endpoints, routes, listeners []cache.Resource

snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
snapshot := cache.NewSnapshot("1.0", endpoints, clusters, routes, listeners)
_ = snapshotCache.SetSnapshot("node1", snapshot)
```
