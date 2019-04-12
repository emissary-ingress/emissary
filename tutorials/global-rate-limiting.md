---
title: Global Rate Limiting
category: Security
reading_time: 5 minutes
technologies_used: Ambassador Pro 
---
Imagine your website gets on the front page of Hacker News. Or, you've announced a major Black Friday sale. Generally, [ecommerce platforms generate 300% more sales on Black Friday](https://www.userreport.com/blog/e-commerce-exploit-increased-traffic-black-friday/). But if your database starts to fail under the increased load, the degraded performance can impact your conversion numbers. Or worse, your website can fail completely. To mitigate performance issues as quickly as possible while you work to scale your backend, you can use global rate limiting.

Suppose we want to limit users to only 10 requests per minute for any requests through Ambassador. We can configure a global rate limit that can rate limit based off a header that identifies this subset of users. Users with the header `x-limited-user: true` will be limited to 10 requests per minute.

1. [Install Ambassador Pro](https://www.getambassador.io/user-guide/ambassador-pro-install/)

2. In the `pro-ref-arch` directory, observe the ambassador `Module` in `ambassador/03-ambassador-service.yaml` we deployed earlier.

   You will see a `default_label` set in the config. This configures Ambassador to label every request through Ambassador with a check for `x-limited-user` so the rate limiting service can check it.

   ```yaml
         ---
         apiVersion: ambassador/v1
         kind: Module
         name: ambassador
         config:
           enable_grpc_web: True
             default_label_domain: ambassador
             default_labels:
               ambassador:
                 defaults:
                 - x_limited_user:
                     header: "x-limited-user"
                     omit_if_not_present: true
   ```

3. Configure the global rate limit

   ```
   kubectl apply -f ratelimit/rl-global.yaml
   ```

   This configures Ambassador's rate limiting service to look for the `x_limited_user` label and, if set to `true`, limit the requests to 10 per minute.

4. Test the rate limit

   We provide a simple way to test that this global rate limit is working. Run the simple shell script `ratelimit-test.sh` in "global" mode to send requests to the `qotm` and `httpbin` endpoints. You will see, after a couple of request, that requests that set `x-limited-user: true` will be returned a 429 by Ambassador after 10 requests but requests with `x-limited-user: false` are allowed.

   ```
   ratelimit/ratelimit-test.sh global
   ```

## Summary
Don't leave your website performance at risk for sudden spikes in traffic. To quickly enable global rate limiting on your website, get started with a [free 14-day trial of Ambassador Pro](https://www.getambassador.io/pro/free-trial), or [contact sales](https://www.getambassador.io/contact) today. 
