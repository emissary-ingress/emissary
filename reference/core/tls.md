# Transport Layer Security (TLS)

The Ambassador Edge Stack's robust TLS support exposes configuration options for different TLS use cases including:

- [Client Certificate Validation](../../tls/client-cert-validation)
- [HTTP -> HTTPS Redirection](../../tls/cleartext-redirection)
- [Mutual TLS](../../tls/mtls)
- [Server Name Indication (SNI)](../../../user-guide/sni)
- [TLS Origination](../../tls/origination)

## `Host`

The `Host` resource has been added to the Ambassador API Gateway and Ambassador Edge Stack in version 1.0.0.

The `Host` resource simplifies TLS by directly connecting how Ambassador performs TLS termination and the domains it serves. With the Ambassador Edge Stack, the `Host` can issue and manage certificates automatically using the ACME protocol. 

For both the Ambassador Edge Stack and API Gateway, it can read a certificate from a Kubernetes secret and use that certificate to terminate TLS on a domain.

The following will configure Ambassador to grab a certificate from a secret named `host-secret` and use that secret for terminating TLS on the `host.example.com` domain:

```yaml
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: example-host
spec:
  hostname: host.example.com
  acmeProvider:
    authority: none
  tlsSecret:
    name: host-secret
```

### `Host`and `TLSContext`

