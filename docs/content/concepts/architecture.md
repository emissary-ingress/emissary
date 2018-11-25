# Ambassador Architecture

## Ambassador is a control plane

Ambassador is a specialized [control plane for Envoy Proxy](https://blog.getambassador.io/the-importance-of-control-planes-with-service-meshes-and-front-proxies-665f90c80b3d). In this architecture, Ambassador translates configuration (in the form of Kubernetes annotations) to Envoy configuration. All actual traffic is directly handled by the high-performance [Envoy Proxy](https://www.envoyproxy.io).

![Architecture](/images/ambassador-arch.png)

## Details

When a user applies a Kubernetes manifest containing Ambassador annotations, the following steps occur:

1. Ambassador is asynchronously notified by the Kubernetes API of the change.
2. Ambassador translates the configuration into an abstract intermediate representation (IR).
3. An Envoy configuration file is generated from the IR.
4. The Envoy configuration file is validated by Ambassador (using Envoy in validation mode).
5. Assuming the file is valid configuration, Ambassador uses Envoy's [hot restart mechanism](https://blog.envoyproxy.io/envoy-hot-restart-1d16b14555b5) to deploy the new configuration and properly drain connections.

## Scaling and availability

Ambassador relies on Kubernetes for scaling, high availability, and persistence. All Ambassador configuration is stored directly in Kubernetes; there is no database. Ambassador is packaged as a single container that contains both the control plane and an Envoy Proxy instance. By default, Ambassador is deployed as a Kubernetes `deployment` and can be scaled and managed like any other Kubernetes deployment.

## Envoy Proxy

Ambassador closely tracks Envoy Proxy releases. A stable branch of Envoy Proxy is maintained that enables the team to cherry pick specific fixes into Ambassador.