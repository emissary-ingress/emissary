## Modules

Modules let you enable and configure special behaviors for Ambassador, in ways that may apply to Ambassador as a whole or which may apply only to some mappings. The actual configuration possible for a given module depends on the module.

### The `ambassador` Module

If present, the `ambassador` module defines system-wide configuration. **You may very well not need this module.** The defaults in the `ambassador` module are, roughly:

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
```

#### `use_remote_address`

**Ambassador is very likely to change to default `use_remote_address` to `true`
very soon**. At present, `use_remote_address` still defaults to `false`; consider setting it to `true` if your application wants to have the incoming source address available. See the [Envoy documentation](https://www.envoyproxy.io/docs/envoy/latest/configuration/http_conn_man/headers.html) for more information here.

#### `use_proxy_proto`

Many load balancers can use the [PROXY protocol](https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt) to convey information about the connection they are proxying. In order to support this in Ambassador, you'll need to set `use_proxy_protocol` to `true`; this is not the default since the PROXY protocol is not compatible with HTTP.

#### Probes

The default liveness and readiness probes map `/ambassador/v0/check_alive` and `ambassador/v0/check_ready` internally to check Envoy itself. If you'd like to, you can change these to route requests to some other service. For example, to have the readiness probe map to the Quote of the Moment's health check, you could do

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

TLS configuration is examined in more detail in the documentation on [TLS termination](/user-guide/tls-termination.md) and [TLS client certificate authentication](/reference/auth-tls-certs).

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
