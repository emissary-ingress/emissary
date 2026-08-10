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
├── flavors.sh                  # isolated Emissary installs + sharded runner (see below)
├── probe.sh                    # shared retry + assert helper (see below)
└── fixtures/
    └── <fixture-name>/
        ├── chainsaw-test.yaml  # the Test resource (apply, probe, catch)
        ├── manifests.yaml      # Deployments/Services/Mappings for the scenario
        └── queries.json        # kat-client query set for the probe
```

Each test gets a fresh, randomly-named namespace (`generateName: e2e-`).
Emissary watches all namespaces, so Mappings/Listeners/TCPMappings created in
those test namespaces are picked up automatically.

## Flavors

Most tests can't share one Emissary. A `Module`, `AuthService`,
`RateLimitService`, `TracingService` or `LogService` configures the *whole
instance*, so two such tests running at once would clobber each other.

A **flavor** is a pre-baked, isolated Emissary install. Isolation is by
`AMBASSADOR_ID`, not by namespace: an Emissary only acts on CRDs whose
`ambassador_id` matches its own, and a CRD that omits the field defaults to
`"default"`. So a flavor is just another helm release of the shipped chart
with `env.AMBASSADOR_ID` set, which also makes the chart stamp the matching
`ambassador_id` onto its default Listeners.

| Flavor    | Namespace      | `AMBASSADOR_ID` | http | https | tcp  |
|-----------|----------------|-----------------|------|-------|------|
| `default` | `emissary`     | *(unset)*       | 80   | 443   | 6789 |
| `f1`      | `emissary-f1`  | `f1`            | 8110 | 8111  | 8112 |
| `f2`      | `emissary-f2`  | `f2`            | 8120 | 8121  | 8122 |
| `f3`      | `emissary-f3`  | `f3`            | 8130 | 8131  | 8132 |
| `f4`      | `emissary-f4`  | `f4`            | 8140 | 8141  | 8142 |

Every fixture declares which one it runs on:

```yaml
metadata:
  labels:
    flavor: f1
```

and every Emissary CRD it creates repeats that id:

```yaml
spec:
  ambassador_id: ["f1"]
```

**Which flavor should a fixture use?**

- **`default`** if the fixture only creates `Mapping`s, `Host`s, `TCPMapping`s
  and the like. These add no global config, so they coexist safely and the
  `default` shard runs them in parallel. Omit `ambassador_id` entirely.
- **A numbered flavor** if the fixture creates a `Module` or any of the global
  services. Only one fixture at a time runs on a given flavor.

`make e2e/run` fans out one `chainsaw` process per flavor plus one for
`default`. Each flavor's shard runs serially (`--parallel 1`); the shards run
concurrently. `flavors.sh check` runs first and fails the suite if any fixture
has a missing or unknown `flavor` label, since such a fixture would match no
shard's selector and be skipped in silence.

> **Why flavors disable the chart's Module.** The chart ships a `Module` named
> `ambassador` (`module.enabled: true`). Emissary keys its module store by name
> alone, so that Module would collide with one a fixture creates: the loser is
> renamed to `ambassador.<namespace>` and `get_module("ambassador")` then
> returns whichever won, silently dropping the fixture's config. Flavors are
> therefore installed with `--set module.enabled=false`, which leaves the
> fixture's own Module as the only one carrying that `ambassador_id`.

> **Adding a flavor.** Edit `FLAVORS` in `flavors.sh`. Ports come out of the
> `8100-8159` block k3d publishes at cluster-creation time, so up to six fit;
> a seventh needs the range widened *and* the cluster recreated.

The probe step shells out to `probe.sh`, which runs the host-built
`kat-client` binary against `$GATEWAY_URL` and retries until a `jq` assertion
passes. The kat-server image is templated into manifests using the Chainsaw
binding `kat_server_image`, which reads `KAT_SERVER_IMAGE` from the
environment.

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
stderr, then exits 1 so Chainsaw runs the fixture's `catch` block.

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
   flavor block mapped to the host loadbalancer, and Traefik disabled.
2. `make images` builds Emissary's container images via goreleaser snapshot.
3. `e2e/install` imports images into k3d, then `helm install`s the CRDs chart
   and the ingress chart pinned to the locally-built image tag, then
   `e2e/install-flavors` installs one extra release per flavor.
4. `e2e/run` shards `chainsaw test` across the flavors.

### Iterating

Once the cluster is up and Emissary is installed, you usually only need:

```
make e2e/run              # re-run every fixture, sharded by flavor
make e2e/run/gzip-minimum # re-run one fixture (flavor read from the file)
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
> 6789 or the `8100-8159` flavor block on a cluster you created before they
> were added), you have to `make e2e/cluster-down && make e2e/cluster-up` to
> pick it up. `helm upgrade` alone won't get traffic in.

