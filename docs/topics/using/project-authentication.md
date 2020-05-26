# Adding Authentication to your `Project`

## This feature is in BETA. Your feedback is greatly appreciated!

Make sure you have configured at least one working [authentication Filter](filters). The [HOWTO section](../../howtos/) has numerous dentailed entries on integrating with specific IDPs.

The following `FilterPolicy` will enable authentication for your `Project`'s production deployment:

```
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
  name: foo
  namespace: default
spec:
  rules:
  - host: <your-hostname>
    path: <your-project-prefix>** # e.g. /foo/**
    filters:
    - name: <your-filter-name>
      namespace: <your-filter-namespace>
```

You can apply the following `FilterPolicy` to enable authentication for your `Project`'s preview deploys. Note that you can use a different authentication filter for previews, and in fact you can omit the project-specific portion of the path if you wish to lock down previews for all `Projects`:

```
apiVersion: getambassador.io/v2
kind: FilterPolicy
metadata:
  name: foo
  namespace: default
spec:
  rules:
  - host: <your-hostname>
    path: /.previews/<your-project-prefix>** # e.g. /.previews/foo/*
    filters:
    - name: <your-filter-name>
      namespace: <your-filter-namespace>
```
