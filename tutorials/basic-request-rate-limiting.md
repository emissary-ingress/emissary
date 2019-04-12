---
title: Basic Request Rate Limiting
category: Security
reading_time: 10 minutes
technologies_used: Ambassador, Prometheus
---

Suppose one of our endpoints is not very resilient to much load at all and we want to limit the amount of requests to it regardless of if the request has `x-limited-user: true` or not.

The QoTM service deployed from the Ambassador directory exposes the routes `/qotm/limited/` and `/qotm/open/`. The `/qotm/limited/` can only handle as much load as is defined by the `REQUEST_LIMIT` environment variable (defaults to 5), meaning that after 5 requests in a minute, the server will return a 500 error. The `/qotm/open/` endpoint, however, can handle higher loads.

You can test this by running the `ratelimit.sh` script in "basic" mode which sends a request every second. After the fifth request you will see the server a 500 error.

To protect our QoTM app, we need to put a rate limit on the number of requests that are allowed to the `/qotm/limited/` route.

This module configures the Pro rate limiting service.

1. [Install Ambassador Pro](https://www.getambassador.io/user-guide/ambassador-pro-install/)

2. Observe the `Mapping`s in `ambassador/05-qotm.yaml` we deployed earlier.

   You will see a `labels` applied to the `qotm_limited_mapping`. This configures Ambassador to label the request with the string `qotm`. We will configure Ambassador to `RateLimit` off this label.

   ```yaml
      ---
      apiVersion: ambassador/v1
      kind: Mapping
      name: qotm_limited_mapping
      prefix: /qotm/limited/
      rewrite: /limited/
      service: qotm
      labels:
        ambassador:
          - string_request_label:
            - qotm
   ```

   **Note:** There is no label applied to the `qotm_open_mapping`.

3. Configure the `RateLimit`:

   ```
   kubectl apply -f rl-basic.yaml
   ```

   We have now configured Ambassador to limit requests containing the label `qotm` to 5 requests per minute.

4. Test the `RateLimit`:

   ```
   ./ratelimit-test.sh basic
   ```

   This is a simple bash script that sends a `cURL` to http://$AMBASSADOR_IP/qotm/limited/ every second. You will notice that after the 5th request, Ambassador is returning a 429 instead of 200 to requests to the `/qotm/limited/` endpoint.

   The `/qotm/open/` endpoint does not have the same load restrictions and therefore does not need to be rate limited.

## Summary
This is a summary, put the summary here.