# Ambassador Edge Stack Early Access Releases

From time to time, Ambassador Edge Stack may ship early access releases to test major changes. **Early access releases are not supported for production use**, but are intended to gain early feedback from our community prior to shipping a release.

Early access releases will always have names that include the string "-ea" followed by a build number, for example `0.50.0-ea1` is the first early access build of Ambassador Edge Stack 0.50.0.

## Early Access Status

There are currently no early access releases available.

### Installing Ambassador Edge Stack Early Access releases

We do not recommend Helm for early access releases. Instead, use a Kubernetes deployment as usual, but use image `quay.io/datawire/ambassador:0.50.0-ea5`.

We recommend testing with shadowing, as documented below, before switching to any new Ambassador Edge Stack release. We also recommend testing with shadowing for all early access releases before deploying in production.

## Testing with shadowing

One strategy for testing early access releases involves using Ambassador Edge Stack ID and traffic shadowing. You can do the following:

1. Install Ambassador Edge Stack Early Access on your cluster with a unique Ambassador Edge Stack ID.
2. Shadow traffic from your production Ambassador E dge Stackinstance to the Ambassador Edge Stack Early Access release.
3. Monitor the Early Access release to determine if there are any problems.


<div style="border: thick solid gray;padding:0.5em"> 

Ambassador Edge Stack is a community supported product with 
[features](getambassador.io/features) available for free and 
limited use. For unlimited access and commercial use of
Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) 
for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
</p>
