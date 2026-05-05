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
└── fixtures/
    └── <fixture-name>/
        ├── chainsaw-test.yaml  # the Test resource (apply, probe, catch)
        └── manifests.yaml      # Deployments/Services/Mappings for the scenario
```

Each test gets a fresh, randomly-named namespace (`generateName: e2e-`).
Emissary watches all namespaces, so Mappings/Listeners/TCPMappings created in
those test namespaces are picked up automatically by the cluster-wide
Emissary install (which still lives in the fixed `emissary` namespace).

The probe step shells out to the host-built `kat-client` binary against
`$GATEWAY_URL`. The kat-server image is templated into manifests using the
Chainsaw binding `kat_server_image`, which reads `KAT_SERVER_IMAGE` from the
environment.

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
   80/443 (HTTP fixtures) and 6789 (TCPMapping fixtures) mapped to the host
   loadbalancer and Traefik disabled.
2. `make images` builds Emissary's container images via goreleaser snapshot.
3. `e2e/install` imports images into k3d, then `helm install`s the CRDs chart
   and the ingress chart pinned to the locally-built image tag.
4. `e2e/run` invokes `chainsaw test` against `test/e2e/fixtures/`.

### Iterating

Once the cluster is up and Emissary is installed, you usually only need:

```
make e2e/run              # re-run fixtures against the existing deployment
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
> 6789 on a cluster you created before it was added), you have to
> `make e2e/cluster-down && make e2e/cluster-up` to pick it up. `helm
> upgrade` alone won't get traffic in.

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
| `E2E_GATEWAY_URL`    | `http://localhost`     | URL the probes target                    |
| `E2E_LOCAL_VERSION`  | `v4.0.0-local`         | short chart VERSION (dirty trees produce strings longer than k8s' 63-char label limit) |

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
3. Write `chainsaw-test.yaml` defining a `Test` resource with:
   - `bindings` for the env-derived values (`kat_server_image`, `kat_client`,
     `gateway_url`).
   - A `try` block that `apply`s `manifests.yaml` (with `template: true` so
     the `($kat_server_image)` substitution happens) and runs a probe via a
     `script` step.
   - A `catch` block that dumps pod logs, describes pods, and lists the
     scenario's CRDs. Existing fixtures are good templates.
4. Run `make e2e/run` and confirm it passes.

No registration step is required. Chainsaw discovers every directory under
`fixtures/` that contains a `chainsaw-test.yaml`.

## How CI runs this

`.github/workflows/test-images.yaml` mirrors the local flow: it spins up
k3d, imports the images built by `build-images`, `helm install`s the charts
produced by `build-charts` (pinned to that same image tag), installs
`chainsaw` into `tools/bin/`, and then runs `chainsaw test
test/e2e/fixtures`. On failure each fixture's `catch` block already dumps
pod logs and the relevant CRDs; the workflow's separate diagnostics step
adds cluster-wide context.

The key difference from local: CI consumes pre-built image and chart
artifacts from upstream jobs instead of running `make images` / `make
charts` itself.
