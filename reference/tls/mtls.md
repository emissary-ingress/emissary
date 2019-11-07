# Mutual TLS (mTLS)

Ambassador Edge Stack can be configured to both provide to and validate certificates from upstream services. This behavior is called mutual TLS (mTLS) and is a commonly done when using a service mesh to enforce end-to-end TLS for all services in your cluster.

To configure mTLS between Ambassador Edge Stack and your upstream services, you need to create a `TLSContext` with certificates that are signed by the Certificate Authority (CA) of your upstream service. 

Below are examples of how to configure Ambassador Edge Stack to do mTLS with two popular service meshes, Istio and Consul Connect.

## Istio mTLS

Istio stores it's TLS certificates as Kubernetes secrets by default, so accessing them is a matter of YAML configuration changes.

1. Load Istio's TLS certificates

Istio creates stores it's tls certificates in a form that Ambassador Edge Stack is currently unable to automatically read. Because of this, you will need to mount the `istio.default` secret in a volume in the Ambassador Edge Stack container. This is done by configuring a `volume` and `volumeMount` in the Ambassador Edge Stack deployment manifest.

   ```yaml
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: ambassador
    spec:
    ...
            volumeMounts:
              - mountPath: /etc/istiocerts/
                name: istio-certs
                readOnly: true
          restartPolicy: Always
          volumes:
          - name: istio-certs
            secret:
              optional: true
              secretName: istio.default
   ```

2. Create a `TLSContext` to load these certificates

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: TLSContext
   name: istio-upstream
   cert_chain_file: /etc/istiocerts/cert-chain.pem
   private_key_file: /etc/istiocerts/key.pem
   cacert_chain_file: /etc/istiocerts/root-cert.pem
   ```

3. Configure Ambassador Edge Stack to use this `TLSContext` when making connections to upstream services.

   The `tls` attribute in a `Mapping` configuration tell's Ambassador Edge Stack to use the `TLSContext` we created above when making connections to upstream services.

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: Mapping
   name: productpage_mapping
   prefix: /productpage/
   rewrite: /productpage
   service: https://productpage:9080
   tls: istio-upstream
   ```

Ambassador Edge Stack will now use the certificate stored in the `istio.default` secret to originate TLS to istio-powered services. See the [Ambassador Edge Stack with Istio](/user-guide/with-istio#istio-mutual-tls) documentation) for an example with more information.

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
   name: ambassador-consul
   hosts: []
   secret: ambassador-consul-connect
   ```

2. Tell Ambassador to use the `TLSContext` when proxying requests by setting the `tls` attribute in a `Mapping`

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: Mapping
   name: qotm_mtls_mapping
   prefix: /qotm-consul-mtls/
   service: https://qotm-proxy
   tls: ambassador-consul
   ```

Ambassador Edge Stack will now use the certificates loaded into the `ambassador-consul` `TLSContext` when proxying requests with `prefix: /qotm-consul-mtls`. See the [Consul example](/user-guide/consul#encrypted-tls) for an example configuration.

**Note:** The Consul connector can be configured with the following environment variables. The defaults will be best for most use-cases.

| Environment Variable | Description | Default |
| -------------------- | ----------- | ------- |
| \_AMBASSADOR\_ID        | Set the Ambassador ID so multiple instances of this integration can run per-Cluster when there are multiple Ambassadors (Required if `AMBASSADOR_ID` is set in your Ambassador deployment) | `""` |
| \_CONSUL\_HOST          | Set the IP or DNS name of the target Consul HTTP API server | `127.0.0.1` |
| \_CONSUL\_PORT          | Set the port number of the target Consul HTTP API server | `8500` |
| \_AMBASSADOR\_TLS\_SECRET\_NAME | Set the name of the Kubernetes `v1.Secret` created by this program that contains the Consul-generated TLS certificate. | `$AMBASSADOR_ID-consul-connect` |
| \_AMBASSADOR\_TLS\_SECRET\_NAMESPACE | Set the namespace of the Kubernetes `v1.Secret` created by this program. | (same Namespace as the Pod running this integration) |

<div style="border: thick solid gray;padding:0.5em"> 

Ambassador Edge Stack is a community supported product with 
[features](getambassador.io/features) available for free and 
limited use. For unlimited access and commercial use of
Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) 
for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
</p>
