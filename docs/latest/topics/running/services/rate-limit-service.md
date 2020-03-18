# RateLimitService Plugin

Rate limiting is a powerful technique to improve the [availability and
resilience of your
services](https://blog.getambassador.io/rate-limiting-a-useful-tool-with-distributed-systems-6be2b1a4f5f4).
In the Ambassador API Gateway, each request can have one or more *labels*. These
labels are exposed to a third party service via a gRPC API. The third-party
service can then rate limit requests based on the request labels.

**Note that `RateLimitService`is only applicable to the Ambassador API Gateway, and not the Edge Stack.**

## Request Labels

Ambassador Edge Stack lets users add one or more labels to a given request. These labels are added as part of a `Mapping` object. For example:

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  catalog
spec:
  prefix: /catalog/
  service: catalog
  request_labels:
    - service: catalog
```

For more information on request labels, see the [Rate Limit reference](../../using/rate-limits/).

## Domains

In Ambassador Edge Stack, each engineer (or team) can be assigned its own *domain*. A domain is a separate namespace for labels. By creating individual domains, each team can assign their own labels to a given request, and independently set the rate limits based on their own labels.

## Default labels

Ambassador Edge Stack allows setting a default label on every request. A default label is set on the `ambassador Module`. For example:

```yaml
---
apiVersion: getambassador.io/v2
kind:  Module
metadata:
  name:  ambassador
spec:
  config:
    default_label_domain: ambassador
    default_labels:
      ambassador:
        defaults:
        - remote_address
```

## External Rate Limit Service

In order for Ambassador Edge Stack to rate limit, you need to implement a gRPC `RateLimitService`, as defined in [Envoy's `rls.proto`][rls.proto] interface. If you do not have the time or resources to implement your own rate limit service, Ambassador Edge Stack integrates a high performance, rate limiting service.

[rls.proto]: https://github.com/datawire/ambassador/tree/master/api/envoy/service/ratelimit/v2/rls.proto

Ambassador Edge Stack generates a gRPC request to the external rate limit service and provides a list of labels on which the rate limit service can base its decision to accept or reject the request:

```
[
  {"source_cluster", "<local service cluster>"},
  {"destination_cluster", "<routed target cluster>"},
  {"remote_address", "<trusted address from x-forwarded-for>"},
  {"generic_key", "<descriptor_value>"},
  {"<some_request_header>", "<header_value_queried_from_header>"}
]
```

If Ambassador Edge Stack cannot contact the rate limit service, it will allow the request to be processed as if there were no rate limit service configuration.

It is the external rate limit service's responsibility to determine whether rate limiting should take place, depending on custom business logic. The rate limit service must simply respond to the request with an `OK` or `OVER_LIMIT` code:

* If Envoy receives an `OK` response from the rate limit service, then Ambassador Edge Stack allows the client request to resume being processed by the normal Ambassador Envoy flow.
* If Envoy receives an `OVER_LIMIT` response, then the Ambassador Edge Stack will return an HTTP 429 response to the client and will end the transaction flow, preventing the request from reaching the backing service.

The headers injected by the [AuthService](../auth-service) can also be passed to the rate limit service since the `AuthService` is invoked before the `RateLimitService`.

## Configuring the Rate Limit Service

A `RateLimitService` manifest configures Ambassador Edge Stack to use an external service to check and enforce rate limits for incoming requests:

```yaml
---
apiVersion: getambassador.io/v2
kind:  RateLimitService
metadata:
  name:  ratelimit
spec:
  service: "example-rate-limit:5000"
```

- `service` gives the URL of the rate limit service.

You may only use a single `RateLimitService` manifest.

## Rate Limit Service and TLS

You can tell Ambassador Edge Stack to use TLS to talk to your service by using a `RateLimitService` with an `https://` prefix. However, you may also provide a `tls` attribute: if `tls` is present and `true`, Ambassador Edge Stack will originate TLS even if the `service` does not have the `https://` prefix.

If `tls` is present with a value that is not `true`, the value is assumed to be the name of a defined TLS context, which will determine the certificate presented to the upstream service.

## Example

The [Ambassador Edge Stack Rate Limiting Tutorial](../../../user-guide/rate-limiting-tutorial) has a simple rate limiting example. For a more advanced example, read the [advanced rate limiting tutorial](../../../user-guide/advanced-rate-limiting) with Ambassador Edge Stack tutorial.

## Further Reading

* [Rate limiting: a useful tool with distributed systems](https://blog.getambassador.io/rate-limiting-a-useful-tool-with-distributed-systems-6be2b1a4f5f4)
* [Rate limiting for API Gateways](https://blog.getambassador.io/rate-limiting-for-api-gateways-892310a2da02)
* [Implementing a Java Rate Limiting Service for Ambassador Edge Stack](https://blog.getambassador.io/implementing-a-java-rate-limiting-service-for-the-ambassador-api-gateway-e09d542455da)
* [Designing a Rate Limit Service for Ambassador Edge Stack](https://blog.getambassador.io/designing-a-rate-limiting-service-for-ambassador-f460e9fabedb)
