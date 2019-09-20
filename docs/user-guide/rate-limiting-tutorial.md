# Rate Limiting

Ambassador can validate incoming requests before routing them to a backing service. In this tutorial, we'll configure Ambassador to use a simple third party rate limit service. If you don't want to implement your own rate limiting service, Ambassador Pro integrates a [powerful, flexible rate-limiting service](/user-guide/advanced-rate-limiting).

## Before You Get Started

This tutorial assumes you have already followed the [Ambassador Getting Started](/user-guide/getting-started) guide. If you haven't done that already, you should do that now.

After completing [Getting Started](/user-guide/getting-started), you'll have a Kubernetes cluster running Ambassador and the Quote of the Moment service. Let's walk through adding rate limiting to this setup.

## 1. Deploy the rate limit service

Ambassador delegates the actual rate limit logic to a third party service. We've written a [simple rate limit service](https://github.com/datawire/ambassador/tree/master/test-services/ratelimit-service) that:

- listens for requests on port 5000;
- handles gRPC `shouldRateLimit` requests;
- allows requests with the `x-ambassador-test-allow: "true"` header; and
- marks all other requests as `OVER_LIMIT`;

Here's the YAML we'll start with:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: example-rate-limit
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind: RateLimitService
      name: ratelimit
      service: "example-rate-limit:5000"
spec:
  type: ClusterIP
  selector:
    app: example-rate-limit
  ports:
  - port: 5000
    name: http-example-rate-limit
    targetPort: http-api
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-rate-limit
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
  selector:
    matchLabels:
      app: example-rate-limit
  template:
    metadata:
      labels:
        app: example-rate-limit
    spec:
      containers:
      - name: example-rate-limit
        image: agervais/ambassador-ratelimit-service:1.0.0
        imagePullPolicy: Always
        ports:
        - name: http-api
          containerPort: 5000
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
```

This configuration tells Ambassador about the rate limit service, notably that it is serving requests at `example-rate-limit:5000`.

Ambassador will see the annotations and reconfigure itself within a few seconds.

## 2. Configure Ambassador Mappings

Ambassador only validates requests on Mappings which set rate limiting descriptors. If Ambassador cannot contact the rate limit service, it will allow the request to be processed as if there were no rate limit service configuration.

We already have the `tour` service running, so let's apply some rate limits to the service. The easiest way to do that is to annotate the `tour` service. While we could use `kubectl patch` for this, it's simpler to just modify the service definition and re-apply.

### v1 API
Ambassador 0.50.0 and later requires the `v1` API Version for rate limiting. The `v1` API uses the `labels` attribute to attach rate limiting descriptors. Review the [Rate Limits configuration documentation](/reference/rate-limits#request-labels) for more information.

Replace the label that is applied to the `tour-backend_mapping` with:

```yaml
labels:
  ambassador:
    - request_label_group:
      - x-ambassador-test-allow:
          header: "x-ambassador-test-allow"
          omit_if_not_present: true
```

so the full service definition will now look like this:
```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: tour
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v1
      kind: Mapping
      name: tour-ui_mapping
      prefix: /
      service: tour:5000
      ---
      apiVersion: ambassador/v1
      kind: Mapping
      name: tour-backend_mapping
      prefix: /backend/
      service: tour:8080
      labels:
        ambassador:
          - request_label_group:
            - x-ambassador-test-allow:
                header: "x-ambassador-test-allow"
                omit_if_not_present: true
spec:
  ports:
  - name: ui
    port: 5000
    targetPort: 5000
  - name: backend
    port: 8080
    targetPort: 8080
  selector:
    app: tour
```

### v0 API
Ambassador versions 0.40.2 and earlier use the `v0` API version which uses the `rate_limits` attribute to set rate limiting descriptors. Review the [Rate Limits configuration documentation](/reference/rate-limits#the-rate_limits-attribute) for more information.


```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: tour
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind: Mapping
      name: tour-ui_mapping
      prefix: /
      service: tour:5000
      ---
      apiVersion: ambassador/v0
      kind: Mapping
      name: tour-backend_mapping
      prefix: /backend/
      service: tour:8080
      rate_limits:
        - descriptor: A test case
          headers:
            - "x-ambassador-test-allow"
spec:
  ports:
  - name: ui
    port: 5000
    targetPort: 5000
  - name: backend
    port: 8080
    targetPort: 8080
  selector:
    app: tour
```

This configuration tells Ambassador about the rate limit rules to apply, notably that it needs the `x-ambassador-test-allow` header, and that it should set "A test case" as the `generic_key` descriptor when performing the gRPC request.

Note that both `descriptor` and `headers` are optional. However, if `headers` are defined, **they must be part of the request in order to be rate limited**.

Ambassador would also perform multiple requests to `example-rate-limit:5000` if we had defined multiple `rate_limits` rules on the mapping.

## 3. Test rate limiting

If we `curl` to a rate-limited URL:

```shell
$ curl -v -H "x-ambassador-test-allow: probably" $AMBASSADORURL/backend/
```

We get a 429, since we are limited.

```shell
HTTP/1.1 429 Too Many Requests
content-type: text/html; charset=utf-8
content-length: 0
```

If we set the correct header value to the service request, we will get a quote successfully:

```shell
$ curl -v -H "x-ambassador-test-allow: true" $AMBASSADORURL/backend/

TCP_NODELAY set
* Connected to 35.196.173.175 (35.196.173.175) port 80 (#0)
> GET /backed HTTP/1.1
> Host: 35.196.173.175
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< content-type: application/json
< date: Thu, 23 May 2019 15:25:06 GMT
< content-length: 172
< x-envoy-upstream-service-time: 0
< server: envoy
< 
{
    "server": "humble-blueberry-o2v493st",
    "quote": "Nihilism gambles with lives, happiness, and even destiny itself!",
    "time": "2019-05-23T15:25:06.544417902Z"
* Connection #0 to host 54.165.128.189 left intact
}
```

## More

For more details about configuring the external rate limit service, read the documentation on [external rate limit](/reference/services/rate-limit-service) and [rate_limits mapping](/reference/rate-limits).
