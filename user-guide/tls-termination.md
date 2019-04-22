# TLS Termination

To enable TLS termination for Ambassador you'll need a few things:

1. You'll need a TLS certificate.
2. For any production use, you'll need a DNS record that matches your TLS certificate's `Common Name`.
3. You'll need to store the certificate in a Kubernetes `secret`.
4. You'll need a basic `tls` module configuration in Ambassador.

All these requirements mean that it's easiest to decide to enable TLS _before_ you configure Ambassador the first time. It's possible to switch after setting up Ambassador, but it's annoying.

## 1. You'll need a TLS certificate.

There are a great many ways to get a certificate; [Let's Encrypt](https://www.letsencrypt.org) is a good option if you're not already working with something else. Check out the "Certificate Manager" section below to get set up with Let's Encrypt on Kubernetes.

Note that requesting a certificate _requires_ a `Common Name` (`CN`) for your Ambassador. The `CN` becomes very important when you try to use HTTPS in practice: if the `CN` does not match the DNS name you use to reach the Ambassador, most TLS libraries will refuse to make the connection. So use a DNS name for the `CN`, and in step 2 make sure everything matches.

## 2. You'll need a DNS name.

As noted above, the DNS name must match the `CN` in the certificate. The simplest way to manage this is to create an `ambassador` Kubernetes service up front, before you do anything else, so that you can point DNS to whatever IP Kubernetes assignes to it -- then don't delete the `ambassador` service, even if you later need to update it or delete and recreate the `ambassador` deployment.

Alternatively you can request a static IP from your cloud provider and set it as `loadBalancerIP` in the service created below.

```shell
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-https.yaml
```

will create a minimal `ambassador` service for this purpose. You can then use its external IP address to configure either a `CNAME` or an `A` record in DNS. Make sure that there's a matching `PTR` record, too.

It's OK to include annotations on the `ambassador` service at this point, if you need to configure additional TLS options (see below for more on this).

Once assigned the external IP can be checked with:

```shell
kubectl get svc -o wide ambassador
```

## 3. You'll need to store the certificate in a Kubernetes `secret`.

Create a Kubernetes `secret` named `ambassador-certs`:

```shell
kubectl create secret tls ambassador-certs --cert=$FULLCHAIN_PATH --key=$PRIVKEY_PATH
```

where `$FULLCHAIN_PATH` is the path to a single PEM file containing the certificate chain for your cert (including the certificate for your Ambassador and all relevant intermediate certs -- this is what Let's Encrypt calls `fullchain.pem`), and `$PRIVKEY_PATH` is the path to the corresponding private key.

When Ambassador starts, it will notice the `ambassador-certs` secret and turn TLS on.

**Important.** Note that the `ambassador-certs` Secret _must_ be in the same Kubernetes namespace as the Ambassador Service.

##### Configuring using a user defined secret

If you do not wish to use a secret named `ambassador-certs`, then you can tell Ambassador to use your own secret. This can be particularly useful if you want to use different secrets for different Ambassador deployments.

Create the secret -
```shell
kubectl create secret tls user-secret --cert=$FULLCHAIN_PATH --key=$PRIVKEY_PATH
```

And then, configure Ambassador's TLS module like the following -

```yaml
apiVersion: ambassador/v1
kind: Module
name: tls
config:
  server:
    enabled: True
    secret: user-secret
```

This will make Ambassador load a secret called `user-secret` to configure TLS termination.

Note: If `ambassador-certs` is present in the cluster and the TLS module is configured to load a custom secret, then `ambassador-certs` will take precedence, and the custom secret will be ignored.

## 4. Configure other Ambassador TLS options using the `tls` module.

You'll need a minimal `tls` module to configure Ambassador TLS, e.g.,

```yaml
---
apiVersion: ambassador/v1
kind: Module
name: tls
config:
  server:
    secret: ambassador-certs
```

The `tls` module supports additional configuration options.

```yaml
---
apiVersion: ambassador/v1
kind: Module
name: tls
config:
  # The 'server' block configures TLS termination. 'enabled' is the only
  # required element.
  server:
    # If 'enabled' is True, TLS termination will be enabled.
    enabled: True

    # If you set 'redirect_cleartext_from' to a port number, HTTP traffic
    # to that port will be redirected to HTTPS traffic. Typically you would
    # use port 80, of course.
    # redirect_cleartext_from: 80

    # These are optional. They should not be present unless you are using
    # a custom Docker build to install certificates onto the container
    # filesystem, in which case YOU WILL STILL NEED TO SET enabled: True
    # above.
    #
    # cert_chain_file: /etc/certs/tls.crt   # remember to set enabled!
    # private_key_file: /etc/certs/tls.key  # remember to set enabled!

    # Enable TLS ALPN protocol, typically HTTP2 to negotiate it with HTTP2
    # clients over TLS. This must be set to be able to use grpc over TLS.
    # alpn_protocols: h2

  # The 'client' block configures TLS client-certificate authentication.
  # 'enabled' is the only required element.
  client:
    # If 'enabled' is True, TLS client-certificate authentication will occur.
    enabled: False

    # If 'cert_required' is True, TLS client certificates will be required
    # for every connection.
    # cert_required: False

    # This is optional. It should not be present unless you are using
    # a custom Docker build to install certificates onto the container
    # filesystem, in which case YOU WILL STILL NEED TO SET enabled: True
    # above.
    #
    # cacert_chain_file: /etc/cacert/tls.crt  # remember to set enabled!
```

In terms of the tls `Module`, it's simplest to include it as an `annotation` on the `ambassador` service itself, like so:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: ambassador
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind: Module
      name: tls
      config:
        server:
          enabled: True
          redirect_cleartext_from: 80
          secret: ambassador-certs
spec:
  ports:
    - name: http
      protocol: TCP
      port: 80
    - name: https
      protocol: TCP
      port: 443
  ...
```

**Important.** Note that the name of the Module is case-sensitive! It must be `name: tls` as opposed to `name: TLS`.

## Redirecting Cleartext

Ambassador can only fully serve traffic for either HTTP or HTTPS traffic. Ambassador can however be configured to issue a 301 redirect for all cleartext traffic received on a port. This port is specified by `redirect_cleartext_from` and should be set to whichever port Ambassador is expecting to see HTTP traffic from (typically 80).

To redirect HTTP traffic on port 80 to HTTPS on port 443:

```yaml
---
apiVersion: ambassador/v1
kind: Module
name: tls
config:
  server:
    enabled: True
    redirect_cleartext_from: 80
    secret: ambassador-certs
```

## Overriding Default Ports

By default, Ambassador will listen for HTTPS on port 443 when the TLS `Module` is configured. If you would like Ambassador to listen on a different port (i.e. 8443), you will need to configure this in the [Ambassador `Module`](/reference/core/ambassador). 

```yaml
apiVersion: ambassador/v1
kind: Module
name: ambassador
config: 
  service_port: 8443
```

## Certificate Manager

Jetstack's [cert-manager](https://github.com/jetstack/cert-manager) lets you easily provision and manage TLS certificates on Kubernetes. See documentation on using [cert-manager with Ambassador](/user-guide/cert-manager).

## Supporting multiple domains

With the setup from above it is not possible to supply separate certificates for different domains.
This can be achieved with Ambassador's [SNI support](/user-guide/sni).
