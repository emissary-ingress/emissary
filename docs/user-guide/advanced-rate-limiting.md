# Advanced Rate Limiting

Ambassador Pro integrates a flexible, high performance Rate Limit Service (RLS). Similar to Ambassador, the RLS features a decentralized configuration model so that individual teams can manage their own rate limits. For example:

* A service owner may want to manage load shedding characteristics, and ensuring specific types of requests take precedence over other types of requests
* An operations engineer may want to ensure service availability overall when request volume is high, and limit the total number of requests being passed to upstream services
* A security engineer may want to protect against denial-of-service attacks from a bad actor

Like Ambassador, the Ambassador RLS is designed so that many different teams, with different requirements, can independently manage and control rate limiting as necessary.

## Request labels and domains

In Ambassador Pro, each request can have multiple *labels*. Labels are arbitrary key/value pairs that contain any metadata about a given request, e.g., its IP, a hard-coded value, the path of the request, and so forth. The Rate Limit Service processes these labels and enforces any limits that are set on a label. Labels can be assigned to *domains*, which are separate namespaces. Typically, different teams would be responsible for different domains.

## Configuring Rate Limiting: The 50,000 foot view

Logically, configuring rate limiting is straightforward.

1. Configure a specific mapping to include one or more request labels.
2. Configure a limit for a given request label with the `RateLimit` resource.

In the examples below, we'll use the QOTM sample service used in the [Getting Started](https://www.getambassador.io/user-guide/getting-started#5-adding-a-service).

## Example 1: Global rate limiting for availability

