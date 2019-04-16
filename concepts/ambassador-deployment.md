# Ambassador Deployment Architecture

Ambassador can be deployed in a variety of configurations. The specific configuration depends on your data center.

## Public Cloud

If you're using a public cloud provider such as Amazon, Azure, or Google, Ambassador can be deployed directly to a Kubernetes cluster running in the data center. Traffic is routed to Ambassador via a cloud-managed load balancer such as an Amazon Elastic Load Balancer or Google Cloud Load Balancer. Typically, this load balancer is transparently managed by Kubernetes in the form of the `LoadBalancer` service type. Ambassador then routes traffic to your services running in Kubernetes.

## Private Data Center

In a private data center, Ambassador is deployed on the Kubernetes cluster. Instead of exposing Ambassador via the `LoadBalancer` service type, Ambassador is exposed as a `NodePort`. Traffic is sent to a specific port on any of the nodes in the cluster, which route the traffic to Ambassador, which then routes the traffic to your services running in Kubernetes. In addition, you'll need to deploy a separate load balancer to route traffic from your core routers to Ambassador. [MetalLB](https://metallb.universe.tf/) is an open source external load balancer for Kubernetes designed for this problem. Other options are traditional TCP load balancers such as F5 or Citrix Netscaler.

## Hybrid data center

Many data centers include services that are running outside of Kubernetes on bare metal or virtual machines. In order for Ambassador to route to services both inside and outside of Kubernetes, Ambassador needs the real-time network location of all services. This problem is known as "service discovery", and Ambassador supports using [Consul](https://www.consul.io). Services in your data center register themselves with Consul, and Ambassador uses Consul-supplied data to dynamically route requests to available services.