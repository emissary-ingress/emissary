# Routing TCP Connections

In addition to managing HTTP, GRPC, and Websockets at layer 7, Ambassador Edge Stack can also manage TCP connections at layer 4. The core abstraction used to support TCP connections is the `TCPMapping`.

## TCPMapping

An Ambassador Edge Stack `TCPMapping` associates TCP connections with Kubernetes _services_. Cleartext TCP connections are defined by destination IP address and/or destination TCP port; TLS TCP connections can also be defined by the hostname presented using SNI. A service is exactly the same as in Kubernetes.

## TCPMapping Configuration

Ambassador Edge Stack supports a number of attributes to configure and customize mappings.

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| `address`         | (optional) the IP address on which Ambassador Edge Stack should listen for connections for this Mapping -- if not present, Ambassador Edge Stack will listen on all addresses )
| `port`            | (required) the TCP port on which Ambassador Edge Stack should listen for connections for this Mapping |
| `idle_timeout_ms` | (optional) the timeout, in milliseconds, after which the connection will be terminated if no traffic is seen -- if not present, no timeout is applied |
| `enable_ipv4` | (optional) if true, enables IPv4 DNS lookups for this mapping's service (the default is set by the [`ambassador Module`](../../running/ambassador)) |
| `enable_ipv6` | (optional) if true, enables IPv6 DNS lookups for this mapping's service (the default is set by the [`ambassador Module`](../../running/ambassador)) |

If both `enable_ipv4` and `enable_ipv6` are set, Ambassador Edge Stack will prefer IPv6 to IPv4. See the [`ambassador Module`](../../running/ambassador) documentation for more information.

