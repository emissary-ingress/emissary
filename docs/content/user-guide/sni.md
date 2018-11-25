# Server Name Indication (SNI)

Ambassador lets you supply separate TLS certificates for different domains, instead of using a single TLS certificate for all domains. This allows Ambassador to serve multiple secure connections on the same IP address without requiring all websites to use the same certificate. Ambassador supports this use case through its support of Server Name Indication, an extension to the TLS protocol.

Note: SNI is only available in the [0.50 early access release](early-access.md).

## Configuring SNI

1. Create a TLS certificate, and store the secret in a Kubernetes secret.
    ```console
    kubectl create secret tls <secret name> --cert <path to the certificate chain> --key <path to the private key>
    ```

2. Create a `TLSContext` resource which points to the certificate, and lists all the different hosts in the certificate. Typically, these would be the Subject Alternative Names you will be using. If you're using a wildcard certificate, you can put in any host values that you wish to use.

    ```yaml
    apiVersion: ambassador/v0
    kind: TLSContext
    name: <TLSContext name>
    hosts: # list of hosts to match against>
    - host1
    - host2
    secret: <Kubernetes secret created in the first step>
    ```

   The `TLSContext` resource is typically added to the main Ambassador `service` where global Ambassador configuration is typically stored.

3. Create additional `TLSContext` resources pointing to additional certificates as necessary.

4. Configure the global TLS configuration (e.g., `redirect_cleartext_from`) in the `tls` module. The `tls` configuration applies to all `TLSContext` resources. For more information on global TLS configuration, see the [reference section on TLS](/reference/core/tls).

## Using SNI

SNI is designed to be configured on a per-mapping basis. This enables application developers or service owners to individually manage how their service gets exposed over TLS. To use SNI, specify your SNI host in the `mapping` resource, e.g.,

    ```yaml
    apiVersion: ambassador/v0
    kind:  Mapping
    name:  example-mapping
    prefix: /example/
    service: example.com:80
    host: <SNI host>
    ```
Ambassador will check if any of the `TLSContext` resources have a matching host, and if it finds one, SNI configuration will be applied to that mapping.

Note that if the mapping does not have the `host` field, all valid SNI configurations will be applied to the given mapping.

## Examples

#### Multiple certificates

In this configuration:

* Requests with `Host: internal.example.com` header set hitting `/httpbin/` prefix get internal TLS certificates.
* Requests with `Host: external.example.com` header set hitting `/httpbin/` prefix get external TLS certificates.
    

Note that the `TLSContext` and `Mapping` objects are on the same `Service` for illustrative purposes; more typically they would be managed separately as noted above.
    
```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind:  Mapping
      name:  httpbin-internal
      prefix: /httpbin/
      service: httpbin.org:80
      host_rewrite: httpbin.org
      host: internal.example.com
      ---
      apiVersion: ambassador/v0
      kind:  Mapping
      name:  httpbin-external
      prefix: /httpbin/
      service: httpbin.org:80
      host_rewrite: httpbin.org
      host: external.example.com
      ---
      apiVersion: ambassador/v0
      kind: TLSContext
      name: internal-context
      hosts:
      - internal.example.com
      secret: internal-secret
      ---
      apiVersion: ambassador/v0
      kind: TLSContext
      name: external-context
      hosts:
      - external.example.com
      secret: external-secret
  name: httpbin
spec:
  ports:
  - port: 80
    targetPort: 80
```
    

#### Multiple mappings with a fallback

In this configuration:

* Requests with `Host: host.httpbin.org` header set hitting `/httpbin/` prefix get httpbin TLS certificates.
* Requests with `Host: host.mockbin.org` header set hitting `/mockbin/` prefix get mockbin TLS certificates
* The `frontend` mapping will be accessible via both via `host.httpbin.org` and `host.mockbin.org`
       
```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind:  Mapping
      name:  httpbin
      prefix: /httpbin/
      service: httpbin.org:80
      host_rewrite: httpbin.org
      host: host.httpbin.org
      ---
      apiVersion: ambassador/v0
      kind:  Mapping
      name:  mockbin
      prefix: /mockbin/
      service: mockbin.org:80
      host_rewrite: mockbin.org
      host: host.mockbin.org
      ---
      apiVersion: ambassador/v0
      kind: TLSContext
      name: httpbin
      hosts:
      - host.httpbin.org
      secret: httpbin-secret
      ---
      apiVersion: ambassador/v0
      kind: TLSContext
      name: mockbin
      hosts:
      - host.mockbin.org
      secret: mockbin-secret
      ---
      # This mapping gets all the available SNI configurations applied to it
      apiVersion: ambassador/v0
      kind:  Mapping
      name:  frontend
      prefix: /
      service: frontend
  name: httpbin
spec:
  ports:
  - port: 80
    targetPort: 80
```