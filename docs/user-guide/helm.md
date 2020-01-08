# Installing Ambassador with Helm

&nbsp;
&nbsp;
&nbsp;

| Warning! |
| :---------------------------:|
|This installation method is not supported for Early Access release. Check back soon! |

&nbsp;
&nbsp;
&nbsp;
&nbsp;

[Helm](https://helm.sh) is a package manager for Kubernetes that automates the release and management of software on Kubernetes. The Ambassador Edge Stack can be installed via a Helm chart with a few simple steps, depending on if you are deploying for the first time, or upgrading from an existing installation.

## Prerequisites

The Ambassador Edge Stack Helm chart is hosted in the
[helm/charts](https://github.com/helm/charts) repository. Add this repo to your
helm client with:

```bash
helm repo add stable https://kubernetes-charts.storage.googleapis.com/
```

**Note** that this will be changed soon as part of the Helm 2 deprecation.

## First Time Installation

If you are installing the Ambassador Edge Stack for the first time on your host, complete the following directions:

1. Install The Ambassador Edge Stack Chart with the following command:

```
helm install --namespace ambassador ambassador stable/ambassador
```

This will install the necessary deployments, RBAC, Custom Resource Definitions, etc. for The Ambassador Edge Stack to route traffic. Details on how to configure Ambassador using the helm chart can be found in the Helm chart [README](https://github.com/helm/charts/tree/master/stable/ambassador).

If you are using `Helm 2`, use the following command instead:

```
helm install --namespace ambassador --name ambassador stable/ambassador
```

2. Enable all Ambassador Edge Stack features with the following `kubectl` command:

```
kubectl apply -f https://www.getambassador.io/early-access/yaml/ambassador/ambassador-edge-stack-resources.yaml # FIXME: this URL is broken
```

The Helm chart deploys The Ambassador Edge Stack but does not create the Edge Stack resources necessary for authentication, rate limiting, automatic HTTPS, etc.

## Upgrade an Existing Installation

Upgrading your existing Ambassador Edge Stack is as simple as upgrading any other Helm release. Complete the following steps:

1. Upgrade the release with the following command:

```
helm upgrade ambassador stable/ambassador
```

This will upgrade the image and deploy and other necessary resources for the Ambassador Edge Stack. 

**Note:** It is strongly encouraged for you to move your Ambassador release to the `ambassador` namespace with the following command: 
`helm upgrade --namespace ambassador ambassador stable/ambassador`

2. Upgrade the Custom Resource Definitions. The Ambassador Edge Stack ships with `v2` Custom Resource Definitions (CRDs). By design, Helm ignores and will not upgrade any CRDs that already exist in the cluster so you must upgrade these manually.

Do so with the following `kubectl` command:

```
kubectl apply -f https://www.getambassador.io/early-access/yaml/aes-crds.yaml
```

If you are using **Helm 2**, CRDs are managed differently and will not be ignored by default. Set `crds.create=false` in your `helm upgrade` command to ignore the CRDs.

3. Enable all Ambassador Edge Stack features. The Helm chart deploys The Ambassador Edge Stack but does not create the Edge Stack resources necessary for authentication, rate limiting, automatic HTTPS, etc. 

Create these features with `kubectl`:

```
kubectl apply -f https://www.getambassador.io/early-access/yaml/ambassador/ambassador-edge-stack-resources.yaml # FIXME: this URL is broken
```