Ambassador Edge Stack can manage TCP connections using TLS:

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| [`host`](../headers/host) | (optional) enables TLS _termination_ and specifies the hostname that must be presented using SNI for this `TCPMapping` to match -- **FORCES TLS TERMINATION**, see below |
| [`tls`](#using-tls)       | (optional) enables TLS _origination_, and may specify the name of a `TLSContext` that will determine the certificate to offer to the upstream service | 

Ambassador Edge Stack supports multiple deployment patterns for your services. These patterns are designed to let you safely release new versions of your service while minimizing its impact on production users.

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| [`weight`](../canary)        | (optional) specifies the (integer) percentage of traffic for this resource that will be routed using this mapping |

The name of the mapping must be unique.

### `TCPMapping` and TLS Termination

**The `host` attribute of a `TCPMapping` determines whether Ambassador Edge Stack will terminate TLS when a client connects.** The `tls` attribute determines whether Ambassador Edge Stack will _originate_ TLS. The two are independent.

This leaves four cases:

#### Neither `host` Nor `tls` are Set

In this case, Ambassador Edge Stack simply proxies bytes between the client and the upstream. TLS may or may not be involved, and Ambassador Edge Stack doesn't care. You should specify the port to use for the upstream connection; if you don't, Ambassador Edge Stack will guess port 80.

Examples:

```yaml
---
apiVersion: getambassador.io/v2
kind:  TCPMapping
metadata:
  name:  ssh
spec:
  port: 2222
  service: upstream:22
```

could be used to relay an SSH connection on port 2222, or


```yaml
apiVersion: getambassador.io/v2
kind:  TCPMapping
metadata:
  name:  cockroach
spec:
  port: 26257
  service: cockroach:26257
```

could proxy a CockroachDB connection.

#### `host` is Set, But `tls` is Not

In this case, Ambassador Edge Stack will terminate the TLS connection, require that the host offered with SNI match the `host` attribute, and then make a **cleartext** connection to the upstream host. You should specify the port to use for the upstream connection; if you don't, Ambassador Edge Stack will guess port **80**.

This can be useful for doing host-based TLS proxying of arbitrary protocols, allowing the upstream to not have to care about TLS.

Note that this case **requires** that you have created a termination `TLSContext` that has a `host` that matches the `host` in the `TCPMapping`. (This is the same rule as TLS termination with SNI in an HTTP `Mapping`.)

Example:

```yaml
---
apiVersion: getambassador.io/v2
kind:  TLSContext
metadata:
  name:  my-context
spec:
  hosts:
  - my-host-1
  - my-host-2
  secret: supersecret
---
apiVersion: getambassador.io/v2
kind:  TCPMapping
metadata:
  name:  my-host-1
spec:
  port: 2222
  host: my-host-1
  service: upstream-host-1:9999
---
apiVersion: getambassador.io/v2
kind:  TCPMapping
metadata:
  name:  my-host-2
spec:
  port: 2222
  host: my-host-2
  service: upstream-host-2:9999
```

The example above will accept a TLS connection with SNI on port 2222. If the client requests SNI host `my-host-1`, the decrypted traffic will be relayed to `upstream-host-1`, port 9999. If the client requests SNI host `my-host-2`, the decrypted traffic will be relayed to `upstream-host-2`, port 9999. Any other SNI host will cause the TLS handshake to fail.

#### `host` and `tls` are Both Set

In this case, Ambassador Edge Stack will terminate the incoming TLS connection, require that the host offered with SNI match the `host` attribute, and then make a **TLS** connection to the upstream host. You should specify the port to use for the upstream connection; if you don't, Ambassador Edge Stack will guess port **443**.

This is useful for doing host routing while maintaining end-to-end encryption.

Note that this case **requires** that you have created a termination `TLSContext` that has a `host` that matches the `host` in the `TCPMapping`. (This is the same rule as TLS termination with SNI in an HTTP `Mapping`.)

Example:

```yaml
---
apiVersion: getambassador.io/v2
kind:  TLSContext
metadata:
  name:  my-context
spec:
  hosts:
  - my-host-1
  - my-host-2
  secret: supersecret
---
apiVersion: getambassador.io/v2
kind:  TLSContext
metadata:
  name:  origination-context
spec:
  secret: othersecret
---
apiVersion: getambassador.io/v2
kind:  TCPMapping
metadata:
  name:  test-1
spec:
  port: 2222
  host: my-host-1
  tls: true
  service: upstream-host-1:9999
---
apiVersion: getambassador.io/v2
kind:  TCPMapping
metadata:
  name:  test-2
spec:
  port: 2222
  host: my-host-2
  tls: origination-context
  service: upstream-host-2:9999
```

The example above will accept a TLS connection with SNI on port 2222.

If the client requests SNI host `my-host-1`, the traffic will be relayed over a TLS connection to `upstream-host-1`, port 9999. No client certificate will be offered for this connection.

If the client requests SNI host `my-host-2`, the decrypted traffic will be relayed to `upstream-host-2`, port 9999. The client certificate from `origination-context` will be offered for this connection.

Any other SNI host will cause the TLS handshake to fail.

#### Host is Not Set, But `tls` is

Here, Ambassador Edge Stack will accept the connection **without terminating TLS**, then relay traffic over a **TLS** connection upstream. This is probably useful only to accept unencrypted traffic and force it to be encrypted when it leaves Ambassador Edge Stack.

Example:

```yaml
---
apiVersion: getambassador.io/v2
kind:  TLSContext
metadata:
  name:  origination-context
spec:
  secret: othersecret
---
apiVersion: getambassador.io/v2
kind:  TCPMapping
metadata:
  name:  test
spec:
  port: 2222
  tls: true
  service: upstream-host:9999
```

The example above will accept **any** connection to port 2222 and relay it over a **TLS** connection to `upstream-host` port 9999. No client certificate will be offered.

#### Summary

- To get a `TCPMapping` to terminate TLS, configure Ambassador Edge Stack with a termination `TLSContext` and list a `host` in the `TCPMapping`.

- To get a `TCPMapping` to originate TLS, use the `tls` attribute in the `TCPMapping`.

- You can mix and match as long as you think about how the protocols interact.

#### Required Attributes for `TCPMapping`s

- `name` is a string identifying the `Mapping` (e.g. in diagnostics)
- `port` is an integer specifying which port to listen on for connections
- `service` is the name of the service handling the resource; must include the namespace (e.g. `myservice.othernamespace`) if the service is in a different namespace than Ambassador Edge Stack

Note that the `service` in a `TCPMapping` should include a port number, and must not include a scheme.

## Namespaces and Mappings

Given that `AMBASSADOR_NAMESPACE` is correctly set, Ambassador Edge Stack can map to services in other namespaces by taking advantage of Kubernetes DNS:

- `service: servicename` will route to a service in the same namespace as the Ambassador Edge Stack, and
- `service: servicename.namespace` will route to a service in a different namespace.
