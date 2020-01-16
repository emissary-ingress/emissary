# Client Certificate Validation

Ambassador Edge Stack can be configured to use a provided CA certificate to validate certificates sent from your clients. This allows for client-side mTLS where both Ambassador Edge Stack and the client provide and validate each other's certificates.

## Configuration

To configure client certificate by creating a secret to hold your client's CA certificate and setting `ca_secret` to the value of that secret:

1. Create a secret to hold the client CA certificate

    ```shell
    kubectl create secret generic client-cacert --from-file=tls.crt=$CACERT_PATH
    ```

2. Configure Ambassador Edge Stack to use this certificate for client certificate validation

    ```yaml
    ---
    apiVersion: getambassador.io/v2
    kind: TLSContext
    metadata:
      name: tls
    spec:
      hosts: ["*"]
      secret: ambassador-cert
      ca_secret: client-cacert
    ```

    **Note**: Client certificate validation requires Ambassador Edge Stack be configured to terminate TLS as well by providing a `secret` with TLS certificates for termination

Ambassador will now be configured to validate certificates that the client provides.

## Require Client Certificate

By default, Ambassador Edge Stack will allow requests through that do not provide client certificates. To tell Ambassador Edge Stack to reject requests that fail to provide a certificate, set `cert_required: true` in the `TLSContext` configuration.

```yaml
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: tls
spec:
  hosts: ["*"]
  secret: ambassador-cert
  ca_secret: client-cacert
  cert_required: true
```
