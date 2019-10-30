# Global Configuration

Ambassador supports a variety of global configuration options in the `ambassador` module.

## The `ambassador` Module

If present, the `ambassador` module defines system-wide configuration. This module can be applied on any Kubernetes service (the `ambassador` service itself is a common choice). **You may very well not need this module.** The defaults in the `ambassador` module are:

```yaml
apiVersion: getambassador.io/v1
kind:  Module
metadata:
  name:  ambassador
spec:
  config:
  # Put the id if you are using multiple ambassadors in the same cluster.
  # For more information: https://www.getambassador.io/reference/running/#ambassador_id.
  # ambassador_id: "<ambassador_id>"
  
  # admin_port is the port where Ambassador's Envoy will listen for
  # low-level admin requests. You should almost never need to change
  # this.
  # admin_port: 8001

  # default_label_domain and default_labels set a default domain and
  # request labels to every request for use by rate limiting. For
  # more on how to use these, see the Rate Limit reference.

  # diag_port is the port where Ambassador will listen for requests
  # to the diagnostic service.
  # diag_port: 8877

  # The diagnostic service (at /ambassador/v0/diag/) defaults on, but
  # you can disable the api route. It will remain accessible on
  # diag_port.
  # diagnostics:
  #   enabled: true

  # Should we automatically add Linkerd `l5d-dst-override` headers?
  # add_linkerd_headers: false

  # Should we enable the gRPC-http11 bridge?
  # enable_grpc_http11_bridge: false

  # Should we enable the grpc-Web protocol?
  # enable_grpc_web: false

  # Should we enable http/1.0 protocol?
  # enable_http10: false

  # Should we do IPv4 DNS lookups when contacting services? Defaults to true,
  # but can be overridden in a [`Mapping`](/reference/mappings).
  # enable_ipv4: true

  # Should we do IPv6 DNS lookups when contacting services? Defaults to false,
  # but can be overridden in a [`Mapping`](/reference/mappings).
  # enable_ipv6: false

  # liveness probe defaults on, but you can disable the api route.
  # It will remain accessible on diag_port.
  # liveness_probe:
  #   enabled: true

  # run a custom lua script on every request. see below for more details.
  # lua_scripts

  # readiness probe defaults on, but you can disable the api route.
  # It will remain accessible on diag_port.
  # readiness_probe:
  #   enabled: true

  # By default Envoy sets server_name response header to 'envoy'
  # Override it with this variable
  # server_name: envoy

  # If present, service_port will be the port Ambassador listens
  # on for microservice access. If not present, Ambassador will
  # use 8443 if TLS is configured, 8080 otherwise.
  # service_port: 8080

  # statsd configures Ambassador statistics. These values can be
  # set in the Ambassador module or in an environment variable.
  # For more information, see the [Statistics reference](/reference/statistics/#exposing-statistics-via-statsd)

  # use_proxy_protocol controls whether Envoy will honor the PROXY
  # protocol on incoming requests.
  # use_proxy_proto: false

  # envoy_log_type defines the type of log envoy will use , currently only support json or text
  # envoy_log_type: text

  # envoy_log_path defines the path of log envoy will use. By default this is standard output
  # envoy_log_path: /dev/fd/1

  # envoy_log_format defines the envoy log line format.
  # see https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log for a complete list of operators
  # envoy_log_format: "ACCESS [%START_TIME%] \"%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%\" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% %DURATION% %RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)% \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(U*    545 SER-AGENT)%\" \"%REQ(X-REQUEST-ID)%\" \"%REQ(:AUTHORITY)%\" \"%UPSTREAM_HOST%\""

  # use_remote_address controls whether Envoy will trust the remote
  # address of incoming connections or rely exclusively on the
  # X-Forwarded_For header.
  # use_remote_address: true

  # xff_num_trusted_hops controls the how Envoy sets the trusted
  # client IP address of a request. If you have a proxy in front
  # of Ambassador, Envoy will set the trusted client IP to the
  # address of that proxy. To preserve the orginal client IP address,
  # setting x_num_trusted_hops: 1 will tell Envoy to use the client IP
  # address in X-Forwarded-For. Please see the envoy documentation for
  # more information: https://www.envoyproxy.io/docs/envoy/v1.11.2/configuration/http_conn_man/headers#x-forwarded-for
  # xff_num_trusted_hops: 0

  # Ambassador lets through only the HTTP requests with
  # `X-FORWARDED-PROTO: https` header set, and redirects all the other
  # requests to HTTPS if this field is set to true. Note that `use_remote_address`
  # must be set to false for this feature to work as expected.
  # x_forwarded_proto_redirect: false

  # load_balancer sets the global load balancing type and policy that
  # Ambassador will use for all mappings, unless overridden in a
  # mapping. Defaults to round robin with Kubernetes.
  # More information at the [load balancer reference](/reference/core/load-balancer)
  # load_balancer:
  #   policy: round_robin/least_request/ring_hash/maglev
  #   ...

  # circuit_breakers sets the global circuit breaking configuration that
  # Ambassador will use for all mappings, unless overridden in a
  # mapping.
  # More information at the [circuit breaking reference](/reference/core/circuit-breaking)
  # circuit_breakers:
  #   max_connections: 2048
  #   ...

  # retry_policy lets you add resilience to your services in case of request failures by performing automatic retries.
  # retry_policy:
  #   retry_on: "5xx"
  #   ...

  # Set default CORS configuration for all mappings in the cluster. See
  # CORS syntax at https://www.getambassador.io/reference/cors.html
  # cors:
  #   origins: http://foo.example,http://bar.example
  #   methods: POST, GET, OPTIONS
  #   ...
  #   ...

  # Set the default upstream-connection idle timeout. If not set (the default), upstream
  # connections will never be closed due to idling.
  # cluster_idle_timeout_ms: 30000

  # Set which regular expression engine to use. See the "Regular Expressions" section below.
  # regex_type: safe 

  # This field controls the RE2 “program size” which is a rough estimate of how complex a compiled regex is to
  # evaluate.  A regex that has a program size greater than the configured value will fail to compile.
  # regex_max_size: 200
```

