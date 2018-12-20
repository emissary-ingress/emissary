# Rate limiting

Ambassador Pro includes a flexible, powerful rate limiting service. 

## Introduction to rate limiting

this is why rate limiting is important

## Descriptors

In Ambassador Pro, a _descriptor_ defines what is rate limited. Descriptors can contain arbitrary metadata about a request.  Ambassador Pro uses this approach instead of using fixed fields (e.g., URLs, client IPs, etc.) to give the end user more control over what exactly is rate limited.

A descriptor is a key/value pair, e.g., `database:users` or `catalog:*`. Each descriptor is configured to have its own rate limit.

## Domains

Different service teams will need to adjust and control their rate limits independently of other teams. To prevent descriptor collision, each descriptor must belong to a `domain`. A `domain` is a top-level namespace for descriptors, and typically maps to different organizations or teams using Ambassador.

## Configuring Rate Limiting: The 50,000 foot view

Logically, configuring rate limiting is straightforward.

1. Configure a specific mapping to include one or more descriptors.
2. Configure the rate limiting service to set a rate limit on a descriptor.

## Example 1: A rusty catalog service

Imagine the `catalog` service is a Rust-y application that can only handle 1 request per second before crashing. While the engineering team really wants to rewrite the `catalog` service in Golang (because Rust isn't fast enough), they haven't had a chance to do so. We want to rate limit all requests for this service to 1 request per second. 

We update the mapping for the `catalog` service to add a descriptor to the route:

```
apiVersion: ambassador/v0
kind: Mapping
name: catalog
prefix: /catalog/
service: catalog
rate_limits:
  - descriptor: catalog
```

We then need to configure the rate limit for the catalog service:

```
domain: catalog_team
descriptors:
  - key: catalog
    rate_limit:
      unit: second
      requests_per_unit: 1
```


