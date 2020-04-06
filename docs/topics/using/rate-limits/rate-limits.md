# Rate Limiting in Ambassador Edge Stack

## The Basic Configuration Objects

Rate limits in Ambassador Edge Stack are composed of two parts:

* Labels applied to requests (a label is basic metadata that is used by the rate limiting service).
* `RateLimits` that set limits based on the labels in the request

### Labels

Edge Stack supports three types of labels:

* `generic_key` which is a simple string
* `remote_address` which is the value of the client IP address, assuming the load balancer configuration is set correctly
* a custom type that can forward the value of a header to Ambassador for rate limiting

## Global vs service-level rate limits

Edge Stack supports both global and service-level rate limits via two different labeling mechanisms.

* Labels applied in the `ambassador` Module are "global" and applied to every single request that goes through Ambassador; these labels are typically managed by operations
* Labels applied at the `Mapping` are at the service-level and applied only to the requests that use that `Mapping`

## An example service-level rate limit

The following `Mapping` resource will add the `{"generic_key": "default_generic_key_label"}` to every request to the `foo-app` service:

```
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: foo-app
spec:
  prefix: /foo/
  service: foo
  labels:
    ambassador:
      - label_group:
        - foo-app_generic_key_label
```

You can then create a default rate limit on every request that matches this label:

```
---
apiVersion: getambassador.io/v2
kind: RateLimit
metadata:
  name: default-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{generic_key: "default_generic_key_label"}]
     rate: 10
     unit: minute
```

Tip: For testing purposes, it is helpful to configure per-minute rate limits before switching the rate limits to per second or per hour.

## Request Labels

Mappings can have multiple `labels` which annotate a given request. 

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

* Request labels must be part of the `ambassador` label domain.
* Each label must have a name, e.g., `one_request_label`
* The `string_request_label` simply adds the string `catalog` to every incoming request to the given mapping. The string is referenced with the key `generic_key`.
* The `header_request_label` adds a specific HTTP header value to the request, in this case, the method. Note that HTTP/2 request headers must be used here (e.g., the `host` header needs to be specified as the `:authority` header).
* Multiple labels can be part of a single named label, e.g., `multi_request_label` specifies two different headers to be added
* When an HTTP header is not present, the entire named label is omitted. The `omit_if_not_present: true` is an explicit notation to remind end-users of this limitation. `false` is *not* a supported value.

Ambassador Edge Stack supports several special labels:

* `remote_address` automatically populates the remote IP address using the trusted IP address from `X-Forwarded-For`
* `request_headers: HEADER` will extract the value from a given HTTP header
* `destination_cluster` populates the name of the Envoy cluster. Typically, there is a 1:1 correspondence between a `service` in a `Mapping` to a `destination_cluster`. You can get the name of the cluster from the diagnostics service.
* `source_cluster` populates the name of the originating cluster (e.g., the Envoy listener).

Note: In Envoy, labels are referred to as descriptors.

## Grouping

Labels can be grouped. This allows for a single request to count against multiple different `RateLimit` resources. For example, imagine the following scenario:

1. Users should be limited on the total number of requests that can be sent to a set of endpoints
2. On a specific service, stricter limits are desirable

The following `Mapping` resources could be configured:

```

---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: foo-app
spec:
  prefix: /foo/
  service: foo
  labels:
    ambassador:
      - foo-app_label_group:
        - foo-app
      - total_requests_group:
        - remote_address
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: bar-app
spec:
  prefix: /bar/
  service: bar
  labels:
    ambassador:
      - bar-app_label_group:
        - bar-app
      - total_requests_group:
        - remote_address
```

Now requests to the `foo-app` and the `bar-app` would be labeled with {{"generic_key": "foo-app"},{"remote_address", 10.10.11.12}} and {{"generic_key": "bar-app"},{"remote_address", 10.10.11.12}}, respectively. Rate limits on these two services could be created as such:

```
---
apiVersion: getambassador.io/v2
kind: RateLimit
metadata:
  name: foo-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{generic_key: "foo-app"}]
     rate: 10
     unit: second
---
apiVersion: getambassador.io/v2
kind: RateLimit
metadata:
  name: bar-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{generic_key: "bar-app"}]
     rate: 20
     unit: second
---
apiVersion: getambassador.io/v2
kind: RateLimit
metadata:
  name: user-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{remote_address: "*"}]
     rate: 100
     unit: minute
```

## Global labels and groups

Global labels are prepended to every single label group. In the above example, if the following global label was added in the `ambassador` Module:

```
—--
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    default_label_domain: ambassador
    default_labels:
      ambassador:
        defaults:
        - default
```

The labels metadata would change from: `{{"generic_key": "foo-app"},{"remote_address", 10.10.11.12}}` and
`{{"generic_key": "bar-app"},{"remote_address", 10.10.11.12}}` to:
`{{"generic_key": "default", "generic_key": "foo-app"},{"generic_key": "default", "remote_address", 10.10.11.12}}` and
`{{"generic_key": "default", "generic_key": "bar-app"},{"generic_key": "default", "remote_address", 10.10.11.12}}`
and thus our `RateLimit`s would need to change to appropriately handle the new labels.