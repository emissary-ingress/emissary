# RateLimitService Plugin

Rate limiting is a powerful technique to improve the [availability and
resilience of your
services](https://blog.getambassador.io/rate-limiting-a-useful-tool-with-distributed-systems-6be2b1a4f5f4).
In Ambassador, each request can have one or more *labels*.  These labels are
exposed to a third-party service via a gRPC API.  The third-party service can
then rate limit requests based on the request labels.

**Note that `RateLimitService` is only applicable to the Ambassador API Gateway,
and not the Ambassador Edge Stack, as the Ambassador Edge Stack includes a
built-in rate limit service.**

## Request Labels

Ambassador lets users add one or more labels to a given request.  These labels
are added as part of a `Mapping` object.  For example:

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

For more information on request labels, see the [Rate Limit reference](../../../using/rate-limits/).

## Domains

In Ambassador, each engineer (or team) can be assigned its own *domain*.  A
domain is a separate namespace for labels.  By creating individual domains, each
team can assign their own labels to a given request, and independently set the
rate limits based on their own labels.

## Default labels

Ambassador allows setting a default label on every request.  A default label is
set on the `ambassador Module`.  For example:

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

In order for the Ambassador API Gateway to rate limit, you need to implement a
gRPC `RateLimitService`, as defined in [Envoy's `v1/rls.proto`][`v1/rls.proto`]
interface.  If you do not have the time or resources to implement your own rate
limit service, the Ambassador Edge Stack integrates a high-performance rate
limiting service.

> Note: *In a future version of Ambassador*, the Ambassador API Gateway will
> change the version of the gRPC service name used to communicate
> `RateLimitService`s from the one defined in [`v1/rls.proto`][]
> (`pb.lyft.ratelimit.RateLimitService`) to the one defined in
> [`v2/rls.proto`][] (`envoy.service.ratelimit.v2.RateLimitService`):
>
> - In some future version of Ambassador, there will be a setting to control
>   which name is used; with the default being the current name; it will be
>   opt-in to the new name.
>
> - In some future version of Ambassador after that, *no sooner than Ambassador
>   1.6.0*, the default value of that setting swill change; making it opt-out
>   from the new name.
>
> - In some future version of Ambassador after that, *no sooner than Ambassador
>   1.7.0*, the setting will go away, and Ambassador will always use the new
>   name.
>
> In the mean-time, implementations of `RateLimitService` are encouraged to
> respond to both both names--they are simply aliases of eachother, registering
> the service under both names is usually a simple 1-or-2-line addition.  For
> example, in Go the change to support both names is:
>
> ```diff
>  import (
>  	envoy_ratelimit_v1 "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v1"
> +	envoy_ratelimit_v2 "github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v2"
>  )
> ...
>  	envoy_ratelimit_v1.RegisterRateLimitServiceServer(myGRPCServer, myRateLimitImplementation)
> +	envoy_ratelimit_v2.RegisterRateLimitServiceServer(myGRPCServer, myRateLimitImplementation)
> ```

[`v1/rls.proto`]: https://github.com/datawire/ambassador/tree/master/api/envoy/service/ratelimit/v1/rls.proto
[`v2/rls.proto`]: https://github.com/datawire/ambassador/tree/master/api/envoy/service/ratelimit/v2/rls.proto

The Ambassador API Gateway generates a gRPC request to the external rate limit
service and provides a list of labels on which the rate limit service can base
its decision to accept or reject the request:

```
[
  {"source_cluster", "<local service cluster>"},
  {"destination_cluster", "<routed target cluster>"},
  {"remote_address", "<trusted address from x-forwarded-for>"},
  {"generic_key", "<descriptor_value>"},
  {"<some_request_header>", "<header_value_queried_from_header>"}
]
```

If the Ambassador API Gateway cannot contact the rate limit service, it will
allow the request to be processed as if there were no rate limit service
configuration.

It is the external rate limit service's responsibility to determine whether rate
limiting should take place, depending on custom business logic.  The rate limit
service must simply respond to the request with an `OK` or `OVER_LIMIT` code:

* If Envoy receives an `OK` response from the rate limit service, then the
  Ambassador API Gateway allows the client request to resume being processed by
  the normal flow.
* If Envoy receives an `OVER_LIMIT` response, then the Ambassador API Gateway
  will return an HTTP 429 response to the client and will end the transaction
  flow, preventing the request from reaching the backing service.

The headers injected by the [AuthService](../auth-service) can also be passed to
the rate limit service since the `AuthService` is invoked before the
`RateLimitService`.

## Configuring the Rate Limit Service

A `RateLimitService` manifest configures the Ambassador API Gateway to use an
external service to check and enforce rate limits for incoming requests:

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

You can tell the Ambassador API Gateway to use TLS to talk to your service by
using a `RateLimitService` with an `https://` prefix.  However, you may also
provide a `tls` attribute: if `tls` is present and `true`, the Ambassador API
Gateway will originate TLS even if the `service` does not have the `https://`
prefix.

If `tls` is present with a value that is not `true`, the value is assumed to be the name of a defined TLS context, which will determine the certificate presented to the upstream service.

## Example

The [Ambassador API Gateway Rate Limiting
Tutorial](../../../../howtos/rate-limiting-tutorial) has a simple rate limiting
example.  For a more advanced example, read the [advanced rate limiting
tutorial](../../../../howtos/advanced-rate-limiting), which uses the rate limit
service that is integrated with the Ambassador Edge Stack.

## Further Reading

* [Rate limiting: a useful tool with distributed systems](https://blog.getambassador.io/rate-limiting-a-useful-tool-with-distributed-systems-6be2b1a4f5f4)
* [Rate limiting for API Gateways](https://blog.getambassador.io/rate-limiting-for-api-gateways-892310a2da02)
* [Implementing a Java Rate Limiting Service for Ambassador Edge Stack](https://blog.getambassador.io/implementing-a-java-rate-limiting-service-for-the-ambassador-api-gateway-e09d542455da)
* [Designing a Rate Limit Service for Ambassador Edge Stack](https://blog.getambassador.io/designing-a-rate-limiting-service-for-ambassador-f460e9fabedb)
