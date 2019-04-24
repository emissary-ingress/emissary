Because we modify the Ambassador config betweend tests, it is deployed
at run-time, instead of here.

Because `04-*.yaml` creates CRD instances, we define the CRDs in
`02-ambassador-pro-crds.yaml` instead of in the real
`03-ambassador.yaml`.
