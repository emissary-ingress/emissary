# Frequently Asked Questions

## General

### Why Ambassador Edge Stack?

Kubernetes shifts application architecture for microservices, as well as the
development workflow for a full-cycle development. Ambassador is designed for
the Kubernetes world with:

* Sophisticated traffic management capabilities (thanks to its use of [Envoy Proxy](https://www.envoyproxy.io)), such as load balancing, circuit breakers, rate limits, and automatic retries.
* API management capabilities such as a developer portal and OpenID Connect integration for Single Sign-On.
* A declarative, self-service management model built on Kubernetes Custom Resource Definitions, enabling GitOps-style continuous delivery workflows.

We've written about [the history of Ambassador](https://blog.getambassador.io/building-ambassador-an-open-source-api-gateway-on-kubernetes-and-envoy-ed01ed520844), [Why Ambassador In Depth](../why-ambassador), [Features and Benefits](../features-and-benefits) and about the [evolution of API Gateways](../../topics/concepts/microservices-api-gateways/).

### What's the difference between the Ambassador API Gateway and the Ambassador Edge Stack?

The Ambassador API Gateway was the name of the original open-source project. As the project evolved, we realized that the functionality we were building had extended far beyond an API Gateway. In particular, the Ambassador Edge Stack is intended to provide all the functionality you need at the edge -- hence, an "edge stack." This includes an API Gateway, ingress controller, load balancer, developer portal, and more.

### How is Ambassador Edge Stack licensed?

The core Ambassador Edge Stack is open source under the Apache Software License 2.0. The GitHub repository for the core is [https://github.com/datawire/ambassador](https://github.com/datawire/ambassador). Some additional features of the Ambassador Edge Stack (e.g., Single Sign-On) are not open source and available under a proprietary license.

### Can I use the add-on features for Ambassador Edge Stack for free?

Yes! The core functionality of the Ambassador Edge Stack is free and has no limits whatsoever. If you wish to use one of our additional, proprietary features such as Single Sign-On, you can get a free community license for up to 5 requests per second. Please contact [sales](/contact/) if you need more than 5 RPS.

For more details on core unlimited features and premium features, see the [editions page](/editions).

### How does Ambassador use Envoy Proxy?

Ambassador uses [Envoy Proxy](https://www.envoyproxy.io) as its core proxy. Envoy is an open-source, high-performance proxy originally written by Lyft. Envoy is now part of the Cloud Native Computing Foundation.

### Is Ambassador Edge Stack production ready?

Yes. Thousands of organizations, large and small, run Edge Stack in production.
Public users include Chick-Fil-A, ADP, Microsoft, NVidia, and AppDirect, among others.

### What is the performance of Edge Stack?

There are many dimensions to performance. We published a benchmark of [Ambassador performance on Kubernetes](/resources/envoyproxy-performance-on-k8s/). Our internal performance regressions cover many other scenarios; we expect to publish more data in the future.

### What's the difference between a service mesh (such as Istio) and Ambassador Edge Stack?

Service meshes focus on routing internal traffic from service to service
("east-west"). Ambassador focuses on traffic into your cluster ("north-south").
While both a service mesh and Ambassador can route L7 traffic, the reality is that
these use cases are quite different. Many users will integrate Ambassador with a
service mesh. Production customers of Ambassador have integrated with Consul,
Istio, and Linkerd2.

## Common Configurations

### How do I disable the 404 landing page?

Established users will want to better control 404 behavior both for usability and 
security.  You can leverage the Mapping resource to implement this functionality to 
your cluster.  Ambassador Edge Stack users can use a 'catch-all' mapping using the '/' 
prefix in a mapping configuration.  The simplest mapping, described below, returns only 404 text.  
To use a custom 404 landing page, simply insert your service and remove the rewrite value.

```yaml
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: "404 fallback"
spec:
  prefix: "/"
  rewrite: "/404/" # This must not map to any existing prefix!
  service: localhost:8500
```

For more information on the Mapping resource, see [Advanced Mapping Configuration](../../topics/using/mappings).

### How do I disable the default Admin mappings?

In a production environment, public access to the console and admin endpoints is not an 
ideal situation.  To solve this, we will be using an Ambassador Module to remove the default 
mappings and create a new, host-based mapping to expose the Admin endpoint more securely.  The 
Ambassador module applies system-wide configuration settings for Ambassador to follow.

```yaml
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    diagnostics:
      enabled: false
```

After applying this module, the admin endpoint is no longer accessible from the outside world.  
We cannot, however, exclude actual administrators from this endpoint, so to create a more managed 
endpoint for them to use, create a mapping to expose the endpoint.

```yaml
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: admin-mapping
spec:
  host: admin.example.com
  prefix: /edge_stack/
  rewrite: /edge_stack_ui/edge_stack/
  service: localhost:8500
```

Now, administrators can connect to the admin console via hostname.  Additional [Mapping](../../topics/using/intro-mappings) and 
[Filter](../../topics/using/filters/index) settings can be appropriately configured to better control access to admin services.  To 
learn more about Ambassador Module configurations, see [Ambassador Module](../../topics/running/ambassador)

## Troubleshooting

### How do I get help for Edge Stack?

We have an online [Slack community](https://d6e.co/slack) with thousands of
users. We try to help out as often as possible, although we can't promise a
particular response time. If you need a guaranteed SLA, we also have commercial
contracts. [Contact sales](/contact/) for more information.

### What do I do when I get the error `no healthy upstream`?

This error means that Ambassador could not connect to your backend service.
Start by verifying that your backend service is actually available and
responding by sending an HTTP response directly to the pod. Then, verify that
Ambassador is routing by deploying a test service and seeing if the mapping
works. Then, verify that your load balancer is properly routing requests to
Ambassador. In general, verifying each network hop between your client and
backend service is critical to finding the source of the problem.
