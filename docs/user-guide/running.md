# Running Ambassador

If you clone the [Ambassador repository](https://github.com/datawire/ambassador), you'll have access to multiple Kubernetes resource files:

- `ambassador-rest.yaml` defines the main Ambassador server itself;
- `ambassador-store.yaml` defines the persistent storage that Ambassador uses to remember which services are running; and, finally,
- `ambassador.yaml` wraps up all of the above.

Additionally, you can choose either

- `ambassador-https.yaml`, which defines an HTTPS-only service for talking to Ambassador and is recommended, or
- `ambassador-http.yaml`, which defines an HTTP-only mechanism to access Ambassador.

### <a name="TLS">The Ambassador Service and TLS</a>

You need to choose up front whether you want to use TLS or not. (If you want more information on TLS, check out our [TLS Overview](../reference/tls-auth.md).) It's possible to switch this later, but you'll likely need to muck about with your DNS and such to do it, so it's a pain.

**We recommend using TLS**, which means speaking to Ambassador only over HTTPS. To do this, you need a TLS certificate, which means you'll need the DNS set up correctly. So start by creating the Ambassador's kubernetes service:

```
kubectl apply -f ambassador-https.yaml
```

This will create an L4 load balancer that will later be used to talk to Ambassador. Once created, you'll be able to set up your DNS to associate a DNS name with this service, which will let you request the cert. Sadly, setting up your DNS and requesting a cert are a bit outside the scope of this README -- if you don't know how to do this, check with your local DNS administrator! (If you _are_ the domain admin, check out our [TLS Overview](../reference/tls-auth.md), and check out [Let's Encrypt](https://letsencrypt.org/) if you're shopping for a new CA.)

Once you have the cert, you can run

```
sh scripts/push-cert $FULLCHAIN_PATH $PRIVKEY_PATH
```

where `$FULLCHAIN_PATH` is the path to a single PEM file containing the certificate chain for your cert (including the certificate for your Ambassador and all relevant intermediate certs -- this is what Let's Encrypt calls `fullchain.pem`), and `$PRIVKEY_PATH` is the path to the corresponding private key. `push-cert` will push the cert into Kubernetes secret storage, for Ambassador's later use.

### Without TLS

If you really, really cannot use TLS, you can do

```
kubectl apply -f ambassador-http.yaml
```

for HTTP-only access.

### Using TLS for Client Auth

If you want to use TLS client-certificate authentication, you'll need to tell Ambassador about the CA certificate chain to use to validate client certificates. This is also best done before starting Ambassador. Get the CA certificate chain - including all necessary intermediate certificates - and use `scripts/push-cacert` to push it into a Kubernetes secret:

```
sh scripts/push-cacert $CACERT_PATH
```

After starting Ambassador, you **must** tell Ambassador about which certificates are allowed, using the `/ambassador/principal/` REST API of Ambassador's [admin interface](administering.md):

```
curl -X POST -H "Content-Type: application/json" \
      -d '{ "fingerprint": "$FINGERPRINT" }' \
      http://localhost:8888/ambassador/principal/flynn
```

`$FINGERPRINT` here is the SHA256 fingerprint of the TLS client cert. The easiest way to get it is with the `openssl` command:

```
openssl x509 -fingerprint -sha256 -in $CERTPATH -noout | cut -d= -f2 | tr -d ':' | tr '[A-Z]' '[a-z]'
```

**NOTE WELL** that the presence of the CA cert chain makes a valid client certificate **mandatory**. If you don't define some valid certificates, Ambassador won't allow any access.

### After the Service

The easy way to get Ambassador fully running once its service is created is

```
kubectl apply -f ambassador.yaml
```

### Once Running

However you started Ambassador, once it's running you'll see pods and services called `ambassador` and `ambassador-store`. Both of these are necessary, and at present only one replica of each should be run.

*ALSO NOTE*: The very first time you start Ambassador, it can take a very long time - like 15 minutes - to get the images pulled down and running. You can use `kubectl get pods` to see when the pods are actually running.
