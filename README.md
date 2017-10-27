Ambassador [![Build Status](https://travis-ci.org/datawire/ambassador.png)](https://travis-ci.org/datawire/ambassador)
==========

Ambassador is an open source Kubernetes-native API Gateway built on [Envoy](https://envoyproxy.github.io), designed for microservices. Key features include:

* Ability to flexibly map public URLs to services running inside a Kubernetes cluster
* Authentication
* Simple setup and configuration via a declarative YAML file
* Integrated monitoring
* All the load balancing, observability, and protocol support of Envoy

Ambassador also takes full advantage of Kubernetes for availability and scalability, dramatically simplifying the architecture of Ambassador.

To get started, visit https://www.getambassador.io, or join our [Gitter channel](https://gitter.im/datawire/ambassador).

Mapping
=======

Ambassador is built around the idea of mapping _resources_ (in the REST sense) to _services_ (in the Kubernetes sense). A `resource` is identified by a URL prefix -- for example, you might declare that any URL beginning with `/user/` identifies a "user" resource. A `service` is code running in Kubernetes that can handle the resource you want to map.

If you're on GKE
================

You'll need to configure RBAC. See:

https://cloud.google.com/container-engine/docs/role-based-access-control

What's in this repo
==================

**If you are just trying to use Ambassador, don't clone this repo! Go to https://www.getambassador.io/ instead!!**

To _build_ Ambassador from source, check out [the build guide](BUILDING.md).
