# Routing TCP connections

In addition to managing HTTP, GRPC, and Websockets at layer 7, Ambassador can also manage TCP connections at layer 4. The core abstraction used to support TCP connections is the `TCPMapping`.

## TCPMapping

An Ambassador `TCPMapping` associates TCP connections with Kubernetes [_services_](#services). Cleartext TCP connections are defined by destination IP address and/or destination TCP port; TLS TCP connections can also by defined by the hostname presented using SNI. A service is exactly the same as in Kubernetes.

## TCPMapping Configuration

Ambassador supports a number of attributes to configure and customize mappings.

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| `address`         | (optional) the IP address on which Ambassador should listen for connections for this Mapping -- if not present, Ambassador will listen on all addresses )
| `port`            | (required) the TCP port on which Ambassador should listen for connections for this Mapping |
| `idle_timeout_ms` | (optional) the timeout, in milliseconds, after which the connection will be terminated if no traffic is seen -- if not present, no timeout is applied |
| `enable_ipv4` | (optional) if true, enables IPv4 DNS lookups for this mapping's service (the default is set by the [Ambassador module](/reference/modules)) |
| `enable_ipv6` | (optional) if true, enables IPv6 DNS lookups for this mapping's service (the default is set by the [Ambassador module](/reference/modules)) |

If both `enable_ipv4` and `enable_ipv6` are set, Ambassador will prefer IPv6 to IPv4. See the [Ambassador module](/reference/modules) documentation for more information.

Ambassador can manage TCP connections using TLS:

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| [`tls`](#using-tls)       | (optional) enables TLS, and specifies the name of the `TLSContext` which will determine the certificate to offer when a client connects |
| [`host`](/reference/host) | (optional) specifies the hostname that must be presented using SNI for this `TCPMapping` to match -- **ONLY AVAILABLE WHEN USING TLS** |

Note that **Ambassador will terminate the TLS connection** when using TLS with a `TCPMapping`. You can use a `TCPMapping` to pass a TLS connection to an upstream service without terminating it, but `host` matching will not be available.

Ambassador supports multiple deployment patterns for your services. These patterns are designed to let you safely release new versions of your service, while minimizing its impact on production users.

| Attribute                 | Description               |
| :------------------------ | :------------------------ |
| [`weight`](/reference/canary)        | (optional) specifies the (integer) percentage of traffic for this resource that will be routed using this mapping |

The name of the mapping must be unique.

## Example `TCPMapping`s

`TCPMapping` definitions are fairly straightforward. Here's an example that simply relays any connection on port 2222 onward to a Kubernetes service:

```yaml
---
apiVersion: ambassador/v1
kind: TCPMapping
name: qotm_mapping
port: 2222
service: qotm
```

Here's the same service, with Ambassador terminating TLS:

```yaml
---
apiVersion: ambassador/v1
kind: TCPMapping
name: quote_mapping
port: 2222
tls: some-context
service: qotm
```

Note that in this example, Ambassador will contact the `qotm` service using cleartext HTTP, not HTTPS. You can explicitly ask Ambassador to use HTTPS:

```yaml
---
apiVersion: ambassador/v1
kind: TCPMapping
name: quote_mapping
port: 2222
tls: some-context
service: https://qotm
```

but realize that Ambassador will decrypt the incoming traffic, then re-encrypt it.

To use SNI as part of the `TCPMapping` match:

```yaml
---
apiVersion: ambassador/v1
kind: TCPMapping
name: quote_mapping_1
port: 2222
tls: some-context
host: host1
service: https://qotm1
---
apiVersion: ambassador/v1
kind: TCPMapping
name: quote_mapping_2
port: 2222
tls: some-context
host: host2
service: https://qotm2
```

Required attributes for mappings:

- `name` is a string identifying the `Mapping` (e.g. in diagnostics)
- `port` is an integer specifying which port to listen on for connections
- `service` is the name of the [service](#services) handling the resource; must include the namespace (e.g. `myservice.othernamespace`) if the service is in a different namespace than Ambassador

## Namespaces and Mappings

Given that `AMBASSADOR_NAMESPACE` is correctly set, Ambassador can map to services in other namespaces by taking advantage of Kubernetes DNS:

- `service: servicename` will route to a service in the same namespace as the Ambassador, and
- `service: servicename.namespace` will route to a service in a different namespace.
