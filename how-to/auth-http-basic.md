# Auth with HTTP Basic Auth

The `authentication` module allows Ambassador to require authentication for specific mappings. Not all mappings must require authentication: a mapping which has no configuration for the `authentication` module will be assumed to require no authentication.

Ambassador's authentication module is built around asking an authentication service for a particular type of authentication. The service is defined globally; the type of auth requested is defined per module. Ambassador provides a built-in authentication service which currently implements only HTTP Basic Authentication; you can also write your own authentication service.

To enable the `authentication` module and tell Ambassador which service to use, you can use one of the following:

```shell
curl -XPUT -H "Content-Type: application/json" \
     -d '{ "ambassador": "basic" }' \
      http://localhost:8888/ambassador/module/authentication
```

to enable Ambassador's built-in authentication service, or

```shell
curl -XPUT -H "Content-Type: application/json" \
     -d '{ "auth_service": "<target>" }' \
      http://localhost:8888/ambassador/module/authentication
```

to use an external auth service, where `target` is a hostname and port, e.g. `authv1:5000`. See [external authentication](#external-authentication) below.

After enabling the module globally, any mapping that should require authentication needs to be told what kind of authentication to request. If you're using the built-in service, "basic" is currently the only type supported:

```shell
curl -XPUT -H "Content-Type: application/json" \
     -d '{ "type": "basic" }' \
      http://localhost:8888/ambassador/mapping/<mapping-name>/module/authentication
```

You can also define this association when creating a mapping, e.g.:

```shell
curl -XPUT -H"Content-Type: application/json" \
     -d'{ "prefix": "/qotm/quote/", "rewrite": "/quote/", "service": "qotm", "modules": { "authentication": { "type": "basic" } } }' \
     http://localhost:8888/ambassador/mapping/qotm_quote_map
```

Finally, the built-in service requires using consumers to tell Ambassador who should be allowed to authenticate. To successfully authenticate using HTTP Basic Auth, the consumer must have an `authentication` module config defining the auth type ("basic") and the password:

```shell
curl -XPUT -H"Content-Type: application/json" \
     -d'{ "type":"basic", "password":"alice" }' \
     http://localhost:8888/ambassador/consumer/<consumer-id>/module/authentication
```

which, again, can be supplied when the consumer is created:

```shell
curl -XPOST -H"Content-Type: application/json" \
     -d'{ "username": "alice", "fullname": "Alice Rules", "modules": { "authentication": { "type":"basic", "password":"alice" } } }' \
     http://localhost:8888/ambassador/consumer
```

## Consumers

Consumers represent human end users of Ambassador, and may be required for some modules to function. For example, the `authentication` module may require defining consumers to let Ambassador know who's allowed to authenticate.

A consumer is created with a `POST` request:

```shell
curl -XPOST -H"Content-Type: application/json" \
     -d<consumer-dict> \
     http://localhost:8888/ambassador/consumer
```

where `consumer-dict` has the details of the new consumer:

```json
{
    "username": <username>,
    "fullname": <full-name>,
    "shortname": <short-name>,
    "modules": <module-dict>
}
```

- `username` is the username to use when logging in, etc.
- `full-name` is the consumer's full name. Ambassador assumes nothing about how full names are formed.
- `short-name` (optional) is a short name that the consumer prefers to be called.
- `module-dict` (optional) defines module configuration for this consumer.

You can get a list of all the consumers that Ambassador knows about with

```shell
curl http://localhost:8888/ambassador/consumer
```

and you can delete a specific consumer with

```shell
curl -XDELETE http://localhost:8888/ambassador/consumer/<consumer-id>
```

You can manipulate specific bits of module information for this consumer, as well (the [modules](#modules) section has more on this):

```shell
curl http://localhost:8888/ambassador/consumer/<consumer-id>/module/<module-name>
```

to read a single module's config;

```shell
curl -XPUT -H "Content-Type: application/json" \
     -d <module-dict> \
     http://localhost:8888/ambassador/consumer/<consumer-id>/module/<module-name>
```

to alter a single module's config (`<module-dict>` is the dictionary of new configuration information for the given consumer and module); and

```shell
curl -XDELETE http://localhost:8888/ambassador/consumer/<consumer-id>/module/<module-name>
```

to delete a single module's config for a given consumer.
