# Features and Benefits

Ambassador is an API Gateway for microservices built on [Envoy](https://lyft.github.io/envoy/). Key features in Ambassador include:

* Self-service mapping of public URLs to services running inside a Kubernetes cluster
* Robust TLS support, including TLS client-certificate authentication
* Simple setup and configuration
* Integrated monitoring

Check out the [Ambassador roadmap](roadmap.md) for what's coming up in the future.

## Self-Service

Ambassador is built from the start to support _self-service_ deployments -- a developer working on a new service doesn't have to go to Operations to get their service added to the mesh, they can do it themselves in a matter of seconds. Likewise, a developer can remove their service from the mesh, or merge services, or separate services, as needed, at their convenience.

## Resource Mapping

At the heart of Ambassador is the idea of mapping _resources_ (in the REST sense) to _services_ (in the Kubernetes sense).

* A `resource` is identified by a URL prefix -- for example, you might declare that any URL beginning with `/user/` identifies a "user" resource.

* A `service` is code running in Kubernetes that can handle the resource you want to map.

For more information, check out the [concepts](concepts.md) section of the Ambassador documentation.

## TLS

Ambassador supports inbound TLS and inbound TLS client-certificate authentication. We **strongly** recommend using TLS with Ambassador, and encourage you to carefully read the [TLS](../how-to/tls-termination.md) section of the Ambassador documentation for more.

At present, a resource can be mapped to only one service, but the same service can be used behind as many different resources as you want. There's no hard limit to the number of mappings Ambassador can handle (though eventually you'll run out of memory).

### CAVEATS

Ambassador is ALPHA SOFTWARE. In particular, in version 0.8.0, there is no authentication mechanism, so anyone who can reach the administrative interface can map or unmap resources -- great for self service, of course, but possibly dangerous. For this reason, the administrative requires a Kubernetes port-forward.

Ambassador is under active development; check frequently for updates, and please file issues for things you'd like to see!
