# End-to-End Tests

Black-box end-to-end tests that exercise a real Emissary-ingress installation
running in a local k3d cluster. Each fixture applies Kubernetes manifests,
runs probes through the gateway, and asserts on the response.

The tests are driven by [Chainsaw](https://kyverno.github.io/chainsaw/), a
declarative Kubernetes test framework. Each fixture is one Chainsaw `Test`
that gets its own ephemeral namespace, applies its manifests, runs a probe,
and is automatically torn down (with diagnostics on failure).

## Layout

```
test/e2e/
├── .chainsaw.yaml              # Chainsaw Configuration (timeouts, parallelism, namespacing)
├── helm-values.yaml            # values for the Emissary helm install
├── slots.sh                    # slot installs + sharded runner (see below)
├── probe.sh                    # shared retry + assert helper (see below)
└── fixtures/
    └── <fixture-name>/
        ├── chainsaw-test.yaml  # the Test resource (apply + probe)
        ├── manifests.yaml      # Deployments/Services/Mappings for the scenario
        └── queries.json        # kat-client query set for the probe
```

Each test gets a fresh, randomly-named namespace (`generateName: e2e-`).
Emissary watches all namespaces, so Mappings/Listeners/TCPMappings created in
those test namespaces are picked up automatically.

## Slots

Most tests can't share one Emissary. A `Module`, `AuthService`,
`RateLimitService`, `TracingService` or `LogService` configures the *whole
instance*, so two such tests running at once would clobber each other.

A **slot** is a pre-baked, isolated Emissary install. Isolation is by
`AMBASSADOR_ID`, not by namespace: an Emissary only acts on CRDs whose
`ambassador_id` matches its own, and a CRD that omits the field defaults to
`"default"`. So a slot is just another helm release of the shipped chart
with `env.AMBASSADOR_ID` set, which also makes the chart stamp the matching
`ambassador_id` onto its default Listeners.

| Slot      | Namespace        | `AMBASSADOR_ID` | http | https | tcp  |
|-----------|------------------|-----------------|------|-------|------|
| `default` | `emissary`       | *(unset)*       | 80   | 443   | 6789 |
| `slot1`   | `emissary-slot1` | `slot1`         | 8100 | 8101  | 8102 |
| `slot2`   | `emissary-slot2` | `slot2`         | 8104 | 8105  | 8106 |
| ...       |                  |                 |      |       |      |
| `slot8`   | `emissary-slot8` | `slot8`         | 8128 | 8129  | 8130 |

The numbered slots are **anonymous and interchangeable**. Nothing distinguishes
one from another except its ports; all the config comes from the fixture's own
CRDs at runtime, so any exclusive fixture can run on any free slot.

## Declaring what a fixture needs

A fixture never names a slot. It declares only *whether* it needs one:

```yaml
metadata:
  labels:
    slot: exclusive     # or: shared
```

- **`shared`** -- the fixture only creates `Mapping`s, `Host`s, `TCPMapping`s
  and the like. It adds no global config, so it can overlap with others on the
  base Emissary.
- **`exclusive`** -- the fixture creates a `Module`, `AuthService`,
  `RateLimitService`, `TracingService` or `LogService`. Those configure the
  whole instance, so it needs a slot to itself for its duration.

Every Emissary CRD then uses the same expression, whichever kind it is:

```yaml
spec:
  ambassador_id: [(env('SLOT'))]
```

`slots.sh` exports `SLOT` per shard: `default` for the shared shard, `slotN`
for an exclusive one. An Emissary with `AMBASSADOR_ID` unset self-identifies as
`"default"`, so the shared case needs no special handling.

Because the line is identical everywhere, switching a fixture between shared
and exclusive is a one-word label change with nothing else to remember.

## How the run is sharded

`make e2e/run` starts one `chainsaw` process per shard, all concurrent:

- **shared** -- every `slot: shared` fixture, `--parallel $E2E_DEFAULT_PARALLEL`
  against the base Emissary.
- **slot1..N** -- the `slot: exclusive` fixtures, dealt round-robin across the
  available slots, each shard `--parallel 1` so a slot hosts one fixture at a
  time. Shards target their fixtures by name via `--include-test-regex`, which
  chainsaw feeds to Go's `-run` and matches against `chainsaw/<name>`.

Assignment happens at run time, so there is no slot bookkeeping in the
fixtures and no way for two of them to collide on a hand-picked number. If
there are more exclusive fixtures than slots they double up and run in
sequence; if there are fewer, the spare slots get no process at all.

`slots.sh check` runs first and fails the suite if a fixture has a missing or
unknown `slot` label, or a name that wouldn't survive being put in a regex,
since any of those would mean the fixture silently never runs.

> **Why slots disable the chart's Module.** The chart ships a `Module` named
> `ambassador` (`module.enabled: true`). Emissary keys its module store by name
> alone, so that Module would collide with one a fixture creates: the loser is
> renamed to `ambassador.<namespace>` and `get_module("ambassador")` then
> returns whichever won, silently dropping the fixture's config. Slots are
> therefore installed with `--set module.enabled=false`, which leaves the
> fixture's own Module as the only one carrying that `ambassador_id`.

> **Adding a slot.** Edit `SLOTS` in `slots.sh`. Ports come out of the
> `8100-8159` block k3d publishes at cluster-creation time; at the 4-port
> stride that holds 15 slots, and going past that needs the range widened
> *and* the cluster recreated.
>
> Slots install concurrently, so adding one costs wall-clock only if it pushes
> the runner past its CPU budget. Sizing: at ~28s per fixture, N slots clear
> the pinned backlog in `(count / N) * 28`s. Returns flatten past N=8, where
> the `default` shard becomes the constraint instead -- raise
> `E2E_DEFAULT_PARALLEL` before adding a ninth slot. The hard ceiling is runner
> CPU (the chart requests `200m` per Emissary; a 4-vCPU runner fits ~16), so
> 8-12 is the practical band.

The probe step shells out to `probe.sh`, which runs the host-built
`kat-client` binary against `$GATEWAY_URL` and retries until a `jq` assertion
passes. Fixtures need no `bindings` at all: `script` steps inherit the process
environment, and manifests call `(env('KAT_SERVER_IMAGE'))` and
`(env('SLOT'))` directly.

## The probe helper

`probe.sh <queries.json> <jq-expr>` is the whole assertion mechanism:

```sh
../../probe.sh queries.json '
  .[0].result.status == 200 and
  .[0].result.json.backend == "http-echo"
'
```

- `queries.json` is a plain [kat-client][] input file. URLs starting with `/`
  are prefixed with `$PROBE_BASE` (defaults to `$GATEWAY_URL`); absolute URLs
  are left alone, so TLS and alternate-port fixtures spell out their own
  scheme and host.
- The `jq` expression is evaluated against kat-client's output array. It maps
  directly onto the old KAT Python assertions: `Query(..., expected=404)`
  becomes `.[0].result.status == 404`, and `r.backend.name == "x"` becomes
  `.[0].result.json.backend == "x"`.
- Chainsaw does not retry `script` steps, and Envoy config propagation is
  async, so the retry lives here. Tune with `PROBE_ATTEMPTS` (default 30) and
  `PROBE_INTERVAL` (default 2s).

On failure it prints the assertion, the last full response, and kat-client's
stderr, then exits 1 so Chainsaw runs the `catch` diagnostics configured in
`.chainsaw.yaml`.

[kat-client]: ../../cmd/kat-client/client.go

## Running locally

Everything is driven through `make` targets defined in `build-aux/e2e.mk`.

### Prerequisites

- Docker running locally (k3d needs it).
- Python venv active so Makefile `python3` invocations resolve project deps:
  ```
  source .venv/bin/activate     # or: uv run make ...
  ```
- `k3d`, `kubectl`, `helm`, and `chainsaw` are fetched automatically into
  `tools/bin/` the first time they're needed.
- `jq` on your `PATH` (used by the probe scripts to parse kat-client JSON).
  Preinstalled on GitHub runners; `brew install jq` on macOS.

The fixtures use the in-tree `kat-client` as the probe and `kat-server` as
the backend. `make e2e/run` builds `kat-client` as a host binary at
`tools/bin/kat-client`, and `kat-server`'s image is the one produced by
`make images`.

### Full cycle from scratch

```
make e2e/local
```

This runs, in order:
1. `e2e/cluster-up` creates a k3d cluster named `emissary-e2e` with ports
   80/443 (HTTP fixtures), 6789 (TCPMapping fixtures) and the `8100-8159`
   slot block mapped to the host loadbalancer, and Traefik disabled.
2. `make images` builds Emissary's container images via goreleaser snapshot.
3. `e2e/install` imports images into k3d, then `helm install`s the CRDs chart
   and the ingress chart pinned to the locally-built image tag, then
   `e2e/install-slots` installs one extra release per slot.
4. `e2e/run` shards `chainsaw test` across the slots.

### Iterating

Once the cluster is up and Emissary is installed, you usually only need:

```
make e2e/run              # re-run every fixture, sharded by slot
make e2e/run/gzip-minimum # re-run one fixture (a slot is allocated for it)
```

If you changed code and want to redeploy without recreating the cluster:

```
make images && make VERSION=v4.0.0-local e2e/install
```

> **Why the `VERSION` override?** `make e2e/install` builds Helm charts whose
> metadata labels embed `VERSION`. A dirty working tree's default version
> (e.g. `4.0.2-0.20260422205059-<sha>-dirty.<ts>`) exceeds Kubernetes' 63-char
> label limit, and `helm install` rejects the CRDs with `metadata.labels:
> Invalid value: ... must be no more than 63 characters`. `make e2e/local`
> applies this override for you automatically; `make e2e/install` on its own
> does not, so pass it explicitly (or set `E2E_LOCAL_VERSION` in the
> environment). Use any short string. `v4.0.0-local` is just the default.

> **Adding a new edge port?** k3d's published ports are fixed at cluster
> creation time. If you add a port to `e2e/cluster-up` (or want the existing
> 6789 or the `8100-8159` slot block on a cluster you created before they
> were added), you have to `make e2e/cluster-down && make e2e/cluster-up` to
> pick it up. `helm upgrade` alone won't get traffic in.

To reinstall just the slots (after changing `SLOTS` in `slots.sh`, say):

```
make e2e/uninstall-slots && make VERSION=v4.0.0-local e2e/install-slots
```

### Teardown

```
make e2e/cluster-down
```

### Overridable variables

All have sensible defaults; override on the command line as needed:

| Variable             | Default                | Purpose                                  |
|----------------------|------------------------|------------------------------------------|
| `E2E_CLUSTER`        | `emissary-e2e`         | k3d cluster name                         |
| `E2E_NAMESPACE`      | `emissary`             | namespace for the Emissary install       |
| `E2E_CRD_NAMESPACE`  | `emissary-system`      | namespace for the CRDs chart             |
| `E2E_GATEWAY_URL`    | `http://localhost`     | host the probes target (slot ports are appended) |
| `E2E_LOCAL_VERSION`  | `v4.0.0-local`         | short chart VERSION (dirty trees produce strings longer than k8s' 63-char label limit) |
| `E2E_SLOTS`          | `slot1:8100 ... slot8:8128` | `<name>:<port-base>` list of isolated Emissary installs |
| `E2E_DEFAULT_PARALLEL` | `8`                  | how many `shared` fixtures run at once |
| `E2E_SLOT`           | *(first free)*         | force `make e2e/run/<name>` onto a specific slot |

Per-fixture probe and apply timeouts are set in `.chainsaw.yaml` and on
individual steps inside each `chainsaw-test.yaml`.

## Adding a new fixture

1. Create `test/e2e/fixtures/<name>/`.
2. Put the resources you want in `manifests.yaml` (Deployment, Service,
   Mapping, whatever the scenario needs). For the locally-built kat-server
   image, and on every Emissary CRD, use:
   ```yaml
   image: (env('KAT_SERVER_IMAGE'))
   ...
   spec:
     ambassador_id: [(env('SLOT'))]
   ```
3. Decide `shared` or `exclusive` (see [Declaring what a fixture
   needs](#declaring-what-a-fixture-needs)). You never pick a slot number.
4. Put the requests in `queries.json` (kat-client format, relative URLs).
5. Write `chainsaw-test.yaml` defining a `Test` resource with:
   - `metadata.labels.slot` set to `shared` or `exclusive`.
   - A `try` block that `apply`s `manifests.yaml` and runs
     `../../probe.sh queries.json '<jq-expr>'` via a `script` step.

   No `bindings`, no `template: true` (templating is on by default), and no
   `catch` -- the diagnostics in `.chainsaw.yaml` apply to every fixture. Add a
   `catch` only for something unusual; it appends to the global one rather than
   replacing it. Existing fixtures are good templates.
6. Run `make e2e/run/<name>` and confirm it passes.

No registration step is required beyond the `slot` label. Chainsaw discovers
every directory under `fixtures/` that contains a `chainsaw-test.yaml`, and
`slots.sh check` fails the suite if that label is missing or unknown.

### Porting a KAT test

The old Python suite under `python/tests/src/tests/kat/` maps over piece by
piece. Each fixture's header comment names the class it came from.

| KAT                                       | Chainsaw                                            |
|-------------------------------------------|-----------------------------------------------------|
| `manifests()` / `config()`                 | `manifests.yaml`                                    |
| `queries()`                                | `queries.json`                                      |
| `Query(url, expected=404)`                 | `.[N].result.status == 404`                         |
| `Query(url, headers={...})`                | `"headers": {...}` in the query                     |
| `check()` / `assert`                       | the `jq` expression passed to `probe.sh`            |
| `r.backend.name == "x"`                    | `.[N].result.json.backend == "x"`                   |
| `r.headers["Server"] == ["x"]`             | `.[N].result.headers["Server"][0] == "x"`           |
| one Emissary per test (`AMBASSADOR_ID`)    | a slot pin                                        |

When porting an assertion about an *absence* (`"X" not in r.headers`), AND in a
positive check too. `has("X") | not` is satisfied by a 404 from an Emissary
that never picked up the route, so on its own it passes for the wrong reason.
`fixtures/gzip-unsupported-content-type` shows the pattern.

## How CI runs this

`.github/workflows/test-images.yaml` mirrors the local flow: it spins up k3d
(with the slot port block published), imports the images built by
`build-images`, `helm install`s the charts produced by `build-charts` (pinned
to that same image tag), calls `slots.sh install` for the slot releases,
installs `chainsaw` into `tools/bin/`, and then runs `slots.sh run`. On
failure each fixture's `catch` block dumps pod logs and the relevant CRDs; the
workflow's diagnostics step then walks every Emissary namespace.

Both CI and `make` call `slots.sh`, so the slot list and its port math have
exactly one definition.

The key difference from local: CI consumes pre-built image and chart
artifacts from upstream jobs instead of running `make images` / `make
charts` itself.
