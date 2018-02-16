# Ambassador Configuration

Ambassador is configured in a declarative fashion, using YAML manifests to describe the state of the world. As with Kubernetes, Ambassador's manifests are identified with `apiVersion`, `kind`, and `name`. The current `apiVersion` is `ambassador/v0`; currently-supported `kind`s are:

- [`Module`](#module) manifests configure things with can apply to Ambassador as a whole. For example, the `ambassador` module can define listener ports, and the `tls` module can configure TLS termination for Ambassador.

- [`AuthService`](#authservice) manifests configures the external authentication service[s] that Ambassador will use.

- [`Mapping`](#mapping) manifests associate REST _resources_ with Kubernetes _services_. Ambassador _must_ have one or more mappings defined to provide access to any services at all.

## Ambassador Configuration

Ambassador assembles its configuration from YAML blocks that may be stored:

- as `annotations` on Kubernetes `service`s (this is the recommended technique);
- as data in a Kubernetes `ConfigMap`; or
- as files in Ambassador's local filesystem.

The data contained within each YAML block is the same no matter where the blocks are stored, and multiple YAML documents are likewise supported no matter where the blocks are stored.

### Running Ambassador Within Kubernetes

When you run Ambassador within Kubernetes:

1. At startup, Ambassador will look for the `ambassador-config` Kubernetes `ConfigMap`. If it exists, its contents will be used as the baseline Ambassador configuration.
2. Ambassador will then scan Kubernetes `service`s in its namespace, looking for `annotation`s named `getambassador.io/config`. YAML from these `annotation`s will be merged into the baseline Ambassador configuration.
3. Whenever any services change, Ambassador will update its `annotation`-based configuration.
4. The baseline configuration, if present, will **never be updated** after Ambassador starts. To effect a change in the baseline configuration, use Kubernetes to force a redeployment of Ambassador.

**Note:** the baseline configuration is not required. It is completely possible - indeed, recommended - to use _only_ `annotation`-based configuration.

### Running Ambassador Within a Custom Image

You can also run Ambassador by building a custom image that contains baked-in configuration:

1. All the configuration data should be collected within a single directory on the filesystem.
2. At image startup, run `ambassador config $configdir $envoy_json_out` where
   - `$configdir` is the path of the directory containing the configuration data, and
   - `$envoy_json_out` is the path to the `envoy.json` to be written.

In this usage, Ambassador will not look for `annotation`-based configuration, and will not update any configuration after startup.

### Best Practices for Configuration

Ambassador's configuration is assembled from multiple YAML blocks, to help enable self-service routing and make it easier for multiple developers to collaborate on a single larger application. This implies a few things:

- Ambassador's configuration should be under version control.

    While you can always read back Ambassador's configuration from `annotation`s or its diagnostic service, it's far better to have a master copy under git or the like. Ambassador doesn't do any versioning of its configuration.

- Be aware that Ambassador tries to not start with a broken configuration, but it's not perfect.

    Gross errors will result in Ambassador refusing to start, in which case `kubectl logs` will be helpful. However, it's always possible to e.g. map a resource to the wrong service, or use the wrong `rewrite` rules. Ambassador can't detect that on its own, although its diagnostic pages can help you figure it out.

- Be careful of mapping collisions.

    If two different developers try to map `/user/` to something, this can lead to unexpected behavior. Ambassador's canary-deployment logic means that it's more likely that traffic will be split between them than that it will throw an error -- again, the diagnostic service can help you here.

## Namespaces

Ambassador supports multiple namespaces within Kubernetes. To make this work correctly, you need to set the `AMBASSADOR_NAMESPACE` environment variable in Ambassador's container. By far the easiest way to do this is using Kubernetes' downward API (this is included in the YAML files from `getambassador.io`):

```yaml
        env:
        - name: AMBASSADOR_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace          
```

Given that `AMBASSADOR_NAMESPACE` is set, Ambassador [mappings](#mapping) can operate within the same namespace, or across namespaces. **Note well** that mappings will have to explictly include the namespace with the service to cross namespaces; see the [mapping](#mappings) documentation for more information.

If you only want Ambassador to only work within a single namespace, set `AMBASSADOR_SINGLE_NAMESPACE` as an environment variable.

## Modules

Modules let you enable and configure special behaviors for Ambassador, in ways that may apply to Ambassador as a whole or which may apply only to some mappings. The actual configuration possible for a given module depends on the module.

### The `ambassador` Module

If present, the `ambassador` module defines system-wide configuration. **You will not normally need this module.** The defaults in the `ambassador` module are, roughly:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  # If present, service_port will be the port Ambassador listens
  # on for microservice access. If not present, Ambassador will
  # use 443 if TLS is configured, 80 otherwise.
  # service_port: 80

  # diag_port is the port where Ambassador will listen for requests
  # to the diagnostic service.
  # diag_port: 8877

  # admin_port is the port where Ambassador's Envoy will listen for
  # low-level admin requests. You should almost never need to change
  # this.
  # admin_port: 8001

  # liveness probe defaults on, but you can disable it.
  # liveness_probe:
  #   enabled: false

  # readiness probe defaults on, but you can disable it.
  # readiness_probe:
  #   enabled: false
```

Everything in this file has a default that should cover most situations; it should only be necessary to include them to override the defaults in highly-custom situations.

#### Probes

The default liveness and readiness probes map `/ambassador/v0/check_alive` and `ambassador/v0/check_ready` internally to check Envoy itself. If you'd like to, you can change these to route
requests to some other service. For example, to have the readiness probe map to the Quote of the Moment's health check, you could do

```yaml
readiness_probe:
  service: qotm
  rewrite: /health
```

The liveness and readiness probe both support `prefix`, `rewrite`, and `service`, with the same meanings as for [mappings](#mappings). Additionally, the `enabled` boolean may be set to `false` (as an the commented-out examples above) to disable support for the probe entirely.

**Note well** that configuring the probes in the `ambassador` module only means that Ambassador will respond to the probes. You must still configure Kubernetes to perform the checks, as shown in the Datawire-provided YAML files.

### The `tls` Module

If present, the `tls` module defines system-wide configuration for TLS.

When running in Kubernetes, Ambassador will enable TLS termination whenever it finds valid TLS certificates stored in the `ambassador-certs` Kubernetes secret, so many Kubernetes installations of Ambassador will not need a `tls` module at all.

The most common case requiring a `tls` module is redirecting cleartext traffic on port 80 to HTTPS on port 443, which can be done with the following `tls` module:

```
---
apiVersion: ambassador/v0
kind:  Module
name:  tls
config:
  server:
    enabled: True
    redirect_cleartext_from: 80
```

TLS configuration is examined in more detail in the documentation on [TLS termination](../how-to/tls-termination.md) and [TLS client certificate authentication](../how-to/auth-tls-certs.md).

### The `authentication` Module

The `authentication` module is now deprecated. Use the `AuthService` manifest type instead.

## AuthService

An `AuthService` manifest configures Ambassador to use an external service to check authentication and authorization for incoming requests:

```yaml
---
apiVersion: ambassador/v0
kind:  AuthService
name:  authentication
config:
  auth_service: "example-auth:3000"
  path_prefix: "/extauth"
  allowed_headers:
  - "x-qotm-session"
```

- `auth_service` gives the URL of the authentication service
- `path_prefix` (optional) gives a prefix prepended to every request going to the auth service
- `allowed_headers` (optional) gives an array of headers that will be incorporated into the upstream request if the auth service supplies them.

You may use multiple `AuthService` manifests to round-robin authentication requests among multiple services. **Note well that all services must use the same `path_prefix` and `allowed_headers`;** if you try to have different values, you'll see an error in the diagnostics service, telling you which value is being used.

## Mappings

Mappings associate REST [_resources_](#resources) with Kubernetes [_services_](#services). A resource, here, is a group of things defined by a URL prefix; a service is exactly the same as in Kubernetes. Ambassador _must_ have one or more mappings defined to provide access to any services at all.

Each mapping can also specify, among other things:

- a [_rewrite rule_](#rewriting) which modifies the URL as it's handed to the Kubernetes service;
- a [_weight_](#weights) specifying how much of the traffic for the resource will be routed using the mapping;
- a [_host_](#host) specifying a required value for the HTTP `Host` header; and
- other [_headers_](#headers) which must appear in the HTTP request.

### Mapping Evaluation Order

Ambassador sorts mappings such that those that are more highly constrained are evaluated before those less highly constrained. The prefix length, the request method and the constraint headers are all taken into account.

If there's any question about how Ambassador is ordering rules, the diagnostic service is a good first place to look: the order in which mappings appear in the diagnostic service is the order in which they are evaluated.

### Optional Fallback Mapping

Ambassador will respond with a `404 Not Found` to any request for which no mapping exists. If desired, you can define a fallback "catch-all" mapping so all unmatched requests will be sent to an upstream service.

For example, defining a mapping with only a `/` prefix will catch all requests previously unhandled and forward them to an external service:

```yaml
---
apiVersion: ambassador/v0
kind: Mapping
name: catch-all
prefix: /
service: https://www.getambassador.io
```

### Defining Mappings

Mapping definitions are fairly straightforward. Here's an example for a REST service which Ambassador will contact using HTTP:

```yaml
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: http://qotm
```

and a REST service which Ambassador will contact using HTTPS:

```
---
apiVersion: ambassador/v0
kind:  Mapping
name:  quote_mapping
prefix: /qotm/quote/
rewrite: /quotation/
service: https://qotm
```

(Note that the 'http://' prefix for an HTTP service is optional.)

Here's an example for a CQRS service (using HTTP):

```yaml
---
apiVersion: ambassador/v0
kind: Mapping
name: cqrs_get_mapping
prefix: /cqrs/
method: GET
service: getcqrs
---
apiVersion: ambassador/v0
kind: Mapping
name: cqrs_put_mapping
prefix: /cqrs/
method: PUT
service: putcqrs
```

Required attributes for mappings:

- `name` is a string identifying the `Mapping` (e.g. in diagnostics)
- `prefix` is the URL prefix identifying your [resource](#resources)
- `service` is the name of the [service](#services) handling the resource; must include the namespace (e.g. `myservice.othernamespace`) if the service is in a different namespace than Ambassador

Common optional attributes for mappings:

- `rewrite` is what to [replace](#rewriting) the URL prefix with when talking to the service
- `host_rewrite`: forces the HTTP `Host` header to a specific value when talking to the service
- `grpc`: if present with a true value, tells the system that the service will be handling gRPC calls
- `method`: defines the HTTP method for this mapping (e.g. GET, PUT, etc. -- must be all uppercase!)
- `method_regex`: if present and true, tells the system to interpret the `method` as a regular expression
- `weight`: if present, specifies the (integer) percentage of traffic for this resource that will be routed using this mapping
- `host`: if present, specifies the value which _must_ appear in the request's HTTP `Host` header for this mapping to be used to route the request
- `headers`: if present, specifies a dictionary of other HTTP headers which _must_ appear in the request for this mapping to be used to route the request
- `tls`: if present and true, tells the system that it should use HTTPS to contact this service. (It's also possible to use `tls` to specify a certificate to present to the service; if this is something you need, please ask for details on [Gitter](https://gitter.im/datawire/ambassador).)
- `cors`: if present, enables Cross-Origin Resource Sharing (CORS) setting on a mapping. For more details about each setting, see [using cors](#using-cors)

Less-common optional attributes for mappings:

- `add_request_headers`: if present, specifies a dictionary of other HTTP headers that should be added to each request when talking to the service. Envoy dynamic `value`s `%CLIENT_IP%` and `%PROTOCOL%` are supported, in addition to static `value`s.
- `auto_host_rewrite`: if present with a true value, forces the HTTP `Host` header to the `service` to which we will route.
- `case_sensitive`: determines whether `prefix` matching is case-sensitive; defaults to True.
- `host_redirect`: if set, this `Mapping` performs an HTTP 301 `Redirect`, with the host portion of the URL replaced with the `host_redirect` value.
- `path_redirect`: if set, this `Mapping` performs an HTTP 301 `Redirect`, with the path portion of the URL replaced with the `path_redirect` value.
- `timeout_ms`: the timeout, in milliseconds, for requests through this `Mapping`. Defaults to 3000.
- `use_websocket`: if present with a true value, tells Ambassador that this service will use websockets.
- `envoy_override`: supplies raw configuration data to be included with the generated Envoy route entry.

The name of the mapping must be unique. If no `method` is given, all methods will be proxied.

#### Using `host_rewrite`

By default, the `Host` header is not altered when talking to the service -- whatever `Host` header the client gave to Ambassador will be presented to the service. For many microservices this will be fine, but if you use Ambassador to route to services that use the `Host` header for routing, it's likely to fail (legacy monoliths are particularly susceptible to this, as well as external services). You can use `host_rewrite` to force the `Host` header to whatever value that such target services need.

An example: the default Ambassador configuration includes the following mapping for `httpbin.org`:

```yaml
---
apiVersion: ambassador/v0
kind: Mapping
name: httpbin_mapping
prefix: /httpbin/
service: httpbin.org:80
host_rewrite: httpbin.org
```

As it happens, `httpbin.org` is virtually hosted, and it simply _will not_ function without a `Host` header of `httpbin.org`, which means that the `host_rewrite` attribute is necessary here.

#### Using `weight`

The `weight` attribute specifies how much traffic for a given resource will be routed using a given mapping. Its value is an integer percentage between 0 and 100. Ambassador will balance weights to make sure that, for every resource, the mappings for that resource will have weights adding to 100%. (In the simplest case, a single mapping is guaranteed to receive 100% of the traffic no matter whether it's assigned a `weight` or not.)

Specifying a weight only makes sense if you have multiple mappings for the same resource, and typically you would _not_ assign a weight to the "default" mapping (the mapping expected to handle most traffic): letting Ambassador assign that mapping all the traffic not otherwise spoken for tends to make life easier when updating weights. Here's an example, which might appear during a canary deployment:

```yaml
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm
---
apiVersion: ambassador/v0
kind: Mapping
name: qotm2_mapping
prefix: /qotm/
service: qotmv2
weight: 10
```

In this case, the `qotm2_mapping` will receive 10% of the requests for `/qotm/`, and Ambassador will assign the remaining 90% to the `qotm_mapping`.

#### Using `host`

A mapping that specifies the `host` attribute will take effect _only_ if the HTTP `Host` header matches the value in the `host` attribute. You may have multiple mappings listing the same resource but different `host` attributes to effect `Host`-based routing. An example:

```yaml
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: qotm1
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
host: qotm.datawire.io
service: qotm2
```

will map requests for `/qotm/` to the `qotm2` service if the `Host` header is `qotm.datawire.io`, and to the `qotm1` service otherwise.

#### Using `headers`

If present, the `headers` attribute must be a dictionary of `header`: `value` pairs, for example:

```yaml
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
headers:
  x-qotm-mode: canary
  x-random-header: datawire
service: qotm
```

will allow requests to `/qotm/` to succeed only if the `x-qotm-mode` header has the value `canary` _and_ the `x-random-header` has the value `datawire`.

#### `headers`, `host`, and `method`

Internally:

- the `host` attribute becomes a `header` match on the `:authority` header; and
- the `method` attribute becomes a `header` match on the `:method` header.

You will see these headers in the diagnostic service if you use the `method` or `host` attributes.

#### Using `tls`

In most cases, you won't need the `tls` attribute: just use a `service` with an `https://` prefix. However, note that if the `tls` attribute is present and `true`, Ambassador will originate TLS even if the `service` does not have the `https://` prefix.

If `tls` is present with a value that is not `true`, the value is assumed to be the name of a defined TLS context, which will determine the certificate presented to the upstream service. TLS context handling is a beta feature of Ambassador at present; please [contact us on Gitter](https://gitter.im/datawire/ambassador) if you need to specify TLS origination certificates.

#### Using `cors`

A mapping that specifies the `cors` attribute will automatically enable the CORS filter. An example:

```yaml
apiVersion: ambassador/v0
kind:  Mapping
name:  cors_mapping
prefix: /cors/
service: cors-example
cors:
  origins: http://foo.example,http://bar.example
  methods: POST, GET, OPTIONS
  headers: Content-Type
  credentials: true
  exposed_headers: X-Custom-Header
  max_age: "86400"
```

CORS settings:

- `origins`: Specifies a comma-separated list of allowed domains for the `Access-Control-Allow-Origin` header. To allow all origins, use the wildcard `"*"` value.
- `methods`: if present, specifies a comma-separated list of allowed methods for the `Access-Control-Allow-Methods` header.
- `headers`: if present, specifies a comma-separated list of allowed headers for the `Access-Control-Allow-Headers` header.
- `credentials`: if present with a true value (boolean), will send a `true` value for the `Access-Control-Allow-Credentials` header.
- `exposed_headers`: if present, specifies a comma-separated list of allowed headers for the `Access-Control-Expose-Headers` header.
- `max_age`: if present, indicated how long the results of the preflight request can be cached, in seconds. This value must be a string.

#### Using `envoy_override`

It's possible that your situation may strain the limits of what Ambassador can do. The `envoy_override` attribute is provided for cases we haven't predicted: any object given as the value of `envoy_override` will be inserted into the Envoy `Route` synthesized for the given mapping. For example, you could enable Envoy's `auto_host_rewrite` by supplying

```yaml
envoy_override:
  auto_host_rewrite: True
```

Note that `envoy_override` cannot, at present, change any element already synthesized in the mapping: it can only add additional information.

#### Namespaces and Mappings

Given that `AMBASSADOR_NAMESPACE` is correctly set, Ambassador can map to services in other namespaces by taking advantage of Kubernetes DNS:

- `service: servicename` will route to a service in the same namespace as the Ambassador, and
- `service: servicename.namespace` will route to a service in a different namespace.

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

A `service` is simply a URL to Ambassador. For example:

- `servicename` assumes that DNS can resolve the bare servicename, and that it's listening on the default HTTP port;
- `servicename.domain` supplies a domain name (for example, you might do this to route across namespaces in Kubernetes); and
- `service:3000` supplies a nonstandard port number.

At present, Ambassador relies on Kubernetes to do load balancing: it trusts that using the DNS to look up the service by name will do the right thing in terms of spreading the load across all instances of the service.

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

### Modifying Ambassador's Underlying Envoy Configuration

Ambassador uses Envoy for the heavy lifting of proxying.

If you wish to use Envoy features that aren't (yet) exposed by Ambassador, you can use your own custom config template. To do this, create a templated `envoy.json` file using the Jinja2 template language. Then, use this template as the value for the key `envoy.j2` in your ConfigMap. This will then replace the [default template](https://github.com/datawire/ambassador/tree/master/ambassador/templates).

Please [contact us on Gitter](https://gitter.im/datawire/ambassador) for more information if this seems necessary for a given use case (or better yet, submit a PR!) so that we can expose this in the future.
