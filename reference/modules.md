# Core Configuration: Modules

Modules let you enable and configure special behaviors for Ambassador, in ways that may apply to Ambassador as a whole or which may apply only to some mappings. The actual configuration possible for a given module depends on the module.

## The `ambassador` Module

If present, the `ambassador` module defines system-wide configuration. **You may very well not need this module.** The defaults in the `ambassador` module are, roughly:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  # If present, service_port will be the port Ambassador listens
  # on for microservice access. If not present, Ambassador will
  # use 443 if TLS is configured, 80 otherwise. In future releases
  # of Ambassador, this will change to 8080 when we run Ambassador
  # as non-root by default.
  # service_port: 80

  # diag_port is the port where Ambassador will listen for requests
  # to the diagnostic service.
  # diag_port: 8877

  # admin_port is the port where Ambassador's Envoy will listen for
  # low-level admin requests. You should almost never need to change
  # this.
  # admin_port: 8001

  # liveness probe defaults on, but you can disable the api route.
  # It will remain accessible on diag_port.
  # liveness_probe:
  #   enabled: true

  # readiness probe defaults on, but you can disable the api route.
  # It will remain accessible on diag_port.
  # readiness_probe:
  #   enabled: true

  # The diagnostic service (at /ambassador/v0/diag/) defaults on, but
  # you can disable the api route. It will remain accessible on 
  # diag_port.
  # diagnostics:
  #   enabled: true

  # use_proxy_protocol controls whether Envoy will honor the PROXY
  # protocol on incoming requests.
  # use_proxy_proto: false

  # use_remote_address controls whether Envoy will trust the remote
  # address of incoming connections or rely exclusively on the 
  # X-Forwarded_For header. 
  #
  # The current default is not to include any use_remote_address setting,
  # but THAT IS LIKELY TO CHANGE SOON.
  # use_remote_address: false

  # Ambassador lets through only the HTTP requests with
  # `X-FORWARDED-PROTO: https` header set, and redirects all the other
  # requests to HTTPS if this field is set to true.
  # x_forwarded_proto_redirect: false

  # Set default CORS configuration for all mappings in the cluster. See CORS syntax at https://www.getambassador.io/reference/cors.html
  # cors:
  #   origins: http://foo.example,http://bar.example
  #   methods: POST, GET, OPTIONS
  #   ...
  #   ...

```

### `use_remote_address`

**Ambassador is very likely to change to default `use_remote_address` to `true`
very soon**. At present, `use_remote_address` still defaults to `false`; consider setting it to `true` if your application wants to have the incoming source address available. See the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http_conn_man/headers.html) for more information here.

### `use_proxy_proto`

Many load balancers can use the [PROXY protocol](https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt) to convey information about the connection they are proxying. In order to support this in Ambassador, you'll need to set `use_proxy_protocol` to `true`; this is not the default since the PROXY protocol is not compatible with HTTP.

### Probes

The default liveness and readiness probes map `/ambassador/v0/check_alive` and `ambassador/v0/check_ready` internally to check Envoy itself. If you'd like to, you can change these to route requests to some other service. For example, to have the readiness probe map to the Quote of the Moment's health check, you could do

```yaml
readiness_probe:
  service: qotm
  rewrite: /health
```

The liveness and readiness probe both support `prefix`, `rewrite`, and `service`, with the same meanings as for [mappings](#mappings). Additionally, the `enabled` boolean may be set to `false` (as in the commented-out examples above) to disable support for the probe entirely.

**Note well** that configuring the probes in the `ambassador` module only means that Ambassador will respond to the probes. You must still configure Kubernetes to perform the checks, as shown in the Datawire-provided YAML files.

## The `authentication` Module

The `authentication` module is now deprecated. Use the [AuthService](services/auth-service) manifest type instead.
