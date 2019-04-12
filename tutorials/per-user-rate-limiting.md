---
title: Per User Rate Limiting
category: Security
reading_time: 5 minutes
technologies_used: Ambassador Pro
---
If you're site is attacked by a malicious user, are you prepared? 

If you notice users who are  abusing their limitless permissions and overwhelming the `httpbin` endpoint. You can ensure fairness and protect your site from vulnerabilities by enabling rate limiting based off the incoming client IP address. We do this with the `remote_address` field that Envoy configures on each request.

1. [Install Ambassador Pro](https://www.getambassador.io/user-guide/ambassador-pro-install/)

2. Configure the label in the `Mapping`

   Observe the `Mapping` in `ambassador/04-httpbin.yaml` we deployed earlier. You will see a `label` applied that labels requests to `/httpbin/` with the requests `remote_address`. Ambassador Pro's rate limiting service will use this label to identify the client IP of the request.

   ```yaml
         ---
         apiVersion: ambassador/v1
         kind:  Mapping
         name:  httpbin_mapping
         prefix: /httpbin/
         service: httpbin.org:80
         host_rewrite: httpbin.org
         labels:
           ambassador:
             - request_label_group:
               - remote_address
   ```

3. Configure the `RateLimit`

   ```
   kubectl apply -f rl-per-user.yaml
   ```

   This will tell Ambassador Pro's rate limiting service to limit the number of requests from each user to 20 requests per minute. This will stop our greedy users from issuing too many requests while not impacting the performance of our more considerate users.

4. Test The rate limit

   We provide a simple way to test that this rate limit is working. By running the `ratelimit-test.sh` script in "local-user" mode you will see that your local machine is issuing a lot of request to the system and Ambassador is responding with a 429 after 20 requests.

   ```
   ./ratelimit-test.sh local-user
   ```

   Now, from another terminal, run the `ratelimit-test.sh` script in "remote-user" mode to verify that only your local machine is being rate limited. This will issue a `kubectl exec` command to issue curl requests from Ambassador running inside your cluster. You will see these requests are allowed through even though your local machine is locked out.

   ```
   ./ratelimit-test.sh remote-user
   ```

## Summary
Protect your site from malicious users. To enable per user rate limiting quickly and easily, get started with a [free 14-day trial of Ambassador Pro](https://www.getambassador.io/pro/free-trial), or [contact sales](https://www.getambassador.io/contact) today. 
