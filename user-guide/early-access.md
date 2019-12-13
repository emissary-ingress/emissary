# Early Access Releases

Looking for early access to the Ambassador Edge Stack? [Click here to get started](../../user-guide/getting-started).

## About

From time to time, the Ambassador Edge Stack may ship early access releases to test major changes. **Early access releases are not supported for production use** but are intended to gain early feedback from our community before shipping a release.

Early access releases will always have names that include the string "-ea" followed by a build number. For example, `$version$`.

## Early Access Status

The current early access build for the Ambassador Edge Stack is `$version$`.

### Installing the Ambassador Edge Stack Early Access releases

Use a Kubernetes deployment as usual, but use the image `quay.io/datawire/aes:$version$`.

We recommend [testing with shadowing](../../reference/shadowing), as documented below, before switching to any new Ambassador Edge Stack release. We also recommend testing with shadowing for all early access releases before deploying in production.

## Testing with shadowing

One strategy for testing early access releases involves using the Ambassador Edge Stack ID and traffic shadowing. You can do the following:

1. Install the Ambassador Edge Stack Early Access on your cluster with a unique Ambassador Edge Stack ID.
2. Shadow traffic from your production Ambassador Edge Stack instance to the Ambassador Edge Stack Early Access release.
3. Monitor the Early Access release to determine if there are any problems.
