Ambassador [![Build Status](https://travis-ci.org/datawire/ambassador.png)](https://travis-ci.org/datawire/ambassador)
==========

**If you are just trying to use Ambassador, don't clone this repo! Go to https://www.getambassador.io/ instead!!**

Ambassador is an open source Kubernetes-native API Gateway built on [Envoy](https://www.envoyproxy.io), designed for microservices. Key features include:

* Self-service mapping of public URLs to services running inside a Kubernetes cluster via Kubernetes annotations
* Flexible canary deployments
* Kubernetes-native architecture
* First class gRPC and HTTP/2 support
* Istio integration
* Authentication
* Integrated diagnostics
* Robust TLS support, including TLS client-certificate authentication
* Simple setup and configuration
* Integrated monitoring

Ambassador also takes full advantage of Kubernetes for availability and scalability, dramatically simplifying the architecture of Ambassador.

To get started, visit https://www.getambassador.io or join our [Gitter channel](https://gitter.im/datawire/ambassador).

Building
========

**If you are just trying to use Ambassador, you don't need to build anything! Go to https://www.getambassador.io/ instead!!**

To _build_ Ambassador from source, check out [the build guide](BUILDING.md).
