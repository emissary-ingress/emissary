# TLS Termination

To enable TLS termination for Ambassador you'll need a few things:

1. You'll need a TLS certificate.
2. For any production use, you'll need a DNS record that matches your TLS certificate's `Common Name`.
3. You'll need to store the certificate in a Kubernetes `secret`.
4. You may need to configure other Ambassador TLS options using the `tls` module.

All these requirements mean that it's easiest to decide to enable TLS _before_ you configure Ambassador the first time. It's possible to switch after setting up Ambassador, but it's annoying.

## 1. You'll need a TLS certificate.

There are a great many ways to get a certificate; [Let's Encrypt](https://www.letsencrypt.org) is a good option if you're not already working with something else. 

Note that requesting a certificate _requires_ a `Common Name` (`CN`) for your Ambassador. The `CN` becomes very important when you try to use HTTPS in practice: if the `CN` does not match the DNS name you use to reach the Ambassador, most TLS libraries will refuse to make the connection. So use a DNS name for the `CN`, and in step 2 make sure everything matches.

## 2. You'll need a DNS name.

As noted above, the DNS name must match the `CN` in the certificate. The simplest way to manage this is to create an `ambassador` Kubernetes service up front, before you do anything else, so that you can point DNS to whatever Kubernetes gives you for it -- then don't delete the `ambassador` service, even if you later need to update it or delete and recreate the `ambassador` deployment.

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-https.yaml
```

will create a minimal `ambassador` service for this purpose; you can then use its external appearance to configure either a `CNAME` or an `A` record in DNS. Make sure that there's a matching `PTR` record, too.

It's OK to include annotations on the `ambassador` service at this point, if you need to configure additional TLS options (see below for more on this).

## 3. You'll need to store the certificate in a Kubernetes `secret`.

Create a Kubernetes `secret` named `ambassador-certs`:

```shell
kubectl create secret tls ambassador-certs --cert=$FULLCHAIN_PATH --key=$PRIVKEY_PATH
```

where `$FULLCHAIN_PATH` is the path to a single PEM file containing the certificate chain for your cert (including the certificate for your Ambassador and all relevant intermediate certs -- this is what Let's Encrypt calls `fullchain.pem`), and `$PRIVKEY_PATH` is the path to the corresponding private key.

When Ambassador starts, it will notice the `ambassador-certs` secret and turn TLS on.

**Important.** If you've already created the `ambassador` deployment in step 2 or you're adding TLS termination to an existing deployment, you **MUST** restart it. Ambassador looks for the `ambassador-certs` when it starts and only watches service changes later on ([#474](https://github.com/datawire/ambassador/issues/474)). If high availability is not an issue, simply delete the `ambassador` pods (after a short downtime, the deployment will start new pods for you):

```shell
kubectl delete pods -l service=ambassador
```

To ensure high availability, you can force a no-op rolling update (https://github.com/kubernetes/kubernetes/issues/27081):

```shell
kubectl patch deployment ambassador -p "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"date\":\"`date +'%s'`\"}}}}}"
```

## 4. You may need to configure other Ambassador TLS options.

If you don't need anything else, you're good to go.

However, you may also configure other options using Ambassador's `tls` module:

```yaml
---
apiVersion: ambassador/v0
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
    # filesystem.
    # cert_chain_file: /etc/certs/tls.crt
    # private_key_file: /etc/certs/tls.key

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
    # filesystem.
    # cacert_chain_file: /etc/cacert/fullchain.pem
```

Of these, `redirect_cleartext_from` is the most likely to be relevant: to make Ambassador redirect HTTP traffic on port 80 to HTTPS on port 443, you _must_ use the `tls` module:

```yaml
---
apiVersion: ambassador/v0
kind:  Module
name:  tls
config:
  server:
    enabled: True
    redirect_cleartext_from: 80
```

is the minimal YAML to do this.

If you need a `tls` module, it's simplest to include it as an `annotation` on the `ambassador` service itself. 

## Legacy configuration options

It's still possible - but not recommended! - to configure Ambassador using a `ConfigMap`, or with YAML files on the container filesystem. If you think you'll need to do this, please contact us on [Gitter](https://gitter.im/datawire/ambassador).

