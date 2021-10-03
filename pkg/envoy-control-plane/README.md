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

1. Go 1.11+

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

1. Take a look at the [example server](internal/example/README.md).


## XDS API versioning

The Envoy xDS APIs follow a well defined [versioning scheme](https://www.envoyproxy.io/docs/envoy/latest/configuration/overview/versioning).
Due to lack of generics and function overloading in golang, creating a new version unfortunately involves code duplication.
Based on the discussion [here](https://docs.google.com/document/d/1ZkHpz6DwEUmAlG0kb2Mgu4iaeQC2Bbb0egMbECoNNKY/edit?ts=5e602993#heading=h.15nsmgmjaaml) and [here](https://envoyproxy.slack.com/archives/C7LDJTM6Z/p1582925082005300), go-control-plane is adopting a mechanism to create a new version from an existing version by running a script.
In order to handle deprecated/new fields between versions, make sure to create a shim such that duplication remains minimal. One such example today is how different [resource urls](https://github.com/envoyproxy/go-control-plane/tree/master/pkg/resource) are handled.

For authoring changes, make changes to v2 and at the end, use `make create_version` to create the v3 specific files.
Make sure to run `make build` and `make test` to identify and fix failures.

When v2 version is frozen in the future, we will change the experience such that changes will need to happen to v3 and autogen will create the v2 version instead.

## Resource caching

Because Envoy clients are assumed to be ephemeral, and thus, can come and go
away arbitrarily, the server relies on a configuration cache to minimize the
client load on the server. There are several caches available in this
repository:

- `Simple` cache is a snapshot-based cache that maintains a consistent view of
  the configuration for each group of proxies. It supports running as an ADS
  server or as regular dis-aggregated xDS servers. In ADS mode, the cache can
  hold responses until the complete set of referenced resources is requested
  (e.g. the entire set of RDS as referenced by LDS). Holding the response
  enables an atomic update of xDS collections.

- `Linear` is an eventually consistent cache for a single type URL collection.
  The cache maintains a single linear version history and a version vector for
  the resources in the cache. For each request, it compares the request version
  against latest versions for the requested resources, and responds with any
  updated resources. This cache assumes the resources are entirely opaque.

- `Mux` cache is a simple cache combinator. It allows mixing multiple caches
  for different type URLs, e.g use a simple cache for LDS/RDS/CDS and a linear
  cache for EDS.

## Usage

The [example server](internal/example/README.md) demonstrates how to integrate the go-control-plane with your code.

