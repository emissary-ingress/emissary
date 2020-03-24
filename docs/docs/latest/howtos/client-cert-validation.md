# Client Certificate Validation

Sometimes, for additional security or authentication purposes, you will want
the server to validate who the client is before establishing an encryopted 
connection.

To support this, Ambassador can be configured to use a provided CA certificate 
to validate certificates sent from your clients. This allows for client-side 
mTLS where both Ambassador and the client provide and validate each other's 
certificates.

## Prerequesites

- [openssl](https://www.openssl.org/source/) For creating client certificates
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- [Ambassador Edge Stack](../../tutorials/getting-started)
- [cURL](https://curl.haxx.se/download.html)


## Configuration

1. Create a certificate and key.

   This can be done with a single command with `openssl`:

   ```
   openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365
   ```

   Enter a passcode for PEM files and fill in the certificate information.
   Since this certificate will only be shared between a client and Ambassador,
   the Common Name must be set to something. Everything else can be left blank.

   **Note:** If using MacOS, 
   [you must](https://curl.haxx.se/mail/archive-2014-10/0053.html) 
   add the certificate and key as a PKCS encoded file to your Keychain. To do 
   this:
   
   1. Encode `cert.pem` and `key.pem` created above in PKCS format

      ```
      openssl pkcs12 -inkey key.pem -in cert.pem -export -out certificate.p12
      ```

   2. Open "Keychain Access" on your system and select "File"->"Import Items..."

   3. Navigate to your working directoy and select the `certificate.p12` file
   we just created above.

2. Create a secret to hold the client CA certificate.

   ```shell
   kubectl create secret generic client-cacert --from-file=tls.crt=cet.pem
   ```

3. Configure Ambassador Edge Stack to use this certificate for client certificate validation.

   First create a `Host` to manage your domain:

   ```yaml
   apiVersion: getambassador.io/v2
   kind: Host
   metadata:
     name: example-host
   spec:
     hostname: host.example.com
     acmeProvider:
       email: julian@example.com
   ```

   Then create a `TLSContext` to configure advanced TLS options like client
   certificate validation:
   
    ```yaml
    ---
    apiVersion: getambassador.io/v2
    kind: TLSContext
    metadata:
      name: example-host-context
    spec:
      hosts:
      - host.example.com
      secret: host.example.com
      ca_secret: client-cacert
      cert_required: false      # Optional: Configures Ambassador to reject the request if the client does not provide a certificate. Default: false
    ```

    **Note**: Client certificate validation requires Ambassador Edge Stack be configured to terminate TLS 

    Ambassador is now be configured to validate certificates that the client provides.

4. Test that Ambassador is validating the client certificates with `curl`

   **Linux**:
   ```
   curl -v --cert cert.pem --key key.pem https://host.example.com/
   ```

   **MacOS**:
   ```
   curl -v --cert certificate.p12:[password] https://host.example.com/
   ```

   Looking through the verbose output, you can see we are sending a client
   certificate and Ambassador is validating it. 

   If you need further proof, simply create a new set of certificates and 
   try sending the curl with those. You will see Ambassador deny the request.
