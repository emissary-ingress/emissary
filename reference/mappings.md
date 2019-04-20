# Configuring Services

Ambassador is designed so that the author of a given Kubernetes service can easily and flexibly configure how traffic gets routed to the service. The core abstraction used to support service authors is a `mapping`, which can apply to HTTP, GRPC, and Websockets at layer 7 via a `Mapping` resource, or to raw TCP connections at layer 4 via a `TCPMapping`.

Ambassador _must_ have one or more mappings defined to provide access to any services at all.

## `Mapping`

An Ambassador `Mapping` associates REST [_resources_](#resources) with Kubernetes [_services_](#services). A resource, here, is a group of things defined by a URL prefix; a service is exactly the same as in Kubernetes. 

Each mapping can also specify, among other things:

- a [_rewrite rule_](/reference/rewrites) which modifies the URL as it's handed to the Kubernetes service;
- a [_weight_](/reference/canary) specifying how much of the traffic for the resource will be routed using the mapping;
- a [_host_](/reference/host) specifying a required value for the HTTP `Host` header;
- a [_shadow_](/reference/shadowing) marker, specifying that this mapping will get a copy of traffic for the resource; and
- other [_headers_](/reference/headers) which must appear in the HTTP request.

## Mapping Configuration

Ambassador supports a number of attributes to configure and customize mappings.

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| [`add_request_headers`](/reference/add_request_headers) | specifies a dictionary of other HTTP headers that should be added to each request when talking to the service |
| [`add_response_headers`](/reference/add_response_headers) | specifies a dictionary of other HTTP headers that should be added to each response when returning response to client |
| [`cors`](/reference/cors)           | enables Cross-Origin Resource Sharing (CORS) setting on a mapping |
| `enable_ipv4` | if true, enables IPv4 DNS lookups for this mapping's service (the default is set by the [Ambassador module](/reference/modules)) |
| `enable_ipv6` | if true, enables IPv6 DNS lookups for this mapping's service (the default is set by the [Ambassador module](/reference/modules)) |
| [`grpc`](/user-guide/grpc) | if true, tells the system that the service will be handling gRPC calls |
| [`headers`](/reference/headers)      | specifies a list of other HTTP headers which _must_ appear in the request for this mapping to be used to route the request |
| [`host`](/reference/host) | specifies the value which _must_ appear in the request's HTTP `Host` header for this mapping to be used to route the request |
| [`host_regex`](/reference/host) | if true, tells the system to interpret the `host` as a [regular expression](http://en.cppreference.com/w/cpp/regex/ecmascript) |
| [`host_rewrite`](/reference/host) | forces the HTTP `Host` header to a specific value when talking to the service |
| [`load_balancer`](/reference/core/load-balancer) | configures load balancer on a mapping
| [`method`](/reference/method)                  | defines the HTTP method for this mapping (e.g. GET, PUT, etc. -- must be all uppercase) |
| `method_regex`            | if true, tells the system to interpret the `method` as a [regular expression](http://en.cppreference.com/w/cpp/regex/ecmascript) |
| `prefix_regex`            | if true, tells the system to interpret the `prefix` as a [regular expression](http://en.cppreference.com/w/cpp/regex/ecmascript) |
| [`rate_limits`](/reference/rate-limits) | specifies a list rate limit rules on a mapping |
| [`regex_headers`](/reference/headers)           | specifies a list of HTTP headers and [regular expressions](http://en.cppreference.com/w/cpp/regex/ecmascript) which _must_ match for this mapping to be used to route the request |
| [`rewrite`](/reference/rewrites)      | replaces the URL prefix with when talking to the service |
| `timeout_ms`              | the timeout, in milliseconds, for requests through this `Mapping`. Defaults to 3000. |
| [`tls`](#using-tls)       | if true, tells the system that it should use HTTPS to contact this service. (It's also possible to use `tls` to specify a certificate to present to the service.) |
| `use_websocket`           | if true, tells Ambassador that this service will use websockets |

If both `enable_ipv4` and `enable_ipv6` are set, Ambassador will prefer IPv6 to IPv4. See the [Ambassador module](/reference/modules) documentation for more information.

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
| [`host_redirect`](/reference/redirects) | if true, this `Mapping` performs an HTTP 301 `Redirect`, with the host portion of the URL replaced with the `service` value. |
| [`path_redirect`](/reference/redirects)           | if set when `host_redirect` is also true, the path portion of the URL will replaced with the `path_redirect` value in the HTTP 301 `Redirect`. |
| [`precedence`](#a-nameprecedencea-using-precedence)           | an integer overriding Ambassador's internal ordering for `Mapping`s. An absent `precedence` is the same as a `precedence` of 0. Higher `precedence` values are matched earlier. |
| `bypass_auth`             | if true, tells Ambassador that this service should bypass `ExtAuth` (if configured) |

The name of the mapping must be unique. If no `method` is given, all methods will be proxied.

## Example Mappings

Mapping definitions are fairly straightforward. Here's an example for a REST service which Ambassador will contact using HTTP:

```yaml
---
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
service: http://qotm
```

and a REST service which Ambassador will contact using HTTPS:

```yaml
---
apiVersion: ambassador/v1
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
apiVersion: ambassador/v1
kind: Mapping
name: cqrs_get_mapping
prefix: /cqrs/
method: GET
service: getcqrs
---
apiVersion: ambassador/v1
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

Ambassador routes traffic to a `service`. A `service` is defined as:

```
[scheme://]service[.namespace][:port]
```

Where everything except for the `service` is optional.

- `scheme` can be either `http` or `https`; if not present, the default is `http`.
- `service` is the name of a service (typically the service name in Kubernetes or Consul); it is not allowed to contain the `.` character. 
- `namespace` is the namespace in which the service is running. If not supplied, it defaults to the namespace in which Ambassador is running. When using a Consul resolver, `namespace` is not allowed.
- `port` is the port to which a request should be sent. If not specified, it defaults to `80` when the scheme is `http` or `443` when the scheme is `https`. Note that the [resolver](/reference/core/resolvers) may return a port in which case the `port` setting is ignored.

Note that while using `service.namespace.svc.cluster.local` may work for Kubernetes resolvers, the preferred syntax is `service.namespace`.

## Mapping Evaluation Order

Ambassador sorts mappings such that those that are more highly constrained are evaluated before those less highly constrained. The prefix length, the request method and the constraint headers are all taken into account.

If absolutely necessary, you can manually set a `precedence` on the mapping (see below). In general, you should not need to use this feature unless you're using the `regex_headers` or `host_regex` matching features. If there's any question about how Ambassador is ordering rules, the diagnostic service is a good first place to look: the order in which mappings appear in the diagnostic service is the order in which they are evaluated.

## Optional Fallback Mapping

Ambassador will respond with a `404 Not Found` to any request for which no mapping exists. If desired, you can define a fallback "catch-all" mapping so all unmatched requests will be sent to an upstream service.

For example, defining a mapping with only a `/` prefix will catch all requests previously unhandled and forward them to an external service:

```yaml
---
apiVersion: ambassador/v1
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

