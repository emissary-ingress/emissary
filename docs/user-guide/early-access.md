# Ambassador Early Access Releases

From time to time, Ambassador may ship early access releases to test major changes. **Early access releases are not supported for production use**, but are intended to gain early feedback from our community prior to shipping a release.

Early access releases will always have names that include the string "-ea.$buildnumber", for example `0.50.0-ea.1` would be the first early access build of Ambassador 0.50.0.

## Early Access Status

There are currently no early access releases available.

### Installing Ambassador Early Access releases

We do not recommend Helm for early access releases. Instead, use a Kubernetes deployment as usual, but use image `quay.io/datawire/ambassador:0.50.0-ea.5`.

We recommend testing with shadowing, as documented below, before switching to any new Ambassador release. We also recommend testing with shadowing for all early access releases before deploying in production.
 
## Testing with shadowing

One strategy for testing early access releases involves using Ambassador ID and traffic shadowing. You can do the following:

1. Install Ambassador Early Access on your cluster with a unique Ambassador ID.
2. Shadow traffic from your production Ambassador instance to the Ambassador Early Access release.
3. Monitor the Early Access release to determine if there are any problems.
