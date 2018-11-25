# Ambassador Early Access Releases

From time to time, Ambassador may ship early access releases to test major changes. **Early access releases are not supported for production use**, but are intended to gain early feedback from our community prior to shipping a release.

Early access releases will always have names that include the string "-ea" followed by a build number, for example `0.50.0-ea1` is the first early access build of Ambassador 0.50.0.

## Ambassador 0.50 Early Access

Ambassador 0.50 is a major revamp of Ambassador's architecture, with support for Envoy v2 configuration, ADS, and significant internal refactoring. For details on Ambassador 0.50, see the [Ambassador 0.50 Early Access blog post](https://blog.getambassador.io/announcing-ambassador-0-50-early-access-1-with-v2-config-ads-support-cd785276a60e).

### Installing Ambassador Early Access releases

We do not recommend Helm for early access releases. Instead, use a Kubernetes deployment as usual, but use image `quay.io/datawire/ambassador:0.50.0-ea5`.

We recommend testing with shadowing, as documented below, before switching to any new Ambassador release. We also recommend testing with shadowing for all early access releases before deploying in production.
 
## Testing with shadowing

One strategy for testing early access releases involves using Ambassador ID and traffic shadowing. You can do the following:

1. Install Ambassador Early Access on your cluster with a unique Ambassador ID.
2. Shadow traffic from your production Ambassador instance to the Ambassador Early Access release.
3. Monitor the Early Access release to determine if there are any problems.
