# Auth with TLS Client Certs

If you want to use TLS client-certificate authentication, you'll need to tell Ambassador about the CA certificate chain to use to validate client certificates. This is also best done before starting Ambassador. Get the CA certificate chain - including all necessary intermediate certificates - and use `scripts/push-cacert` to push it into a Kubernetes secret:

```shell
sh scripts/push-cacert $CACERT_PATH
```

After starting Ambassador, you **must** tell Ambassador about which certificates are allowed, using the `/ambassador/principal/` REST API of Ambassador's [admin interface](administering.md):

```shell
curl -X POST -H "Content-Type: application/json" \
      -d '{ "fingerprint": "$FINGERPRINT" }' \
      http://localhost:8888/ambassador/principal/flynn
```

`$FINGERPRINT` here is the SHA256 fingerprint of the TLS client cert. The easiest way to get it is with the `openssl` command:

```shell
openssl x509 -fingerprint -sha256 -in $CERTPATH -noout | cut -d= -f2 | tr -d ':' | tr '[A-Z]' '[a-z]'
```

**NOTE WELL** that the presence of the CA cert chain makes a valid client certificate **mandatory**. If you don't define some valid certificates, Ambassador won't allow any access. We'll be improving on this in a later release.
