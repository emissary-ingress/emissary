# Server Name Indication (SNI)

Ambassador lets you supply separate TLS certificates for different domains, instead of using a single TLS certificate for all domains. This allows Ambassador to serve multiple secure connections on the same IP address without requiring all websites to use the same certificate.

Note: Configuring SNI is only available in [early access releases](early-access.md).

## Configuring SNI

1. Specify the host in the `host` field of the mapping you want to apply the SNI configuration to
    ```yaml
    apiVersion: ambassador/v0
    kind:  Mapping
    name:  example-mapping
    prefix: /example/
    service: example.com:80
    host: <SNI host>
    ```
    
2. Store TLS certificates in a Kubernetes secret
    ```console
    kubectl create secret tls <secret name> --cert <path to the certificate chain> --key <path to the private key>
    ```

3. Create a `TLSContext` resource which looks like -

    ```yaml
    apiVersion: ambassador/v0
    kind: TLSContext
    name: <TLSContext name>
    hosts: # list of hosts to match against>
    - host1
    - host2
    secret: <Kubernetes secret created in the first step>
    ```

That's all! Ambassador will check if any of the `TLSContext` resources have a matching host, and if it finds one, SNI configuration will be applied to that mapping.

Note: If `TLSContext` resources are configured, then all the mappings that do not have the `host` field set, will get all SNI configurations applied to them.

### Examples

1. Requests with `Host: internal.example.com` header set hitting `/httpbin/` prefix get internal TLS certificates.

    Requests with `Host: external.example.com` header set hitting `/httpbin/` prefix get external TLS certificates
    
    An example Kubernetes service will look like the following, with TLS certificates stored in respective Kubernetes secrets -
    
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
    
2. Requests with `Host: host.httpbin.org` header set hitting `/httpbin/` prefix get httpbin TLS certificates.

    Requests with `Host: host.mockbin.org` header set hitting `/mockbin/` prefix get mockbin TLS certificates
    
    All other mappings with any or no `Host` header set get all available SNI configurations applied to them (httpbin and mockin in this case)
    
    An example Kubernetes service will look like the following, with TLS certificates stored in respective Kubernetes secrets -
    
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

3. All mappings get a given SNI configuration.

    An example Kubernetes service will look like the following, with TLS certificates stored in respective Kubernetes secrets -
    
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
          ---
          apiVersion: ambassador/v0
          kind:  Mapping
          name:  mockbin
          prefix: /mockbin/
          service: mockbin.org:80
          host_rewrite: mockbin.org
          ---
          apiVersion: ambassador/v0
          kind: TLSContext
          name: example-context
          hosts:
          - example.com
          - www.example.com
          secret: example-secret
      name: httpbin
    spec:
      ports:
      - port: 80
        targetPort: 80
    ```