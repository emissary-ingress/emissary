# Table of Contents

* [Blog](https://blog.getambassador.io)

----

## Getting Started

* [Why Ambassador?](about/why-ambassador.md)
* [Features and Benefits](about/features-and-benefits.md)
* [Installing Ambassador](user-guide/install.md)
  * [Kubernetes (YAML)](user-guide/getting-started.md)
  * [Kubernetes (Helm)](user-guide/helm.md)
  * [Docker Quickstart](about/quickstart.md)
  * [Docker Compose](user-guide/docker-compose.md)

## Concepts

* [Decentralized configuration](concepts/developers.md)
* [Ambassador Architecture](concepts/architecture.md)
  * [Why Ambassador uses Envoy Proxy](https://blog.getambassador.io/envoy-vs-nginx-vs-haproxy-why-the-open-source-ambassador-api-gateway-chose-envoy-23826aed79ef)
* [Microservices API Gateways](about/microservices-api-gateways.md)
* [Using Ambassador in your organization](concepts/using-ambassador-in-org.md)

## Guides

* [Adding Authentication](user-guide/auth-tutorial.md)
* [Adding Rate Limiting](user-guide/rate-limiting-tutorial.md)
* [Adding Tracing](user-guide/tracing-tutorial.md)
* [Use gRPC with Ambassador](user-guide/grpc.md)
* [TLS Termination](user-guide/tls-termination.md)
* [Istio and Ambassador](user-guide/with-istio.md)
* [More guides](https://blog.getambassador.io/howto/home)

## Reference

* [Configuring Ambassador](reference/configuration.md)
  * [Core Configuration](reference/modules.md)
    * [TLS and X-Forwarded-Proto](reference/core/tls.md)
  * [Configuring Services](reference/mappings.md)
    * [Canary Releases](reference/canary.md)
    * [Cross Origin Resource Sharing](reference/cors.md)
    * [Custom Envoy config](reference/override.md)
    * [Header-based routing](reference/headers.md)
    * [Host Header](reference/host.md)
    * [Rate Limits](reference/rate-limits.md)
    * [Redirects](reference/redirects.md)
    * [Request Headers](reference/add_request_headers.md)
    * [Rewrites](reference/rewrites.md)
    * [Traffic Shadowing](reference/shadowing.md)
  * [External Services](reference/services/services.md)
    * [Authentication](reference/services/auth-service.md)
    * [Rate Limiting](reference/services/rate-limit-service.md)
    * [Tracing](reference/services/tracing-service.md)
  * [Advanced configuration topics](reference/advanced.md)
* [Running Ambassador](reference/running.md)
  * [Diagnostics](reference/diagnostics.md)
  * [Ambassador with AWS](reference/ambassador-with-aws.md)
* [Upgrading Ambassador](reference/upgrading.md)
* [Statistics and Monitoring](reference/statistics.md)


## Developers

* [Building Ambassador (GitHub)](https://github.com/datawire/ambassador/blob/master/BUILDING.md)
* [Changelog (GitHub)](https://github.com/datawire/ambassador/blob/master/CHANGELOG.md)

## Need Help?

* [Ask on Slack](https://d6e.co/slack)
* [File a GitHub Issue](https://github.com/datawire/ambassador/issues/new)
* [Visit Datawire.io](https://www.datawire.io)
