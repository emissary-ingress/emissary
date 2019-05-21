# Transport Layer Security (TLS)

Ambassador supports both terminating TLS and originating TLS. By default, Ambassador will enable TLS termination whenever it finds valid TLS certificates stored in the `ambassador-certs` Kubernetes secret.

## `TLSContext`

The `TLSContext` type manages TLS configurations. The `TLSContext` replaces the original tls `Module` in older versions of Ambassador (pre-0.50).

```yaml
---
apiVersion: ambassador/v1
kind: TLSContext
name: tls-context-1

# 'hosts' defines which the hosts for which this TLSContext is relevant.
# It ties into SNI. A TLSContext without "hosts" is useful only for 
# originating TLS. It must be an array of strings.
# hosts: [ "*" ]

# 'secret' defines a Kubernetes Secret that contains the TLS certificate we
# use for origination or termination.
# secret: ambassador-certs

# 'ca_secret' defines a Kubernetes Secret that contains the TLS certificate we
# use for verifying incoming TLS client certificates.
# ca_secret: None

# 'alpn_protocols' is used to enable the TLS ALPN protocol. It is required
# if you want to do GRPC over TLS; typically it will be set to "h2" for that
# case.
# alpn_protocols: None

# 'cert_required' can be set to true to _require_ TLS client certificate
# authentication.
# cert_required: false

# 'min_tls_version' sets the minimum acceptable TLS version: v1.0, v1.1, 
# v1.2, or v1.3. It defaults to v1.0.
# min_tls_version: v1.0

# 'max_tls_version' sets the maximum acceptable TLS version: v1.0, v1.1, 
# v1.2, or v1.3. It defaults to v1.3.
# max_tls_version: v1.3

# If you are building a custom Docker image, you can configure a TLSContext
# using paths instead of Secrets.
# cert_chain_file: None
# private_key_file: None
# cacert_chain_file: None
```

## `alpn_protocols`

The `alpn_protocols` setting configures the TLS ALPN protocol. To use gRPC over TLS, set `alpn_protocols: h2`. If you need to support HTTP/2 upgrade from HTTP/1, set `alpn_protocols: h2,http/1.1` in the configuration.

### HTTP/2 Support

The `alpn_protocols` setting is also required for HTTP/2 support.

```yaml
apiVersion: ambassador/v1
kind:  Module
name:  tls
config:
  server:
    alpn_protocols: h2[, http/1.1]
```
Without setting setting alpn_protocols as shown above, HTTP2 will not be available via negotiation and will have to be explicitly requested by the client.

If you leave off http/1.1, only HTTP2 connections will be supported.

## `redirect_cleartext_from`

The most common case still requiring a `tls` module is redirecting cleartext traffic on port 8080 to HTTPS on port 8443, which can be done with the following configuration:

```yaml
---
apiVersion: ambassador/v1
kind:  Module
name:  tls
config:
  server:
    enabled: True
    redirect_cleartext_from: 8080
```

## `x_forwarded_proto_redirect`

In cases when TLS is being terminated at an external layer 7 load balancer, then you would want to redirect only the originating HTTP requests to HTTPS, and let the originating HTTPS requests pass through.

This distinction between an originating HTTP request and an originating HTTPS request is done based on the `X-FORWARDED-PROTO` header that the external layer 7 load balancer adds to every request it forwards after TLS termination.

To enable this `X-FORWARDED-PROTO` based HTTP to HTTPS redirection, add a `x_forwarded_proto_redirect: true` field to the Ambassador module configuration, e.g.,

```yaml
apiVersion: ambassador/v1
kind: Module
name: ambassador
config:
  x_forwarded_proto_redirect: true
```

Note: Setting `x_forwarded_proto_redirect: true` will impact all your Ambassador mappings. Requests that contain have `X-FORWARDED-PROTO` set to `https` will be passed through. Otherwise, for all other values of `X-FORWARDED-PROTO`, they will be redirected to TLS.


