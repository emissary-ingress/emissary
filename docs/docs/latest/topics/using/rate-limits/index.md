# Rate Limiting Concepts at the Edge

Rate limiting at the edge is a technique that is used to prevent a sudden or sustained increase in user traffic from breaking an API or underlying service. On the Internet, users can do whatever they want to your APIs, as you have no direct control over these end-users. Whether it’s intentional or not, these users can impact the availability, responsiveness, and scalability of your service.

## Two Approaches: Rate Limiting and Load Shedding

Rate limiting use cases that fall into this scenario range from implementing functional requirements related to a business scenario -- for example, where requests from paying customers is prioritized over requests from non-paying trial users -- to implementing cross-functional requirements, such as resilience from a malicious actor attempting to issue a denial-of-service (DoS) attack.

A closely related technique to rate limiting is load shedding, and this can be used to selectively prioritize traffic (by dropping requests) based on the state of the entire system. For example, if a backend data store has become overloaded and slow to respond, it may be appropriate to drop (or “shed”) low priority requests or requests that are not time sensitive.

## Use Cases and Scenarios

The table below outlines several scenarios where rate limiting and load shedding can provide an effective solution to a range of functional and cross-functional requirements. The “Type of Rate Limiter” column provides a summary of the category of rate limiting that would be most appropriate for the scenario, and the “Specifics” column outlines what business or system properties would be involved in computing rate limiting decisions.

| Scenario | Type of Rate Limiter | &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Specifics&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
| --- | --- | --- |
**Fairness.** One or more users are sending large volumes of requests, and thus impacting other users of the API | **User request rate limiting -** restricts each user to a predetermined number of requests per time unit.<br/><br/>**Concurrent user request limiting -** limits the number of concurrent user requests that can be inflight at any given point in time. | <ul><li>User ID rate limiter</li><li>User property rate limiter (IP address, organisation, device etc)</li><li>Geographic rate limiter</li><li>Time-based rate limiter</li></ul> 
**Prioritisation.** The business model depends on handling high priority requests over others | **User request rate limiting** |<ul><li>User ID rate limiter</li><li>User property rate limiter (IP address, organisation, device, free vs non-free user etc)</li></ul>
**Resilience.** The API backend cannot scale rapidly enough to meet request demand due to a technical issue. | **Backend utilisation load shedder -** rate limit based upon utilisation of aggregate backend instances.<br/><br/>**Node/server utilisation load shedder -** rate limit based upon utilisation of individual or isolated groups of compute nodes/servers. |<ul><li>User ID rate limiter</li><li>User property rate limiter (IP address, organisation, device etc)</li></ul>
**Security.** Prevent bad actors from using a DoS attack to overwhelm services, fuzzing, or brute force attacks |**User request rate limiting**<br/><br/>**Node/server utilisation load shedder** | <ul><li>User ID rate limiter</li><li>User property rate limiter (IP address, organisation, device etc)</li><li>Service identifier load shedder e.g. login service, audit service</li></ul>
**Responsiveness.** As per the Reactive Manifesto, responsive systems focus on providing rapid and consistent response times, establishing reliable upper bounds so they deliver a consistent quality of service | **Concurrent user request limiting**<br/><br/>**Backend utilisation load shedder**<br/><br/>**Node/server utilisation load shedder** | <ul><li>User ID rate limiter</li><li>User property rate limiter (IP address, organisation, device etc)</li><li>Service identifier load shedder e.g. login service, audit service</li></ul>

## Avoiding Contention with Rate Limiting Configuration: Decoupling Dev and Ops

One of the core features of Ambassador Edge Stack is the decentralization of configuration, allowing operations and development teams to independently control Ambassador Edge Stack, as well as individual application development teams to minimize collaboration when configuring independently deployable services. This same approach applies to rate limiting configuration.

The Ambassador Edge Stack rate limiting configuration allows centralized operations teams to define and implement global rate limiting and load shedding policies to protect the system, while still allowing individual application teams to define rate limiting policies that enforce business rules, for example, around paying and non-paying customers (perhaps implementing the so-called “freemium” model). See [Advanced Rate Limiting](../../../howtos/advanced-rate-limiting) documentation for examples.

## Benefits of Applying a Rate Limiter to the Edge

Modern applications and APIs can experience floods of traffic over a short time period (e.g. from achieving a HackerNews front page link), and increasingly bad actors and cyber criminals are targeting public-facing services.

By implementing rate limiting and load shedding capabilities at the edge, a large amount of scenarios that are bad for business can be mitigated. These capabilities also make the life of the operations and development team that much easier, as the need to constantly firefight ingress traffic is reduced.
