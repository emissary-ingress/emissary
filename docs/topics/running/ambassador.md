# Global Configuration

## The Ambassador `Module`

If present, the `Module` defines system-wide configuration. This module can be applied to any Kubernetes service (the `ambassador` service itself is a common choice). **You may very well not need this Module.** To apply the `Module` to an Ambassador `Service`, it MUST be named 'ambassador', otherwise it will be ignored.  To create multiple `ambassador Modules` in the same namespace, they should be put in the annotations of each separate Ambassador `Service`.

The defaults in the `Module` are:

```yaml
apiVersion: getambassador.io/v2
kind:  Module
metadata:
  name:  ambassador
spec:
# Use ambassador_id only if you are using multiple ambassadors in the same cluster.
# For more information: ../../running#ambassador_id.
  # ambassador_id: "<ambassador_id>"
  config:
# Use the following table for config fields
```

| ID | Definition &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; | Example |
| :----- | :----- | :-- |
| `add_linkerd_headers` | Should we automatically add Linkerd `l5d-dst-override` headers? | `add_linkerd_headers: false` |
| `admin_port` | The port where Ambassador's Envoy will listen for low-level admin requests. You should almost never need to change this. | `admin_port: 8001` |
| `ambassador_id` | Use only if you are using multiple ambassadors in the same cluster. [Learn more](#ambassador_id). | `ambassador_id: "<ambassador_id>"` |
| `cluster_idle_timeout_ms` | Set the default upstream-connection idle timeout. Default is 1 hour. | `cluster_idle_timeout_ms: 30000` |
| `default_label_domain  and default_labels` | Set a default domain and request labels to every request for use by rate limiting. For more on how to use these, see the [Rate Limit reference](../../using/rate-limits/rate-limits##an-example-with-global-labels-and-groups). | None |
| `defaults` | The `defaults` element allows setting system-wide defaults that will be applied to various Ambassador resources. See [using defaults](../../using/defaults) for more information. | None |
| `diagnostics.enabled` | Enable or disable the [Edge Policy Console](../../using/edge-policy-console) and `/ambassador/v0/diag/` endpoints.  See below for more details. | None |
| `enable_grpc_http11_bridge` | Should we enable the gRPC-http11 bridge? | `enable_grpc_http11_bridge: false` |
| `enable_grpc_web` | Should we enable the grpc-Web protocol? | `enable_grpc_web: false` |
| `enable_http10` | Should we enable http/1.0 protocol? | `enable_http10: false` |
| `enable_ipv4`| Should we do IPv4 DNS lookups when contacting services? Defaults to true, but can be overridden in a [`Mapping`](../../using/mappings). | `enable_ipv4: true` |
| `enable_ipv6` | Should we do IPv6 DNS lookups when contacting services? Defaults to false, but can be overridden in a [`Mapping`](../../using/mappings). | `enable_ipv6: false` |
| `envoy_log_format` | Defines the envoy log line format. See [this page](https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log/access_log) for a complete list of operators. | See [this page](https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log/usage#default-format-string) for the standard log format. |
| `envoy_log_path` | Defines the path of log envoy will use. By default this is standard output. | `envoy_log_path: /dev/fd/1` |
| `envoy_log_type` | Defines the type of log envoy will use, currently only support json or text. | `envoy_log_type: text` |
| `envoy_validation_timeout` | Defines the timeout, in seconds, for validating a new Envoy configuration. The default is 10; a value of 0 disables Envoy configuration validation. Most installations will not need to use this setting. | `envoy_validation_timeout: 30` |
| `ip_allow`       | Defines HTTP source IP address ranges to allow; all others will be denied. `ip_allow` and `ip_deny` may not both be specified. See below for more details. | None |
| `ip_deny`        | Defines HTTP source IP address ranges to deny; all others will be allowed. `ip_allow` and `ip_deny` may not both be specified. See below for more details. | None |
| `listener_idle_timeout_ms` | Controls how Envoy configures the tcp idle timeout on the http listener. Default is 1 hour. | `listener_idle_timeout_ms: 30000` |
| `lua_scripts` | Run a custom lua script on every request. see below for more details. | None |
| `grpc_stats` | Enables telemetry of gRPC calls using the "gRPC Statistics" Envoy filter. see below for more details. |  |
| `proper_case` | Should we enable upper casing for response headers? For more information, see [the Envoy docs](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/core/protocol.proto#envoy-api-msg-core-http1protocoloptions-headerkeyformat). | `proper_case: false` |
| `regex_max_size` | This field controls the RE2 "program size" which is a rough estimate of how complex a compiled regex is to evaluate. A regex that has a program size greater than the configured value will fail to compile.    | `regex_max_size: 200` |
| `regex_type` | Set which regular expression engine to use. See the "Regular Expressions" section below. | `regex_type: safe` |
| `server_name` | By default Envoy sets server_name response header to `envoy`. Override it with this variable. | `server_name: envoy` |
| `service_port` | If present, service_port will be the port Ambassador listens on for microservice access. If not present, Ambassador will use 8443 if TLS is configured, 8080 otherwise. | `service_port: 8080` |
| `statsd` | Configures Ambassador statistics. These values can be set in the Ambassador module or in an environment variable. For more information, see the [Statistics reference](../statistics#exposing-statistics-via-statsd). | None |
| `use_proxy_proto` | Controls whether Envoy will honor the PROXY protocol on incoming requests. | `use_proxy_proto: false` |
| `use_remote_address` | Controls whether Envoy will trust the remote address of incoming connections or rely exclusively on the X-Forwarded-For header. | `use_remote_address: true` |
| `use_ambassador_namespace_for_service_resolution` | Controls whether Ambassador will resolve upstream services assuming they are in the same namespace as the element referring to them, e.g. a Mapping in namespace `foo` will look for its service in namespace `foo`. If `true`, Ambassador will resolve the upstream services assuming they are in the same namespace as Ambassador, unless the service explicitly mentions a different namespace. | `use_ambassador_namespace_for_service_resolution: false` |
| `x_forwarded_proto_redirect` | Ambassador lets through only the HTTP requests with `X-FORWARDED-PROTO: https` header set, and redirects all the other requests to HTTPS if this field is set to true. Note that `use_remote_address` must be set to false for this feature to work as expected. | `x_forwarded_proto_redirect: false` |
| `xff_num_trusted_hops` | Controls the how Envoy sets the trusted client IP address of a request. If you have a proxy in front of Ambassador, Envoy will set the trusted client IP to the address of that proxy. To preserve the orginal client IP address, setting `x_num_trusted_hops: 1` will tell Envoy to use the client IP address in `X-Forwarded-For`. Please see the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.11.2/configuration/http_conn_man/headers#x-forwarded-for) for more information. | `xff_num_trusted_hops: 0` |
| `preserve_external_request_id` | Controls whether to override the `X-REQUEST-ID` header or keep it as it is coming from incomming request. Note that `preserve_external_request_id` must be set to true for this feature to work. Default value will be false. | `preserve_external_request_id: false` |

### Additional `config` Field Examples

The Ambassador `Module` can set global configurations for circuit-breaking, cors, keepalive, load-balancing, and retry policy. Setting any of these values in a `Mapping` will overwrite this behavior.

#### Circuit Breaking

`circuit_breakers` sets the global circuit breaking configuration that Ambassador will use for all mappings, unless overridden in a mapping. More information at the [circuit breaking reference](../../using/circuit-breakers).

```
circuit_breakers
  max_connections: 2048
  ...
```

#### Cross Origin Resource Sharing (CORS)

`cors` sets the default CORS configuration for all mappings in the cluster. See the [CORS syntax](../../using/cors).

```
cors:
  origins: http://foo.example,http://bar.example
  methods: POST, GET, OPTIONS
  ...
```

#### `ip_allow` and `ip_deny`

`ip_allow` specifies IP source ranges from which HTTP requests will be allowed, with all others being denied. `ip_deny` specifies IP source ranges from which HTTP requests will be denied, with all others being allowed. If both are present, it is an error: `ip_allow` will be honored and `ip_deny` will be ignored.

Both take a list of IP address ranges with a keyword specifying how to interpret the address, for example:

```yaml
ip_allow:
- peer: 127.0.0.1
- remote: 99.99.0.0/16
```

The keyword `peer` specifies that the match should happen using the IP address of the other end of the network connection carrying the request: `X-Forwarded-For` and the `PROXY` protocol are both ignored. Here, our example specifies that connections originating from the Ambassador pod itself should always be allowed.

The keyword `remote` specifies that the match should happen using the IP address of the HTTP client, taking into account `X-Forwarded-For` and the `PROXY` protocol if they are allowed (if they are not allowed, or not present, the peer address will be used instead). This permits matches to behave correctly when, for example, Ambassador is behind a layer 7 load balancer. Here, our example specifies that HTTP clients from the IP address range `99.99.0.0` - `99.99.255.255` will be allowed.

You may specify as many ranges for each kind of keyword as desired.

#### Keepalive

`keepalive` sets the global keepalive settings. Ambassador will use for all mappings unless overridden in a mapping. No default value is provided by Ambassador. More information at https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/core/address.proto#envoy-api-msg-core-tcpkeepalive.

```
keepalive:
  time: 2
  interval: 2
  probes: 100
...
```

#### Load Balancer

`load_balancer` sets the global load balancing type and policy that Ambassador will use for all mappings unless overridden in a mapping. Defaults to round-robin with Kubernetes. More information at the [load balancer reference](../load-balancer).

```
load_balancer:
  policy: round_robin/least_request/ring_hash/maglev
  ...
```

#### Retry Policy

`retry_policy` lets you add resilience to your services in case of request failures by performing automatic retries.

```
retry_policy:
  retry_on: "5xx"
  ...
```

### Linkerd Interoperability (`add_linkerd_headers`)

When using Linkerd, requests going to an upstream service need to include the `l5d-dst-override` header to ensure that Linkerd will route them correctly. Setting `add_linkerd_headers` does this automatically; see the [Mapping](../../using/mappings) documentation for more details.

### Upstream Idle Timeout (`cluster_idle_timeout_ms`)

If set, `cluster_idle_timeout_ms` specifies the timeout (in milliseconds) after which an idle connection upstream is closed. If no `cluster_idle_timeout_ms` is specified, upstream connections will never be closed due to idling.

### Defaults (`defaults`)

The `defaults` element is a dictionary of default values that will be applied to various Ambassador resources. See [using defaults](../../using/defaults) for more information.

### Diagnostics (`diagnostics`)

- Both the API Gateway and the Edge Stack provide low-level diagnostics at `/ambassador/v0/diag/`.
- The Ambassador Edge Stack also provides the higher-level Edge Policy Console at `/edge_stack/admin/`.

By default, both services are enabled:

```
diagnostics:
  enabled: true
```

Setting `diagnostics.enabled` to `false` will disable the routes for both services (they will remain accessible from inside the Ambassador pod on port 8877):

```
diagnostics:
  enabled: false
```

When configured this way, diagnostics are only available from inside the Ambassador pod(s) via `localhost` networking. You can use Kubernetes port forwarding to set up remote access temporarily:

```
kubectl port-forward -n ambassador deploy/ambassador 8877
```

If you want to expose the diagnostics page but control them via `Host` based routing, you can set `diagnostics.enabled` to false and create mappings as specified in the [FAQ](../../../about/faq#how-do-i-disable-the-default-admin-mappings).

### gRPC HTTP/1.1 bridge (`enable_grpc_http11_bridge`)

Ambassador supports bridging HTTP/1.1 clients to backend gRPC servers. When an HTTP/1.1 connection is opened and the request content type is `application/grpc`, Ambassador will buffer the response and translate into gRPC requests. For more details on the translation process, see the [Envoy gRPC HTTP/1.1 bridge documentation](https://www.envoyproxy.io/docs/envoy/v1.11.2/configuration/http_filters/grpc_http1_bridge_filter.html). This setting can be enabled by setting `enable_grpc_http11_bridge: true`.

### gRPC-Web (`enable_grpc_web`)

gRPC is a binary HTTP/2-based protocol. While this allows high performance, it is problematic for any programs that cannot speak raw HTTP/2 (such as JavaScript in a browser). gRPC-Web is a JSON and HTTP-based protocol that wraps around the plain gRPC to alleviate this problem and extend benefits of gRPC to the browser, at the cost of performance.

The gRPC-Web specification requires a server-side proxy to translate between gRPC-Web requests and gRPC backend services. Ambassador can serve as the service-side proxy for gRPC-Web when `enable_grpc_web: true` is set. Find more on the gRPC Web client [GitHub](https://github.com/grpc/grpc-web).

### HTTP/1.0 support (`enable_http10`)

Enable/disable the handling of incoming HTTP/1.0 and HTTP 0.9 requests.

### `enable_ivp4` and `enable_ipv6`

If both IPv4 and IPv6 are enabled, Ambassador Edge Stack will prefer IPv6. This can have strange effects if Ambassador Edge Stack receives `AAAA` records from a DNS lookup, but the underlying network of the pod doesn't actually support IPv6 traffic. For this reason, the default is IPv4 only.

A `Mapping` can override both `enable_ipv4` and `enable_ipv6`, but if either is not stated explicitly in a `Mapping`, the values here are used. Most Ambassador Edge Stack installations will probably be able to avoid overriding these settings in `Mapping`s.

### Envoy Access Logs (`envoy_log_format`, `envoy_log_path`, and `envoy_log_type`)

Ambassador allows for two types of logging output, json and text (`envoy_log_type`). These logs can be formatted using Envoy [operators](https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log/usage#command-operators) to display specific information about an incoming request. For example, a log of type `json` could use the following to show only the protocol and duration of a request:

```
envoy_log_format:
  {
    "protocol": "%PROTOCOL%",
    "duration": "%DURATION%"
  }
```

Additionally, a file path can be specified to output logs instead of standard out using `envoy_log_path`.

### Listener Idle Timeout (`listener_idle_timeout_ms`)

Controls how Envoy configures the tcp idle timeout on the http listener. Default is no timeout (TCP connection may remain idle indefinitely). This is useful if you have proxies and/or firewalls in front of Ambassador and need to control how Ambassador initiates closing an idle TCP connection. Please see the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.12.2/api-v2/api/v2/core/protocol.proto#envoy-api-msg-core-httpprotocoloptions) for more information.

### Readiness and Liveness probes (`readiness_probe` and `liveness_probe`)

The default liveness and readiness probes map `/ambassador/v0/check_alive` and `ambassador/v0/check_ready` internally to check Envoy itself. If you'd like to, you can change these to route requests to some other service. For example, to have the readiness probe map to the quote application's health check, you could do

```yaml
readiness_probe:
  enabled: true
  service: quote
  rewrite: /backend/health
```

The liveness and readiness probes both support `prefix`, `rewrite`, and `service`, with the same meanings as for [mappings](../../using/mappings). Additionally, the `enabled` boolean may be set to `false` to disable API support for the probe.  It will, however, remain accessible on port 8877.

### Lua Scripts (`lua_scripts`)

Ambassador Edge Stack supports the ability to inline Lua scripts that get run on every request. This is useful for simple use cases that mutate requests or responses, e.g., add a custom header. Here is a sample:

```yaml
lua_scripts: |
  function envoy_on_response(response_handle)
    response_handle:headers():add("Lua-Scripts-Enabled", "Processed")
  end
```

For more details on the Lua API, see the [Envoy Lua filter documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/lua_filter.html).

**Some caveats around the embedded scripts:**

* They run in-process, so any bugs in your Lua script can break every request
* They're inlined in the Ambassador Edge Stack YAML, so you likely won't want to write complex logic in here
* They're run on every request/response to every URL

If you need more flexible and configurable options, Ambassador Edge Stack supports a [pluggable Filter system](../../using/filters/).

### gRPC Statistics (`grpc_stats`)

Use the Envoy filter to enable telemetry of gRPC calls. [gRPC Statistics Filter](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/grpc_stats_filter)

Supported parameters:
* `all_methods`
* `services`
* `upstream_stats`

Available metrics:
* `envoy_cluster_grpc_<service>_<status_code>`
* `envoy_cluster_grpc_<service>_request_message_count`
* `envoy_cluster_grpc_<service>_response_message_count`
* `envoy_cluster_grpc_<service>_success`
* `envoy_cluster_grpc_<service>_total`
* `envoy_cluster_grpc_upstream_<stats>` - **only when `upstream_stats: true`**

Please note that `<service>` will only be present if `all_methods` is set or the service and the method are present under `services`.
If `all_methods` is false or the method is not on the list, the available metrics will be in the format
`envoy_cluster_grpc_<stats>`.

##### all_methods
If set to true, emit stats for all service/method names.
If set to false, emit stats for all service/message types to the same stats without including the service/method in the name.
**This option is only safe if all clients are trusted. If this option is enabled with untrusted clients, the clients could cause unbounded growth in the number
of stats in Envoy, using unbounded memory and potentially slowing down stats pipelines.**

##### services
If set, specifies an allowlist of service/methods that will have individual stats emitted for them. Any call that does not match the allowlist will be
counted in a stat with no method specifier (generic metric).

**If both `all_methods` and `services` are present, `all_methods` will be ignored.**

##### upstream_stats
If true, the filter will gather a histogram for the request time of the upstream.

```yaml
---
apiVersion: getambassador.io/v2
kind:  Module
metadata:
  name: ambassador
spec:
  config:
    grpc_stats:
      upstream_stats: true
      services:
        - name: <package>.<service>
          method_names: [<method>]
```

### Header Case (`proper_case`)

To enable upper casing of response headers by proper casing words: the first character and any character following a special character will be capitalized if it’s an alpha character. For example, “content-type” becomes “Content-Type”. Please see the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/api-v2/api/v2/core/protocol.proto#envoy-api-msg-core-http1protocoloptions-headerkeyformat)

### Regular Expressions (`regex_type`)

If `regex_type` is unset (the default), or is set to any value other than `unsafe`, Ambassador Edge Stack will use the [RE2](https://github.com/google/re2/wiki/Syntax) regular expression engine. This engine is designed to support most regular expressions, but keep bounds on execution time. **RE2 is the recommended regular expression engine.**

If `regex_type` is set to `unsafe`, Ambassador Edge Stack will use the [modified ECMAScript](https://en.cppreference.com/w/cpp/regex/ecmascript) regular expression engine. **This is not recommended** since the modified ECMAScript engine can consume unbounded CPU in some cases (mostly relating to backreferences and lookahead); it is provided for backward compatibility if necessary.

### Overriding Default Ports (`service_port`)

By default, Ambassador Edge Stack listens for HTTP or HTTPS traffic on ports 8080 or 8443 respectively. This value can be overridden by setting the `service_port` in the Ambassador `Module`:

```yaml
service_port: 4567
```

This will configure Ambassador Edge Stack to listen for traffic on port 4567 instead of 8080.

### Allow Proxy Protocol (`use_proxy_proto`)

Many load balancers can use the [PROXY protocol](https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt) to convey information about the connection they are proxying. In order to support this in Ambassador Edge Stack, you'll need to set `use_proxy_protocol` to `true`; this is not the default since the PROXY protocol is not compatible with HTTP.

### Trust Downstream Client IP (`use_remote_address`)

In Ambassador 0.50 and later, the default value for `use_remote_address` is set to `true`. When set to `true`, Ambassador Edge Stack will append to the `X-Forwarded-For` header its IP address so upstream clients of Ambassador Edge Stack can get the full set of IP addresses that have propagated a request.  You may also need to set `externalTrafficPolicy: Local` on your `LoadBalancer` as well to propagate the original source IP address.  See the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers) and the [Kubernetes documentation](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip) for more details.

  **Note well** that if you need to use `x_forwarded_proto_redirect`, you **must** set `use_remote_address` to `false`. Otherwise, unexpected behaviour can occur.

### `X-Forwarded-For` Trusted Hops (`xff_num_trusted_hops`)

The value of `xff_num_trusted_hops` indicates the number of trusted proxies in front of Ambassador Edge Stack. The default setting is 0 which tells Envoy to use the immediate downstream connection's IP address as the trusted client address. The trusted client address is used to populate the `remote_address` field used for rate limiting and can affect which IP address Envoy will set as `X-Envoy-External-Address`.

`xff_num_trusted_hops` behavior is determined by the value of `use_remote_address` (which defaults to `true` in Ambassador Edge Stack).

* If `use_remote_address` is `false` and `xff_num_trusted_hops` is set to a value N that is greater than zero, the trusted client address is the (N+1)th address from the right end of XFF. (If the XFF contains fewer than N+1 addresses, Envoy falls back to using the immediate downstream connection’s source address as a trusted client address.)

* If `use_remote_address` is `true` and `xff_num_trusted_hops` is set to a value N that is greater than zero, the trusted client address is the Nth address from the right end of XFF. (If the XFF contains fewer than N addresses, Envoy falls back to using the immediate downstream connection’s source address as a trusted client address.)

Refer to [Envoy's documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_conn_man/headers.html#x-forwarded-for) for some detailed examples of this interaction.

**NOTE:** This value is not dynamically configurable in Envoy. A restart is required changing the value of `xff_num_trusted_hops` for Envoy to respect the change.