### Overriding Default Ports

By default, Ambassador listens for HTTP or HTTPS traffic on ports 8080 or 8443 respectively. This value can be overridden by setting the `service_port` in the Ambassador `Module`:

```yaml
---
apiVersion: getambassador.io/v1
kind: Module
metadata:
  name: ambassador
spec:
  config:
    service_port: 4567
```

This will configure Ambassador to listen for traffic on port 4567 instead of 8080.

### Regular Expressions (`regex_type`)

If `regex_type` is unset (the default), or is set to any value other than `unsafe`, Ambassador will use the
[RE2](https://github.com/google/re2/wiki/Syntax) regular expression engine. This engine is designed to support
most regular expressions, but keep bounds on execution time. **RE2 is the recommended regular expression engine.**

If `regex_type` is set to `unsafe`, Ambassador will use the [modified ECMAScript](https://en.cppreference.com/w/cpp/regex/ecmascript)
regular expression engine. **This is not recommended** since the modified ECMAScript engine can consume unbounded 
CPU in some cases (mostly relating to backreferences and lookahead); it is provided for backward compatibility if
necessary.

### Lua Scripts (`lua_scripts`)

Ambassador supports the ability to inline Lua scripts that get run on every request. This is useful for simple use cases that mutate requests or responses, e.g., add a custom header. Here is a sample:

```yaml
---
apiVersion: getambassador.io/v1
kind: Module
metadata:
  name: ambassador
spec:
  config:
    lua_scripts: |
      function envoy_on_response(response_handle)
        response_handle:headers():add("Lua-Scripts-Enabled", "Processed")
      end
```

For more details on the Lua API, see the [Envoy Lua filter documentation](https://www.envoyproxy.io/docs/envoy/v1.11.2/configuration/http_filters/lua_filter).

Some caveats around the embedded scripts:

* They run in-process, so any bugs in your Lua script can break every request
* They're inlined in the Ambassador YAML, so you likely won't want to write complex logic in here
* They're run on every request/response to every URL

If you need more flexible and configurable options, Ambassador Pro supports a [pluggable Filter system](/reference/filter-reference).

### Linkerd Interoperability (`add_linkerd_headers`)

When using Linkerd, requests going to an upstream service need to include the `l5d-dst-override` header to ensure that Linkerd will route them correctly. Setting `add_linkerd_headers` does this automatically; see the [Mapping](/reference/mappings) documentation for more details.

### Upstream Idle Timeout (`cluster_idle_timeout_ms`)

If set, `cluster_idle_timeout_ms` specifies the timeout (in milliseconds) after which an idle connection upstream will closed. If no `cluster_idle_timeout_ms` is specified, upstream connections will never be closed due to idling.

### gRPC HTTP/1.1 bridge (`enable_grpc_http11_bridge`)

Ambassador supports bridging HTTP/1.1 clients to backend gRPC servers. When an HTTP/1.1 connection is opened and the request content type is `application/grpc`, Ambassador will buffer the response and translate into gRPC requests. For more details on the translation process, see the [Envoy gRPC HTTP/1.1 bridge documentation](https://www.envoyproxy.io/docs/envoy/v1.11.2/configuration/http_filters/grpc_http1_bridge_filter.html). This setting can be enabled by setting `enable_grpc_http11_bridge: true`.

### gRPC-Web (`enable_grpc_web`)

gRPC-Web is a protocol built on gRPC that extends the benefits of gRPC to the browser. The gRPC-Web specification requires a server-side proxy to translate between gRPC-Web requests and gRPC backend services. Ambassador can serve as the service-side proxy for gRPC-Web when `enable_grpc_web: true` is set.

### HTTP/1.0 support (`enable_http10`)

Enable/disable handling of incoming HTTP/1.0 and HTTP 0.9 requests.

### `enable_ivp4` and `enable_ipv6`

If both IPv4 and IPv6 are enabled, Ambassador will prefer IPv6. This can have strange effects if Ambassador receives
`AAAA` records from a DNS lookup, but the underlying network of the pod doesn't actually support IPv6 traffic. For this
reason, the default for 0.50.0 is IPv4 only.

A `Mapping` can override both `enable_ipv4` and `enable_ipv6`, but if either is not stated explicitly in a `Mapping`,
the values here are used. Most Ambassador installations will probably be able to avoid overridding these setting in `Mapping`s.

### Readiness and Liveness probes (`readiness_probe` and `liveness_probe`)

The default liveness and readiness probes map `/ambassador/v0/check_alive` and `ambassador/v0/check_ready` internally to check Envoy itself. If you'd like to, you can change these to route requests to some other service. For example, to have the readiness probe map to the tour application's health check, you could do

```yaml
readiness_probe:
  service: tour
  rewrite: /backend/health
```

The liveness and readiness probe both support `prefix`, `rewrite`, and `service`, with the same meanings as for [mappings](/reference/mappings). Additionally, the `enabled` boolean may be set to `false` (as in the commented-out examples above) to disable support for the probe entirely.

**Note well** that configuring the probes in the `ambassador` module only means that Ambassador will respond to the probes. You must still configure Kubernetes to perform the checks, as shown in the Datawire-provided YAML files.

### `use_remote_address`

In Ambassador 0.50 and later, the default value for `use_remote_address` to `true`. When set to `true`, Ambassador will append to the `X-Forwarded-For` header its IP address so upstream clients of Ambassador can get the full set of IP addresses that have propagated a request.  You may also need to set `externalTrafficPolicy: Local` on your `LoadBalancer` as well to propagate the original source IP address..  See the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/v1.11.2/configuration/http_conn_man/headers.html) and the [Kubernetes documentation](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip) for more details.

**Note well** that if you need to use `X-Forwarded-Proto`, you **must** set `use_remote_address` to `false`.

### `use_proxy_proto`

Many load balancers can use the [PROXY protocol](https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt) to convey information about the connection they are proxying. In order to support this in Ambassador, you'll need to set `use_proxy_protocol` to `true`; this is not the default since the PROXY protocol is not compatible with HTTP.

### `xff_num_trusted_hops`

The value of `xff_num_trusted_hops` indicates the number of trusted proxies in front of Ambassador. The default setting is 0 which tells Envoy to use the immediate downstream connection's IP address as the trusted client address. The trusted client address is used to populate the `remote_address` field used for rate limiting and can affect which IP address Envoy will set as `X-Envoy-External-Address`.

`xff_num_trusted_hops` behavior is determined by the value of `use_remote_address` (which defaults to `true` in Ambassador).

- If `use_remote_address` is `false` and `xff_num_trusted_hops` is set to a value N that is greater than zero, the trusted client address is the (N+1)th address from the right end of XFF. (If the XFF contains fewer than N+1 addresses, Envoy falls back to using the immediate downstream connection’s source address as trusted client address.)

- If `use_remote_address` is `true` and `xff_num_trusted_hops` is set to a value N that is greater than zero, the trusted client address is the Nth address from the right end of XFF. (If the XFF contains fewer than N addresses, Envoy falls back to using the immediate downstream connection’s source address as trusted client address.)

Refer to [Envoy's documentation](https://www.envoyproxy.io/docs/envoy/v1.11.2/configuration/http_conn_man/headers#x-forwarded-for) for some detailed examples on this interaction.

**NOTE:** This value is not dynamically configurable in Envoy. A restart is required  changing the value of `xff_num_trusted_hops` for Envoy to respect the change.
