# Mappings Services

Ambassador Edge Stack is designed so that the author of a given Kubernetes service can easily and flexibly configure how traffic gets routed to the service. The core abstraction used to support service authors is a `mapping`, which can apply to HTTP, GRPC, and Websockets at layer 7 via a `Mapping` resource, or to raw TCP connections at layer 4 via a `TCPMapping`.

Ambassador Edge Stack _must_ have one or more mappings defined to provide access to any services at all. The name of the mapping must be unique.

## Mapping Configuration

Ambassador Edge Stack supports a number of attributes to configure and customize mappings.

### Required attributes for mappings

| Required attribute        | Description               |
| :------------------------ | :------------------------ |
| `name`                    | is a string identifying the `Mapping` (e.g. in diagnostics) |
| [`prefix`](#resources)    | is the URL prefix identifying your [resource](#resources) |
| [`service`](#services)    | is the name of the [service](#services) handling the resource; must include the namespace (e.g. `myservice.othernamespace`) if the service is in a different namespace than Ambassador |

### Additional Attributes

| Attribute | Description &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; | Output |
|-------------------------|------------------------------------------------------------------------------------------------------------|-------------|
| `add_linkerd_headers` | When true, Ambassador Edge Stack adds the `l5d-dst-overrideheader` to the request and the service field is used as a value. Note that when `add_linkerd_headers` is set to true in the `ambassador` Module, the configuration will be applied to all mappings, including auth. `ambassador` Module and individual mapping configurations can be used together, and the latest will always take precedence over the module's configurations. | boolean |
| `add_request_headers` | Specifies a dictionary of other HTTP headers that should be added to each request when talking to the service. | string list |
| `add_response_headers` | Specifies a dictionary of other HTTP headers that should be added to each response when returning a response to the client. | string list |
| `cluster_idle_timeout_ms` | The timeout, in milliseconds, before an idle connection upstream is closed (may be set on a Mapping, AuthService, or in the `ambassador` Module). | integer |
| `connect_timeout_ms` | The timeout, in milliseconds, for requests coming through the Cluster for this Mapping. Defaults to 3000. | integer |
| `cors` | Enables Cross-Origin Resource Sharing (CORS) setting on a mapping | dictionary |
| `circuit_breakers` | Configures circuit breaking on a mapping. | dictionary |
| `enable_ipv4` | If true, enables IPv4 DNS lookups for this mapping's service (the default is set by the ambassadorModule) | boolean |
| `enable_ipv6` | If true, enables IPv6 DNS lookups for this mapping's service (the default is set by the `ambassador` Module). | boolean |
| `grpc` | If true, tells the system that the service will be handling gRPC calls. | boolean |
| `headers` | Specifies a list of other HTTP headers which must appear in the request for this mapping to be used to route the request. | string list |
| `host` | Specifies the value which must appear in the request's HTTP Host header for this mapping to be used to route the request. | string |
| `host_regex`| If true, tells the system to interpret the host as a regular expression. | boolean |
| `host_rewrite` | Forces the HTTP Host header to a specific value when talking to the service. | string |
| `idle_timeout_ms` | The timeout, in milliseconds, after which connections through this Mapping will be terminated if no traffic is seen. Defaults to 300000 (5 minutes). | integer |
| `load_balancer` | Configures load balancer on a mapping. | dictionary |
| `method` | Defines the HTTP method for this mapping (e.g. GET, PUT, etc. -- must be all uppercase). | string |
| `method_regex` | If true, tells the system to interpret the method as a regular expression. | boolean |
| `prefix_regex` | If true, tells the system to interpret the prefix as a regular expression and requires that the entire path must match the regex, not just the prefix. | boolean |
| `rate_limits` | Specifies a list rate limit rules on a mapping. | dictionary |
| `remove_request_headers` | Specifies a list of HTTP headers that are dropped from the request before sending upstream. | string list |
| `remove_response_headers` | Specifies a list of HTTP headers that are dropped from the response before sending to the client. | string list |
| `regex_headers` | Specifies a list of HTTP headers and regular expressions that must match for this mapping to be used to route the request. | string list |
| `rewrite` | Replaces the URL prefix with when talking to the service. Defaults to `"/"`, meaning the prefix is stripped. | string |
| `retry_policy` | Performs automatic retries upon request failures. | dictionary |
| `timeout_ms` | The timeout, in milliseconds, for requests through this Mapping. Defaults to 3000. | integer |
| `tls` | If true, tells the system that it should use HTTPS to contact this service. (It's also possible to use TLS to specify a certificate to present to the service.) | boolean |
| `use_websocket` | If true, tells Ambassador Edge Stack that this service will use WebSockets. | boolean |

If both `enable_ipv4` and `enable_ipv6` are set, Ambassador Edge Stack will prefer IPv6 to IPv4. See the `ambassador`[Module](../modules) documentation for more information.

Ambassador Edge Stack supports multiple deployment patterns for your services. These patterns are designed to let you safely release new versions of your service, while minimizing its impact on production users.

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| [`shadow`](../shadowing)     | if true, a copy of the resource's traffic will go the `service` for this `Mapping`, and the reply will be ignored. |
| [`weight`](../canary)        | specifies the (integer) percentage of traffic for this resource that will be routed using this mapping |

These attributes are less commonly used, but can be used to override Ambassador Edge Stack's default behavior in specific cases.

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| `auto_host_rewrite`       | if true, forces the HTTP `Host` header to the `service` to which Ambassador Edge Stack routes |
| `case_sensitive`          | determines whether `prefix` matching is case-sensitive; defaults to True |
| [`host_redirect`](../redirects) | if true, this `Mapping` performs an HTTP 301 `Redirect`, with the host portion of the URL replaced with the `service` value. |
| [`path_redirect`](../redirects)           | if set when `host_redirect` is also true, the path portion of the URL will replaced with the `path_redirect` value in the HTTP 301 `Redirect`. |
| [`precedence`](#using-precedence)           | an integer overriding Ambassador Edge Stack's internal ordering for `Mapping`s. An absent `precedence` is the same as a `precedence` of 0. Higher `precedence` values are matched earlier. |
| `bypass_auth`             | if true, tells Ambassador Edge Stack that this service should bypass `ExtAuth` (if configured) |

If no `method` is given, the Mapping will apply to all HTTP `methods`.

## Mapping Resources and CRDs

Mapping resources can be defined as annotations on Kubernetes services or as independent custom resource definitions.

For example, here is a `Mapping` on a Kubernetes `service`:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: qu
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v2
      kind:  Mapping
      name:  httpbin-mapping
      prefix: /httpbin/
      service: http://httpbin.org
spec:
  ports:
  - name: httpbin
    port: 80
```

The same `Mapping` can be created as an independent resource:

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  httpbin-mapping
spec:
  prefix: /httpbin/
  service: http://httpbin.org
```

If you're new to Ambassador, start with the CRD approach. Note that you *must* use the `getambassador.io/v2` `apiVersion` as noted above.

## Additional Example Mappings

Here's an example service for implementing the CQRS pattern (using HTTP):

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  cqrs-get
spec:
  prefix: /cqrs/
  method: GET
  service: getcqrs
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  cqrs-put
spec:
  prefix: /cqrs/
  method: PUT
  service: putcqrs
```

## Resources

To Ambassador, a `resource` is a group of one or more URLs that all share a common prefix in the URL path. For example:

```shell
https://ambassador.example.com/resource1/foo
https://ambassador.example.com/resource1/bar
https://ambassador.example.com/resource1/baz/zing
https://ambassador.example.com/resource1/baz/zung
```

all share the `/resource1/` path prefix, so it can be considered a single resource. On the other hand:

```shell
https://ambassador.example.com/resource1/foo
https://ambassador.example.com/resource2/bar
https://ambassador.example.com/resource3/baz/zing
https://ambassador.example.com/resource4/baz/zung
```

share only the prefix `/` -- you _could_ tell Ambassador Edge Stack to treat them as a single resource, but it's probably not terribly useful.

Note that the length of the prefix doesn't matter: if you want to use prefixes like `/v1/this/is/my/very/long/resource/name/`, go right ahead, Ambassador Edge Stack can handle it.

Also note that Ambassador Edge Stack does not actually require the prefix to start and end with `/` -- however, in practice, it's a good idea. Specifying a prefix of

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

Ambassador Edge Stack routes traffic to a `service`. A `service` is defined as:

```
[scheme://]service[.namespace][:port]
```

Where everything except for the `service` is optional.

- `scheme` can be either `http` or `https`; if not present, the default is `http`.
- `service` is the name of a service (typically the service name in Kubernetes or Consul); it is not allowed to contain the `.` character.
- `namespace` is the namespace in which the service is running. Starting with Ambassador 1.0.0, if not supplied, it defaults to the namespace in which the Mapping resource is defined. The default behavior can be configured using the [`ambassador` Module](../core/ambassador). When using a Consul resolver, `namespace` is not allowed.
- `port` is the port to which a request should be sent. If not specified, it defaults to `80` when the scheme is `http` or `443` when the scheme is `https`. Note that the [resolver](../core/resolvers) may return a port in which case the `port` setting is ignored.

Note that while using `service.namespace.svc.cluster.local` may work for Kubernetes resolvers, the preferred syntax is `service.namespace`.

## `Mapping`

An Ambassador Edge Stack `Mapping` associates REST [_resources_](#resources) with Kubernetes [_services_](#services). A resource, here, is a group of things defined by a URL prefix; a service is exactly the same as in Kubernetes.

Each mapping can also specify, among other things:

- a [_rewrite rule_](../rewrites) which modifies the URL as it's handed to the Kubernetes service;
- a [_weight_](../canary) specifying how much of the traffic for the resource will be routed using the mapping;
- a [_host_](../host) specifying a required value for the HTTP `Host` header;
- a [_shadow_](../shadowing) marker, specifying that this mapping will get a copy of traffic for the resource; and
- other [_headers_](../headers) which must appear in the HTTP request.

## Mapping Evaluation Order

Ambassador Edge Stack sorts mappings such that those that are more highly constrained are evaluated before those less highly constrained. The prefix length, the request method, and the constraint headers are all taken into account.

If absolutely necessary, you can manually set a `precedence` on the mapping (see below). In general, you should not need to use this feature unless you're using the `regex_headers` or `host_regex` matching features. If there's any question about how Ambassador Edge Stack is ordering rules, the diagnostic service is a good first place to look: the order in which mappings appear in the diagnostic service is the order in which they are evaluated.

## Optional Fallback Mapping

Ambassador Edge Stack will respond with a `404 Not Found` to any request for which no mapping exists. If desired, you can define a fallback "catch-all" mapping so all unmatched requests will be sent to an upstream service.

For example, defining a mapping with only a `/` prefix will catch all requests previously unhandled and forward them to an external service:

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  catch-all
spec:
  prefix: /
  service: https://www.getambassador.io
```

### Using `precedence`

Ambassador Edge Stack sorts mappings such that those that are more highly constrained are evaluated before those less highly constrained. The prefix length, the request method, and the constraint headers are all taken into account. These mechanisms, however, may not be sufficient to guarantee the correct ordering when regular expressions or highly complex constraints are in play.

For those situations, a `Mapping` can explicitly specify the `precedence`. A `Mapping` with no `precedence` is assumed to have a `precedence` of 0; the higher the `precedence` value, the earlier the `Mapping` is attempted.

If multiple `Mapping`s have the same `precedence`, Ambassador Edge Stack's normal sorting determines the ordering within the `precedence`; however, there is no way that Ambassador Edge Stack can ever sort a `Mapping` with a lower `precedence` ahead of one at a higher `precedence`.

### Using `tls`

In most cases, you won't need the `tls` attribute: just use a `service` with an `https://` prefix. However, note that if the `tls` attribute is present and `true`, Ambassador Edge Stack will originate TLS even if the `service` does not have the `https://` prefix.

If `tls` is present with a value that is not `true`, the value is assumed to be the name of a defined TLS context, which will determine the certificate presented to the upstream service. TLS context handling is a beta feature of Ambassador Edge Stack at present; please [contact us on Slack](https://d6e.co/slack) if you need to specify TLS origination certificates.

## Namespaces and Mappings

If `AMBASSADOR_NAMESPACE` is correctly set, Ambassador Edge Stack can map to services in other namespaces by taking advantage of Kubernetes DNS:

- `service: servicename` will route to a service in the same namespace as the Ambassador Edge Stack, and
- `service: servicename.namespace` will route to a service in a different namespace.

### Linkerd Interoperability (`add_linkerd_headers`)

When using Linkerd, requests going to an upstream service need to include the `l5d-dst-override` header to ensure that Linkerd will route them correctly. Setting `add_linkerd_headers` does this automatically, based on the `service` attribute in the `Mapping`. 

If `add_linkerd_headers` is not specified for a given `Mapping`, the default is taken from the `ambassador`[Module](../modules). The overall default is `false`: you must explicitly enable `add_linkerd_headers` for Ambassador Edge Stack to add the header for you (although you can always add it yourself with `add_request_headers`, of course).
