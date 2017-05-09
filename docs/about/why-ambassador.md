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

Ambassador is an API Gateway for microservices. Its purpose in life is to sit in between your microservices application and anything that uses your application, routing and mediating requests to the various microservices. 

At first glance, that may seem to be pure overhead. But while people seem to get that the microservices architecture is helpful for moving more quickly, and for increasing the leverage that a single person can bring to bear on a problem, it's still often a challenge to go from a world where everything is centralized to the distributed world of microservices applications. 

Back in the world of the monolith, the nature of the application provided a centralized clearinghouse not just for deployment and testing, but for a host of other things:

- authentication
- authorization
- routing requests to the correct service
- maintaining health information for all your services

Authentication and authorization can be particularly unpleasant in the distributed world. Checking these at every call to every microservice is appropriate in some situations, but can be expensive both in development and at runtime. Checking only at the perimeter works really well in many other situations, but you have to have a perimeter -- and that's where the API gateway comes in. It's the perfect place to stand to easily manage not just routing, but also authentication, authorization, stats collection, performance measurements, the whole nine yards.

There are a stack of other API gateways out there, of course. The [Amazon API Gateway](https://aws.amazon.com/api-gateway/) is a hosted Gateway that runs in Amazon. [Traefik](https://traefik.io/), [NGINX](http://nginx.org/), [Kong](https://getkong.org/), or [HAProxy](http://www.haproxy.org/) are all open source options. And of course, there's [Envoy](lyft.github.io/envoy/), which is interesting because it provides the reverse proxy semantics you need to implement an API Gateway, and it also [supports the features](https://www.datawire.io/guide/traffic/getting-started-lyft-envoy-microservices-resilience/) you need for distributed architecture. (In fact, another project of note is [Istio](https://istio.io), which is also built on Envoy, but is designed to give you a full-blown services mesh rather than focusing on the API gateway case.)

As we explored, though, we found that none of these systems really gave us the elegance and low friction we wanted -- so we decided to build it ourselves, using Envoy for the heavy lifting. 

Datawire's Ambassador is an easy-to-deploy, self-service API gateway, meant to allow all of your microservices to rely on the gateway to correctly handle all the foundational capabilities above. Since Ambassador is built atop Envoy, the usage Envoy gets in the real world gives us confidence in its production-worthiness. It has built-in support for authentication with TLS client certificates, and it has built-in support for using `statsd` to collect statistics and ease monitoring of the service mesh as a whole. 

Ambassador makes all of those things easy to configure and use, and of course we'll be extending Ambassador to take advantage more and more of Envoy's functionality as we go.
Check out the [Ambassador roadmap](roadmap.md) for more.
