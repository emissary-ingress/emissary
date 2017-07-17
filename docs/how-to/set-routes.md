# Mappings, Routes, and Rewriting

Make sure access to Ambassador's admin port is set up.

## Mappings

You use `PUT` requests to the admin interface to map a resource to a service:

```shell
curl -XPUT -H "Content-Type: application/json" \
      -d <mapping-dict> \
      http://localhost:8888/ambassador/mapping/<mapping-name>
```

where `<mapping-name>` is a unique name that identifies this mapping, and `<mapping-dict>` is a dictionary that defines the mapping:

```json
{
    "prefix": <url-prefix>,
    "service": <service-name>,
    "rewrite": <rewrite-as>,
    "modules": <module-dict>
}
```

- `<url-prefix>` is the URL prefix identifying your [resource](#resources)
- `<service-name>` is the name of the [service](#services) handling the resource
- `<rewrite-as>` (optional) is what to [replace](#rewriting) the URL prefix with when talking to the service
- `<module-dict>` (optional) defines any relevant module configuration for this mapping.

The `mapping-name` is used to delete mappings later, and to identify mappings in statistics and such.

The `url-prefix` should probably begin and end with `/` to avoid confusion. An URL prefix of `man` would match the URL `https://getambassador.io/manifold`, which is probably not what you want -- using `/man/` is more clear.

The `service-name` **must** match the name of a service defined in Kubernetes.

The `rewrite-as` part is optional: if not given, it defaults to `/`. Whatever it's set to, the `url-prefix` gets replaced with `rewrite-as` when the request is forwarded:

- If `url-prefix` is `/v1/user/` and `rewrite-as` is `/`, then `/v1/user/foo` will appear to the service as `/foo`.

- If `url-prefix` is `/v1/user/` and `rewrite-as` is `/v2/`, then `/v1/user/foo` will appear to the service as `/v2/foo`.

- If `url-prefix` is `/v1/` and `rewrite-as` is `/v2/`, then `/v1/user/foo` will appear to the service as `/v2/user/foo`.

etc.

Ambassador updates Envoy's configuration five seconds after any mapping change. If another change arrives during that time, the timer is restarted.

### Listing Mappings

You can list all the extant mappings with

```shell
curl http://localhost:8888/ambassador/mapping
```

### Creating a Mapping

An example mapping:

```shell
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/v1/user/", "service": "usersvc" }' \
      http://localhost:8888/ambassador/mapping/user
```

will create a mapping named `user` that will cause requests for any resource with a URL starting with `/v1/user/` to be sent to the `usersvc` Kubernetes service, with the `/v1/user/` part replaced with `/` -- `/v1/user/alice` would appear to the service as simply `/alice`.

If instead you did

```shell
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "/v1/user/", "service": "usersvc", "rewrite": "/v2/" }' \
      http://localhost:8888/ambassador/mapping/user
```

then `/v1/user/alice` would appear to the service as `/v2/alice`.

### Deleting a Mapping

To remove a mapping, use a `DELETE` request:

```shell
curl -XDELETE http://localhost:8888/ambassador/mapping/user
```

will delete the mapping from above.

### Checking for a Mapping

To check whether the `user` mapping exists, you can simply

```shell
curl http://localhost:8888/ambassador/mapping/user
```
