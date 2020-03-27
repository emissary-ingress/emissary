# TLS Termination and Enabling HTTPS

TLS encryption is one of the basic requirements of having a secure system.
Ambassador Edge Stack [automatically enables TLS termination/HTTPs
](../../topics/host-crd#acme-and-tls-settings), making TLS encryption easy
and centralizing TLS termination for all of your services in Kubernetes.

While this automatic certificate management in the Ambassador Edge Stack helps
simply TLS configuration in your cluster, the Open-Source Ambassador API
Gateway still requires you provide your own certificate to enable TLS.

The following will walk you through the process of enabling TLS with a 
self-signed certificate created with the `openssl` utility. 

**Note** these instructions also work if you would like to provide your own
certificate to the Ambassador Edge Stack.

## Prerequisites

This guide requires you have the following installed:

- A Kubernetes cluster v1.11 or newer
- The Kubernetes command-line tool, 
[`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [openssl](https://www.openssl.org/source/)

## Install Ambassador Edge Stack

[Install Ambassador Edge Stack in Kubernetes](../../topics/install).

## Create a Self-Signed Certificate

OpenSSL is a tool that allows us to create self-signed certificates for opening
a TLS encrypted connection. The `openssl` command below will create a 
create a certificate and private key pair that Ambassador can use for TLS
termination.

- Create a private key and certificate.

   ```
   openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -subj '/CN=ambassador-cert' -nodes
   ```

   The above command will create a certificate and private key with the common
   name `ambassador`. Since this certificate is self-signed and only used for testing,
   the other information requested can be left blank.

- Verify the `key.pem` and `cert.pem` files were created

   ```
   ls *.pem
   cert.pem	key.pem
   ```

## Store the Certificate and Key in a Kubernetes Secret

Ambassador Edge Stack dynamically loads TLS certificates by reading them from
Kubernetes secrets. Use `kubectl` to create a `tls` secret to hold the pem 
files we created above.

```
kubectl create secret tls tls-cert --cert=cert.pem --key=key.pem
```

## Tell Ambassador Edge Stack to Use this Secret for TLS Termination

Now that we have stored our certificate and private key in a Kubernetes secret
named `tls-cert`, we need to tell Ambassador Edge Stack to use this certificate
for terminating TLS on a domain. A `Host` is used to tell Ambassador which
certificate to use for TLS termination on a domain.

Create the following `Host` to have Ambassador use the `Secret` we created
above for terminating TLS on all domains.

```yaml
---
apiVersion: getambassador.io/v2
kind: Host
metadata:
  name: wildcard-host
spec:
  hostname: "*"
  acmeProvider:
    authority: none
  tlsSecret:
    name: tls-cert
  selector:
    matchLabels:
      hostname: wildcard-host
```

Apply the `Host` configured above with `kubectl`:

```
kubectl apply -f wildcard-host.yaml
```

Ambassador is now configured to listen for TLS traffic on port `8443` and
terminate TLS using the self-signed certificate we created.

## Send a Request Over HTTPS

We can now send encrypted traffic over HTTPS.

First, make sure the Ambassador service is listening on `443` and forwarding
to port `8443`. Verify this with `kubectl`:

```
kubectl get service ambassador -o yaml

apiVersion: v1
kind: Service
...
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 8080
  - name: https
    port: 443
    protocol: TCP
    targetPort: 8443
...
```

If the output to the `kubectl` command is not similar to the example above, 
edit the Ambassador service to add the `https` port.

After verifying Ambassador Edge Stack is listening on port 443, send a request
to your backend service with curl:

```
curl -Lk https://{{AMBASSADOR_IP}}/backend/

{
    "server": "trim-kumquat-fccjxh8x",
    "quote": "Abstraction is ever present.",
    "time": "2019-07-24T16:36:56.7983516Z"
}
```

**Note:** Since we are using a self-signed certificate, you must set the `-k`
flag in curl to disable hostname validation.

## Next Steps

This guide walked you through how to enable basic TLS termination in Ambassador
Edge Stack using a self-signed certificate for simplicity.

### Get a Valid Certificate from a Certificate Authority

While a self-signed certificate is a simple and quick way to get Ambassador Edge Stack to terminate TLS, it should not be used by production systems. In order to serve HTTPS traffic without being returned a security warning, you will need to get a certificate from an official Certificate Authority like Let's Encrypt.

With the Ambassador Edge Stack, this can be simply done by requesting a
certificate using the built in [ACME support](../../topics/host-crd#acme-support)

For the Open-Source API Gateway, Jetstack's `cert-manager` provides a simple
way to manage certificates from Let's Encrypt. See our documentation for more
information on how to [use `cert-manager` with Ambassador Edge Stack
](../cert-manager).

### Enable Advanced TLS options

Ambassador Edge Stack exposes configuration for many more advanced options
around TLS termination, origination, client certificate validation, and SNI
support. See the full [TLS reference](../../topics/running/tls) for more
information.
