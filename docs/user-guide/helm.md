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

1. Create the `ambassador` namespace for the Ambassador Edge Stack:

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

## Upgrading an Existing Edge Stack Installation

**Note:** If your existing installation is not already running the Ambassador **Edge Stack** as opposed to Ambassador API Gateway, **do not use these instructions**. See "Migrating to the Ambassador Edge Stack" below.

Upgrading an existing installation of the Ambassador Edge Stack is a two-step process:

1. First, apply any CRD updates (as of Helm 3, this is not supported in the chart itself):

```
kubectl apply -f https://www.getambassador.io/yaml/aes-crds.yaml
```

2. Next, upgrade the Ambassador Edge Stack itself:

```
helm upgrade ambassador datawire/ambassador
```

This will upgrade the image and deploy and other necessary resources for the Ambassador Edge Stack. 

## Migrating to the Ambassador Edge Stack

If you have an existing Ambassador API Gateway installation, but are not yet running the Ambassador Edge Stack, the upgrade process is somewhat different than above.

**Note:** It is strongly encouraged for you to move your Ambassador release to the `ambassador` namespace as shown below. If this isn't an option for you, remove the `--namespace ambassador` argument to `helm upgrade`.

1. Upgrade CRDs for the Ambassador Edge Stack. 

To take full advantage of the Ambassador Edge Stack, you'll need the new `Host` CRD, and you'll need the new `getambassador.io/v2` version of earlier CRDs. To upgrade all the CRDs, run


```
kubectl apply -f https://www.getambassador.io/yaml/aes-crds.yaml
```

2. Upgrade your Ambassador installation.

If you're using Helm 3, simply run

```
helm upgrade --namespace ambassador ambassador datawire/ambassador
```

If you're using Helm 2, you need to modify the command slightly:

```
helm upgrade --set crds.create=false --namespace ambassador ambassador datawire/ambassador
```

At this point, the Ambassador Edge Stack should be running with the same functionality as Ambassador API Gateway as well as the added features of the Ambassador Edge Stack. It's safe to do any validation required and roll-back if neccessary. 

**Note:**
The Ambassador Edge Stack will be installed with an `AuthService` and `RateLimitService`. If you are using these plugins, set `authService.create=false` and/or `rateLimit.create=false` to avoid any conflict while testing the upgrade.
