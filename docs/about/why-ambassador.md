---
layout: doc
weight: 1
title: "Why Ambassador?"
categories: about
---

<link rel="stylesheet" href="{{ "/css/mermaid.css" | prepend: site.baseurl }}">
<script src="{{ "/js/mermaid.min.js" | prepend: site.baseurl }}"></script>
<script>mermaid.initialize({
   startOnLoad: true,
   cloneCssStyles: false,
 });
</script>

Ambassador is an API Gateway for microservices. Its purpose in life is to sit in between your microservices application and your users, which might seem like the last place you want to insert extra moving parts. 

People seem to get that the microservices architecture is helpful for moving more quickly, and for increasing the leverage that a single person can bring to bear on a problem, but going from a world where everything is centralized to the distributed microservices world is often still challenging.

The challenge tends to start with deployment -- one monolith is often bad enough to deploy, much less many microservices! At Datawire, we've used Kubernetes to get a handle on this issue: getting a new microservice up and running in a Kubernetes cluster is usually a matter of minutes for us now. 

That's just the start, though. The monolith - by its nature - is a centralized clearinghouse not just for deployment and testing, but for a host of other things:

- authentication
- authorization
- routing requests to the correct service
- maintaining health information for all your services

Authentication and authorization can be particularly unpleasant in the distributed world. Checking these at every call to every microservice is appropriate in some situations, but can be expensive both in development and at runtime. Checking only at the perimeter works really well in many other situations, but you have to have a perimeter! 

Interestingly enough, a perimeter isn't actually hard in the Kubernetes world.

If you define your services as type `NodePort`, they're reachable within your cluster, but not from outside. You can then have an API gateway proxy to your services -- and the API gateway is the perfect place to stand to easily manage not just routing, but also authentication, authorization, stats collection, performance measurements, the whole nine yards.

We decided to make it easy to do exactly this. Datawire's Ambassador is an easy-to-deploy, self-service API gateway, so that all of your microservices can rely on the gateway to correctly handle all the foundational capabilities above.

Ambassador uses Lyft's Envoy for the heavy lifting. The usage Envoy gets in the real world gives us confidence in its production-worthiness, it has built-in support for authentication with TLS client certs, and it has built-in support for using statsd to collect statistics and ease monitoring of the service mesh as a whole. Ambassador makes all of those things easy to configure and use, and of course we'll be extending Ambassador to take advantage more and more of Envoy's functionality as we go.


: a façade that sits between the consumers and producers of an API, and implements cross-cutting functionality such as authentication, monitoring, and traffic management so that your microservices can remain blissfully unaware of these details. Additionally, the API gateway provides a single rendezvous point for all of the microservices that make up an application, allowing consumers to use the application without worrying about the details of which microservice handles which functionality exactly.

For a really concrete example of why this can be useful, consider migrating from a monolith to microservices (Datawire gets asked about this a lot).


There are dozens of different options for API Gateways, depending on your requirements. The [Amazon API Gateway](https://aws.amazon.com/api-gateway/) is a hosted Gateway that runs in Amazon. [Traefik](https://traefik.io/),
[NGINX](http://nginx.org/), [Kong](https://getkong.org/), or [HAProxy](http://www.haproxy.org/) are all open source options.

And of course, there's [Envoy](lyft.github.io/envoy/). Envoy is interesting because not only does it provide the reverse proxy semantics you need to implement an API Gateway, but it also [supports the features](https://www.datawire.io/guide/traffic/getting-started-lyft-envoy-microservices-resilience/) you need for distributed architecture. (In fact, another project of note is [Istio](https://istio.io), which is also built on Envoy, but is designed to give you a full-blown services mesh rather than focusing on the API gateway case.)

When we started digging into Envoy, though, we found that deploying it as a working microservice API gateway was kind of tricky. To get everything going, we had to

1. deploy our services
2. deploy Envoy
3. deploy Envoy's SDS
4. configure Envoy to use our SDS
5. configure Envoy to relay requests for our services

and of those five steps, only one has to do with our real application -- the other _four_ have to do with Envoy. That seemed backward, and that's exactly the problem that Ambassador is designed to solve.





## Introducing Ambassador

While you could do all these steps manually, we thought there should be an easier way, so wrote Ambassador, an Envoy-based API Gateway. So, we're going to pick up where we left off with our simple application, and get it running in Kubernetes on AWS.

So what does Ambassador do?

* Simplifies setting up Envoy as an API Gateway, so you don't need to worry about steps 2 - 5 above
* Self-service configuration for your microservices, so you don't need to edit Envoy configuration by hand every time a service API changes
* Easily accessible health monitoring and statistics

