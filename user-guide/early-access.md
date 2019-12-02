# Ambassador Edge Stack Early Access Releases

From time to time, the Ambassador Edge Stack may ship early access releases to test major changes. **Early access releases are not supported for production use**, but are intended to gain early feedback from our community prior to shipping a release.

Early access releases will always have names that include the string "-ea" followed by a build number, for example `0.50.0-ea1` is the first early access build of the Ambassador Edge Stack 0.50.0.

## Early Access Status

There are currently no early access releases available.

### Installing the Ambassador Edge Stack Early Access releases

We do not recommend Helm for early access releases. Instead, use a Kubernetes deployment as usual, but use image `quay.io/datawire/ambassador:0.50.0-ea5`.

We recommend testing with shadowing, as documented below, before switching to any new Ambassador Edge Stack release. We also recommend testing with shadowing for all early access releases before deploying in production.

## Testing with shadowing

One strategy for testing early access releases involves using the Ambassador Edge Stack ID and traffic shadowing. You can do the following:

1. Install the Ambassador Edge Stack Early Access on your cluster with a unique Ambassador Edge Stack ID.
2. Shadow traffic from your production Ambassador Edge Stack instance to the Ambassador Edge Stack Early Access release.
3. Monitor the Early Access release to determine if there are any problems.
