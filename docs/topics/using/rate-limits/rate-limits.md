# Rate Limiting in the Ambassador Edge Stack

Rate limiting in the Ambassador Edge Stack is composed of two parts:

* Labels that get attached to requests; a label is basic metadata that
  is used by the `RateLimitService` to decide which limits to apply to
  the request.
* `RateLimit`s configure the Ambassador Edge Stack's built-in
  `RateLimitService`, and set limits based on the labels on the
  request.

## Attaching labels to requests

<!-- TODO(lukeshu): This section applies both to Ambassador Edge Stack
and to Ambassador API Gateway; move it to a shared location instead of
putting it alongside the AES `RateLimit` docs. -->

There are two ways of setting labels on a request:

1. Per [`Mapping`](../../mappings#configuring-mappings).  Labels set
   here will only apply to requests that use that Mapping

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: Mapping
   metadata:
     name: foo-app
   spec:
     prefix: /foo/
     service: foo
     labels:
       "my_first_label_domain":
       - "my_first_label_group":
         - "my_label_specifier_1"
         - "my_label_specifier_2"
       - "my_second_label_group":
         - "my_label_specifier_3"
         - "my_label_specifier_4"
       "my_second_label_domain":
       - ...
   ```

2. Globally, in the [`ambassador`
   `Module`](../../../running/ambassador).  Labels set here are
   applied to every single request that goes through Ambassador.  This
   includes requests go through a `Mapping` that sets more labels; for
   those requests, the global labels are prepended to each of the
   Mapping's label groups for the matching domain; otherwise the
   global labels are put in to a new label group named "default" for
   that domain.

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: Module
   metadata:
     name: ambassador
   spec:
     config:
       default_labels:
         "my_first_label_domain":
           defaults:
           - "my_label_specifier_a"
           - "my_label_specifier_b"
         "my_second_label_domain":
           defaults:
           - "my_label_specifier_c"
           - "my_label_specifier_d"
   ```

*Labels* on a request are lists of key/value pairs, organized in to
*label groups*.  Because a label group is a *list* of key/value pairs
(rather than a map),
- it is possible to have multiple labels with the same key
- the order of labels matters

Your Module and Mappings contain *label specifiers* that tell
Ambassador what labels to set on the request.

> Note: The terminology used by the Envoy documentation differs from
> the terminology used by Ambassador:
>
> | Ambassador      | Envoy             |
> |-----------------|-------------------|
> | label group     | descriptor        |
> | label           | descriptor entry  |
> | label specifier | rate limit action |

The Mappings' listing of the groups of specifiers have names for the
groups; the group names are useful for humans dealing with the YAML,
but are ignored by Ambassador, all Ambassador cares about are the
*contents* of the groupings of label specifiers.

There are 5 types of label specifiers in Ambassador:

<!-- This table is ordered the same way as the protobuf fields in
  `route_components.proto`.  There's also a 6th action:
  "header_value_match" (since Envoy 1.2), but Ambassador doesn't
  support it?  -->

| #             | Label Specifier                        | Action, in human terms                                                                                                                  | Action, in [Envoy gRPC terms][`envoy.api.v2.route.RateLimit.Action`]           |
|---------------|----------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------|
| 1             | `"source_cluster"`                     | Sets the label "`source_cluster`=«Envoy source cluster name»"                                                                           | `{ "source_cluster": {} }`                                                     |
| 2             | `"destination_cluster"`                | Sets the label "`destination_cluster`=«Envoy destination cluster name»"                                                                 | `{ "destination_cluster": {} }`                                                |
| 3             | `{ "my_key": { "header": "my_hdr" } }` | If the `my_hdr` header is set, then set the label "«`my_key`»=«Value of the `my_hdr` header»"; otherwise skip applying this label group | `{ "request_headers": { "header_name": "my_hdr", descriptor_key: "my_key" } }` |
| 4             | `"remote_address"`                     | Sets the label "`remote_address`=«IP address of the client»"                                                                            | `{ "remote_address": {} }`                                                     |
| 5             | `{ "generic_key": "my_val" }`          | Sets the label "`generic_key`=«`my_val`»"                                                                                               | `{ "generic_key": { "descriptor_value": "my_val" } }`                          |
| 5 (shorthand) | `"my_val"`                             | Shorthand for `{ "generic_key": "my_val" }`                                                                                             |                                                                                |

[`envoy.api.v2.route.RateLimit.Action`]: https://github.com/datawire/ambassador/blob/$branch$/api/envoy/api/v2/route/route_components.proto#L1328-L1439

1. The Envoy source cluster name is the name of the Envoy listener
   cluster that the request name in on.
2. The Envoy destination cluster is the name of the Envoy cluster that
   the Mapping routes the request to.  Typically, there is a 1:1
   correspondence between upstream services (pointed to by Mappings)
   and clusters.  You can get the name for a cluster from the
   diagnostics service or Edge Policy Console.
3. When setting a label from an HTTP request header, be aware that if
   that header is not set in the request, then the entire label group
   is skipped.
4. The IP address of the HTTP client could be the actual IP of the
   client talking directly to Ambassador, or it could be the IP
   address from `X-Forwarded-For` if Ambassador is configured to trust
   the `X-Fowarded-For` header.
5. `generic_key` allows you to apply a simple string label to requests
   flowing through that Mapping.

## Rate limiting requests based on their labels

A `RateLimit` resource defines a list of limits that apply to
different requests.

```yaml
---
apiVersion: getambassador.io/v2
kind: RateLimit
metadata:
  name: example-limits
spec:
  domain: "my_domain"
  limits:
  - name: per-minute-limit         # optional; default is the `$name.$namespace-$idx` where name is the name of the CRD and idx is the index into the limits array
    pattern:
    - "my_key1": "my_value1"
      "my_key2": "my_value2"
    - "my_key3": "my_value3"
    rate: 5
    unit: "minute"
    injectRequestHeaders:          # optional
    - name: "header-name-string-1"   # required
      value: "go-template-string"    # required
    - name: "header-name-string-2"   # required
      value: "go-template-string"    # required
    injectResponseHeaders:         # optional
    - name: "header-name-string-1"   # required
      value: "go-template-string"    # required
    errorResponse:                 # optional
      headers:                       # optional; default is [], adding no additional headers
      - name: "header-name-string"     # required
        value: "go-template-string"    # required
      bodyTemplate: "string"         # optional; default is "", returning no response body
  - name: per-second-limit
    pattern:
    - "my_key4": ""   # check the key but not the value
    - "my_key5": "*"  # check the key but not the value
    rate: 5
    unit: "second"
  ...
```

It makes no difference whether limits are defined together in one
`RateLimit` resource or are defined separately in many `RateLimit`
resources.

<!-- FIXME(lukeshu): I'm only mostly sure that I'm describing the
algorithm correctly.  The thing to reference is
`vendor-ratelimit/src/config/config_impl.go:rateLimitConfigImpl.GetLimit()`
and/or `lib/rltypes/rls.go:Config.Add()` -->

 - `name`: The symbolic name for this ratelimit. Used to set dynamic metadata that can be referenced in the Envoy access log.

 - `pattern`: Each limit has a *pattern* that matches against a label
   group on a request to decide if that limit should apply to that
   request.  For a pattern to match, the request's label group must
   start with exactly the labels specified in the pattern, in order.
   If a label in a pattern has an empty string or `"*"` as the value,
   then it only checks the key of that label on the request; not the
   value.  If a list item in the pattern has multiple key/value pairs,
   if any of them match the label then it is considered a match.

   For example, the pattern

   ```yaml
   pattern:
   - "key1": "foo"
     "key1": "bar"
   - "key2": ""
   ```

   matches the label group

   ```yaml
   - key1: foo
   - key2: baz
   - otherkey: knob
   ```

   and

   ```yaml
   - key1: bar
   - key2: baz
   - otherkey: knob
   ```

   but not the label group

   ```yaml
   - key0: frob
   - key1: foo
   - key2: baz
   ```

   If a label group is matched by multiple patterns, the pattern with
   the longest list of items wins.

   If a request has multiple label groups, then multiple limits may apply
   to that request; if *any* of the limits are being hit, then Ambassador
   will reject the request as an HTTP 429.

 - `rate`, `unit`: The limit itself is specified as an integer number
   of requests per a unit of time.  Valid units of time are `second`,
   `minute`, `hour`, or `day` (all case-insensitive).

   So for example

   ```yaml
   rate: 5
   unit: minute
   ```

   would allow 5 requests per minute, and any requests in excess of
   that would result in HTTP 429 errors.

 - `injectRequestHeaders`, `injectResponseHeaders`: If this limit's
   pattern matches the request, then `injectRequestHeaders` injects
   HTTP header fields in to the request before sending it to the
   upstream service (assuming the limit even allows the request to go
   to the upstream service), and `injectResponseHeaders` injects
   headers in to the response sent back to the client (whether the
   response came from the upstream service or is an HTTP 429 response
   because it got rate limited).  This is very similar to
   `injectRequestHeaders` in a [`JWT` Filter][].  The header value is
   specified as a [Go `text/template`][] string, with the following
   data made available to it:

    * `.RateLimitResponse.OverallCode` → `int` : `1` for OK, `2` for
      OVER_LIMIT.
    * `.RateLimitResponse.Statuses` →
      [`[]*RateLimitResponse_DescriptorStatus]`][`v2.RateLimitResponse_DescriptorStatus`]
      The itemized status codes for each limit that was selected for
      this request.
    * `.RetryAfter` → `time.Duration` the amount of time until all of
      the limits would allow access again (0 if they all currently
      allow access).

   Also available to the template are the [standard functions available
   to Go `text/template`s][Go `text/template` functions], as well as:

    * a `hasKey` function that takes the a string-indexed map as arg1,
      and returns whether it contains the key arg2.  (This is the same
      as the [Sprig function of the same name][Sprig `hasKey`].)

    * a `doNotSet` function that causes the result of the template to
      be discarded, and the header field to not be adjusted.  This is
      useful for only conditionally setting a header field; rather
      than setting it to an empty string or `"<no value>"`.  Note that
      this does _not_ unset an existing header field of the same name.

 - `errorResponse` allows templating the error response, overriding the default json error format.  Make sure you validate and test your template, not to generate server-side errors on top of client errors.
    * `headers` sets extra HTTP header fields in the error response. The value is specified as a [Go `text/template`][] string, with the same data made available to it as `bodyTemplate` (below). It does not have access to the `json` function.
    * `bodyTemplate` specifies body of the error; specified as a [Go `text/template`][] string, with the following data made available to it:

       * `.status_code` → `integer` the HTTP status code to be returned
       * `.message` → `string` the error message string
       * `.request_id` → `string` the Envoy request ID, for correlation (hidden from `{{ . | json "" }}` unless `.status_code` is in the 5XX range)
       * `.RateLimitResponse.OverallCode` → `int` : `1` for OK, `2` for
         OVER_LIMIT.
       * `.RateLimitResponse.Statuses` →
         [`[]*RateLimitResponse_DescriptorStatus]`][`v2.RateLimitResponse_DescriptorStatus`]
         The itemized status codes for each limit that was selected for
         this request.
       * `.RetryAfter` → `time.Duration` the amount of time until all of
         the limits would allow access again (0 if they all currently
         allow access).

      Also availabe to the template are the [standard functions
      available to Go `text/template`s][Go `text/template` functions],
      as well as:

       * a `json` function that formats arg2 as JSON, using the arg1
         string as the starting indentation.  For example, the
         template `{{ json "indent>" "value" }}` would yield the
         string `indent>"value"`.

[`JWT` Filter]: ../../filters/jwt
[Go `text/template`]: https://golang.org/pkg/text/template/
[Go `text/template` functions]: https://golang.org/pkg/text/template/#hdr-Functions
[`v2.RateLimitResponse_DescriptorStatus`]: https://godoc.org/github.com/datawire/ambassador/pkg/api/envoy/service/ratelimit/v2#RateLimitResponse_DescriptorStatus
[Sprig `hasKey`]: https://masterminds.github.io/sprig/dicts.html#haskey

## Logging RateLimits

It is often desirable to know which RateLimit, if any, is applied to a client's request. This can be achieved by leveraging dynamic metadata available to Envoy's access log.

The following dynamic metadata keys are available under the `envoy.filters.http.ratelimit` namespace. See https://www.envoyproxy.io/docs/envoy/latest/configuration/observability/access_log/usage for more on Envoy's access log format.

* `aes.ratelimit.name` - The symbolic `name` of the `Limit` on a `RateLimit` object that triggered the ratelimit action.
* `aes.ratelimit.retry_after` - The time in seconds until the `Limit` resets. Equivalent to the value of the `Retry-After` returned to the client if the limit was enforced.

Note that if multiple `Limit`s were exceeded by a request, only the `Limit` with the longest time until reset (i.e. its Retry-After value) will be available as dynamic metadata above.

### An example access log specification for RateLimit dynamic metadata

Module:

```yaml
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    envoy_log_format: 'ratelimit %DYNAMIC_METADATA(envoy.filters.http.ratelimit:aes.ratelimit.name)% took action %DYNAMIC_METADATA(envoy.filters.http.ratelimit:aes.ratelimit.action)'
```

## RateLimit Examples

### An example service-level rate limit

The following `Mapping` resource will add a
`my_default_generic_key_label` `generic_key` label to every request to
the `foo-app` service:

```yaml
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
      - my_default_generic_key_label
```

You can then create a default RateLimit for every request that matches
this label:

```yaml
---
apiVersion: getambassador.io/v2
kind: RateLimit
metadata:
  name: default-rate-limit
spec:
  domain: ambassador
  limits:
  - pattern:
    - generic_key: "my_default_generic_key_label"
    rate: 10
    unit: minute
```

> Tip: For testing purposes, it is helpful to configure per-minute
> rate limits before switching the rate limits to per second or per
> hour.

### An example with multiple labels

Mappings can have multiple `labels` which annotate a given request.

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: catalog
spec:
  prefix: /catalog/
  service: catalog
  labels:
    ambassador:                     # the label domain
    - string_request_label:           # the label group name -- useful for humans, ignored by Ambassador
      - catalog                         # annotate the request with `generic_key=catalog`
    - header_request_label:           # another label group name
      - headerkey:                      # The name of the label
          header: ":method"               # annotate the request with the specific HTTP method used
    - multi_request_label_group:
      - authorityheader:
          header: ":authority"
          omit_if_not_present: true
      - xuserheader:
          header: "x-user"
          omit_if_not_present: true
```

<!--

The above example used to say

    omit_if_not_present: true       # if the header is not present, omit the label

on all of the header labels, but I've removed it from the example
because "omit_if_not_present" doesn't actually work right now and is
commented out in the code.

-->

Let's digest the above example:

* Request labels must be part of the "ambassador" label domain.  Or
  rather, it must match the domain in your
  `RateLimitService.spec.domain` which defaults to
  `Module.spec.default_label_domain` which defaults to `ambassador`;
  but normally you should accept the default and just accept that the
  domain on the Mappings must be set to "ambassador".
* Each label must have a name, e.g., `one_request_label`
* The `string_request_label` simply adds the string `catalog` to every
  incoming request to the given mapping.  The string is referenced
  with the key `generic_key`.
* The `header_request_label` adds a specific HTTP header value to the
  request, in this case, the method.  Note that HTTP/2 request headers
  must be used here (e.g., the `host` header needs to be specified as
  the `:authority` header).
* Multiple labels can be part of a single named label, e.g.,
  `multi_request_label` specifies two different headers to be added
* When an HTTP header is not present, the entire named label is
  omitted.  The `omit_if_not_present: true` is an explicit notation to
  remind end-users of this limitation.  `false` is *not* a supported
  value.

### An example with multiple limits

Labels can be grouped.  This allows for a single request to count
against multiple different `RateLimit` resources.  For example,
imagine the following scenario:

1. Users should be limited on the total number of requests that can be
   sent to a set of endpoints
2. On a specific service, stricter limits are desirable

The following `Mapping` resources could be configured:

```yaml
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

Now requests to the `foo-app` and the `bar-app` would be labeled with
```yaml
- "generic_key": "foo-app"
- "remote_address": "10.10.11.12"
```
and
```yaml
- "generic_key": "bar-app"
- "remote_address": "10.10.11.12"
```
respectively.  `RateLimit`s on these two services could be created as
such:

```yaml
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

### An example with global labels and groups

Global labels are prepended to every single label group.  In the above
example, if the following global label was added in the `ambassador`
Module:

```yaml
---
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
        - "my_default_label"
```

The labels metadata would change

 - from
   ```yaml
   - "generic_key": "foo-app"
   - "remote_address": "10.10.11.12"
   ```
   to
   ```yaml
   - "generic_key": "my_default_label"
   - "generic_key": "foo-app"
   - "remote_address": "10.10.11.12"
   ```

and

 - from
   ```yaml
   - "generic_key": "bar-app"
   - "remote_address": "10.10.11.12"
   ```
   to
   ```yaml
   - "generic_key": "my_default_label"
   - "generic_key": "bar-app"
   - "remote_address": "10.10.11.12"
   ```

respectively.

And thus our `RateLimit`s would need to change to appropriately handle
the new labels.
