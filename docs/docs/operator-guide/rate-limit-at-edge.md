# Rate Limiting at the Edge

Rate limiting is a technique that is used to prevent a sudden or sustained increase in user traffic from breaking services exposed at the edge. A closely related technique to rate limiting is load shedding, and this can be used to selectively prioritise traffic (by dropping requests) based on the state of the entire system. 

Some of the scenarios where rate limiting can be used include:

* A user, or series of users, are sending large volumes of low priority or time insensitive requests, which are causing the delay in processing of high priority requests.
* A user is inadvertently generating a spike in traffic, which is compromising the behaviour of the system for other users.
* A client application is misbehaving, and before a fix can by deployed (and users convinced to upgrade) traffic from this source must be limited.
* A bad actor is generating large amounts of traffic with the result of prevent other users from accessing the API, crashing the backend, or exposing security vulnerabilities.
* The API backend is experiencing technical issues or increased latency with request processing. For example, a backend data store has become overloaded, or a third-party dependency is experiencing latency.
* The API backend cannot scale rapidly enough to meet request demand due to a technical issue (e.g. GitHub being down, which is preventing the execution of backend instance deployment pipelines), which is causing requests to either experience increased latency or to fail.

As the above list demonstrates, although rate limiting is often only thought of in the context of limiting user requests, there are actually many different use cases that the different types of rate limiting can be applied to.

## Types of Rate Limiting
There are fundamentally two approaches to rate limiting: user request rate limiting and concurrent user request limiting. These approaches can also be combined with geographic and time-based metadata to make appropriate decisions. Load shedding can be implemented based on the aggregation utilisation of the API backend or an individual node/server instance.

## Implementation Options
There are three primary components when dealing with request/response-based traffic: the source, the sink, and middleware (literally a service in the middle of the source and sink). Rate limiting can be implemented within all three components, but when dealing with source that is not under direct control -- for example, a user’s API client or a third-party application -- rate limiting cannot be guaranteed within the source. Even if you control the user application it is recommended to rate limit in order to guard against bugs that cause excess API request, and also against bad actors who may attempt to subvert client applications.

[images]

Fundamentally rate limiting is simple. For each request property to be limited against, we simply keep a count of the number of times each unique instance of the property seen, and reject the associated request if this is over the specified count per time unit. For example, to limit the amount of requests each client made would be implemented by using the “client identified” property (perhaps set via the request string key “clientID”, or included in the request header), and keep a count for each identifier. 

A maximum number of requests per time unit can be specified, and an algorithm defined for how the request count is decremented, rather than simply resetting the counter at the start of each unit (more on this later). When a request arrives at the API gateway it will increment the appropriate request count and check to see if this increase would mean that the maximum allowables request per time unit has been exceeded. If so, then this request would be rejected, most commonly returning a “Too Many Requests” HTTP 429 status code to the calling client.

[images]

With load shedding, the primary difference here is that the decision to reject traffic is not based on a property of an individual request (e.g. the clientId), but on the overall state of the application (e.g. database under heavy load). Implementing the ability to shed load at the point of ingress can save a major customer incident if the system is still partially up and running but needs time to recover (or fix).

## Benefits of applying a rate limiter to the edge
Modern applications and APIs can experience floods of traffic over a short time period (e.g. from achieving a HackerNews front page link), and increasingly bad actors and cyber criminals are targeting public facing services. By implementing rate limiting and load shedding capabilities at the edge a large amount of scenarios that are bad for business can be mitigated. These capabilities also make the life of the operations and development team that much easier, as the need to constantly firefight ingress traffic is reduced.

