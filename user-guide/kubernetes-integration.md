# Kubernetes Integration: Ambassador Edge Stack Architecture Overview

## Ambassador Edge Stack is a control plane

Ambassador Edge Stack is a specialized [control plane for Envoy Proxy](https://blog.getambassador.io/the-importance-of-control-planes-with-service-meshes-and-front-proxies-665f90c80b3d). In this architecture, Ambassador Edge Stack translates configuration (in the form of Kubernetes annotations) to Envoy configuration. All actual traffic is directly handled by the high-performance [Envoy Proxy](https://www.envoyproxy.io).

![Architecture](/doc-images/ambassador-arch.png)

## Details

When a user applies a Kubernetes manifest containing Ambassador Edge Stack annotations, the following steps occur:

1. Ambassador Edge Stack is asynchronously notified by the Kubernetes API of the change.
2. Ambassador Edge Stack translates the configuration into an abstract intermediate representation (IR).
3. An Envoy configuration file is generated from the IR.
4. The Envoy configuration file is validated by Ambassador Edge Stack (using Envoy in validation mode).
5. Assuming the file is valid configuration, Ambassador Edge Stack uses Envoy's [Aggregated Discovery Service](https://www.envoyproxy.io/docs/envoy/latest/configuration/overview/v2_overview#aggregated-discovery-service) to deploy the new configuration and properly drain connections.

## Scaling and availability

Ambassador relies on Kubernetes for scaling, high availability, and persistence. All Ambassador Edge Stack configuration is stored directly in Kubernetes; there is no database. Ambassador Edge Stack is packaged as a single container that contains both the control plane and an Envoy Proxy instance. By default, Ambassador Edge Stack is deployed as a Kubernetes `deployment` and can be scaled and managed like any other Kubernetes deployment.

## Envoy Proxy

Ambassador Edge Stack closely tracks Envoy Proxy releases. A stable branch of Envoy Proxy is maintained that enables the team to cherry pick specific fixes into Ambassador Edge Stack.

<div style="border: thick solid gray;padding:0.5em"> 

Ambassador Edge Stack is a community supported product with 
[features](getambassador.io/features) available for free and 
limited use. For unlimited access and commercial use of
Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) 
for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
</p>
