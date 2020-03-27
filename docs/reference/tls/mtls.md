# Mutual TLS (mTLS)

Ambassador Edge Stack can be configured to both provide certificates from upstream services, and to validate them. This behavior is called mutual TLS (mTLS) and is commonly done when using a service mesh to enforce end-to-end TLS for all services in your cluster.

To configure mTLS between Ambassador Edge Stack and your upstream services, you need to create a `TLSContext` with certificates that are signed by the Certificate Authority (CA) of your upstream service.

Below are examples of how to configure Ambassador Edge Stack to do mTLS with two popular service meshes, Istio and Consul Connect.

## Istio mTLS

Istio stores its TLS certificates as Kubernetes secrets by default, so accessing them is a matter of YAML configuration changes.

1. Load Istio's TLS certificates

Istio creates and stores its TLS certificates in Kubernetes secrets. In order to use those secrets you can set up a `TLSContext` to read directly from Kubernetes:

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: TLSContext
   metadata:
     name: istio-upstream
   spec:
     secret: istio.default
     secret_namespacing: False
   ```

Please note that if you are using RBAC you may need to reference the `istio` secret for your service account, e.g. if your service account is `ambassador` then your target secret should be `istio.ambassador`. See the [Ambassador Edge Stack with Istio](../../../user-guide/with-istio#istio-mutual-tls) documentation for an example with more information.

2. Configure Ambassador Edge Stack to use this `TLSContext` when making connections to upstream services

   The `tls` attribute in a `Mapping` configuration tells Ambassador Edge Stack to use the `TLSContext` we created above when making connections to upstream services:

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: Mapping
   metadata:
     name: productpage
   spec:
     prefix: /productpage/
     rewrite: /productpage
     service: https://productpage:9080
     tls: istio-upstream
   ```

Ambassador Edge Stack will now use the certificate stored in the secret to originate TLS to Istio-powered services. See the [Ambassador Edge Stack with Istio](../../../user-guide/with-istio#istio-mutual-tls) documentation for an example with more information.

## Consul mTLS

Since Consul does not expose TLS Certificates as Kubernetes secrets, we will need a way to export those from Consul.

1. Install the Ambassador Edge Stack Consul connector.

   ```
   kubectl apply -f https://www.getambassador.io/yaml/consul/ambassador-consul-connector.yaml
   ```

   This will grab the certificate issued by Consul CA and store it in a Kubernetes secret named `ambassador-consul-connect`. It will also create a Service named `ambassador-consul-connector` which will configure the following `TLSContext`:

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: TLSContext
   metadata:
     name: ambassador-consul
   spec:
     hosts: []
     secret: ambassador-consul-connect
   ```

2. Tell Ambassador to use the `TLSContext` when proxying requests by setting the `tls` attribute in a `Mapping`

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: Mapping
   metadata:
     name: qotm-mtls
   spec:
     prefix: /qotm-consul-mtls/
     service: https://qotm-proxy
     tls: ambassador-consul
   ```

Ambassador Edge Stack will now use the certificates loaded into the `ambassador-consul` `TLSContext` when proxying requests with `prefix: /qotm-consul-mtls`. See the [Consul example](../../../user-guide/consul#encrypted-tls) for an example configuration.

**Note:** The Consul connector can be configured with the following environment variables. The defaults will be best for most use-cases.

| Environment Variable | Description | Default |
| -------------------- | ----------- | ------- |
| \_AMBASSADOR\_ID        | Set the Ambassador ID so multiple instances of this integration can run per-Cluster when there are multiple Ambassadors (Required if `AMBASSADOR_ID` is set in your Ambassador deployment) | `""` |
| \_CONSUL\_HOST          | Set the IP or DNS name of the target Consul HTTP API server | `127.0.0.1` |
| \_CONSUL\_PORT          | Set the port number of the target Consul HTTP API server | `8500` |
| \_AMBASSADOR\_TLS\_SECRET\_NAME | Set the name of the Kubernetes `v1.Secret` created by this program that contains the Consul-generated TLS certificate. | `$AMBASSADOR_ID-consul-connect` |
| \_AMBASSADOR\_TLS\_SECRET\_NAMESPACE | Set the namespace of the Kubernetes `v1.Secret` created by this program. | (same Namespace as the Pod running this integration) |
