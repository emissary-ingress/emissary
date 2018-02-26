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

TLS configuration is examined in more detail in the documentation on [TLS termination](../how-to/tls-termination.md) and [TLS client certificate authentication](/reference/auth-tls-certs).

### The `authentication` Module

The `authentication` module is now deprecated. Use the `AuthService` manifest type instead.

## AuthService

An `AuthService` manifest configures Ambassador to use an external service to check authentication and authorization for incoming requests:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
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

### Modifying Ambassador's Underlying Envoy Configuration

Ambassador uses Envoy for the heavy lifting of proxying.

If you wish to use Envoy features that aren't (yet) exposed by Ambassador, you can use your own custom config template. To do this, create a templated `envoy.json` file using the Jinja2 template language. Then, use this template as the value for the key `envoy.j2` in your ConfigMap. This will then replace the [default template](https://github.com/datawire/ambassador/tree/master/ambassador/templates).

Please [contact us on Gitter](https://gitter.im/datawire/ambassador) for more information if this seems necessary for a given use case (or better yet, submit a PR!) so that we can expose this in the future.
