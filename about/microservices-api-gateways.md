# Microservices API Gateways vs traditional API Gateways

A microservices API gateway is an API gateway designed to accelerate the development workflow of independent services teams. A microservices API gateway provides all the functionality for a team to independently publish, monitor, and update a microservice.

This focus on accelerating the development workflow is distinct from the purpose of traditional API gateways, which focus on the challenges of managing APIs. Over the past decade, organizations have worked to expose internal systems through well-defined APIs. The challenge of safely exposing hundreds or thousands of APIs to end users (both internal and external) led to the emergence of API Gateways. Over time, API Gateways have become centralized, mission critical pieces of infrastructure that control access to these APIs.

In this article, we'll discuss how the difference in business objective (productivity vs management) results in a very different API gateway.

## Microservices organization

In a microservices organization, small teams of developers work independently from each other to rapidly deliver functionality to the customer. In order for a services team to work independently, with a productive workflow, a services team needs to be able to:

1. Publish their service, so that others can use the service
2. Monitor their service, to see how well it's working
3. Test and update their service, so they can keep on improving the service

*without* requiring assistance from another team. (As soon as a services team requires another team, they're no longer working independently from another team, and creating bottlenecks.)

For service publication, a microservices API gateway provides a static address for consumers, and dynamically route requests to the appropriate service address. In addition, providing authentication and TLS termination for security are typical considerations in exposing a service to other consumers.

Understanding the end user experience of a service is crucial to improving the service. For example, a software update could inadvertently impact the latency of certain requests. A microservices API gateway is well situated to collect key observability metrics on end user traffic as it routes traffic to the end service.

A microservices API gateway supports dynamically routing user requests to different service versions for canary testing. By routing a small fraction of end user requests to a new version of a service, service teams can safely test the impact of new updates to a small subset of users.

## Microservices API gateways versus traditional API Gateways

At first blush, the use case described above may be fulfilled with a traditional API Gateway. Let's look at the differences a little more closely.

| Use case      | API Gateway       | microservices API gateway                |
|---------------|-------------------|------------------------------|
| Publishing    | Operations registers/updates new services | Service team registers/updates new services |
| Monitoring    | Measure API calls per consumer, for metering | Measure L7 latency, throughput, availability |
| Rate limiting | Cut off API calls per consumer when a consumer exceeds its quota | Limit API calls when service is not responding, for resilience |
| Test & Update | API versioning for stability | Canary routing for dynamic testing

## Self-service publishing

A service team needs to be able to publish a new service to customers without requiring an operations team ("self-service"). While a traditional API gateway may provide a simple mechanism (e.g., REST API) for publishing a new service, in practice, the usage is limited to operations. The primary reason for limiting publication to operations teams is to provide an additional (human) safety mechanism: an errant API call could have potentially disastrous effects on production. microservices API gateways utilize mechanisms that enable service teams to easily *and* safely publish new services. One example approach is to attach the routing metadata directly to service objects, which eliminate the possibility that a service team will inadvertently affect another service.

## Monitoring & Rate limiting

A common business model for APIs is metering, where a consumer is charged different fees depending on API usage. Traditional API gateways excel in this use case: they provide functionality for monitoring per-client usage of an API, and the ability to limit usage when the client exceeds their quota.

A microservice also requires monitoring and rate limiting, but for different reasons. Monitoring user-visible metrics such as throughput, latency, and availability are important to insure that new updates don't impact the end user. Robust end user metrics are critical to allowing rapid, incremental updates. Rate limiting is used to improve the overall resilience of a service. When a service is not responding as expected, an API gateway can throttle incoming requests to allow a service to recover and prevent a cascade failure.

## Testing and updates

A microservices application has multiple services, each of which is being independently updated. Synthetic testing of a moving target is necessary but not sufficient for microservices. Canary testing, where a small percentage of traffic is routed to a new service version, is an important tool to help test an update. By limiting a new service version to a small percentage of users, the impact of a service failure is limited.

In a traditional API gateway, routing is used to manage changing API versions. Microservices API gateways integrate canary routing directly into the routing rules so that service teams can quickly and safely rollout new versions of their service.

# Summary

Traditional API gateways are designed to solve the challenges of API management. While they may appear to solve some of the challenges of adopting microservices, the reality is that a microservices workflow creates a different set of requirements. Integrating a microservices API gateway into your development workflow empowers service teams to self-publish, monitor, and update their service, quickly and safely. This will enable your organization to ship software faster than ever before.
