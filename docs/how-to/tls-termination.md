# TLS Termination

You need to choose up front whether you want to use TLS or not. (If you want more information on TLS, check out our [TLS Overview](../reference/tls-auth.md).) It's possible to switch this later, but you'll likely need to muck about with your DNS and such to do it, so it's a pain.

**We recommend using TLS**, which means speaking to Ambassador only over HTTPS. To do this, you need a TLS certificate, which means you'll need the DNS set up correctly. So start by creating the Ambassador's kubernetes service:

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-https.yaml
```

This will create an L4 load balancer that will later be used to talk to Ambassador. Once created, you'll be able to set up your DNS to associate a DNS name with this service, which will let you request the cert. Sadly, setting up your DNS and requesting a cert are a bit outside the scope of this README -- if you don't know how to do this, check with your local DNS administrator! (If you _are_ the domain admin, check out our [TLS Overview](../reference/tls-auth.md), and check out [Let's Encrypt](https://letsencrypt.org/) if you're shopping for a new CA.)

## RECOMMENDED: Configuring Using Annotations

**This is the easiest way to use TLS, and as such is highly recommended.** If you're unfamiliar with using annotations to configure Ambassador, check out the [Ambassador Getting Started](/user-guide/getting-started.html).

If you're using annotations, all you need to do is store your certificate in a Kubernetes `secret` named `ambassador-certs`:

```shell
kubectl create secret tls ambassador-certs --cert=$FULLCHAIN_PATH --key=$PRIVKEY_PATH
```

where `$FULLCHAIN_PATH` is the path to a single PEM file containing the certificate chain for your cert (including the certificate for your Ambassador and all relevant intermediate certs -- this is what Let's Encrypt calls `fullchain.pem`), and `$PRIVKEY_PATH` is the path to the corresponding private key.

When Ambassador starts, it will notice the `ambassador-certs` secret and turn TLS on.

## Configuring Using a `ConfigMap`

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
      # These are optional: if not present, they take the values listed here,
      # which match what's in ambassador-proxy.yaml. 
      # cert_chain_file: /etc/certs/tls.crt
      # private_key_file: /etc/certs/tls.key
```

`cert_chain_file` and `private_key_file` are optional: if you copy your certificate files into `/etc/certs` as shown above, you needn't include them. If you put the certificate files somewhere else, you'll need to update the paths to match.
