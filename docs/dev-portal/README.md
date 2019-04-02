# Dev Portal Implementation notes

See [Installation documentation](INSTALL.md) for install instructions.

## Dev Portal Web Server

API endpoints:

### Listing

`/openapi/services` returns a JSON list of objects of the form:

```json
{
    service_name: "myservice",
    service_namespace: "default",
    routing_prefix: "/myservice",
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
    openapi_doc: {... openapi doc ...}
}
```

`openapi_doc` can also be `null`, if there is no documentation.

