# Core Configuration: Modules

Modules let you enable and configure special behaviors for Ambassador, in ways that may apply to Ambassador as a whole or which may apply only to some mappings. The actual configuration possible for a given module depends on the module.

## The `ambassador` Module

If present, the `ambassador` module defines system-wide configuration. **You may very well not need this module.** The defaults in the `ambassador` module are:

```yaml
---
apiVersion: ambassador/v1
kind:  Module
name:  ambassador
config:
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

  # readiness probe defaults on, but you can disable the api route.
  # It will remain accessible on diag_port.
  # readiness_probe:
  #   enabled: true

  # If present, service_port will be the port Ambassador listens
  # on for microservice access. If not present, Ambassador will
  # use 443 if TLS is configured, 80 otherwise. In future releases
  # of Ambassador, this will change to 8080 when we run Ambassador
  # as non-root by default.
  # service_port: 80

  # statsd configures Ambassador statistics. These values can be
  # set in the Ambassador module or in an environment variable.
  # For more information, see the [Statistics reference](/reference/statistics/#exposing-statistics-via-statsd)

  # use_proxy_protocol controls whether Envoy will honor the PROXY
  # protocol on incoming requests.
  # use_proxy_proto: false

  # use_remote_address controls whether Envoy will trust the remote
  # address of incoming connections or rely exclusively on the 
  # X-Forwarded_For header. 
  # use_remote_address: true

  # Ambassador lets through only the HTTP requests with
  # `X-FORWARDED-PROTO: https` header set, and redirects all the other
  # requests to HTTPS if this field is set to true.
  # x_forwarded_proto_redirect: false

  # Set default CORS configuration for all mappings in the cluster. See 
  # CORS syntax at https://www.getambassador.io/reference/cors.html
  # cors:
  #   origins: http://foo.example,http://bar.example
  #   methods: POST, GET, OPTIONS
  #   ...
  #   ...
```

### `enable_ivp4` and `enable_ipv6`

If both IPv4 and IPv6 are enabled, Ambassador will prefer IPv6. This can have strange effects if Ambassador receives
`AAAA` records from a DNS lookup, but the underlying network of the pod doesn't actually support IPv6 traffic. For this
reason, the default for 0.50.0 is IPv4 only.

A `Mapping` can override both `enable_ipv4` and `enable_ipv6`, but if either is not stated explicitly in a `Mapping`,
the values here are used. Most Ambassador installations will probably be able to avoid overridding these setting in `Mapping`s.

### `use_remote_address`

In Ambassador 0.50 and later, the default value for `use_remote_address` to `true`. When set to `true`, Ambassador will append to the `X-Forwarded-For` header its IP address so upstream clients of Ambassador can get the full set of IP addresses that have propagated a request.  You may also need to set `externalTrafficPolicy: Local` on your `LoadBalancer` as well to propagate the original source IP address..  See the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http_conn_man/headers.html) and the [Kubernetes documentation](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip) for more details.

### `use_proxy_proto`

Many load balancers can use the [PROXY protocol](https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt) to convey information about the connection they are proxying. In order to support this in Ambassador, you'll need to set `use_proxy_protocol` to `true`; this is not the default since the PROXY protocol is not compatible with HTTP.

### Probes

The default liveness and readiness probes map `/ambassador/v0/check_alive` and `ambassador/v0/check_ready` internally to check Envoy itself. If you'd like to, you can change these to route requests to some other service. For example, to have the readiness probe map to the Quote of the Moment's health check, you could do

```yaml
readiness_probe:
  service: qotm
  rewrite: /health
```

The liveness and readiness probe both support `prefix`, `rewrite`, and `service`, with the same meanings as for [mappings](/reference/mappings). Additionally, the `enabled` boolean may be set to `false` (as in the commented-out examples above) to disable support for the probe entirely.

**Note well** that configuring the probes in the `ambassador` module only means that Ambassador will respond to the probes. You must still configure Kubernetes to perform the checks, as shown in the Datawire-provided YAML files.

## The `authentication` Module

The `authentication` module is now deprecated. Use the [AuthService](/reference/services/auth-service) manifest type instead.