## `min_tls_version` and `max_tls_version`

The `min_tls_version` setting configures the minimum TLS protocol version that Ambassador will use to establish a secure connection. When a client using a lower version attempts to connect to the server, the handshake will result in the following error: `tls: protocol version not supported`.

The `max_tls_version` setting configures the maximum TLS protocol version that Ambassador will use to establish a secure connection. When a client using a higher version attempts to connect to the server, the handshake will result in the following error: `tls: server selected unsupported protocol version`.

```yaml
---
apiVersion: ambassador/v1
kind: TLSContext
name: ambassador-secure
hosts: []
secret: ambassador-secret
min_tls_version: v1.0
max_tls_version: v1.3
```


## Authentication with TLS Client Certificates

Ambassador also supports TLS client-certificate authentcation. After enabling TLS termination, collect the full CA certificate chain (including all necessary intermediate certificates) into a single file. Store the CA certificate chain used to validate the client certificate into a Kubernetes `secret` named `ambassador-cacert`:

```shell
kubectl create secret generic ambassador-cacert --from-file=tls.crt=$CACERT_PATH
```

where `$CACERT_PATH` is the path to the single file mentioned above.

If you want to _require_ client-cert authentication for every connection, you can add the `cert_required` key:

```shell
kubectl create secret generic ambassador-cacert --from-file=tls.crt=$CACERT_PATH --from-literal=cert_required=true
```

When Ambassador starts, it will notice the `ambassador-cacert` secret and turn TLS client-certificate auth on (assuming that TLS termination is enabled).

### Using a user defined secret

If you do not wish to use a secret named `ambassador-cacert`, then you can specify your own secret. This can be particularly useful if you want to use different secrets for different Ambassador deployments in your cluster.

Create the secret -
```shell
kubectl create secret generic user-secret --from-file=tls.crt=$CACERT_PATH
```

And then, configure Ambassador's TLS module like the following -

```yaml
apiVersion: ambassador/v1
kind:  Module
name:  tls
config:
  client:
    enabled: True
    secret: user-secret
```

Note: If `ambassador-cacert` is present in the cluster and the TLS module is configured to load a custom secret, then `ambassador-cacert` will take precedence, and the custom secret will be ignored.