To reinstall just the flavors (after changing `FLAVORS` in `flavors.sh`, say):

```
make e2e/uninstall-flavors && make VERSION=v4.0.0-local e2e/install-flavors
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
| `E2E_GATEWAY_URL`    | `http://localhost`     | host the probes target (flavor ports are appended) |
| `E2E_LOCAL_VERSION`  | `v4.0.0-local`         | short chart VERSION (dirty trees produce strings longer than k8s' 63-char label limit) |
| `E2E_FLAVORS`        | `f1:8110 ... f4:8140`  | `<name>:<port-base>` list of isolated Emissary installs |
| `E2E_DEFAULT_PARALLEL` | `4`                  | how many `default`-flavor fixtures run at once |

Per-fixture probe and apply timeouts are set in `.chainsaw.yaml` and on
individual steps inside each `chainsaw-test.yaml`.

## Adding a new fixture

1. Create `test/e2e/fixtures/<name>/`.
2. Put the resources you want in `manifests.yaml` (Deployment, Service,
   Mapping, whatever the scenario needs). To reference the locally-built
   kat-server image, use the Chainsaw binding `($kat_server_image)`:
   ```yaml
   image: ($kat_server_image)
   ```
3. Pick a flavor (see [Flavors](#flavors)). If the fixture creates a `Module`
   or a global service, use a numbered flavor and put `ambassador_id: ["<f>"]`
   on every Emissary CRD in `manifests.yaml`. Otherwise use `default` and set
   no `ambassador_id`.
4. Put the requests in `queries.json` (kat-client format, relative URLs).
5. Write `chainsaw-test.yaml` defining a `Test` resource with:
   - `metadata.labels.flavor` naming the flavor from step 3.
   - `bindings` for the env-derived values (`kat_server_image`, `kat_client`,
     `gateway_url`, or `gateway_tcp_url` for TCPMapping fixtures).
   - A `try` block that `apply`s `manifests.yaml` (with `template: true` so
     the `($kat_server_image)` substitution happens) and runs
     `../../probe.sh queries.json '<jq-expr>'` via a `script` step.
   - A `catch` block that dumps pod logs, describes pods, and lists the
     scenario's CRDs. Existing fixtures are good templates.
6. Run `make e2e/run/<name>` and confirm it passes.

No registration step is required beyond the `flavor` label. Chainsaw discovers
every directory under `fixtures/` that contains a `chainsaw-test.yaml`, and
`flavors.sh check` fails the suite if that label is missing or unknown.

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
| one Emissary per test (`AMBASSADOR_ID`)    | a flavor pin                                        |

When porting an assertion about an *absence* (`"X" not in r.headers`), AND in a
positive check too. `has("X") | not` is satisfied by a 404 from an Emissary
that never picked up the route, so on its own it passes for the wrong reason.
`fixtures/gzip-unsupported-content-type` shows the pattern.

## How CI runs this

`.github/workflows/test-images.yaml` mirrors the local flow: it spins up k3d
(with the flavor port block published), imports the images built by
`build-images`, `helm install`s the charts produced by `build-charts` (pinned
to that same image tag), calls `flavors.sh install` for the flavor releases,
installs `chainsaw` into `tools/bin/`, and then runs `flavors.sh run`. On
failure each fixture's `catch` block dumps pod logs and the relevant CRDs; the
workflow's diagnostics step then walks every Emissary namespace.

Both CI and `make` call `flavors.sh`, so the flavor list and its port math have
exactly one definition.

The key difference from local: CI consumes pre-built image and chart
artifacts from upstream jobs instead of running `make images` / `make
charts` itself.
