# Installing Ambassador with Helm

[Helm](https://helm.sh) is a package manager for Kubernetes that automates the release and management of software on Kubernetes. The Ambassador Edge Stack can be installed via a Helm chart with a few simple steps, depending on if you are deploying for the first time, or upgrading from an existing installation.

## Prerequisites

The Ambassador Edge Stack Helm chart is hosted by Datawire and published at https://getambassador.io.
Start by adding this repo to your helm client with:

```bash
helm repo add datawire https://www.getambassador.io
```

## First Time Installation

If you are installing the Ambassador Edge Stack for the first time on your host, complete the following directions:

1. Create the `ambassador` namespace for Ambassador Edge Stack:

```
kubectl create namespace ambassador
```

2. If you are using Helm 3, install the Ambassador Edge Stack Chart with the following command:

```
helm install ambassador --namespace ambassador datawire/ambassador
```

If you are using Helm 2, use the following command instead:

```
helm install --name ambassador --namespace ambassador datawire/ambassador
```

This will install the necessary deployments, RBAC, Custom Resource Definitions, etc. for the Ambassador Edge Stack to route traffic. Details on how to configure Ambassador using the Helm chart can be found in the Helm chart [README](https://github.com/datawire/ambassador-chart/tree/master).

**Note:** The Helm chart deploys the Ambassador Edge Stack, but it does not create the Edge Stack resources necessary for authentication, rate limiting, automatic HTTPS, etc.

## Upgrading an Existing Edge Stack Installation

**Note:** If your existing installation is not already running Ambassador **Edge Stack** as opposed to Ambassador Open Source, **do not use these instructions**. See "Migrating to Ambassador Edge Stack" below.

Upgrading an existing installation of Ambassador Edge Stack is a two-step process:

1. First, apply any CRD updates (as of Helm 3, this is not supported in the chart itself):

```
kubectl apply -f https://www.getambassador.io/yaml/aes-crds.yaml
```

2. Next, upgrade the Ambassador Edge Stack itself:

```
helm upgrade ambassador datawire/ambassador
```

This will upgrade the image and deploy and other necessary resources for the Ambassador Edge Stack. 

## Migrating to Ambassador Edge Stack

If you have an existing Ambassador Open Source installation, but are not yet running Ambassador Edge Stack, the upgrade process is somewhat different than above.

**Note:** It is strongly encouraged for you to move your Ambassador release to the `ambassador` namespace as shown below. If this isn't an option for you, remove the `--namespace ambassador` argument to `helm upgrade`.

1. Upgrade your Ambassador installation.

If you're using Helm 3, simply run

```
helm upgrade --namespace ambassador ambassador datawire/ambassador
```

If you're using Helm 2, you need to modify the command slightly:

```
helm upgrade --set crds.create=false --namespace ambassador ambassador datawire/ambassador
```

(Helm 3 will not upgrade CRDs that already exist in the cluster, and we don't want Helm 2 to upgrade them yet either.)

At this point, Ambassador Edge Stack should be running with the same functionality as Ambassador Open Source, and it's safe to do any validation required. To enable the full functionality of Ambassador Edge Stack requires two more steps:

2. Upgrade CRDs for Ambassador Edge Stack. 

To take full advantage of Ambassador Edge Stack, you'll need the new `Host` CRD, and you'll need the new `getambassador.io/v2` version of earlier CRDs. To upgrade all the CRDs, run


```
kubectl apply -f https://www.getambassador.io/yaml/aes-crds.yaml
```

3. Apply additional Ambassador Edge Stack resources.

The Helm chart deploys Ambassador Edge Stack itself, but does not create the additional resources necessary for authentication, rate limiting, automatic HTTPS, etc. To take full advantage of Ambassador Edge Stack, you'll need
create these resources with `kubectl`:

```
kubectl apply -f https://www.getambassador.io/yaml/ambassador/ambassador-edge-stack-resources.yaml # FIXME: this URL is broken
```
