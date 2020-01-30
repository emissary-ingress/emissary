# Server Name Indication (SNI)

Ambassador Edge Stack lets you supply separate TLS certificates for different domains, instead of using a single TLS certificate for all domains. This allows Ambassador Edge Stack to serve multiple secure connections on the same IP address without requiring all websites to use the same certificate. Ambassador Edge Stack supports this use case through its support of Server Name Indication, an extension to the TLS protocol.

## Configuring SNI

SNI gives you the ability to host multiple domains behind the Ambassador Edge Stack and use different TLS certificates for each domain. It is designed to be configured on a per-mapping basis, enabling application developers or service owners to individually manage how their service gets exposed over TLS.

To use SNI, you simply need to:

1. Create a `TLSContext` for the domain

    ```yaml
    apiVersion: getambassador.io/v2
    kind: TLSContext
    metadata:
      name: example-tls
    spec:
      hosts: 
      - example.com
      secret: example-cert
    ```

2. Configure the `host` value on `Mapping`s associated with that domain

    ```yaml
    apiVersion: getambassador.io/v2
    kind:  Mapping
    metadata:
      name:  example-mapping
    spec:
      prefix: /example/
      service: example-service
      host: example.com
    ```

Ambassador Edge Stack will check if any of the `TLSContext` resources have a matching host, and if it finds one, SNI configuration will be applied to that mapping. 

**Note**: If a `Mapping` does not specify a `host`, Ambassador Edge Stack will interpret it as `hosts: "*"` meaning that `Mapping` will be available for all domains.

## Examples

### Multiple Certificates

In this configuration:

* Requests with `Host: internal.example.com` header set hitting `/httpbin/` prefix get internal TLS certificates.
* Requests with `Host: external.example.com` header set hitting `/httpbin/` prefix get external TLS certificates.    

Note that the `TLSContext` and `Mapping` objects are on the same `Service` for illustrative purposes; more typically they would be managed separately as noted above.

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  httpbin-internal
spec:
  prefix: /httpbin/
  service: httpbin.org:80
  host_rewrite: httpbin.org
  host: internal.example.com
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  httpbin-external
spec:
  prefix: /httpbin/
  service: httpbin.org:80
  host_rewrite: httpbin.org
  host: external.example.com
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: internal-context
spec:
  hosts:
  - internal.example.com
  secret: internal-secret
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: external-context
spec:
  hosts:
  - external.example.com
  secret: external-secret
```

### Multiple Mappings with a Fallback

In this configuration:

* Requests with `Host: host.httpbin.org` header set hitting `/httpbin/` prefix get httpbin TLS certificates.
* Requests with `Host: host.mockbin.org` header set hitting `/mockbin/` prefix get mockbin TLS certificates
* The `frontend` mapping will be accessible via both `host.httpbin.org` and `host.mockbin.org`

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  httpbin
spec:
  prefix: /httpbin/
  service: httpbin.org:80
  host_rewrite: httpbin.org
  host: host.httpbin.org
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  mockbin
spec:
  prefix: /mockbin/
  service: mockbin.org:80
  host_rewrite: mockbin.org
  host: host.mockbin.org
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: mockbin
spec:
  hosts:
  - host.mockbin.org
  secret: mockbin-secret
---
apiVersion: getambassador.io/v2
kind: TLSContext
metadata:
  name: httpbin
spec:
  hosts:
  - host.httpbin.org
  secret: httpbin-secret
---
# This mapping gets all the available SNI configurations applied to it
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: frontend
spec:
  prefix: /
  service: frontend
```