## TLS Origination
Ambassador is also able to originate a TLS connection with backend services. This can be easily configured by telling Ambassador to route traffic to a service over HTTPS or setting the [tls](/reference/mappings#using-tls) attribute to `true`.

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: qotm
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind:  Mapping
      name:  qotm_mapping
      prefix: /qotm/
      service: https://qotm
spec:
  selector:
    app: qotm
  ports:
  - port: 443
    name: http-qotm
    targetPort: http-api
```
Ambassador will assume it can trust the services in your cluster so will default to not validating the backend's certificates. This allows for your backend services to use self-signed certificates with ease.

### Mutual TLS
Ambassador can be configured to do mutual TLS with backend services as well. To accomplish this, you will need to provide certificates for Ambassador to use with the backend. You will need to create a Kubernetes secret for Ambassador to load the certificates from with a `TLSContext`.

This can be a necessary requirement for using the Consul service mesh and some Istio setups. 

#### Istio mTLS

Istio stores it's TLS certificates as Kubernetes secrets by default, so accessing them is a matter of YAML configuration changes.

1. You will need to mount the `istio.default` secret in a volume in the Ambassador container. This is done by configuring a `volume` and `volumeMount` in the Ambassador deployment manifest.

   ```yaml
    ---
    apiVersion: extensions/v1beta1
    kind: Deployment
    metadata:
      name: ambassador
    spec:
    ...
            volumeMounts:
              - mountPath: /etc/istiocerts/
                name: istio-certs
                readOnly: true
          restartPolicy: Always
          volumes:
          - name: istio-certs
            secret:
              optional: true
              secretName: istio.default
   ```

2. You can now configure Ambassador to use the certificate stored in this secret by specifying an `upstream` in a `TLSContext`:

   ```yaml
   ---
   apiVersion: ambassador/v1
   kind: TLSContext
   name: istio-upstream
   cert_chain_file: /etc/istiocerts/cert-chain.pem
   private_key_file: /etc/istiocerts/key.pem
   cacert_chain_file: /etc/istiocerts/root-cert.pem
   ```

3. Finally, we will tell Ambassador to use these certificates for originating TLS to specific upstream services by setting the `tls` attribute in a `Mapping`

   ```yaml
   ---
   apiVersion: ambassador/v1
   kind: Mapping
   name: productpage_mapping
   prefix: /productpage/
   rewrite: /productpage
   service: https://productpage:9080
   tls: istio-upstream
   ```

Ambassador will now use the certificate stored in the `istio.default` secret to originate TLS to istio-powered services. See the [Ambassador with Istio](/user-guide/with-istio#istio-mutual-tls) documentation) for and example with more information.

#### Consul mTLS

Since Consul does not expose TLS Certificates as Kubernetes secrets, we will need a way to export those from Consul.

1. Install the Ambassador Consul connector. 

   ```
   kubectl apply -f https://www.getambassador.io/yaml/consul/ambassador-consul-connector.yaml
   ```

   This will grab the certificate issued by Consul CA and store it in a Kubernetes secret named `ambassador-consul-connect`. It will also create a Service named `ambassador-consul-connector` which will configure the following `TLSContext`:

   ```yaml
   ---
   apiVersion: ambassador/v1
   kind: TLSContext
   name: ambassador-consul
   hosts: []
   secret: ambassador-consul-connect
   ```

2. Tell Ambassador to use the `TLSContext` when proxying requests by setting the `tls` attribute in a `Mapping`

   ```yaml
   ---
   apiVersion: ambassador/v1
   kind: Mapping
   name: qotm_mtls_mapping
   prefix: /qotm-consul-mtls/
   service: https://qotm-proxy
   tls: ambassador-consul
   ```

Ambassador will now use the certificates loaded into the `ambassador-consul` `TLSContext` when proxying requests with `prefix: /qotm-consul-mtls`. See the [Consul example](/user-guide/consul#encrypted-tls) for an example configuration.

**Note:** The Consul connector can be configured with the following environment variables. The defaults will be best for most use-cases.

| Environment Variable | Description | Default |
| -------------------- | ----------- | ------- |
| \_AMBASSADOR\_ID        | Set the Ambassador ID so multiple instances of this integration can run per-Cluster when there are multiple Ambassadors (Required if `AMBASSADOR_ID` is set in your Ambassador deployment) | `""` |
| \_CONSUL\_HOST          | Set the IP or DNS name of the target Consul HTTP API server | `127.0.0.1` |
| \_CONSUL\_PORT          | Set the port number of the target Consul HTTP API server | `8500` |
| \_AMBASSADOR\_TLS\_SECRET\_NAME | Set the name of the Kubernetes `v1.Secret` created by this program that contains the Consul-generated TLS certificate. | `$AMBASSADOR_ID-consul-connect` |
| \_AMBASSADOR\_TLS\_SECRET\_NAMESPACE | Set the namespace of the Kubernetes `v1.Secret` created by this program. | (same Namespace as the Pod running this integration) |


## The `tls` module

The `tls` module defines system-wide configuration for TLS when additional configuration is needed.

*Note well*: while the `tls` module is not yet deprecated, `TLSContext` resources are definitely
preferred where possible.

```yaml
---
apiVersion: ambassador/v1
kind:  Module
name:  tls
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


## More reading

The [TLS termination guide](/user-guide/tls-termination) provides a tutorial on getting started with TLS in Ambassador. For more informatiom on configuring Ambassador with external L4/L7 load balancers, see the [documentation on AWS](/reference/ambassador-with-aws). Note that this document, while intended for AWS users, has information also applicable to other cloud providers.

