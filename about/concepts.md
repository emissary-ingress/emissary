# Ambassador Concepts

At the heart of Ambassador are the ideas of [_mappings_](#mappings), [_modules_](#modules), and [_consumers_](#consumers):

- [Mappings](#mappings) associate REST _resources_ with Kubernetes _services_. Ambassador _must_ have one or more mappings defined to provide access to any services at all.

- [Modules](#modules) let you enable and configure special behaviors for Ambassador, in ways which may apply to Ambassador as a whole or which may apply only to some mappings. For example, the `authentication` module allows Ambassador to require authentication per mapping.

- [Consumers](#consumers) represent human end users of Ambassador, and may be required for some modules to function. For example, the `authentication` module may require defining consumers to let Ambassador know who's allowed to authenticate.

## Mappings

Mappings associate REST [_resources_](#resources) with Kubernetes [_services_](#services). A resource, here, is a group of things defined by a URL profix; a service is exactly the same as in Kubernetes. Ambassador _must_ have one or more mappings defined to provide access to any services at all.

Each mapping can also specify a [_rewrite rule_](#rewriting) which modifies the URL as it's handed to the Kubernetes service, and a set of [_module configuration_](#modules) specific to that mapping

### Defining Mappings

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

You can get a list of all the mappings that Ambassador knows about with

```shell
curl http://localhost:8888/ambassador/mapping
```

and you can delete a specific mapping with

```shell
curl -XDELETE http://localhost:8888/ambassador/mapping/<mapping-name>
```

You can manipulate specific bits of module information for this mapping, as well (the [modules](#modules) section has more on this):

```shell
curl http://localhost:8888/ambassador/mapping/<mapping-name>/module/<module-name>
```

to read a single module's config;

```shell
curl -XPUT -H "Content-Type: application/json" \
     -d <module-dict> \
     http://localhost:8888/ambassador/mapping/<mapping-name>/module/<module-name>
```

to alter a single module's config (`<module-dict>` is the dictionary of new configuration information for the given mapping and module); and

```shell
curl -XDELETE http://localhost:8888/ambassador/mapping/<mapping-name>/module/<module-name>
```

to delete a single module's config for a given mapping.

Also, the `mapping-name` identifies the mapping in statistics output and such.

### Resources

To Ambassador, a `resource` is a group of one or more URLs that all share a common prefix in the URL path. For example:

```shell
https://ambassador.example.com/resource1/foo
https://ambassador.example.com/resource1/bar
https://ambassador.example.com/resource1/baz/zing
https://ambassador.example.com/resource1/baz/zung
```

all share the `/resource1/` path prefix, so can be considered a single resource. On the other hand:

```shell
https://ambassador.example.com/resource1/foo
https://ambassador.example.com/resource2/bar
https://ambassador.example.com/resource3/baz/zing
https://ambassador.example.com/resource4/baz/zung
```

share only the prefix `/` -- you _could_ tell Ambassador to treat them as a single resource, but it's probably not terribly useful.

Note that the length of the prefix doesn't matter: if you want to use prefixes like `/v1/this/is/my/very/long/resource/name/`, go right ahead, Ambassador can handle it.

Also note that Ambassador does not actually require the prefix to start and end with `/` -- however, in practice, it's a good idea. Specifying a prefix of

```shell
/man
```

would match all of the following:

```shell
https://ambassador.example.com/man/foo
https://ambassador.example.com/mankind
https://ambassador.example.com/man-it-is/really-hot-today
https://ambassador.example.com/manohmanohman
```

which is probably not what was intended.

### Services

A `service` is exactly the same thing to Ambassador as it is to Kubernetes. When you tell Ambassador to map a resource to a service, it requires there to be a Kubernetes service with _exactly_ the same name, and it trusts whatever Kubernetes has to say about ports and such.

At present, Ambassador relies on Kubernetes to do load balancing: it trusts that using the DNS to look up the service by name will do the right thing in terms of spreading the load across all instances of the service. This will change shortly, in order to gain better control of load balancing.

### Rewrite Rules

Once Ambassador uses a prefix to identify the service to which a given request should be passed, it can rewrite the URL before handing it off to the service. By default, the `prefix` is rewritten to `/`, so e.g. if we map `/prefix1/` to the service `service1`, then

```shell
http://ambassador.example.com/prefix1/foo/bar
```

would effectively be written to

```shell
http://service1/foo/bar
```

when it was handed to `service1`.

You can change the rewriting: for example, if you choose to rewrite the prefix as `/v1/` in this example, the final target would be

```shell
http://service1/v1/foo/bar
```

And, of course, you can choose to rewrite the prefix to the prefix itself, so that

```shell
http://ambassador.example.com/prefix1/foo/bar
```

would be "rewritten" as

```shell
http://service1/prefix1/foo/bar
```

## Modules

Modules let you enable and configure special behaviors for Ambassador, in ways that may apply to Ambassador as a whole or which may apply only to some mappings. The actual configuration possible for a given module depends on the module: at present, the only supported module is the [`authentication` module](#the-authentication-module).

You use `PUT` requests to the admin interface to save or update a module's global configuration:

```shell
curl -XPUT -H "Content-Type: application/json" -d <module-dict> \
      http://localhost:8888/ambassador/module/<module-name>
```

where `<module-name>` is the name of the module from the list below, and `<module-dict>` is a dictionary of configuration information. Which information is needed depends on the module.

Module configuration information can also be associated with [mappings](#mappings) and [consumers](#consumers). These are also be set and updated using `PUT` requests:

```shell
curl -XPUT -H "Content-Type: application/json" -d <module-dict> \
      http://localhost:8888/ambassador/mapping/<mapping-name>/module/<module-name>
```

and

```shell
curl -XPUT -H "Content-Type: application/json" -d <module-dict> \
      http://localhost:8888/ambassador/consumer/<consumer-id>/module/<module-name>
```

You can get a list of all the modules for which Ambassador knows about configuration information with `GET` requests:

```shell
curl http://localhost:8888/ambassador/module
```

for global configuration, and

```shell
curl http://localhost:8888/ambassador/mapping/<mapping-name>/module
```

or

```shell
curl http://localhost:8888/ambassador/consumer/<consumer-id>/module
```

for mapping- or consumer-specific configuration.

Finally, you can delete module configuration with `DELETE` requests:

```shell
curl -XDELETE http://localhost:8888/ambassador/module/<module-name>
curl -XDELETE http://localhost:8888/ambassador/mapping/<mapping-name>/module/<module-name>
curl -XDELETE http://localhost:8888/ambassador/consumer/<consumer-id>/module/<module-name>
```

### The `authentication` module

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

```shell
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
