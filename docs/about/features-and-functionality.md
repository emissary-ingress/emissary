---
layout: doc
weight: 1
title: "Features"
categories: about
---

Ambassador is an API Gateway for microservices built on [Envoy](https://lyft.github.io/envoy/). Key features in Ambassador include:

* Ability to flexibly map public URLs to services running inside a Kubernetes cluster
* Simple setup and configuration
* Integrated monitoring

Ambassador is built around the idea of mapping _resources_ (in the REST sense) to _services_ (in the Kubernetes sense). A `resource` is identified by a URL prefix -- for example, you might declare that any URL beginning with `/user/` identifies a "user" resource. A `service` is code running in Kubernetes that can handle the resource you want to map.

At present, a resource can be mapped to only one service, but the same service can be used behind as many different resources as you want. There's no hard limit to the number of mappings Ambassador can handle (though eventually you'll run out of memory).

### CAVEATS

Ambassador is ALPHA SOFTWARE. In particular, in version 0.7.0, there is no authentication mechanism, so anyone who can reach the administrative interface can map or unmap resources -- great for self service, of course, but possibly dangerous. For this reason, the administrative requires a Kubernetes port-forward.

Ambassador is under active development; check frequently for updates, and please file issues for things you'd like to see!

