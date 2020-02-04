# Installing the Ambassador Edge Stack with Helm

[Helm](https://helm.sh) is a package manager for Kubernetes that automates the release and management of software on Kubernetes. The Ambassador Edge Stack can be installed via a Helm chart with a few simple steps, depending on if you are deploying for the first time, upgrading the Ambassador Edge Stack from an existing installation, or migrating from the Ambassador API Gateway .

## Before You Begin

The Ambassador Edge Stack Helm chart is hosted by Datawire and published at `https://www.getambassador.io`.

Start by adding this repo to your helm client with the following command:

```bash
helm repo add datawire https://www.getambassador.io
```

## Install with Helm

When you run the Helm chart, it installs the Ambassador Edge Stack. You can deploy it with either the Helm 2 or Helm 3 chart.

1. If you are installing the Ambassador Edge Stack **for the first time on your cluster**, create the `ambassador` namespace for the Ambassador Edge Stack:

   ```
   kubectl create namespace ambassador
   ```

2. **Helm 3 users:** Install the Ambassador Edge Stack Chart with the following command:

   ```
   helm install ambassador --namespace ambassador datawire/ambassador
   ```

3. **Helm 2 users**: Install the Ambassador Edge Stack Chart with the following command:

   ```
   helm install --name ambassador --namespace ambassador datawire/ambassador
   ```

4. Finish the installation by running the following command: `edgectl install`
5. Provide an email address when prompted to receive notices if your domain or TLS certificate is about to expire.

Your terminal should print something similar to the following:
```
   $ edgectl install
   -> Installing the Ambassador Edge Stack 1.0.
   -> Existing Ambassador Edge Stack installation detected.
   -> Automatically configuring TLS.
   Please enter an email address. We’ll use this email address to notify you prior to domain and certification expiration [None]: john@example.com.
   -> Obtaining a TLS certificate from Let’s Encrypt.

   Congratulations, you’ve successfully installed the Ambassador Edge Stack in your Kubernetes cluster. Visit https://random-word.edgestack.me to access your Edge Stack installation and for additional configuration.
```

[Edge Control](/reference/edge-control) (`edgectl`) automatically configures TLS for your instance and provisions a domain name for your Ambassador Edge Stack.

This will install the necessary deployments, RBAC, Custom Resource Definitions, etc. for the Ambassador Edge Stack to route traffic. Details on how to configure Ambassador using the Helm chart can be found in the Helm chart [README](https://github.com/datawire/ambassador-chart/tree/master).

## Upgrading an Existing Ambassador Edge Stack Installation

**Note:** If your existing installation is running the Ambassador API Gateway, **do not use these instructions**. See [Migrating to the Ambassador Edge Stack](#migrating-to-the-ambassador-edge-stack) instead.

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

If you have an existing Ambassador API Gateway installation but are not yet running the Ambassador Edge Stack, the upgrade process is somewhat different than above.

**Note:** It is strongly encouraged for you to move your Ambassador release to the `ambassador` namespace as shown below. If this isn't an option for you, remove the `--namespace ambassador` argument to `helm upgrade`.

1. Upgrade CRDs for the Ambassador Edge Stack.

To take full advantage of the Ambassador Edge Stack, you'll need the new `Host` CRD, and you'll need the new `getambassador.io/v2` version of earlier CRDs. To upgrade all the CRDs, run

   ```
   kubectl apply -f https://www.getambassador.io/yaml/aes-crds.yaml
   ```

2. Upgrade your Ambassador installation.

If you're using **Helm 3**, simply run

   ```
   helm upgrade --namespace ambassador ambassador datawire/ambassador
   ```

If you're using **Helm 2**, you need to modify the command slightly:

   ```
   helm upgrade --set crds.create=false --namespace ambassador ambassador datawire/ambassador
   ```

At this point, the Ambassador Edge Stack should be running with the same functionality as Ambassador API Gateway as well as the added features of the Ambassador Edge Stack. It's safe to do any validation required and roll-back if necessary.

**Note:** The Ambassador Edge Stack will be installed with an `AuthService` and `RateLimitService`. If you are using these plugins, set `authService.create=false` and/or `rateLimit.create=false` to avoid any conflict while testing the upgrade.
