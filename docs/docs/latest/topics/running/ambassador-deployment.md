# The Ambassador Edge Stack Deployment Architecture

The Ambassador Edge Stack can be deployed in a variety of configurations. The specific configuration depends on your data center.

## Public Cloud

If you're using a public cloud provider such as Amazon, Azure, or Google, the Ambassador Edge Stack can be deployed directly to a Kubernetes cluster running in the data center. Traffic is routed to the Ambassador Edge Stack via a cloud-managed load balancer such as an Amazon Elastic Load Balancer or Google Cloud Load Balancer. Typically, this load balancer is transparently managed by Kubernetes in the form of the `LoadBalancer` service type. The Ambassador Edge Stack then routes traffic to your services running in Kubernetes.

## On-Premise Data Center

In an on-premise data center, the Ambassador Edge Stack is deployed on the Kubernetes cluster. Instead of exposing it via the `LoadBalancer` service type, the Ambassador Edge Stack is exposed as a `NodePort`. Traffic is sent to a specific port on any of the nodes in the cluster, which route the traffic to the Ambassador Edge Stack, which then routes the traffic to your services running in Kubernetes. You'll also need to deploy a separate load balancer to route traffic from your core routers to Ambassador Edge Stack. [MetalLB](https://metallb.universe.tf/) is an open-source external load balancer for Kubernetes designed for this problem. Other options are traditional TCP load balancers such as F5 or Citrix Netscaler.

## Hybrid Data Center

Many data centers include services that are running outside of Kubernetes on virtual machines. For the Ambassador Edge Stack to route to services both inside and outside of Kubernetes, it needs the real-time network location of all services. This problem is known as "[service discovery](https://www.datawire.io/guide/traffic/service-discovery-microservices/)" and the Ambassador Edge Stack supports using [Consul](https://www.consul.io). Services in your data center register themselves with Consul, and the Ambassador Edge Stack uses Consul-supplied data to dynamically route requests to available services.

## Hybrid On-premise Data Center

The diagram below details a common network architecture for a hybrid on-premise data center. Traffic flows from core routers to MetalLB, which routes to the Ambassador Edge Stack running in Kubernetes. The Ambassador Edge Stack routes traffic to individual services running on both Kubernetes and VMs. Consul tracks the real-time network location of the services, which the Ambassador Edge Stack uses to route to the given services.

![Architecture](../../../images/consul-ambassador.png)
