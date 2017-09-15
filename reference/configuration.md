# Ambassador Configuration

At the heart of Ambassador are the ideas of [_resources_](#resources)], [_mappings_](#mappings), [_modules_](#modules), and soon [_consumers_](#consumers). There may also be an `ambassador.yaml` file.

- [The `ambassador.yaml` file](#ambassador-yaml) is optional. If present, it defines system-wide things like TLS certificates.

- [Mappings](#mappings) associate REST _resources_ with Kubernetes _services_. Ambassador _must_ have one or more mappings defined to provide access to any services at all.

- [Modules](#modules) let you enable and configure special behaviors for Ambassador, in ways which may apply to Ambassador as a whole or which may apply only to some mappings. For example, the `authentication` module allows Ambassador to require authentication per mapping.

- [Consumers](#consumers) represent human end users of Ambassador, and may be required for some modules to function. For example, the `authentication` module may require defining consumers to let Ambassador know who's allowed to authenticate.

Ambassador assembles its configuration from YAML files contained within a single directory on the filesystem. When run as part of an image build, the caller must tell Ambassador the path to the directory; when run as a proxy pod within Kubernetes, Ambassador assumes that its configuration has been published as a `ConfigMap` named `ambassador-config`. The easiest way to create such a `ConfigMap` is to assemble a directory of appropriate YAML files, and use 

```shell
kubectl create configmap ambassador-config --from-file config-dir-path
```

to publish the configuration.

### Best Practices for Configuration Files

Ambassador uses a directory structure for its configuration to allow multiple developers to more easily collaborate on a microservice application: developers working on a given microservice can create a mapping file for their single microservice without having to worry about stepping on other developers, etc. This implies a few things:

- Ambassador's configuration should be under version control.

    Having the configuration directory under git or the like is an obvious thought here. Ambassador doesn't do any versioning of its configuration.

- Be aware that Ambassador tries to not start with a broken configuration, but it's not perfect.

    Gross errors will result in Ambassador refusing to start, in which case `kubectl logs` will be helpful. However, it's always possible to e.g. map a resource to the wrong service, or using the wrong `rewrite` rules. Ambassador can't detect that on its own.

- Be careful of mapping collisions.

    If two different developers try to map `/user/` to something, Ambassador should catch it and refuse to start, but it's still not what you want (obviously).

## The `ambassador.yaml` File

If present, the `ambassador.yaml` file defines system-wide configuration. **You will not normally need this file.**

```yaml
ambassador:
  # If present, service_port will be the port Ambassador listens
  # on for microservice access. If not present, Ambassador will
  # use 443 if TLS is configured, 80 otherwise.
  service_port: 80

  # admin_port is where we'll listen for administrative requests.
  admin_port: 8001

  # TLS setup
  tls:
    cert_chain_file: ...
    private_key_file: ...
```

Everything in this file has a sane default; you should need to supply it _only_ to override defaults in highly-custom situations.

## Mappings

Mappings associate REST [_resources_](#resources) with Kubernetes [_services_](#services). A resource, here, is a group of things defined by a URL profix; a service is exactly the same as in Kubernetes. Ambassador _must_ have one or more mappings defined to provide access to any services at all.

Each mapping can also specify a [_rewrite rule_](#rewriting) which modifies the URL as it's handed to the Kubernetes service, and a set of [_module configuration_](#modules) specific to that mapping

### Defining Mappings

**NOTE WELL: The mapping definition format is very likely to change shortly.**

One or more mappings may be in YAML files with names starting with "mapping-", for example a file named `mapping-qotm.yaml` might have mappings associated with the `qotm` service:

```yaml
mappings:
  qotm_mapping:
    prefix: /qotm/
    service: qotm
  quote_mapping:
    prefix: /qotm/quote/
    service: qotm
    rewrite: /quote/
```

The above example defines two mappings (`qotm_mapping` and `quote_mapping`) which map two resources over to the `qotm` service. Valid attributes for mappings:

- `prefix` is the URL prefix identifying your [resource](#resources)
- `rewrite` (optional) is what to [replace](#rewriting) the URL prefix with when talking to the service
- `service` is the name of the [service](#services) handling the resource
- `method` (optional) defines the HTTP method for this mapping (e.g. GET, PUT, etc. -- must be all uppercase!)
- `method_regex` (optional) if present and true, tells the system to interpret the `method` as a regular expression
- `grpc` (optional) if present with a true value, tells the system that the service will be handling gRPC calls

The name of the mapping must be unique. If no `method` is given, all methods will be proxied.

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

Modules let you enable and configure special behaviors for Ambassador, in ways that may apply to Ambassador as a whole or which may apply only to some mappings. The actual configuration possible for a given module depends on the module: at present, the only supported module is the [`authentication` module](../how-to/auth-http-basic.md), which is configured using the `module-authentication.yaml` file:

```yaml
module:
  auth_service: "example-auth:3000"
  path_prefix: "/extauth"
  allowed_headers:
  - "x-qotm-session"
```

- `auth_service` gives the URL of the authentication service
- `path_prefix` (optional) gives a prefix prepended to every request going to the auth service
- `allowed_headers` (optional) gives an array of headers that will be incorporated into the upstream request if the auth service supplies them.
