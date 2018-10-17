# Configuring Services

Ambassador is designed so that the author of a given Kubernetes service can easily and flexibly configure how traffic gets routed to the service. The core abstraction used to support service authors is a `mapping`.

## Mappings

Mappings associate REST [_resources_](#resources) with Kubernetes [_services_](#services). A resource, here, is a group of things defined by a URL prefix; a service is exactly the same as in Kubernetes. Ambassador _must_ have one or more mappings defined to provide access to any services at all.

Each mapping can also specify, among other things:

- a [_rewrite rule_](/reference/rewrites) which modifies the URL as it's handed to the Kubernetes service;
- a [_weight_](/reference/canary) specifying how much of the traffic for the resource will be routed using the mapping;
- a [_host_](/reference/host) specifying a required value for the HTTP `Host` header;
- a [_shadow_](/reference/shadowing) marker, specifying that this mapping will get a copy of traffic for the resource; and
- other [_headers_](/reference/headers) which must appear in the HTTP request.

## Defining Mappings

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

## Configuring Mappings

Ambassador supports a number of additional attributes to configure and customize mappings.

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| [`add_request_headers`](/reference/add_request_headers) | specifies a dictionary of other HTTP headers that should be added to each request when talking to the service |
| [`cors`](/reference/cors)           | enables Cross-Origin Resource Sharing (CORS) setting on a mapping | 
| [`grpc`](/user-guide/grpc) | if true, tells the system that the service will be handling gRPC calls |
| [`headers`](/reference/headers)      | specifies a list of other HTTP headers which _must_ appear in the request for this mapping to be used to route the request |
| [`host`](/reference/host) | specifies the value which _must_ appear in the request's HTTP `Host` header for this mapping to be used to route the request |
| [`host_regex`](/reference/host) | if true, tells the system to interpret the `host` as a [regular expression](http://en.cppreference.com/w/cpp/regex/ecmascript) |
| [`host_rewrite`](/reference/host) | forces the HTTP `Host` header to a specific value when talking to the service |
| [`method`](/reference/method)                  | defines the HTTP method for this mapping (e.g. GET, PUT, etc. -- must be all uppercase) |
| `method_regex`            | if true, tells the system to interpret the `method` as a [regular expression](http://en.cppreference.com/w/cpp/regex/ecmascript) |
| `prefix_regex`            | if true, tells the system to interpret the `prefix` as a [regular expression](http://en.cppreference.com/w/cpp/regex/ecmascript) |
| [`rate_limits`](/reference/rate-limits) | specifies a list rate limit rules on a mapping |
| [`regex_headers`](/reference/headers)           | specifies a list of HTTP headers and [regular expressions](http://en.cppreference.com/w/cpp/regex/ecmascript) which _must_ match for this mapping to be used to route the request |
| [`rewrite`](/reference/rewrites)      | replaces the URL prefix with when talking to the service |
| `timeout_ms`              | the timeout, in milliseconds, for requests through this `Mapping`. Defaults to 3000. |
| [`tls`](#using-tls)       | if true, tells the system that it should use HTTPS to contact this service. (It's also possible to use `tls` to specify a certificate to present to the service.) |
| `use_websocket`           | if true, tells Ambassador that this service will use websockets |

Ambassador supports multiple deployment patterns for your services. These patterns are designed to let you safely release new versions of your service, while minimizing its impact on production users.

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| [`shadow`](/reference/shadowing)     | if true, a copy of the resource's traffic will go the `service` for this `Mapping`, and the reply will be ignored. |
| [`weight`](/reference/canary)        | specifies the (integer) percentage of traffic for this resource that will be routed using this mapping |

These attributes are less commonly used, but can be used to override Ambassador's default behavior in specific cases.

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| `auto_host_rewrite`       | if true, forces the HTTP `Host` header to the `service` to which Ambassador routes |
| `case_sensitive`          | determines whether `prefix` matching is case-sensitive; defaults to True |
| [`envoy_override`](/reference/override) | supplies raw configuration data to be included with the generated Envoy route entry. |
| [`host_redirect`](/reference/redirects) | if true, this `Mapping` performs an HTTP 301 `Redirect`, with the host portion of the URL replaced with the `service` value. |
| [`path_redirect`](/reference/redirects)           | if set when `host_redirect` is also true, the path portion of the URL will replaced with the `path_redirect` value in the HTTP 301 `Redirect`. |
| [`precedence`](#a-nameprecedencea-using-precedence)           | an integer overriding Ambassador's internal ordering for `Mapping`s. An absent `precedence` is the same as a `precedence` of 0. Higher `precedence` values are matched earlier. |

The name of the mapping must be unique. If no `method` is given, all methods will be proxied.

## Mapping Evaluation Order

Ambassador sorts mappings such that those that are more highly constrained are evaluated before those less highly constrained. The prefix length, the request method and the constraint headers are all taken into account.

If absolutely necessary, you can manually set a `precedence` on the mapping (see below). In general, you should not need to use this feature unless you're using the `regex_headers` or `host_regex` matching features. If there's any question about how Ambassador is ordering rules, the diagnostic service is a good first place to look: the order in which mappings appear in the diagnostic service is the order in which they are evaluated.

## Optional Fallback Mapping

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

###  <a name="precedence"></a> Using `precedence`

Ambassador sorts mappings such that those that are more highly constrained are evaluated before those less highly constrained. The prefix length, the request method and the constraint headers are all taken into account. These mechanisms, however, may not be sufficient to guarantee the correct ordering when regular expressions or highly complex constraints are in play.

For those situations, a `Mapping` can explicitly specify the `precedence`. A `Mapping` with no `precedence` is assumed to have a `precedence` of 0; the higher the `precedence` value, the earlier the `Mapping` is attempted.

If multiple `Mapping`s have the same `precedence`, Ambassador's normal sorting determines the ordering within the `precedence`; however, there is no way that Ambassador can ever sort a `Mapping` with a lower `precedence` ahead of one at a higher `precedence`.

###  <a name="using-tls"></a> Using `tls`

In most cases, you won't need the `tls` attribute: just use a `service` with an `https://` prefix. However, note that if the `tls` attribute is present and `true`, Ambassador will originate TLS even if the `service` does not have the `https://` prefix.

If `tls` is present with a value that is not `true`, the value is assumed to be the name of a defined TLS context, which will determine the certificate presented to the upstream service. TLS context handling is a beta feature of Ambassador at present; please [contact us on Slack](https://d6e.co/slack) if you need to specify TLS origination certificates.

## Namespaces and Mappings

Given that `AMBASSADOR_NAMESPACE` is correctly set, Ambassador can map to services in other namespaces by taking advantage of Kubernetes DNS:

- `service: servicename` will route to a service in the same namespace as the Ambassador, and
- `service: servicename.namespace` will route to a service in a different namespace.

## Resources

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

## Services

A `service` is simply a URL to Ambassador. For example:

- `servicename` assumes that DNS can resolve the bare servicename, and that it's listening on the default HTTP port;
- `servicename.domain` supplies a domain name (for example, you might do this to route across namespaces in Kubernetes); and
- `service:3000` supplies a nonstandard port number.

At present, Ambassador relies on Kubernetes to do load balancing: it trusts that using the DNS to look up the service by name will do the right thing in terms of spreading the load across all instances of the service.

