# Dev Portal Implementation notes

See [Installation documentation](INSTALL.md) for install instructions.

Run ambassador image in pod, but with env variable to disable (in entrypoint.sh) ambx and envory, so just run diagd. call diagd endpoint to get. also need to add the `--no-envoy --no-checks` flags to the `diagd` command line.

query diagd every 60 seconds: $AMBASSADOR_POD/ambassador/v0/diag/?json=true

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

