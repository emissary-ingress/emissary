# Microservices API Gateways vs. Traditional Enterprise API Gateways

A microservices API gateway is an API gateway designed to accelerate the development workflow of independent services teams. A microservices API gateway provides all the functionality for a team to independently publish, monitor, and update a microservice.

This focus on accelerating the development workflow is distinct from the purpose of traditional API gateways, which focus on the challenges of managing APIs. Over the past decade, organizations have worked to expose internal systems through well-defined APIs. The challenge of safely exposing hundreds or thousands of APIs to end-users (both internal and external) led to the emergence of API gateways. Over time, API gateways have become centralized, mission critical pieces of infrastructure that control access to these APIs.

In this article, we'll discuss how the difference in business objective (productivity vs management) results in a very different API gateway.

## Microservices Organization

In a microservices organization, small teams of developers work independently from each other to rapidly deliver functionality to the customer. In order for each service team to work independently, with a productive workflow, a services team needs to be able to:

1. Publish their service, so that others can use the service
2. Monitor their service, to see how well it's working
3. Test and update their service, so they can keep on improving the service

The team needs to do all of this *without* requiring assistance from another operations or platform team--as soon as a services team requires another team, they're no longer working independently, and this can lead to bottlenecks.

For service publication, a microservices API gateway provides a static address for consumers, and dynamically route requests to the appropriate service address. In addition, providing authentication and TLS termination for security are typical considerations in exposing a service to other consumers.

Understanding the end-user experience of a service is crucial to improving the service. For example, a software update could inadvertently impact the latency of certain requests. A microservices API gateway is well situated to collect key observability metrics on end-user traffic as it routes traffic to the end service.

A microservices API gateway also supports dynamically routing user requests to different service versions for canary testing. By routing a small fraction of end-user requests to a new version of a service, service teams can safely test the impact of new updates to a small subset of users.

## Microservices API Gateways vs. Enterprise API Gateways

At first glance, the use case described above may be fulfilled with an enterprise-focused API gateway. While this may be true, the actual emphasis of enterprise API gateways and microservices API gateways are somewhat different:

| Use case      | Traditional Enterprise API gateway       | Microservices API gateway                |
|---------------|-------------------|------------------------------|
| Primary Purpose  | Expose, compose, and manage internal business APIs | Expose and observe internal business services |
| Publishing Functionality | API management team or service team registers / updates gateway via admin API | Service team registers / updates gateway via declarative code as part of the deployment process |
| Monitoring | Admin and operations focused e.g. meter API calls per consumer, report errors (e.g. internal 5XX). | Developer focused e.g. latency, traffic, errors, saturation |
| Handling and Debugging Issues | L7 error-handling (e.g. custom error page or payload). Run gateway/API with additional logging. Troubleshoot issue in staging environment | Configure more detailed monitoring. Enable traffic shadowing and / or canarying |
| Testing | Operate multiple environments for QA, Staging, and Production. Automated integration testing, and gated API deployment. Use client-driven API versioning for compatibility and stability (e.g. semver) | Facilitate canary routing for dynamic testing (taking care with data mutation side effects). Use developer-driven service versioning for upgrade management |
| Local Development | Deploy gateway locally (via installation script, Vagrant or Docker), and attempt to mitigate infrastructure differences with production. Use language-specific gateway mocking and stubbing frameworks | Deploy gateway locally via service orchestration platform (e.g. Kubernetes) |

## Self-Service Publishing

A team needs to be able to publish a new service to customers without requiring an operations or API management team. This ability to self-service for deployment and publication enables the team to keep the feature release velocity high. While a traditional enterprise API gateway may provide a simple mechanism (e.g., REST API) for publishing a new service, in practice, the usage is often limited to the use of a dedicated team that is responsible for the gateway. The primary reason for limiting publication to a single team is to provide an additional (human) safety mechanism: an errant API call could have potentially disastrous effects on production.

Microservices API gateways utilize mechanisms that enable service teams to easily *and* safely publish new services, with the inherent understanding that the producing team are responsible for their service, and will fix an issue if one occurs. A microservices gateway provides configurable monitoring for issue detection, and provides hooks for debugging, such as inspecting traffic or traffic shifting/duplication.

## Monitoring & Rate Limiting

A common business model for APIs is metering, where a consumer is charged different fees depending on API usage. Traditional enterprise API gateways excel in this use case: they provide functionality for monitoring per-client usage of an API, and the ability to limit usage when the client exceeds their quota.

A microservice gateway also requires monitoring and rate limiting, but for different reasons. Monitoring user-visible metrics such as throughput, latency, and availability, are important to ensure that new updates don't impact the end-user. Robust end-user metrics are critical to allowing rapid, incremental updates. Rate limiting is used to improve the overall resilience of a service. When a service is not responding as expected, an API gateway can throttle incoming requests to allow a service to recover and prevent a cascade failure.

## Testing and Updates

A microservices application has multiple services, each of which is being independently updated. Automated pre-production testing of a moving target is necessary but not sufficient for microservices. Canary testing, where a small percentage of production traffic is routed to a new service version, is an important tool to help test an update. By limiting a new service version to a small percentage of users, the impact of a service failure is limited.

In a traditional enterprise API gateway, routing is used to isolate or compose/aggregate changing API versions. Automated pre-production testing and manual post-production verification and exploration is required.

## Summary

Traditional enterprise API gateways are designed to solve the challenges of API management. While they may appear to solve some of the challenges of adopting microservices, the reality is that a microservices workflow creates a different set of requirements. Integrating a microservices API gateway into your development workflow empowers service teams to self-publish, monitor, and update their service, quickly and safely. This will enable your organization to ship software more rapidly, and with more reliability than ever before.

For further reading on how an API Gateway can accelerate continuous delivery, read [this blog post](https://blog.getambassador.io/continuous-delivery-how-can-an-api-gateway-help-or-hinder-1ff15224ec4d).