Imagine the `qotm` service is a Rust-y application that can only handle 3 requests per minute before crashing. While the engineering team really wants to rewrite the `qotm` service in Golang (because Rust isn't fast enough), they haven't had a chance to do so. We want to rate limit all requests for this service to 3 requests per minute. (ProTip: Using requests per minute simplifies testing.)

We update the mapping for the `qotm` service to add a request label `qotm` to the route as part of a `request_label_group`:

```yaml
apiVersion: ambassador/v1
kind: Mapping
name: qotm
prefix: /qotm/
service: qotm
labels:
  ambassador:
    - request_label_group:
      - qotm
```

*Note* If you're modifying an existing mapping, make sure you to update the apiVersion to `v1` as above from `v0`.

We then need to configure the rate limit for the qotm service. Create a new YAML file, `qotm-ratelimit.yaml`, and put the following configuration into the file.

```yaml
apiVersion: getambassador.io/v1beta1
kind: RateLimit
metadata:
  name: qotm-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{generic_key: qotm}]
     rate: 3
     unit: minute
```

`generic_key` in the example above is a special, hard-coded value that is used when a single string label is added to a request.

Deploy the rate limit with `kubectl apply -f qotm-ratelimit.yaml`. (Make sure you always `kubectly apply` your original `qotm` mapping as well.)

## Example 2: Per user rate limiting

Suppose you've rewritten the `qotm` service in Golang, and it's humming along nicely. You then discover that some users are taking advantage of this speed to sometimes cause a big spike in requests. You want to make sure that your API doesn't get overwhelmed by any single user. We use the `remote_address` special value in our mapping, which will automatically label all requests with the calling IP address:

```yaml
apiVersion: ambassador/v1
kind: Mapping
name: qotm
prefix: /qotm/
service: qotm
labels:
  ambassador:
    - request_label_group:
      - remote_address
```

We then update our rate limits to limit on `remote_address`:

```yaml
apiVersion: getambassador.io/v1beta1
kind: RateLimit
metadata:
  name: qotm-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{remote_address: "*"}]
     rate: 3
     unit: minute
```

Note for this to work, you need to make sure you've properly configured Ambassador to [propagate your original client IP address](https://www.getambassador.io/reference/modules/#use_remote_address).

## Example 3: Load shedding GET requests

You've dramatically improved availability of the `qotm` service, thanks to the per-user rate limiting. However, you've realized that on occasion the queries (e.g., the 'GET' requests) cause so much volume that updates to the qotm (e.g., the 'POST' requests) don't get processed. So we're going to add a more sophisticated load shedding strategy:

* We're going to rate limit per user.
* We're going to implement a global rate limit on `GET` requests, but not `POST` requests.

```yaml
apiVersion: ambassador/v1
kind: Mapping
name: qotm
prefix: /qotm/
service: qotm
labels:
    ambassador:
      - request_label_group:
        - remote_address
        - qotm_http_method:
            header: ":method"
            omit_if_not_present: true
```

When we add multiple criteria to a pattern, the entire pattern matches when ANY of the rules match (i.e., a logical OR). A pattern match then triggers a rate limit event. Our rate limiting configuration becomes:

```yaml
apiVersion: getambassador.io/v1beta1
kind: RateLimit
metadata:
  name: qotm-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{remote_address: "*"}, {qotm_http_method: GET}]
     rate: 3
     unit: minute
```

## Example 4: Global Rate Limiting
Suppose, like [Example 2](/user-guide/advanced-rate-limiting#example-2-per-user-rate-limiting), you want to ensure a single user cannot overload your server with too many requests to any service. You need to add a request label to every request so you can rate limit off every request a calling IP makes. This can be configured with a [global rate limit](/reference/rate-limits#global-rate-limiting) that add the `remote_address` special value to every request:

```yaml
---
apiVersion: ambassador/v1
kind: Module
name: ambassador
config:
  use_remote_address: true
  default_label_domain: ambassador
  default_labels:
    ambassador:
      defaults:
      - remote_address
```

We can then configure a global `RateLimit` object that limits on `remote_address`:

```yaml
apiVersion: getambassador.io/v1beta1
kind: RateLimit
metadata:
  name: global-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{remote_address: "*"}]
     rate: 10
     unit: minute
```

### Bypassing a Global Rate Limit
Sometimes, you may have an API that cannot handle as much load as others in your cluster. In this case, a global rate limit may not be enough to ensure this API is not overloaded with requests from a user. To protect this API, you will need to create a label that tells Ambassador Pro to apply a stricter limit on requests. With the above global rate limit configuration rate limiting based off `remote_address`, you will need to add a request label to the services `Mapping`: 

```yaml
apiVersion: ambassador/v1
kind: Mapping
name: qotm
prefix: /qotm/
service: qotm
labels:
  ambassador:
    - request_label_group:
      - qotm
```

Now, the `request_label_group`, contains both the `generic_key: qotm` *and* the `remote_address` key applied from the global rate limit. This allows us to create a separate `RateLimit` object for this route:

```yaml
apiVersion: getambassador.io/v1beta1
kind: RateLimit
metadata:
  name: qotm-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{remote_address: "*"}, {generic_key: qotm}]
     rate: 3
     unit: minute
```
Now, requests will `/qotm/` will be rate limited after only 3 requests.

## Rate limiting matching rules

The following rules apply to the rate limit patterns:

* Patterns are order-sensitive, and must respect the order in which a request is labeled. For example, in #3 above, the `remote_address` pattern must come before the `qotm_http_method` pattern. Switching the two will fail to match.
* Every label in a label group must exist in the pattern in order for matching to occur.
* By default, any type of failure will let the request pass through (fail open).
* Ambassador sets a hard timeout of 20ms on the rate limiting service. If the rate limit service does not respond within the timeout period, the request will pass through.
* If a pattern does not match, the request will pass through.

## Troubleshooting rate limiting

The most common source of failure of the rate limiting service will occur when the labels generated by Ambassador do not match the rate limiting pattern. By default, the rate limiting service will log all incoming labels from Ambassador. Use a tool such as [Stern](https://github.com/wercker/stern) to watch the rate limiting logs from Ambassador, and ensure the labels match your descriptor.

## More

For more on rate limiting, see the [rate limit reference](/reference/rate-limits).