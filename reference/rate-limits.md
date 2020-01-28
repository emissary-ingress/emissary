# Rate Limits

Rate limits are a powerful way to improve availability and scalability for your microservices. With Ambassador Edge Stack, individual requests can be annotated with metadata, called labels.  These labels can then be passed to a third party [rate limiting service](../services/rate-limit-service) which can then rate limit based on this data. If you do not want to write your own rate limiting service, [Ambassador Edge Stack](../../user-guide/install) includes an integrated, flexible rate limiting service.

## Request Labels

In Ambassador 0.50 and later, each mapping in Ambassador Edge Stack can have multiple `labels` which annotate a given request. These labels are then passed to a rate limiting service through a gRPC interface. These labels have the `labels` attribute:

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  catalog
spec:
  prefix: /catalog/
  service: catalog
  labels:
    ambassador:
      - string_request_label:         # a specific request label group
        - catalog                     # annotate the request with the string `catalog`
      - header_request_label:
        - headerkey:                  # The name of the label
            header: ":method"         # annotate the request with the specific HTTP method used
            omit_if_not_present: true # if the header is not present, omit the label
      - multi_request_label_group:
        - authorityheader:
            header: ":authority"
            omit_if_not_present: true
        - xuserheader:
            header: "x-user"
            omit_if_not_present: true
```

Let's digest the above example:

* Request labels must be part of the `ambassador` namespace. This limitation will be removed in future versions of Ambassador Edge Stack.
* Each label must have a name, e.g., `one_request_label`
* The `string_request_label` simply adds the string `catalog` to every incoming request to the given mapping. The string is referenced with the key `generic_key`.
* The `header_request_label` adds a specific HTTP header value to the request, in this case, the method. Note that HTTP/2 request headers must be used here (e.g., the `host` header needs to be specified as the `:authority` header).
* Multiple labels can be part of a single named label, e.g., `multi_request_label` specifies two different headers to be added
* When an HTTP header is not present, the entire named label is omitted. The `omit_if_not_present: true` is an explicit notation to remind end-users of this limitation. `false` is *not* a supported value. This limitation will be removed in future versions of Ambassador Edge Stack.

Ambassador Edge Stack supports several special labels:

* `remote_address` automatically populates the remote IP address using the trusted IP address from `X-Forwarded-For`
* `request_headers: HEADER` will extract the value from a given HTTP header
* `destination_cluster` populates the name of the Envoy cluster. Typically, there is a 1:1 correspondence between a `service` in a `Mapping` to a `destination_cluster`. You can get the name of the cluster from the diagnostics service.
* `source_cluster` populates the name of the originating cluster (e.g., the Envoy listener).

Note: In Envoy, labels are referred to as descriptors.

### Global Rate Limiting

Rate limit labels can be configured on a global level within the [`ambassador Module`](../modules#the-ambassador-module).

```yaml
---
apiVersion: getambassador.io/v2
kind: Module
name: ambassador
config:
  use_remote_address: true
  default_label_domain: ambassador
  default_labels:
    ambassador:
      defaults:
      - default
```

This will annotate every request with the string `default`, creating a key for a rate limiting service based on the appropriate rate limit.

## The `rate_limits` attribute

In pre-0.50 versions of the Ambassador API Gateway, a mapping can specify the `rate_limits` list attribute and at least one `rate-limit` rule which will call the external [RateLimitService](../services/rate-limit-service) before proceeding with the request. Read about Envoy's rate limit service [here](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/rate_limit_filter.html?highlight=rate%20limit%20header).

An example:

```yaml
apiVersion: getambassador.io/v0
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

- `descriptor`: if present, specifies a string identifying the triggered `rate-limit` rule. This descriptor will be sent to the `RateLimitService`.
- `headers`: if present, specifies a list of other HTTP headers which **must** appear in the request for the rate limiting rule to apply. These headers will be sent to the `RateLimitService`.

As with request labels, you must use the internal HTTP/2 request header names in `rate_limits` rules.
