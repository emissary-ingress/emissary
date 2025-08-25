# Contributing to the Emissary Ingress Helm Chart

This Helm chart is used to install the Emissary Ingress.

## Developing

All work on the helm chart should be done in a separate branch off `master` and
contributed with a Pull Request targeting `master`.

**Note**: All updates to the chart require you update the `version` in
`Chart.yaml`.

## Testing

The `ci.in` directory contains scripts that will be run on PRs to `master`.

- `make test-chart` run from the top of the Git checkout will run the
  chart tests.  You need to set the `DEV_KUBECONFIG` environment
  variable to point to the cluster that you would like the tests to
  run in.

## Releasing

TODO
