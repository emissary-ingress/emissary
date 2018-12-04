# Advanced Rate Limiting

Ambassador Pro integrates a flexible, high performance Rate Limit Service (RLS). Similar to Ambassador, the RLS features a decentralized configuration model so that individual teams can manage their own rate limits. For example:

* A service owner may want to manage load shedding characteristics, and insuring specific types of requests take precedence over other types of requests
* An operations engineer may want to insure service availability overall when request volume is high, and limit the total number of requests being passed to upstream services
* A security engineer may want to protect against denial-of-service attacks from a bad actor

Like Ambassador, the Ambassador RLS is designed so that many different teams, with different requirements, can independently manage and control rate limiting as necessary.

## Request labels and domains

In Ambassador Pro, each request can have multiple *labels*. Labels are arbitrary key/value pairs that contain any metadata about a given request, e.g., its IP, a hard-coded value, the path of the request, and so forth. The Rate Limit Service processes these labels and enforces any limits that are set on a label. Labels can be assigned to *domains*, which are separate namespaces. Typically, different teams would be responsible for different domains.

## Configuring Rate Limiting: The 50,000 foot view

Logically, configuring rate limiting is straightforward.

1. Configure a specific mapping to include one or more request labels.
2. Configure a limit for a given request label with the `RateLimit` resource.

## Example 1: Global rate limiting for availability

Imagine the `catalog` service is a Rust-y application that can only handle 3 requests per minute before crashing. While the engineering team really wants to rewrite the `catalog` service in Golang (because Rust isn't fast enough), they haven't had a chance to do so. We want to rate limit all requests for this service to 3 requests per minute. (ProTip: Using requests per minute simplifies testing.)

We update the mapping for the `catalog` service to add a request label to the route:

```
apiVersion: ambassador/v0
kind: Mapping
name: catalog
prefix: /catalog/
service: catalog
labels:
  - ambassador:
    - request_label_group:
      - string_label:
          catalog
```

We then need to configure the rate limit for the catalog service. Create a new YAML file, `catalog-ratelimit.yaml`, and put the following configuration into the file.

```
apiVersion: getambassador.io/v1beta1
kind: RateLimit
metadata:
  name: catalog_rate_limit
spec:
  limits:
   - pattern: [{generic_key: catalog}]
     rate: 3
     unit: minute
```

Deploy the rate limit with `kubectl apply -f catalog-ratelimit.yaml`.

## Example 2: Per user rate limiting

Suppose you've rewritten the `catalog` service in Golang, and it's humming along nicely. You then discover that some users are taking advantage of this speed to sometimes cause a big spike in requests. You want to make sure that your API doesn't get overwhelmed by any single user. We do the following:

```
apiVersion: getambassador.io/v1beta1
kind: RateLimit
metadata:
  name: catalog_rate_limit
spec:
  limits:
   - pattern: [{remote_address: "*"}]
     rate: 3
     unit: minute
```

## Example 3: Load shedding GET requests

You've dramatically improved availability of the `catalog` service, thanks to the per-user rate limiting. However, you've realized that on occasion the queries (e.g., the 'GET' requests) cause so much volume that updates to the catalog (e.g., the 'POST' requests) don't get processed. So we're going to add a more sophisticated rate limiting strategy:

* We're going to rate limit per user.
* We're going to implement a global rate limit on `GET` requests, but not `POST` requests.

```
apiVersion: ambassador/v0
kind: Mapping
name: catalog
prefix: /catalog/
service: catalog
labels:
  - ambassador:
    - request_label_group:
      - string_label:
          catalog
      - method_label:
        - header: ":method"
          omit_if_not_present: true
```

Our rate limiting configuration becomes:

```
apiVersion: getambassador.io/v1beta1
kind: RateLimit
metadata:
  name: catalog_rate_limit
spec:
  limits:
   - pattern: [{remote_address: "*"}]
     rate: 3
     unit: minute
   - pattern: [{method: GET}]
```

## More

For more on rate limiting, see the [rate limit reference](/reference/rate-limits).