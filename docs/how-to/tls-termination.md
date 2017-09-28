# TLS Termination

You need to choose up front whether you want to use TLS or not. (If you want more information on TLS, check out our [TLS Overview](../reference/tls-auth.md).) It's possible to switch this later, but you'll likely need to muck about with your DNS and such to do it, so it's a pain.

**We recommend using TLS**, which means speaking to Ambassador only over HTTPS. To do this, you need a TLS certificate, which means you'll need the DNS set up correctly. So start by creating the Ambassador's kubernetes service:

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-https.yaml
```

This will create an L4 load balancer that will later be used to talk to Ambassador. Once created, you'll be able to set up your DNS to associate a DNS name with this service, which will let you request the cert. Sadly, setting up your DNS and requesting a cert are a bit outside the scope of this README -- if you don't know how to do this, check with your local DNS administrator! (If you _are_ the domain admin, check out our [TLS Overview](../reference/tls-auth.md), and check out [Let's Encrypt](https://letsencrypt.org/) if you're shopping for a new CA.)

Once you have the cert, you can run

```shell
kubectl create secret tls ambassador-certs --cert=$FULLCHAIN_PATH --key=$PRIVKEY_PATH
```

where `$FULLCHAIN_PATH` is the path to a single PEM file containing the certificate chain for your cert (including the certificate for your Ambassador and all relevant intermediate certs -- this is what Let's Encrypt calls `fullchain.pem`), and `$PRIVKEY_PATH` is the path to the corresponding private key.

### `ambassador.yaml`

In addition to installing the certificate, you'll need to make sure that the `ambassador` [module](../about/concepts.md#modules) is configured to allow TLS:

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

The simplest possible TLS block is thus:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  ambassador
config:
  tls:
    server:
      enabled: True
```

which simply enables TLS termination using the default map for certificates. 

Note well that if you change the pathnames in the TLS configuration, you'll have to make sure that your secret mounting matches your edits! Beyond that, Ambassador doesn't care what paths you use.

### Starting Ambassador with TLS

After all of the above, you can [configure Ambassador's mappings, etc.](../reference/configuration.md), then start Ambassador running with

```
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-proxy.yaml
```

Note that `ambassador-proxy.yaml` includes liveness and readiness probes that assume that Ambassador is listening on port 443. This won't work for an HTTP-only Ambassador.