The `Host` will configure basic TLS termination settings in Ambassador. If you need more advanced TLS options on a domain, such as setting the minimum TLS version, you can create a [`TLSContext`](#tlscontext) with the name `{{NAME_OF_HOST}}-context` to configure more advanced TLS options.

For example, to enforce a minimum TLS version on the `Host` above, create a `TLSContext` named `example-host-context` with the following configuration:

```yaml
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: example-host-context
spec:
  hosts:
  - host.example.com
  secret: host-secret
  min_tls_version: v1.2
```

## TLSContext

You control TLS configuration in Ambassador Edge Stack using `TLSContext` resources. Multiple `TLSContext`s can be defined in your cluster and can be used for any combination of TLS use cases.

A full schema of the `TLSContext` can be found below with descriptions of the different configuration options.

```yaml
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: tls-context-1
spec:
  # 'hosts' defines the hosts for which this TLSContext is relevant.
  # It ties into SNI. A TLSContext without "hosts" is useful only for 
  # originating TLS. 
  # type: array of strings
  #
  # hosts: []

  # 'secret' defines a Kubernetes Secret that contains the TLS certificate we
  # use for origination or termination. If not specified, Ambassador will look
  # at the value of cert_chain_file and private_key_file.
  # type: string
  #
  # secret: None

  # 'ca_secret' defines a Kubernetes Secret that contains the TLS certificate we
  # use for verifying incoming TLS client certificates.
  # type: string
  #
  # ca_secret: None

  # Tells Ambassador whether to interpret a "." in the secret name as a "." or 
  # a namespace identifier.
  # type: boolean
  #
  # secret_namespacing: true

  # If you set 'redirect_cleartext_from' to a port number, HTTP traffic
  # to that port will be redirected to HTTPS traffic. Make sure that the
  # port number you specify matches the port on which Ambassador is
  # listening!
  # redirect_cleartext_from: 8080

  # 'cert_required' can be set to true to _require_ TLS client certificate
  # authentication.
  # type: boolean
  #
  # cert_required: false

  # 'alpn_protocols' is used to enable the TLS ALPN protocol. It is required
  # if you want to do GRPC over TLS; typically it will be set to "h2" for that
  # case.
  # type: string (comma-separated list)
  #
  # alpn_protocols: None

  # 'min_tls_version' sets the minimum acceptable TLS version: v1.0, v1.1,
  # v1.2, or v1.3. It defaults to v1.0.
  # min_tls_version: v1.0

  # 'max_tls_version' sets the maximum acceptable TLS version: v1.0, v1.1,
  # v1.2, or v1.3. It defaults to v1.3.
  # max_tls_version: v1.3

  # Tells Ambassador to load TLS certificates from a file in its container.
  # type: string
  #
  # cert_chain_file: None
  # private_key_file: None
  # cacert_chain_file: None
```

### `alpn_protocols`

The `alpn_protocols` setting configures the TLS ALPN protocol. To use gRPC over TLS, set `alpn_protocols: h2`. If you need to support HTTP/2 upgrade from HTTP/1, set `alpn_protocols: h2,http/1.1` in the configuration.

#### HTTP/2 Support

The `alpn_protocols` setting is also required for HTTP/2 support.

```yaml
apiVersion: getambassador.io/v2
kind:  TLSContext
metadata:
  name:  tls
spec:
  secret: ambassador-certs
  hosts: ["*"]
  alpn_protocols: h2[, http/1.1]
```
Without setting alpn_protocols as shown above, HTTP2 will not be available via negotiation and will have to be explicitly requested by the client.

If you leave off http/1.1, only HTTP2 connections will be supported.

### TLS Parameters

The `min_tls_version` setting configures the minimum TLS protocol version that Ambassador Edge Stack will use to establish a secure connection. When a client using a lower version attempts to connect to the server, the handshake will result in the following error: `tls: protocol version not supported`.

The `max_tls_version` setting configures the maximum TLS protocol version that Ambassador Edge Stack will use to establish a secure connection. When a client using a higher version attempts to connect to the server, the handshake will result in the following error: `tls: server selected unsupported protocol version`.

The `cipher_suites` setting configures the supported [cipher list](https://commondatastorage.googleapis.com/chromium-boringssl-docs/ssl.h.html#Cipher-suite-configuration) when negotiating a TLS 1.0-1.2 connection. This setting has no effect when negotiating a TLS 1.3 connection.  When a client does not support a matching cipher a handshake error will result.

The `ecdh_curves` setting configures the supported ECDH curves when negotiating a TLS connection.  When a client does not support a matching ECDH a handshake error will result.

```yaml
---
apiVersion: getambassador.io/v2
kind:  TLSContext
metadata:
  name:  tls
spec:
  hosts: ["*"]
  secret: ambassador-certs
  min_tls_version: v1.0
  max_tls_version: v1.3
  cipher_suites:
  - "[ECDHE-ECDSA-AES128-GCM-SHA256|ECDHE-ECDSA-CHACHA20-POLY1305]"
  - "[ECDHE-RSA-AES128-GCM-SHA256|ECDHE-RSA-CHACHA20-POLY1305]"
  ecdh_curves:
  - X25519
  - P-256
```

## TLS `Module` (*Deprecated*)

The TLS `Module` is deprecated. `TLSContext` should be used when using Ambassador version 0.50.0 and above.

For users of the Ambassador Edge Stack, see the [Host CRD](/reference/host-crd) reference for more information.

```yaml
---
apiVersion: getambassador.io/v2
kind:  Module
metadata:
  name:  tls
spec:
  config:
    # The 'server' block configures TLS termination. 'enabled' is the only
    # required element.
    server:
      # If 'enabled' is not True, TLS termination will not happen.
      enabled: True

      # If you set 'redirect_cleartext_from' to a port number, HTTP traffic
      # to that port will be redirected to HTTPS traffic. Make sure that the
      # port number you specify matches the port on which Ambassador is
      # listening!
      # redirect_cleartext_from: 8080

      # These are optional. They should not be present unless you are using
      # a custom Docker build to install certificates onto the container
      # filesystem, in which case YOU WILL STILL NEED TO SET enabled: True
      # above.
      #
      # cert_chain_file: /etc/certs/tls.crt   # remember to set enabled!
      # private_key_file: /etc/certs/tls.key  # remember to set enabled!

      # Enable TLS ALPN protocol, typically HTTP2 to negotiate it with
      # HTTP2 clients over TLS.
      # This must be set to be able to use grpc over TLS.
      # alpn_protocols: h2

    # The 'client' block configures TLS client-certificate authentication.
    # 'enabled' is the only required element.
    client:
      # If 'enabled' is not True, TLS client-certificate authentication will
      # not happen.
      enabled: False

      # If 'cert_required' is True, TLS client certificates will be required
      # for every connection.
      # cert_required: False

      # This is optional. It should not be present unless you are using
      # a custom Docker build to install certificates onto the container
      # filesystem, in which case YOU WILL STILL NEED TO SET enabled: True
      # above.
      #
      # cacert_chain_file: /etc/cacert/tls.crt  # remember to set enabled!
```
