# Sample service exposing OpenAPI docs

This service publishes OpenAPI docs at `/.ambassador-internal/openapi-docs` in order to help test the Dev Portal.

To run it:

```
$ kubectl apply -f server.yaml
```

The OpenDoc API changes its public name every time it is regenerated, so you can tell when the Dev Portal last queried it.
