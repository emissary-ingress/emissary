# Ambassador Architecture

## Ambassador is a control plane

Ambassador is a specialized [control plane for Envoy Proxy](https://blog.getambassador.io/the-importance-of-control-planes-with-service-meshes-and-front-proxies-665f90c80b3d). In this architecture, Ambassador translates configuration (in the form of Kubernetes annotations) to Envoy configuration. All actual traffic is directly handled by the high-performance [Envoy Proxy](https://www.envoyproxy.io).

![Architecture](/doc-images/ambassador-arch.png)

## Details

1. Service owner defines configuration in Kubernetes manifests.
2. When the manifest is applied to the cluster, the Kubernetes API notifies Ambassador of the change.
3. Ambassador parses the change and transforms the configuration to a semantic intermediate representation. Envoy configuration is generated from this IR.
4. The new configuration is passed to Envoy via the gRPC-based Aggregated Discovery Service (ADS) API.
5. Traffic flows through the reconfigured Envoy, without dropping any connections.

## Scaling and availability

Ambassador relies on Kubernetes for scaling, high availability, and persistence. All Ambassador configuration is stored directly in Kubernetes; there is no database. Ambassador is packaged as a single container that contains both the control plane and an Envoy Proxy instance. By default, Ambassador is deployed as a Kubernetes `deployment` and can be scaled and managed like any other Kubernetes deployment.

## Envoy Proxy

Ambassador closely tracks Envoy Proxy releases. A stable branch of Envoy Proxy is maintained that enables the team to cherry pick specific fixes into Ambassador.