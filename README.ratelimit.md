# Rate limiting

Ambassador Pro includes a flexible, powerful rate limiting service.

## Introduction to rate limiting

this is why rate limiting is important

## Descriptors

In Ambassador Pro, a _descriptor_ defines what is rate
limited. Descriptors can contain arbitrary metadata about a request.
Ambassador Pro uses this approach instead of using fixed fields (e.g.,
URLs, client IPs, etc.) to give the end user more control over what
exactly is rate limited.

A descriptor is a key/value pair, e.g., `database:users` or
`catalog:*`. Each descriptor is configured to have its own rate limit.

## Domains

Different service teams will need to adjust and control their rate
limits independently of other teams. To prevent descriptor collision,
each descriptor must belong to a `domain`. A `domain` is a top-level
namespace for descriptors, and typically maps to different
organizations or teams using Ambassador.

## Configuring Rate Limiting: The 50,000 foot view

Logically, configuring rate limiting is straightforward.

1. Configure a specific mapping to include one or more descriptors.
2. Configure the rate limiting service to set a rate limit on a
   descriptor.

## Example 1: A rusty catalog service

Imagine the `catalog` service is a Rust-y application that can only
handle 1 request per second before crashing. While the engineering
team really wants to rewrite the `catalog` service in Golang (because
Rust isn't fast enough), they haven't had a chance to do so. We want
to rate limit all requests for this service to 1 request per second.

We update the mapping for the `catalog` service to add a descriptor to
the route:

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

## Developing

This is an integration of the lyft ratelimit service into a formfactor
suitable for ambassador. This means:

 - deploying it beside ambassador as a sidecar
 - using CRDs to define it's configuration
 - supplying a basic controller to reload on config changes

The intention (for now) is to make only very minor codechanges to the
lyft ratelimit service itself, and so the Makefile pulls in a (pegged)
version of the lyft dependency that is very lightly patched. (See
comments in the Makefile around the `make diff` target for more
details.)

To get started:

1. Type `make deploy`. This will build a container, acquire a
   kubernaut cluster, and spin up ambassador, redis, and the ratelimit
   service. It will take a while the first time. It will be quicker
   subsequent times.

2. Type `make proxy` to start teleproxy. This will start teleproxy in
   the background. To stop it at any point type `make unproxy`.

The remaining steps all assume teleproxy is running. To query the
ratelimit service in the cluster:

1. Type `make lyft-build` in order to build the ratelimit binaries:

   - ratelimit: the ratelimit service itself
   - ratelimit_client: a client for querying the ratelimit service

2. Run: `./ratelimit_client -dial_string ratelimit:81 -domain test -descriptors a=b`

To modify the ratelimit service config in the cluster:

1. Edit the `k8s/limits.yaml` or add your own limits in another file
   underneath `k8s`.

2. Run `make apply`.

To see the descriptors that ambassador produces:

1. `curl ambassador/rl/`

2. Look at the logs of the ratelimit container in the ambassador pod.
