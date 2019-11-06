# Operator Guide

Unlike traditional API gateways, Ambassador Edge Stack has been designed to allow developers and operations to work independently from each other, with an operations team more focused towards the global deployment and configuration of the gateway. This section of the documentation focuses on the core functionality of Ambassador Edge Stack for operations and sysadmin teams.

# Why Should Operators or Sysadmins Use Ambassador Edge Stack?

Ambassador Edge Stack allows developers to manage individual service/API deployments, and frees time for operations to focus on global, cross-cutting concerns such as authentication, continuous delivery integration, rate limiting and tracing.

In more detail, Ambassador Edge Stack supports operations in the following ways:

* Is simple to [deploy and operate](/user-guide/kubernetes-integration), relying entirely on Envoy and Kubernetes for routing and scaling
* Has extensive support for [TLS termination](/user-guide/tls-termination) and redirects
* Integrated [diagnostics](/reference/statistics) and [tracing](/user-guide/tracing-tutorial) for troubleshooting
* Supports running multiple Ambassador Edge Stacks in a cluster, with different versions, simplifying upgrades and testing
* [Integrates with Istio](/user-guide/with-istio), if you need a service mesh

<div style="border: solid gray;padding:0.5em">

Ambassador Edge Stack is a community supported product with [features](getambassador.io/features) available for free and limited use. For unlimited access and commercial use of Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>

