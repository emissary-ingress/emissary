# Advanced Rate Limiting

The Ambassador Edge Stack integrates a flexible, high-performance Rate Limit Service (RLS). Similar to Ambassador, the RLS features a decentralized configuration model so that individual teams can manage their own rate limits. For example:

* A service owner may want to manage load shedding characteristics and ensuring specific types of requests take precedence over other types of requests
* An operations engineer may want to ensure service availability overall when request volume is high and limit the total number of requests being passed to upstream services
* A security engineer may want to protect against denial-of-service attacks from a bad actor

Like Ambassador, the Ambassador RLS is designed so that many different teams, with different requirements, can independently manage and control rate limiting as necessary.

## Request labels and domains

In the Ambassador Edge Stack, each request can have multiple *labels*. Labels are arbitrary key/value pairs that contain any metadata about a given request, e.g., its IP, a hard-coded value, the path of the request, and so forth. The Rate Limit Service processes these labels and enforces any limits that are set on a label. Labels can be assigned to *domains*, which are separate namespaces. Typically, different teams would be responsible for different domains.

## Configuring Rate Limiting: The 50,000 Foot View

Logically, configuring rate limiting is straightforward.

1. Configure a specific mapping to include one or more request labels.
2. Configure a limit for a given request label with the `RateLimit` resource.

In the examples below, we'll use the backend service of the quote sample application.

## Example 1: Global Rate Limiting for Availability

Imagine the backend service is a Rust-y application that can only handle 3 requests per minute before crashing. While the engineering team really wants to rewrite the backend service in Golang (because Rust isn't fast enough), they haven't had a chance to do so. We want to rate limit all requests for this service to 3 requests per minute. (ProTip: Using requests per minute simplifies testing.)

We update the mapping for the `quote` service to add a request label `backend` to the route as part of a `request_label_group`:

```yaml
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: quote-backend
spec:
  prefix: /backend/
  service: quote
  labels:
    ambassador:
      - request_label_group:
        - backend
```

*Note* If you're modifying an existing mapping, make sure you to update the apiVersion to `v2` as above from `v1`.

We then need to configure the rate limit for the backend service. Create a new YAML file, `backend-ratelimit.yaml`, and put the following configuration into the file.

```yaml
apiVersion: getambassador.io/v2
kind: RateLimit
metadata:
  name: backend-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{generic_key: backend}]
     rate: 3
     unit: minute
```

`generic_key` in the example above is a special, hard-coded value that is used when a single string label is added to a request.

Deploy the rate limit with `kubectl apply -f backend-ratelimit.yaml`. (Make sure you `kubectly apply` the modified `quote-backend` mapping as well.)

## Example 2: Per user rate limiting

Suppose you've rewritten the quote backend service in Golang, and it's humming along nicely. You then discover that some users are taking advantage of this speed to sometimes cause a big spike in requests. You want to make sure that your API doesn't get overwhelmed by any single user. We use the `remote_address` special value in our mapping, which will automatically label all requests with the calling IP address:

```yaml
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: quote-backend
spec:
  prefix: /backend/
  service: quote
  labels:
    ambassador:
      - request_label_group:
        - remote_address
```

We then update our rate limits to limit on `remote_address`:

```yaml
apiVersion: getambassador.io/v2
kind: RateLimit
metadata:
  name: backend-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{remote_address: "*"}]
     rate: 3
     unit: minute
```

Note for this to work, you need to make sure you've properly configured Ambassador to [propagate your original client IP address](../../reference/core/ambassador#use_remote_address).

## Example 3: Load Shedding GET Requests

You've dramatically improved availability of the quote backend service, thanks to the per-user rate limiting. However, you've realized that on occasion the queries (e.g., the 'GET' requests) cause so much volume that updates to the backend (e.g., the 'POST' requests) don't get processed. So we're going to add a more sophisticated load shedding strategy:

* We're going to rate limit per user.
* We're going to implement a global rate limit on `GET` requests, but not `POST` requests.

```yaml
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: quote-backend
spec:
  prefix: /backend/
  service: quote
  labels:
    ambassador:
      - request_label_group:
        - remote_address
        - backend_http_method:
            header: ":method"
            omit_if_not_present: true
```

When we add multiple criteria to a pattern, the entire pattern matches when ANY of the rules match (i.e., a logical OR). A pattern match then triggers a rate limit event. Our rate limiting configuration becomes:

```yaml
apiVersion: getambassador.io/v2
kind: RateLimit
metadata:
  name: backend-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{remote_address: "*"}, {backend_http_method: GET}]
     rate: 3
     unit: minute
```

## Example 4: Global Rate Limiting

Suppose, like [Example 2](#example-2-per-user-rate-limiting), you want to ensure a single user cannot overload your server with too many requests to any service. You need to add a request label to every request so you can rate limit off every request a calling IP makes. This can be configured with a [global rate limit](../topics/using/rate-limits) that add the `remote_address` special value to every request:

```yaml
---
apiVersion: getambassador.io/v2
kind: Module
matadata:
  name: ambassador
spec:
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
apiVersion: getambassador.io/v2
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

Sometimes, you may have an API that cannot handle as much load as others in your cluster. In this case, a global rate limit may not be enough to ensure this API is not overloaded with requests from a user. To protect this API, you will need to create a label that tells Ambassador Edge Stack to apply a stricter limit on requests. With the above global rate limit configuration rate limiting based on `remote_address`, you will need to add a request label to the services `Mapping`.

```yaml
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: quote-backend
spec:
  prefix: /backend/
  service: quote
  labels:
    ambassador:
      - request_label_group:
        - backend
```

Now, the `request_label_group`, contains both the `generic_key: backend` *and* the `remote_address` key applied from the global rate limit. This allows us to create a separate `RateLimit` object for this route:

```yaml
apiVersion: getambassador.io/v2
kind: RateLimit
metadata:
  name: backend-rate-limit
spec:
  domain: ambassador
  limits:
   - pattern: [{remote_address: "*"}, {generic_key: backend}]
     rate: 3
     unit: minute
```

Now, requests to `/backend/` will be rate limited after only 3 requests.

## Rate limiting Matching Rules

The following rules apply to the rate limit patterns:

* Patterns are order-sensitive, and must respect the order in which a request is labeled.
* Every label in a label group must exist in the pattern in order for matching to occur.
* By default, any type of failure will let the request pass through (fail open).
* The Ambassador Edge Stack sets a hard timeout of 20ms on the rate limiting service. If the rate limit service does not respond within the timeout period, the request will pass through.
* If a pattern does not match, the request will pass through.

## Troubleshooting rate limiting

The most common source of failure of the rate limiting service will occur when the labels generated by the Ambassador Edge Stack do not match the rate limiting pattern. By default, the rate limiting service will log all incoming labels from the Ambassador Edge Stack. Use a tool such as [Stern](https://github.com/wercker/stern) to watch the rate limiting logs from the Ambassador Edge Stack, and ensure the labels match your descriptor.

## More

For more on rate limiting, see the [rate limit guide](../topics/using/rate-limits/).
