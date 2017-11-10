# Auth with TLS Client Certificates

If you want to use TLS client-certificate authentication, you'll first need to enable [TLS termination](tls-termination.html): TLS client-certificate auth will not function if TLS termination isn't enabled.

Once that's done, you'll need to tell Ambassador about the CA certificate chain to use to validate client certificates. Start by collecting the chain - including all necessary intermediate certificates - into a single file.

## RECOMMENDED: Configuring Using Annotations

**This is the easiest way to use TLS, and as such is highly recommended.** If you're unfamiliar with using annotations to configure Ambassador, check out the [Ambassador Getting Started](/user-guide/getting-started.html).

If you're using annotations, all you need to do is store your certificate in a Kubernetes `secret` named `ambassador-certs`:

```shell
kubectl create secret generic ambassador-cacert --from-file=tls.crt=$CACERT_PATH
```

where `$FULLCHAIN_PATH` is the path to the single file mentioned above.

If you want to _require_ client-cert authentication for every connection, you can add the `cert_required` key:

```shell
kubectl create secret generic ambassador-cacert --from-file=tls.crt=$CACERT_PATH --from-literal=cert_required=true
```

When Ambassador starts, it will notice the `ambassador-cacert` secret and turn TLS client-certificate auth on (assuming that TLS termination is enabled).

## Configuring Using a `ConfigMap`

If you're using the `ambassador-config` Kubernetes `ConfigMap` that was required in earlier versions of Ambassador, you'll need to create the `ambassador-cacert` Kubernetes `secret` as above, but you'll also need to make sure that the `ambassador` [module](../about/concepts.md#modules) is configured for TLS client-cert auth:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  tls:
    # The 'server' block configures TLS termination. 'enabled' is the only required element.
    server:
      enabled: True
    client:
      enabled: True
      # cert_required: True   # Optional -- leave this off if you don't need to require client certs
```

Included `cert_required: True` will require client certs for every connection. If not present, client certs will be validated if present, but not required.

Earlier versions of Ambassador required the `secret` to be mounted as a volume. This is **no longer required**, but should still work if you're already using it.

## Configuring Using Files in an Image

Finally, if you're building your own custom Ambassador image, you'll need to copy the certificate files into your image, and make sure your `ambassador` [module](../about/concepts.md#modules) is configured properly:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  tls:
    # The 'server' block configures TLS termination. 'enabled' is the only required element.
    server:
      enabled: True
    client:
      enabled: True
      # cert_required: True   # Optional -- leave this off if you don't need to require client certs
      # cacert_chain_file: /etc/cacert/fullchain.pem
```

`cacert_chain_file` is optional: if you copy your certificate chain into `/etc/cacert/fullchain` as shown above, you needn't include it. If you put the certificate chain somewhere else, you'll need to update the path to match.

**NOTE WELL** that the default is "fullchain.pem", not "tls.crt", for historical reasons. Ambassador doesn't care about the actual name, as long as you set the `cacert_chain_file` path to match where the chain was actually copied.
