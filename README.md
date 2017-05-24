Ambassador
==========

Ambassador is an API Gateway for microservices built on [Envoy](https://lyft.github.io/envoy/). Key features in Ambassador include:

* Ability to flexibly map public URLs to services running inside a Kubernetes cluster
* Simple setup and configuration
* Integrated monitoring
* All the load balancing, observability, and protocol support of Envoy

To get started, visit http://www.getambassador.io, or join our [Gitter channel](https://gitter.im/datawire/ambassador).

Mapping
=======

Ambassador is built around the idea of mapping _resources_ (in the REST sense) to _services_ (in the Kubernetes sense). A `resource` is identified by a URL prefix -- for example, you might declare that any URL beginning with `/user/` identifies a "user" resource. A `service` is code running in Kubernetes that can handle the resource you want to map.

What's in this repo
==================

If you clone this repository, you'll have access to multiple Kubernetes resource files:

- `ambassador-rest.yaml` defines the main Ambassador server itself;
- `ambassador-store.yaml` defines the persistent storage that Ambassador uses to remember which services are running;
- `ambassador-sds.yaml` defines the Envoy Service Discovery Service that Ambassador relies on; and finally,
- `ambassador.yaml` wraps up all of the above.

Additionally, you can choose either

- `ambassador-https.yaml`, which defines an HTTPS-only service for talking to Ambassador and is recommended, or
- `ambassador-http.yaml', which defines an HTTP-only mechanism to access Ambassador.
