# Dev Portal Implementation notes

See [Installation documentation](INSTALL.md) for install instructions.

## Architecture

We run a pod that has two containers:

* The Ambassador image, but with Envoy disabled, in order to get diagd access.
* The dev portal image.

### Update loop

The Dev Portal runs a loop every 60 seconds:

1. The Dev Portal queries its local diagd to get all Services registered with Ambassador.
2. For each Service is sends a query (via the global Ambassador) to that service's `/.well-known/openapi-docs` path, to get the OpenAPI documentation for that service, if any.

## Dev Portal Web Server

API endpoints:

### Listing

`/openapi/services` returns a JSON list of objects of the form:

```json
{
    service_name: "myservice",
    service_namespace: "default",
    routing_prefix: "/myservice",
    routing_base_url: "https://api.example.com",
    has_doc: true
}
```

### Get OpenAPI documentation

`/openapi/services/{namespace}/{name}/openapi.json` for a given Service's namespace and name returns the OpenAPI docs as JSON.

### Set metadata

POST to `/openapi/services` with:


```json
{
    service_name: "myservice",
    service_namespace: "default",
    routing_prefix: "/myservice",
    routing_base_url: "https://api.example.com",
    openapi_doc: {... openapi doc ...}
}
```

`openapi_doc` can also be `null`, if there is no documentation.

