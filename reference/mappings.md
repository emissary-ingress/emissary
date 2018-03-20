## Mappings

Mappings associate REST [_resources_](#resources) with Kubernetes [_services_](#services). A resource, here, is a group of things defined by a URL prefix; a service is exactly the same as in Kubernetes. Ambassador _must_ have one or more mappings defined to provide access to any services at all.

Each mapping can also specify, among other things:

- a [_rewrite rule_](#rewriting) which modifies the URL as it's handed to the Kubernetes service;
- a [_weight_](#weights) specifying how much of the traffic for the resource will be routed using the mapping;
- a [_host_](#host) specifying a required value for the HTTP `Host` header; and
- other [_headers_](#headers) which must appear in the HTTP request.

### Mapping Evaluation Order

Ambassador sorts mappings such that those that are more highly constrained are evaluated before those less highly constrained. The prefix length, the request method and the constraint headers are all taken into account.

If absolutely necessary, you can manually set a `precedence` on the mapping (see below). In general, you should not need to use this feature unless you're using the `regex_headers` or `host_regex` matching features. If there's any question about how Ambassador is ordering rules, the diagnostic service is a good first place to look: the order in which mappings appear in the diagnostic service is the order in which they are evaluated.

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

```yaml
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

- `prefix_regex`: if present and true, tells the system to interpret the `prefix` as a regular expression
- `rewrite` is what to [replace](#rewriting) the URL prefix with when talking to the service
- `host_rewrite`: forces the HTTP `Host` header to a specific value when talking to the service
- `grpc`: if present with a true value, tells the system that the service will be handling gRPC calls
- `method`: defines the HTTP method for this mapping (e.g. GET, PUT, etc. -- must be all uppercase!)
- `method_regex`: if present and true, tells the system to interpret the `method` as a regular expression
- `weight`: if present, specifies the (integer) percentage of traffic for this resource that will be routed using this mapping
- `host`: if present, specifies the value which _must_ appear in the request's HTTP `Host` header for this mapping to be used to route the request
- `host_regex`: if present and true, tells the system to interpret the `host` as a regular expression
- `headers`: if present, specifies a list of other HTTP headers which _must_ appear in the request for this mapping to be used to route the request
- `regex_headers`: if present, specifies a list of HTTP headers and regular expressions which they _must_ match 
for this mapping to be used to route the request
- `tls`: if present and true, tells the system that it should use HTTPS to contact this service. (It's also possible to use `tls` to specify a certificate to present to the service; if this is something you need, please ask for details on [Gitter](https://gitter.im/datawire/ambassador).)
- `cors`: if present, enables Cross-Origin Resource Sharing (CORS) setting on a mapping. For more details about each setting, see [using cors](#using-cors)

Less-common optional attributes for mappings:

- `add_request_headers`: if present, specifies a dictionary of other HTTP headers that should be added to each request when talking to the service. Envoy dynamic `value`s `%CLIENT_IP%` and `%PROTOCOL%` are supported, in addition to static `value`s.
- `auto_host_rewrite`: if present with a true value, forces the HTTP `Host` header to the `service` to which we will route.
- `case_sensitive`: determines whether `prefix` matching is case-sensitive; defaults to True.
- `host_redirect`: if set, this `Mapping` performs an HTTP 301 `Redirect`, with the host portion of the URL replaced with the `host_redirect` value.
- `path_redirect`: if set, this `Mapping` performs an HTTP 301 `Redirect`, with the path portion of the URL replaced with the `path_redirect` value.
- `precedence`: an integer overriding Ambassador's internal ordering for `Mapping`s. An absent `precedence` is the same as a `precedence` of 0. Higher `precedence` values are matched earlier.
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

#### Using `host` and `host_regex`

A mapping that specifies the `host` attribute will take effect _only_ if the HTTP `Host` header matches the value in the `host` attribute. If `host_regex` is `true`, the `host` value is taken to be a regular expression, otherwise an exact string match is required.

You may have multiple mappings listing the same resource but different `host` attributes to effect `Host`-based routing. An example:

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
---
apiVersion: ambassador/v0
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
host: "^qotm[2-9]\\.datawire\\.io$"
host_regex: true
service: qotm3
```

will map requests for `/qotm/` to 

- the `qotm2` service if the `Host` header is `qotm.datawire.io`;
- the `qotm3` service if the `Host` header matches `^qotm[2-9]\\.datawire\\.io$`; and to
- the `qotm1` service otherwise.

**Note well** that enclosing regular expressions in quotes can be important to prevent backslashes from being doubled.

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

#### Using `precedence`

Ambassador sorts mappings such that those that are more highly constrained are evaluated before those less highly constrained. The prefix length, the request method and the constraint headers are all taken into account. These mechanisms, however, may not be sufficient to guarantee the correct ordering when regular expressions or highly complex constraints are in play.

For those situations, a `Mapping` can explicitly specify the `precedence`. A `Mapping` with no `precedence` is assumed to have a `precedence` of 0; the higher the `precedence` value, the earlier the `Mapping` is attempted.

If multiple `Mapping`s have the same `precedence`, Ambassador's normal sorting determines the ordering within the `precedence`; however, there is no way that Ambassador can ever sort a `Mapping` with a lower `precedence` ahead of one at a higher `precedence`.

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
