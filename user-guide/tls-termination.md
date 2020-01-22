# TLS Termination and Enabling HTTPS

TLS encryption is one of the basic requirements of having a secure system. Ambassador Edge Stack automatically enables TLS termination/HTTPs, making TLS encryption easy and centralizing TLS termination for all of your services in Kubernetes automatically during configuration if you have a fully qualified domain name (FQDN).

However, if you don't have an FQDN for your Ambassador Edge Stack, you can manually enable TLS. This guide will show you how to quickly enable TLS termination in Ambassador Edge Stack with a self-signed certificate.

**Note** that these instructions do not work with the Ambassador API Gateway.

## Prerequisites

This guide requires you have the following installed:

- A Kubernetes cluster v1.11 or newer
- The Kubernetes command line tool, [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [openssl](https://www.openssl.org/source/)

## Install Ambassador Edge Stack

Install Ambassador Edge Stack in Kubernetes using the [YAML manifests](../install).

## Create a Self-Signed Certificate

OpenSSL is a tool that allows us to create self-signed certificates for opening a TLS encrypted connection. The following commands will quickly create a certificate we can use for this purpose.

- Create a private key.

   ```
   openssl genrsa -out key.pem 2048
   ```

- Create a certificate signed by the private key just created

   ```
   openssl req -x509 -key key.pem -out cert.pem -days 365 -new -subj '/CN=ambassador-cert'
   ```

- Verify the `key.pem` and `cert.pem` files were created

   ```
   ls *.pem
   cert.pem	key.pem
   ```

## Store the Certificate and Key in a Kubernetes Secret

Ambassador Edge Stack dynamically loads TLS certificates by reading them from Kubernetes secrets. Use `kubectl` to create a `tls` secret to hold the pem files we created above.

```
kubectl create secret tls tls-cert --cert=cert.pem --key=key.pem
```

## Tell Ambassador Edge Stack to Use this Secret for TLS Termination

Now that we have stored our certificate and private key in a Kubernetes secret named `tls-cert`, we need to tell Ambassador Edge Stack to use this certificate for terminating TLS. This is done with a `TLSContext`.

Run the following command to create a `TLSContext` CRD that configures Ambassador Edge Stack to use the certificates stored in the `tls-cert` secret for terminating TLS for all hosts and endpoints.

```shell
cat << EOF | kubectl apply -f -
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: tls
spec:
  hosts: ["*"]
  secret: tls-cert
EOF
```

## Send a Request Over HTTPS

We can now send encrypted traffic over HTTPS.

First, make sure the Ambassador Edge Stack service is listening on 443 and forwarding to port 8443. Verify this with `kubectl`:

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

If the output to the `kubectl` command is not similar to the example above, edit the Ambassador Edge Stack service to add the `https` port.

After verifying Ambassador Edge Stack is listening on port 443, send a request to your backend service with curl:

```
curl -Lk https://{{AMBASSADOR_IP}}/backend/

{
    "server": "trim-kumquat-fccjxh8x",
    "quote": "Abstraction is ever present.",
    "time": "2019-07-24T16:36:56.7983516Z"
}
```

**Note:** Since we are using a self-signed certificate, you must set the `-k` flag in curl to disable hostname validation.

## Next Steps

This guide walked you through how to enable basic TLS termination in Ambassador Edge Stack using a self-signed certificate for simplicity. 

### Get a Valid Certificate from a Certificate Authority

While a self-signed certificate is a simple and quick way to get Ambassador Edge Stack to terminate TLS, it should not be used by production systems. In order to serve HTTPS traffic without being returned a security warning, you will need to get a certificate from an official Certificate Authority like Let's Encrypt.

In Kubernetes, Jetstack's `cert-manager` provides a simple way to manage certificates from Let's Encrypt. See our documentation for more information on how to [use `cert-manager` with Ambassador Edge Stack](../cert-manager).

### Enable advanced TLS options

Ambassador Edge Stack exposes configuration for many more advanced options around TLS termination, origination, client certificate validation, and SNI support. See the full [TLS reference](../../reference/core/tls) for more information.
