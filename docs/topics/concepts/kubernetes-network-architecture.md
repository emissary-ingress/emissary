# Kubernetes Network Architecture

## Kubrernetes has its own isolated network

Each Kubernetes cluster provides its own isolated network namespace. This approach has a number of benefits. For example, each pod can be easily accessed with its own IP address. One of the consequences of this approach, however, is that a network bridge is required in order to route traffic from outside the Kubernetes cluster to services inside the cluster.

## Routing traffic to your Kubernetes cluster

While there are a number of techniques for routing traffic to a Kubernetes cluster, by far the most common and popular method involves deploying an in-cluster edge proxy / ingress controller along with an external load balancer. In this architecture, the network topology looks like this:

```
Load balancer --> Edge Proxy / Ingress Controller --> Kubernetes services/pods
```

Each of the components in this topology is discussed in further detail below.

### Load balancer

The load balancer is deployed outside of the Kubernetes cluster. Typically, the load balancer also has one or more static IP addresses assigned to it. A DNS entry is then created to map a domain name (e.g., example.com) to the static IP address.

Cloud infrastructure providers such as Amazon Web Services, Azure, Digital Ocean, and Google make it easy to create these load balancers directly from Kubernetes. This is done by creating a Kubernetes service of `type: LoadBalancer`. When this service is created, the cloud provider will use the metadata contained in the Kubernetes service definition to provision a load balancer.

If the Kubernetes cluster is deployed in a private data center, an external load balancer is still generally used. Provisioning of this load balancer usually requires the involvement of the data center operations team.

In both the private data center and cloud infrastructure case, the external load balancer should be configured to point to the edge proxy.

### Edge Proxy / Ingress Controller

The Edge Proxy is typically a Layer 7 proxy that is deployed directly in the cluster. The core function of the edge proxy is to accept incoming traffic from the external load balancer and route the traffic to Kubernetes services. The edge proxy should be configured using Kubernetes manifests. This enables a common management workflow for both the edge proxy and Kubernetes services.

The most popular approach to configuring edge proxies is with the Kubernetes ingress resource. When an edge proxy can process ingress resources, it is called an ingress controller. Not all edge proxies are ingress controllers (because they can't process ingress resources), but all ingress controllers are edge proxies.

The ingress resource is a Kubernetes standard. As such, it is a lowest common denominator resource. In practice, users find that the ingress resource is insufficient in scope to address the requirements for edge routing. Semantics such as TLS termination, redirecting to TLS, timeouts, rate limiting, and authentication are all beyond the scope of the ingress resource.

The Ambassador Edge Stack can function as an ingress controller (i.e., it reads ingress resources), although it also includes many other capabilities that are beyond the scope of the ingress specification. Most Ambassador Edge Stack users find that the various additional capabilities of Ambassador are essential, and end up using Ambassador's extensions to the ingress resource, instead of using ingress resources themselves.

### Kubernetes services and pods

Each instance of your application is deployed in a Kubernetes pod. As the workload on your application increases or decreases, Kubernetes can automatically add or remove pods. A Kubernetes _service_ represents a group of pods that comprise the same version of a given application. Traffic can be routed to the pods via a Kubernetes service, or it can be routed directly to the pods.

When traffic is routed to the pods via a Kubernetes service, Kubernetes uses a built-in mechanism called `kube-proxy` to load balance traffic between the pods. Due to its implementation, `kube-proxy` is a Layer 4 proxy, i.e., it load balances at a connection level. For particular types of traffic such as HTTP/2 and gRPC, this form of load balancing is particularly problematic as it can easily result in a very uneven load balancing configuration.

Traffic can also be routed directly to pods, bypassing the Kubernetes service. Since pods are much more ephemeral than Kubernetes services, this approach requires an edge proxy that is optimized for this use case. In particular, the edge proxy needs to support real-time discovery of pods, and be able to dynamically update pod locations without downtime.

The Ambassador Edge Stack supports routing both to Kubernetes services and directly to pods.

## Further reading

* [Kubernetes Ingress 101](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d)
* [Envoy Proxy Performance on Kubernetes](/resources/envoyproxy-performance-on-k8s/)