# Running Ambassador

The simplest way to run Ambassador is **not** to build it! Instead, just use the YAML files published at https://www.getambassador.io, and start by deciding whether you want to use TLS or not. (If you want more information on TLS, check out our [TLS Overview](../reference/tls-auth.md).) It's possible to switch this later, but it's a pain, and may well involve mucking about with your DNS and such to do it, so it's better to decide up front.

## <a name="TLS">Using TLS</a>

**We recommend using TLS**, which means speaking to Ambassador only over HTTPS. To do this, you need a TLS certificate, which means you'll need the DNS set up correctly.

### TLS, DNS, and the Ambassador Service

In order to set up DNS, we need to know the external IP address or hostname for Ambassador. In order to know that, we need to start by creating the Ambassador's kubernetes service:

```
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-https.yaml
```

This will create an L4 load balancer that will later be used to talk to Ambassador. Once created, you'll be able to set up your DNS to associate a DNS name with this service, and it should be stable **as long as you don't delete the service**.

### Setting up Ambassador's TLS Certificate

Given Ambassador's DNS name, you can request a certificate for it. Sadly, setting up your DNS and requesting a cert are a bit outside the scope of this document -- if you don't know how to do this, check with your local DNS administrator! (If you _are_ the domain admin, check out our [TLS Overview](../reference/tls-auth.md), and check out [Let's Encrypt](https://letsencrypt.org/) if you're shopping for a new CA.)

Once you have the cert, you can run

```shell
kubectl create secret tls ambassador-certs --cert=$FULLCHAIN_PATH --key=$PRIVKEY_PATH
```

where `$FULLCHAIN_PATH` is the path to a single PEM file containing the certificate chain for your cert (including the certificate for your Ambassador and all relevant intermediate certs -- this is what Let's Encrypt calls `fullchain.pem`), and `$PRIVKEY_PATH` is the path to the corresponding private key.

The `ambassador-certs` secret tells Ambassador to provide HTTPS on port 443, and gives it the certificate to present to a client contacting Ambassador. 

[//]:# (
### Using TLS for Client Auth
--
If you want to use TLS client-certificate authentication, you'll need to tell Ambassador about the CA certificate chain to use to validate client certificates. This is also best done before starting Ambassador. Get the CA certificate chain - including all necessary intermediate certificates - and create a Kubernetes secret with it:
--
```shell
kubectl create secret generic ambassador-cacert --from-file=fullchain.pem=$CACERT_PATH
```
--
**NOTE WELL** that the presence of the CA cert chain makes a valid client certificate **mandatory**. If you don't define some valid certificates, Ambassador won't allow any access.  
)

### Starting Ambassador with TLS

After all of the above, you can configure Ambassador's mappings, etc., then start Ambassador running with

```
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-proxy.yaml
```

### Without TLS

If you really, really cannot use TLS, you can set up your Ambassador configuration, then do

```
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador.yaml
```

to create an HTTP-only Ambassador service, then start Ambassador running.

## Once Running

However you started Ambassador, once it's running you'll see pods and services called `ambassador`. By default three replicas of the `ambassador` proxy will be run.

*ALSO NOTE*: The very first time you start Ambassador, it can take a very long time - like 15 minutes - to get the images pulled down and running. You can use `kubectl get pods` to see when the pods are actually running.
