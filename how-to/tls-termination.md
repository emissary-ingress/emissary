# TLS Termination

You need to choose up front whether you want to use TLS or not. (If you want more information on TLS, check out our [TLS Overview](../reference/tls-auth.md).) It's possible to switch this later, but you'll likely need to muck about with your DNS and such to do it, so it's a pain.

**We recommend using TLS**, which means speaking to Ambassador only over HTTPS. To do this, you need a TLS certificate, which means you'll need the DNS set up correctly. So start by creating the Ambassador's kubernetes service:

```shell
kubectl apply -f ambassador-https.yaml
```

This will create an L4 load balancer that will later be used to talk to Ambassador. Once created, you'll be able to set up your DNS to associate a DNS name with this service, which will let you request the cert. Sadly, setting up your DNS and requesting a cert are a bit outside the scope of this README -- if you don't know how to do this, check with your local DNS administrator! (If you _are_ the domain admin, check out our [TLS Overview](../reference/tls-auth.md), and check out [Let's Encrypt](https://letsencrypt.org/) if you're shopping for a new CA.)

Once you have the cert, you can run

```shell
sh scripts/push-cert $FULLCHAIN_PATH $PRIVKEY_PATH
```

where `$FULLCHAIN_PATH` is the path to a single PEM file containing the certificate chain for your cert (including the certificate for your Ambassador and all relevant intermediate certs -- this is what Let's Encrypt calls `fullchain.pem`), and `$PRIVKEY_PATH` is the path to the corresponding private key. `push-cert` will push the cert into Kubernetes secret storage, for Ambassador's later use.
