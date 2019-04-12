# Transport Layer Security (TLS)

Ambassador supports both terminating TLS and originating TLS. By default, Ambassador will enable TLS termination whenever it finds valid TLS certificates stored in the `ambassador-certs` Kubernetes secret. 

## The `tls` module

The `tls` module defines system-wide configuration for TLS when additional configuration is needed.

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
    # to that port will be redirected to HTTPS traffic. Typically you would
    # use port 80, of course.
    # redirect_cleartext_from: 80

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

## `alpn_protocols`

The `alpn_protocols` setting configures the TLS ALPN protocol. To use gRPC over TLS, set `alpn_protocols: h2`. If you need to support HTTP/2 upgrade from HTTP/1, set `alpn_protocols: h2,http/1.1` in the configuration.

## Redirecting from cleartext to TLS

The most common case requiring a `tls` module is redirecting cleartext traffic on port 80 to HTTPS on port 443, which can be done with the following configuration:

```yaml
---
apiVersion: ambassador/v1
kind:  Module
name:  tls
config:
  server:
    enabled: True
    redirect_cleartext_from: 80
```

## X-FORWARDED-PROTO Redirect

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
Ambassador can be configured to do mutual TLS with backend services as well. To accomplish this, you will need to provide certificates for Ambassador to use with the backend. An example of this is given in the [Ambassador with Istio](/user-guide/with-istio#istio-mutual-tls) documentation.

## More reading

The [TLS termination guide](/user-guide/tls-termination) provides a tutorial on getting started with TLS in Ambassador. For more informatiom on configuring Ambassador with external L4/L7 load balancers, see the [documentation on AWS](/reference/ambassador-with-aws). Note that this document, while intended for AWS users, has information also applicable to other cloud providers.

