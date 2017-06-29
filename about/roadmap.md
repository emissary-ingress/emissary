# Roadmap

Ambassador is an API Gateway for microservices built on [Envoy](https://lyft.github.io/envoy/). Key features in Ambassador include:

* Self-service mapping of public URLs to services running inside a Kubernetes cluster
* Robust TLS support, including TLS client-certificate authentication
* Simple setup and configuration
* Integrated monitoring

Planned features for future releases include the following:

* More authentication mechanisms, including at least:
   * HTTP Basic auth
   * OAuth2
   * JWT
* Per-service authorization
* Configurable rate limiting (global and per-service)
* Embeddable custom plugins, probably in Lua or Python

If you have requests or suggestions for the roadmap, let us know!
