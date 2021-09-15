# Contributing to the Emissary Ingress Helm Chart

This Helm chart is used to install the Emissary Ingress.

## Developing

All work on the helm chart should be done in a separate branch off `master` and
contributed with a Pull Request targeting `master`.

**Note**: All updates to the chart require you update the `version` in
`Chart.yaml`.

## Testing

The `ci/` directory contains scripts that will be run on PRs to `master`.

- `make chart-test` run from this directory will run the chart tests.

## Releasing

TODO
