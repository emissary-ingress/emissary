# TLS Termination

You need to choose up front whether you want to use TLS or not. It's possible to switch this later, but you'll likely need to muck about with your DNS and such to do it, so it's a pain.

## 1. Get a certificate, and store it in Kubernetes

You'll need a certificate for TLS. With the certificate:

* Make sure that the `CN` matches the DNS name of your service.
* Ambassador needs the full certificate chain, so concatenate the server certificate and any intermediate certificates into a single file.

Create a Kubernetes `secret` named `ambassador-certs`:

```shell
kubectl create secret tls ambassador-certs --cert=$FULLCHAIN_PATH --key=$PRIVKEY_PATH
```

where `$FULLCHAIN_PATH` is the path to a single PEM file containing the certificate chain for your cert (including the certificate for your Ambassador and all relevant intermediate certs -- this is what Let's Encrypt calls `fullchain.pem`), and `$PRIVKEY_PATH` is the path to the corresponding private key.

When Ambassador starts, it will notice the `ambassador-certs` secret and turn TLS on.

## 2. Create the Ambassador service

1. Create the `ambassador` service in Kubernetes, and don't delete it even if you need to delete and recreate the `ambassador` deployment. This will give you a stable external appearance that you can use for DNS records.

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-https.yaml
```

2. Either a `CNAME` or an `A` is fine.
3. Make sure that there's a valid `PTR` record.

## (Legacy) Configuring Using a `ConfigMap`

If you're using the `ambassador-config` Kubernetes `ConfigMap` that was required in earlier versions of Ambassador, you'll need to create the `ambassador-certs` Kubernetes `secret` as above, but you'll also need to make sure that the `ambassador` [module](../about/concepts.md#modules) is configured to allow TLS:

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
```

Earlier versions of Ambassador required the `secret` to be mounted as a volume; this is **no longer required**, but should still function.

## Configuring Using Files in an Image

If you're building your own custom Ambassador image, you'll need to copy the certificate files into your image, and make sure your `ambassador` [module](../about/concepts.md#modules) is configured properly:

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
      # These are optional: if not present, they take the values listed here,
      # which match what's in ambassador-proxy.yaml.
      # cert_chain_file: /etc/certs/tls.crt
      # private_key_file: /etc/certs/tls.key
```

`cert_chain_file` and `private_key_file` are optional: if you copy your certificate files into `/etc/certs` as shown above, you needn't include them. If you put the certificate files somewhere else, you'll need to update the paths to match.
