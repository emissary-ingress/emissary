# Rate Limiting

Ambassador can validate incoming requests before routing them to a backing service. In this tutorial, we'll configure Ambassador to use an external third party rate limit service.

## Before You Get Started

This tutorial assumes you have already followed the [Ambassador Getting Started](/user-guide/getting-started) guide. If you haven't done that already, you should do that now.

After completing [Getting Started](/user-guide/getting-started), you'll have a Kubernetes cluster running Ambassador and the Quote of the Moment service. Let's walk through adding rate limiting to this setup.

## 1. Deploy the rate limit service

Ambassador delegates the actual rate limit logic to a third party service. We've written a [simple rate limit service](https://github.com/datawire/ambassador/tree/master/end-to-end/ratelimit-service) that:

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
      apiVersion: ambassador/v0
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
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: example-rate-limit
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
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

Ambassador only validates requests on Mappings which define a `rate_limits` attribute. If Ambassador cannot contact the rate limit service, it will allow the request to be processed as if there were no rate limit service configuration.

We already have the `qotm` service running, so let's apply some rate limits to the service. The easiest way to do that is to annotate the `qotm` service. While we could use `kubectl patch` for this, it's simpler to just modify the service definition and re-apply. Here's the new YAML:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: qotm
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind: Mapping
      name: qotm_mapping
      prefix: /qotm/
      service: qotm
      rate_limits:
        - descriptor: A test case
          headers:
            - "x-ambassador-test-allow"
spec:
  type: ClusterIP
  selector:
    app: qotm
  ports:
  - port: 80
    name: http-qotm
    targetPort: http-api
```

This configuration tells Ambassador about the rate limit rules to apply, notably that it needs the `x-ambassador-test-allow` header, and that it should set "A test case" as the `generic_key` descriptor when performing the gRPC request.

Note that both `descriptor` and `headers` are optional. However, if `headers` are defined, **they must be part of the request in order to be rate limited**.

Ambassador would also perform multiple requests to `example-rate-limit:5000` if we had defined multiple `rate_limits` rules on the mapping.

## 3. Test rate limiting

If we `curl` to a rate-limited URL:

```shell
$ curl -v -H "x-ambassador-test-allow: probably" $AMBASSADORURL/qotm/quote/1
```

We get a 429, since we are limited.

```shell
HTTP/1.1 429 Too Many Requests
content-type: text/html; charset=utf-8
content-length: 0
```

If we set the correct header value to the service request, we will get a quote successfully:

```shell
$ curl -v -H "x-ambassador-test-allow: true" $AMBASSADORURL/qotm/quote/1

TCP_NODELAY set
* Connected to 35.196.173.175 (35.196.173.175) port 80 (#0)
> GET /qotm/quote/1 HTTP/1.1
> Host: 35.196.173.175
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< content-type: application/json
< x-qotm-session: 069da5de-5433-46c0-a8de-d266e327d451
< content-length: 172
< server: envoy
< date: Wed, 27 Sep 2017 18:53:38 GMT
< x-envoy-upstream-service-time: 25
<
{
 \"hostname\": \"qotm-1827164760-gf534\",
 \"ok\": true,
 \"quote\": \"A late night does not make any sense.\",
 \"time\": \"2017-09-27T18:53:39.376073\",
 \"version\": \"1.1\"
}
```

## More

For more details about configuring the external rate limit service, read the documentation on [external rate limit](/reference/services/rate-limit-service) and [rate_limits mapping](/reference/rate-limits).
