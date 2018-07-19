# Auth with TLS Client Certificates

If you want to use TLS client-certificate authentication, you'll first need to enable [TLS termination](tls-termination.html): TLS client-certificate auth will not function if TLS termination isn't enabled.

Once that's done, you'll need to tell Ambassador about the CA certificate chain to use to validate client certificates. Start by collecting the chain - including all necessary intermediate certificates - into a single file.

Assuming you're using annotations to configure Ambassador, all you need to do to enable TLS client-certificate authentication is store to store your CA certificate chain in a Kubernetes `secret` named `ambassador-cacert`:

```shell
kubectl create secret generic ambassador-cacert --from-file=tls.crt=$CACERT_PATH
```

where `$CACERT_PATH` is the path to the single file mentioned above.

If you want to _require_ client-cert authentication for every connection, you can add the `cert_required` key:

```shell
kubectl create secret generic ambassador-cacert --from-file=tls.crt=$CACERT_PATH --from-literal=cert_required=true
```

When Ambassador starts, it will notice the `ambassador-cacert` secret and turn TLS client-certificate auth on (assuming that TLS termination is enabled).

You can also configure TLS client-certificate authentication using the `tls` module. For details here, see the documentation on [TLS termination](tls-termination.html).

##### Configuring using a user defined secret

If you do not wish to use a secret named `ambassador-cacert`, then you can tell Ambassador to use your own secret. This can be particularly useful if you want to use different secrets for different Ambassador deployments.

Create the secret -
```shell
kubectl create secret generic user-secret --from-file=tls.crt=$CACERT_PATH
```

And then, configure Ambassador's TLS module like the following -

```yaml
apiVersion: ambassador/v0
kind:  Module
name:  tls
config:
  client:
    enabled: True
    secret: user-secret
```

Note: If `ambassador-cacert` is present in the cluster and the TLS module is configured to load a custom secret, then `ambassador-cacert` will take precedence, and the custom secret will be ignored.
