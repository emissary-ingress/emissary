Ambex: Ambassador Experimental^H^H^H^H^H^H^H^H^H^H^H^H ADS service
=============================

Ambassador prior to v0.50.0 worked by writing out Envoy v1 JSON
configuration files, then triggering a hot restart of Envoy.  This
works, but it has some unpleasant limitations:

- Restarts take awhile, and as a result you can't change the
  configuration very quickly.
- Restarts can drop connections (cf Envoy #2776; more on this later).
- Envoy itself is deprecating the v1 configuration, and will only
  support v2 in a bit.

To get around these limitations, and generally go for a better
experience all 'round, we want to switch to the so-called `xDS` model,
in which Envoy's configuration is supplied by its various "*D*iscovery
*S*ervices": e.g. the `CDS` is the Cluster Discovery Service; the
`EDS` is the Endpoint Discovery Service.  For Ambassador, the
Aggregated Discovery Service or `ADS` is the one we want to use --
basically, it brings the other services together under one aegis and
lets you just tell Envoy "get everything dynamically."

However, the whole `ADS` thing is a bit of a pain:

- Envoy makes a bidirectional gRPC stream to the `ADS` server.
- The `ADS` then makes gRPC calls _to the Envoy_ to feed the Envoy
  configuration elements, but:
- The `ADS` has to carefully order things such that the configuration
  elements match what Envoy expects for consistency.

Rather than do all that logic by hand, we'll use the Envoy
`go-control-plane`[^1] for the heavy lifting.  This is also something of a
pain, given that it's not well documented, but here's the deal:

- The root of the world is a `SnapshotCache`:
  - `import github.com/datawire/ambassador/pkg/envoy-control-plane/cache`,
    then refer to `cache.SnapshotCache`.
  - A collection of internally consistent configuration objects is a
    `Snapshot` (`cache.Snapshot`).
  - `Snapshot`s are collected in the `SnapshotCache`.
  - A given `SnapshotCache` can hold configurations for multiple
    Envoys, identified by the Envoy `nodeID`, which must be configured
    for the Envoy.

- The `SnapshotCache` can only hold `go-control-plane` configuration
  objects, so you have to build these up to hand to the
  `SnapshotCache`.

- The gRPC stuff is handled by a `Server`:
  - `import github.com/datawire/ambassador/pkg/envoy-control-plane/server`,
    then refer to `server.Server`.
  - Our `runManagementServer` function (largely ripped off from the
    `go-control-plane` tests) gets this running.  It takes a
    `SnapshotCache` (cleverly called `config` for no reason I (Flynn)
    understand) and a standard Go `gRPCServer` as arguments.
  - _ALL_ the gRPC madness is handled by the `Server`, with the
    assistance of the methods in its `callback` object.

- Once the `Server` is running, Envoy can open a gRPC stream to it.
  - On connection, Envoy will get handed the most recent `Snapshot`
    that the `Server`'s `SnapshotCache` knows about.
  - Whenever a newer `Snapshot` is added to the `SnapshotCache`, that
    `Snapshot` will get sent to the Envoy.

- We manage the `SnapshotCache` by loading Envoy configuration files
  on disk:
   - it ignores files that start with a `.` (hidden files)
   - it interprets `*.json` files as [JSON-encoded protobuf](https://developers.google.com/protocol-buffers/docs/proto3#json)
   - it interprets `*.pb` files as [text-encoded protobuf](https://pkg.go.dev/google.golang.org/protobuf/encoding/prototext)
   - all other files are ignored
  As for when it loads those files:
   - By default when we get a SIGHUP we reload the configuration.
   - When passed the `--watch` argument we reload whenever any file in
     the directory changes.  Be careful about updating files
     atomically if you use this!

[^1]: The Envoy `go-control-plane` usually refers to
      `github.com/envoyproxy/go-control-plane`, but we've "forked" it
      as `github.com/datawire/ambassador/pkg/envoy-control-plane` in
      order to build it against the protobufs for our patched Envoy.

Running Ambex
=============

You'll need the Go toolchain, and will want to have a functioning `envoy`.

Then, you can run the ambex CLI using `busyambassador`:

```shell
go run github.com/datawire/ambassador/cmd/busyambassador ambex ARGs...
```

If you're on a platform other than GNU/Linux, in order to have a
functioning `envoy`, you may want to run all of this in the builder
shell: `make shell`.

Try it out
----------

You'll want to run both `ambex` and an instance of `envoy` with a
boostrap config pointing at that `ambex.

 1. First, start the `ambex`:

    ```shell
    go run github.com/datawire/ambassador/cmd/busyambassador ambex ./example/ambex/
    ```

    or

    ```shell
    go run github.com/datawire/ambassador/cmd/busyambassador ambex --watch ./example/ambex/
    ```

 2. Second, in another shell, start the `envoy`:

    ```shell
    envoy -l debug -c ./example/envoy/bootstrap-ads.yaml
    ```

You should now be able to run some curls in (yet) another shell:

```console
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

Edit and/or add more files to the `./example/ambex/` directory in
order to play with more configurations and see them reload
_instantaneously_ (if you used the `--watch` flag), or when-triggered
(if you didn't use the `--watch` flag; trigger a relead by signaling
the process with `killall -HUP ambex`).

Clean up
--------

Kill Ambex and Envoy with `Ctrl-C`.
