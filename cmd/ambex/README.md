Ambex: Ambassador Experimental ADS service
==========================================

Ambassador v1 works by writing out Envoy v1 JSON configuration files, then triggering a hot restart of Envoy. This works, but it has some unpleasant limitations:

- Restarts take awhile, and as a result you can't change the configuration very quickly.
- Restarts can drop connections (cf Envoy #2776; more on this later).
- Envoy itself is deprecating the v1 configuration, and will only support v2 in a bit.

To get around these limitations, and generally go for a better experience all 'round, we want to switch to the so-called `xDS` model,  in which Envoy's configuration is supplied by its various "*D*iscovery *S*ervices": e.g. the `CDS` is the Cluster Discovery Service; the `EDS` is the Endpoint Discovery Service. For Ambassador, the Aggregated Discovery Service or `ADS` is the one we want to use -- basically, it brings the other services together under one aegis and lets you just tell Envoy "get everything dynamically."

However, the whole `ADS` thing is a bit of a pain:

- Envoy makes a bidirectional gRPC stream to the `ADS` server.
- The `ADS` then makes gRPC calls _to the Envoy_ to feed the Envoy configuration elements, but:
- The `ADS` has to carefully order things such that the configuration elements match what Envoy expects for consistency.

Rather than do all that logic by hand, we'll use the Envoy `go-control-plane` for the heavy lifting. This is also something of a pain, given that it's not well documented, but here's the deal:

- The root of the world is a `SnapshotCache`:
  - `import github.com/datawire/ambassador/pkg/envoy-control-plane/cache`, then refer to `cache.SnapshotCache`.
  - A collection of internally consistent configuration objects is a `Snapshot` (`cache.Snapshot`).
  - `Snapshot`s are collected in the `SnapshotCache`.
  - A given `SnapshotCache` can hold configurations for multiple Envoys, identified by the Envoy `nodeID`, which must be configured for the Envoy.
- The `SnapshotCache` can only hold `go-control-plane` configuration objects, so you have to build these up to hand to the `SnapshotCache`.
- The gRPC stuff is handled by a `Server`:
  - `import github.com/datawire/ambassador/pkg/envoy-control-plane/server`, then refer
    to `server.Server`.
  - Our `runManagementServer` function (largely ripped off from the `go-control-plane` tests) gets this running. It takes a `SnapshotCache` (cleverly called `config` for no reason I understand) and a standard Go `gRPCServer` as arguments.
  - _ALL_ the gRPC madness is handled by the `Server`, with the assistance of the methods in its `callback` object.
- Once the `Server` is running, Envoy can open a gRPC stream to it.
  - On connection, Envoy will get handed the most recent `Snapshot` that the `Server`'s `SnapshotCache` knows about.
  - Whenever a newer `Snapshot` is added to the `SnapshotCache`, that `Snapshot` will get sent to the Envoy.
- We manage the `SnapshotCache` by loading envoy configuration files from json or protobuf files on disk.
  - By default when we get a SIGHUP we reload the configuration.
  - When passed the -watch argument we reload whenever any file in the directory changes.

Running Ambex
=============

You'll need the Go toolchain and [Glide](https://glide.sh/).

Linux
-----

Run `make build` to build the ambex binary. Then in one window run

```shell
./bin_$(go env GOOS)_$(go env GOARCH)/ambex bootstrap_image/example
```

or

```shell
./bin_$(go env GOOS)_$(go env GOARCH)/ambex -watch bootstrap_image/example
```

to start the ADS server.

And in another window run

```shell
make run
```

to launch envoy with a bootstrap configuration that points to the locally running ambex.
This uses Docker `--net=host` networking, which does not work on MacOS or Windows.

Everything in Docker
--------------------

You can run everything in Docker. This works on MacOS too. In the first shell run Envoy:

```shell
make run_envoy
```

In a second shell run Ambex:

```shell
make run_ambex
```

Note that the `run_ambex` recipe assumes that `make run_envoy` has already been executed.
It uses `docker exec` to launch Ambex in the same container.

Try it out
----------

You should now be able to run some curls in (yet) another shell:

```shell
$ curl localhost:8080/hello
Hello ADS!!!

$ curl localhost:8080/get
{
  "args": {}, 
  "headers": {
    "Accept": "*/*", 
    "Connection": "close", 
    "Host": "httpbin.org", 
    "User-Agent": "curl/7.54.0", 
    "X-Envoy-Expected-Rq-Timeout-Ms": "15000"
  }, 
  "origin": "72.74.69.156", 
  "url": "http://httpbin.org/get"
}
```

Edit and/or add more files to the example directory in order to play
with more configurations and see them reload _instantaneously_.

Note that instantaneous reloads require the `-watch` flag; otherwise
you can force a reload by signaling the process
(`killall -HUP ambex`).

If you're running everything in Docker, you will need to edit/add files
in `/application/example` in the container. You can do this directly in the container:

```shell
$ curl localhost:8080/hello
Hello ADS!!!

$ docker exec -it ambex-envoy sed -i "s/ADS/ADS World/" /application/example/listener.json

$ curl localhost:8080/hello
Hello ADS World!!!
```

For more serious in-container editing, you may want to start with something like this:

```shell
$ docker exec -it ambex-envoy bash
root@04d063adb905:/application# apt-get -qq update
root@04d063adb905:/application# apt-get -qq install vim > /dev/null
root@04d063adb905:/application# vim example/listener.json
...
```

You can also edit on the host machine and then copy:

```shell
$ curl localhost:8080/hello
Hello ADS World!!!

$ docker cp bootstrap_image/example ambex-envoy:/application/

$ curl localhost:8080/hello
Hello ADS!!!
```

Clean up
--------

Kill Ambex and Envoy with `Ctrl-C`.
Use `make clean` to clean up the filesystem and Docker images.
