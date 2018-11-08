# Testing Safely with Production Traffic

No code is ever truly proven or tested until it’s running in production. This is particularly true of today’s cloud-native applications. Applications today are continuously deployed, and have multiple layers of dependencies. Thus, traditional strategies such as mocks, staging environments, and integration testing are no longer sufficient for testing cloud-native applications.

## Testing in production

The problem with testing in production is the possibility that real-world users could be impacted by any errors in a given update. Over the past few years, two approaches for testing against production traffic have gained increasing acceptance. These patterns are:

* Traffic shadowing: this approach “shadows” or mirrors a small amount of real user traffic from production to your application under test:
  * Although the shadowed results can be verified (both the downstream response and associated upstream side effects) they are not returned to the user -- the user only sees the results from the currently released application. Typically any side effects are not persisted or are executed as a no-op and verified (much like setting up a mock, and verifying that a method/function was called with the correct parameters).
  * This allows verification that the application is not crashing or otherwise behaving badly from an operational perspective when dealing with real user traffic and behaviour (and the larger the percentage of traffic shadowed, the higher the probability of confidence).
* Canary releasing: this approach shifts a small amount of real user traffic from production to your application under test:
  * The user will see the direct response from the canary version of application from any traffic that is shifted to the canary release, and they will not trigger or see the response from the current production released version of the application. The canary results can also be verified (both the downstream response and associated upstream side effects), but it is key to understand that any side effects will be persisted.
  * In addition to allowing verification that the application is not crashing or otherwise behaving badly from an operational perspective when dealing with real user traffic and behaviour, canary releasing allows user validation. For example, if a business KPI performs worse for all canaried requests, then this most likely indicates that the canaried application should not be fully released in its current form.

## Observability is a prerequisite for testing in production

Observability is a critical requirement for testing in production. In any canary or shadow deployment, collecting key metrics around latency, traffic, errors, and saturation (the [“Four Golden Signals of Monitoring”](https://landing.google.com/sre/sre-book/chapters/monitoring-distributed-systems/#xref_monitoring_golden-signals)) provides valuable insight into the stability and performance of a new version of the service. Moreover, application developers can compare the metrics (e.g., latency) between the production version and update version. If the metrics are similar, then updates can proceed with much greater confidence.

## Benefits of testing in production

Modern cloud applications are continuously deployed, as different teams rapidly update their respective services. Deploying and testing updates in a pre-production staging environment introduces either a bottleneck to the speed of iteration, or provides little feedback due to the fact that staging is not representative of what will be running in production when the deployment actually occurs.  Testing in production addresses both of these challenges: developers evaluate their changes in the real-world environment, enabling rapid iteration.