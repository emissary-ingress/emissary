# Advanced Rate Limiting

Ambassador Pro integrates a flexible, high performance Rate Limit Service (RLS). Similar to Ambassador, the RLS features a decentralized configuration model so that individual teams can manage their own rate limits. Some of the use cases that can be addressed with Ambassador Pro's rate limiting service include:

* Improving availability against denial-of-service attacks by insuring that traffic to a given service doesn't exceed its capacity (request rate limiter)
* Insuring fairness between different clients, insuring that a single client doesn't consume a disproportionate amount of resources (user requests limiter)
* Insuring that high priority traffic is serviced before lower priority requests (load shedder)

## Domains

As your organization grows, different groups within your organization may have different rate limit requirements. For example:

* A service owner may want to manage load shedding characteristics, and insuring specific types of requests take precedence over other types of requests
* An operations engineer may want to insure service availability overall when request volume is high, and limit the total number of requests being passed to upstream services
* A security engineer may want to protect against denial-of-service attacks from a bad actor

In this scenario, each engineer will need to _independently_ monitor and manage the rate limits. In Ambassador, each engineer (or team) can be assigned its own *domain*. A domain is a separate namespace for labels. By creating individual domains, each team can assign their own labels to a given request, and independently set the rate limits based on their own labels.

## Request labels

In Ambassador Pro, each request can have multiple *labels*. Labels are arbitrary key/value pairs that contain any metadata about a given request, e.g., its IP, a hard-coded value, the path of the request, and so forth. The Rate Limit Service processes these labels and enforces any limits that are set on a label.

## Domains: XXX FIXME

Different service teams will need to adjust and control their rate limits independently of other teams. To prevent descriptor collision, each descriptor must belong to a `domain`. A `domain` is a top-level namespace for descriptors, and typically maps to different organizations or teams using Ambassador.

## Configuring Rate Limiting: The 50,000 foot view

Logically, configuring rate limiting is straightforward.

1. Configure a specific mapping to include one or more request labels.
2. Configure a limit for a given request label.

## Example 1: Global rate limiting for availability

Imagine the `catalog` service is a Rust-y application that can only handle 1 request per second before crashing. While the engineering team really wants to rewrite the `catalog` service in Golang (because Rust isn't fast enough), they haven't had a chance to do so. We want to rate limit all requests for this service to 1 request per second. 

We update the mapping for the `catalog` service to add a descriptor to the route:

```
apiVersion: ambassador/v0
kind: Mapping
name: catalog
prefix: /catalog/
service: catalog
request_labels:
  - service: catalog
```

We then need to configure the rate limit for the catalog service:

```
domain: catalog_team
descriptors:
  - key: service
    value: catalog
    rate_limit:
      unit: second
      requests_per_unit: 1
```

## Example 2: Per user rate limiting

Suppose you've rewritten the `catalog` service in Golang, and it's humming along nicely. You then discover that some users are taking advantage of this speed to sometimes cause a big spike in requests. You want to make sure that your API doesn't get overwhelmed by any single user. We do the following:

## Example 3: Load shedding GET requests

You've dramatically improved availability of the `catalog` service, thanks to the per-user rate limiting. However, you've realized that on occasion the queries (e.g., the 'GET' requests) cause so much volume that updates to the catalog (e.g., the 'POST' requests) don't get processed. So we're going to add a more sophisticated rate limiting strategy:

* We're going to rate limit per user.
* We're going to implement a global rate limit on `GET` requests, but not `POST` requests.

```
some yaml here
```


## Reference

### Matching rules

The Rate Limit Service uses the following matching rules:

* If no value is specified in the matching descriptor, it is treated as a wildcard.
* If multiple descriptors are sent to the RLS, the RLS will rate limit against all matching rules (i.e., logical OR). In other words, the RLS will return the result of `isRateLimited(all_matching_descriptors)`.
* The most specific match will be attempted.


## Notes for Noah

1. https://www.getambassador.io/reference/rate-limits will need to be updated once we figure out how Ambassador is supposed to work. But the descriptors here are broken. I think we'll need something like:

```
apiVersion: ambassador/v0
kind: Mapping
name: rate_limits_mapping
prefix: /rate-limit/
service: rate-limit-example
rate_limits:
  - descriptors:
    - service: catalog
    - foo: bar 
```

2. Installation: we'll need a Redis service, plus the rate limit sidecar.