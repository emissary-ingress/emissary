# Rate Limits

Rate limits are a powerful way to improve availability and scalability for your microservices. With Ambassador, individual requests can be annotated with metadata, called labels.  These labels can then be passed to a third party [rate limiting service](/services/rate-limit-service) which can then rate limit based on this data. If you do not want to write your own rate limiting service, [Ambassador Pro](https://www.getambassador.io/pro) includes an integrated, flexible rate limiting service.

## Request labels

In Ambassador 0.50 and later, each mapping in Ambassador can have multiple *labels* which annotate a given request. These labels are then passed to a rate limiting service through a gRPC interface. These labels are specified with the `labels` annotation:

```
apiVersion: ambassador/v0
kind: Mapping
name: catalog
prefix: /catalog/
service: catalog
labels:
  - ambassador:
    - request_label: # a specific request label
      - catalog      # annotate the request with the value `catalog`
    - request_label:
      - header: ":method"          # annotate the request with the specific HTTP method used
        omit_if_not_present: true  # if the header is not present, omit the label
```

Request labels must be part of the `ambassador` namespace. This limitation will be removed in future versions of Ambassador.

HTTP/2 request headers can be used in request labels, as shown in the example above. For example:
- the `host` header should be specified as the `:authority` header; and
- the `method` header should be specified as the `:method` header.

## The `rate_limits` attribute

In pre-0.50 versions of Ambassador, a mapping can specify the `rate_limits` list attribute and at least one `rate_limits` rule which will call the external [RateLimitService](/reference/services/rate-limit-service) before proceeding with the request. An example:

```yaml
apiVersion: ambassador/v0
kind: Mapping
name: rate_limits_mapping
prefix: /rate-limit/
service: rate-limit-example
rate_limits:
  - {}
  - descriptor: a rate-limit descriptor
    headers:
    - matching-header
```

Rate limit rule settings:

- `descriptor`: if present, specifies a string identifying the triggered rate limit rule. This descriptor will be sent to the `RateLimitService`.
- `headers`: if present, specifies a list of other HTTP headers which **must** appear in the request for the rate limiting rule to apply. These headers will be sent to the `RateLimitService`.

As with request labels, you must use the internal HTTP/2 request header names in `rate_limits` rules.