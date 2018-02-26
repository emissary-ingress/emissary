Ambassador [![Build Status](https://travis-ci.org/datawire/ambassador.png)](https://travis-ci.org/datawire/ambassador)
==========

[Ambassador](https://www.getambassador.io) is an open source Kubernetes-native API Gateway built on [Envoy](https://www.envoyproxy.io), designed for microservices. Key features include:

* Self-service configuration, via Kubernetes annotations
* First class gRPC and HTTP/2 support
* Support for CORS, timeouts, rate-limiting, weighted round robin (canary), and more
* Istio integration
* Authentication
* Robust TLS support, including TLS client-certificate authentication

Architecture
============

Ambassador deploys the Envoy Proxy for L7 traffic management. Configuration of Ambassador is via Kubernetes annotations. Ambassador relies on Kubernetes for scaling and resilience. For more on Ambassador's architecture and motivation, read [this blog post](https://blog.getambassador.io/building-ambassador-an-open-source-api-gateway-on-kubernetes-and-envoy-ed01ed520844).

Getting Started
===============

You can get Ambassador up and running in less than a minute by running it locally with Docker. Follow the instructions here: https://www.getambassador.io#get-started.

For production usage, Ambassador runs in Kubernetes. For a Kubernetes deployment, follow the instructions at https://www.getambassador.io/user-guide/getting-started.

Community
=========

Ambassador is an open source project, and welcomes any and all contributors. To get started:

* Join our [Gitter channel](https://gitter.im/datawire/ambassador)
* Read the [developer guide](BUILDING.md)
* Check out the [Ambassador documentation](https://www.getambassador.io/about/why-ambassador)

If you're interested in contributing, here are some ways:

* Write a blog post for [our blog](https://blog.getambassador.io)
* Investigate an [open issue](https://github.com/datawire/ambassador/issues)
* Add [more tests](https://github.com/datawire/ambassador/tree/develop/end-to-end)
