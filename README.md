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
  - `import github.com/envoyproxy/go-control-plane/pkg/cache`, then refer to `cache.SnapshotCache`.
  - A collection of internally consistent configuration objects is a `Snapshot` (`cache.Snapshot`).
  - `Snapshot`s are collected in the `SnapshotCache`.
  - A given `SnapshotCache` can hold configurations for multiple Envoys, identified by the Envoy `nodeID`, which must be configured for the Envoy.
- The `SnapshotCache` can only hold `go-control-plane` configuration objects, so you have to build these up to hand to the `SnapshotCache`.
- The gRPC stuff is handled by a `Server`:
  - `import github.com/envoyproxy/go-control-plane/pkg/server`, then refer
    to `server.Server`.
  - Our `runManagementServer` function (largely ripped off from the `go-control-plane` tests) gets this running. It takes a `SnapshotCache` (cleverly called `config` for no reason I understand) and a standard Go `gRPCServer` as arguments.
  - _ALL_ the gRPC madness is handled by the `Server`, with the assistance of the methods in its `callback` object.
- Once the `Server` is running, Envoy can open a gRPC stream to it.
  - On connection, Envoy will get handed the most recent `Snapshot` that the `Server`'s `SnapshotCache` knows about.
  - Whenever a newer `Snapshot` is added to the `SnapshotCache`, that `Snapshot` will get sent to the Envoy.
- We manage the `SnapshotCache` using our `configurator` object, which knows how to listen to `stdin` and HTTP requests to change objects and post new Snapshots.
  - Obviously this piece has to be rewritten for the real world.

Running Ambex
=============

Run `make` to build everything. Then in one window

```shell
docker run -it --rm --name ambex-shell \
       -p8000:8000 -p9000:9000 -v $(pwd)/mountpoint:/application \
       datawire/ambassador-envoy-alpine-stripped:v1.5.0-232-g6557e9ea7 \
       /bin/sh
```

to start a shell running in a Docker container.

In that shell:

```shell
cd /application
./ambex -debug
```

and you'll have the Ambex server running.

In another window, 

```shell
docker exec -it ambex-shell /bin/sh
```

to get another shell running in the Docker container. In that shell, start Envoy:

```shell
cd /application
./envoy-stripped-binary -l debug -c bootstrap-ads.yaml
```

and you should see Ambex and Envoy working out Envoy configuration.

Back at the host, you can test this by starting two QoTM services, in two other windows, from the main Ambex directory (you'll need Python 3 and Flask for this). One should be

```shell
python qotm.py 5000 v-one
```

and the other

```shell
python qotm.py 6000 v-two
```

Then you can test from yet another host window:

```shell
curl http://localhost:8000/qotm/
```

which should always return `v-one` as the hostname for now.

After that, fire up the full-on test script:

```shell
python loop.py http://127.0.0.1:8000/qotm/
```

and then start up the evil swapping loop:

```shell
while true; do
    curl -v 'http://localhost:9000/mapping?name=qotm-mapping&prefix=%2Fqotm%2F&cluster=qotm-2&update=true'
    sleep 1
    curl -v 'http://localhost:9000/mapping?name=qotm-mapping&prefix=%2Fqotm%2F&cluster=qotm-1&update=true'
    sleep 1
done
```

You should see the test script swapping between "1"s and "2"s in its output, but it should happily keep running. (You may see it hang without crashing -- this seems to be a flakiness in the Docker setup we're using, rather than something horrible. It's much less likely if you run Envoy without any `-l` argument.)
