# Rate Limiting with the RateLimitService

Rate limiting is a powerful technique to improve the [availability and resilience of your services](https://blog.getambassador.io/rate-limiting-a-useful-tool-with-distributed-systems-6be2b1a4f5f4). In Ambassador, each request can have one or more *labels*. These labels are exposed to a third party service via a gRPC API. The third party service can then rate limit requests based on the request labels.

## Request labels

Ambassador lets users add one or more labels to a given request. These labels are added as part of a `Mapping` object. For example:

```
apiVersion: ambassador/v1
kind: Mapping
name: catalog
prefix: /catalog/
service: catalog
request_labels:
  - service: catalog
```

For more information on request labels, see the [Rate Limit reference](/reference/rate-limits).

## Domains

In Ambassador, each engineer (or team) can be assigned its own *domain*. A domain is a separate namespace for labels. By creating individual domains, each team can assign their own labels to a given request, and independently set the rate limits based on their own labels.

## Default labels

Ambassador allows setting a default label on every request. A default label is set on the `ambassador` module. For example:

```yaml
---
apiVersion: ambassador/v1
kind: Module
name: ambassador
config:
  default_label_domain: ambassador
  default_labels:
    ambassador:
      defaults:
      - remote_address
```

## External Rate Limit Service

In order for Ambassador to rate limit, you need to implement a gRPC service that supports the Envoy [ratelimit.proto](https://github.com/datawire/ambassador/blob/master/ambassador/common/ratelimit/ratelimit.proto) interface. If you do not have the time or resources to implement your own rate limit service, [Ambassador Pro](/pro) integrates a high performance, rate limiting service. 

Ambassador generates a gRPC request to the external rate limit service and provides a list of labels on which the rate limit service can base its decision to accept or reject the request:

```
[
  {"source_cluster", "<local service cluster>"},
  {"destination_cluster", "<routed target cluster>"},
  {"remote_address", "<trusted address from x-forwarded-for>"},
  {"generic_key", "<descriptor_value>"},
  {"<some_request_header>", "<header_value_queried_from_header>"}
]
```

If Ambassador cannot contact the rate limit service, it will allow the request to be processed as if there were no rate limit service configuration.

It is the external rate limit service's responsibility to determine whether rate limiting should take place, depending on custom business logic. The rate limit service must simply respond to the request with an `OK` or `OVER_LIMIT` code:
* If Envoy receives an `OK` response from the rate limit service, then Ambassador allows the client request to resume being processed by the normal Ambassador Envoy flow.
* If Ambassador receives an `OVER_LIMIT` response, then Ambassador will return an HTTP 429 response to the client and will end the transaction flow, preventing the request from reaching the backing service.

The headers injected by the [AuthService](/reference/services/auth-service) can also be passed to the rate limit service since the `AuthService` is invoked before the `RateLimitService`.

## Configuring the Rate Limit Service

A `RateLimitService` manifest configures Ambassador to use an external service to check and enforce rate limits for incoming requests:

```yaml
---
apiVersion: ambassador/v1
kind: RateLimitService
name: ratelimit
service: "example-rate-limit:5000"
```

- `service` gives the URL of the rate limit service.

You may only use a single `RateLimitService` manifest.

## Example

The [Ambassador Rate Limiting Tutorial](/user-guide/rate-limiting-tutorial) has a simple rate limiting example. For a more advanced example, read the [advanced rate limiting tutorial](/user-guide/advanced-rate-limiting) with Ambassador Pro tutorial.

## Further reading

* [Rate limiting: a useful tool with distributed systems](https://blog.getambassador.io/rate-limiting-a-useful-tool-with-distributed-systems-6be2b1a4f5f4)
* [Rate limiting for API Gateways](https://blog.getambassador.io/rate-limiting-for-api-gateways-892310a2da02)
* [Implementing a Java Rate Limiting Service for Ambassador](https://blog.getambassador.io/implementing-a-java-rate-limiting-service-for-the-ambassador-api-gateway-e09d542455da)
* [Designing a Rate Limit Service for Ambassador](https://blog.getambassador.io/designing-a-rate-limiting-service-for-ambassador-f460e9fabedb)


