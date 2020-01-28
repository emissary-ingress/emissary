# Installing the Ambassador Edge Stack

The Ambassador Edge Stack can be installed in a variety of ways:

## Kubernetes

Kubernetes via YAML is the most common approach to install with our default, customizable manifest. The Ambassador Edge Stack is designed to run in Kubernetes for production. If you're new to Kubernetes and/or Ambassador, we recommend using this method.

See the [Quick Start](../../user-guide/getting-started) installation guide to get started now.

[![YAML](../../doc-images/kubernetes.png)](../../user-guide/getting-started)

## Other Methods

You can also install the Ambassador Edge Stack using Helm, Docker, or Bare Metal.

| [![Helm](../../doc-images/helm.png)](../../user-guide/helm) | [![Docker](../../doc-images/docker.png)](../../about/quickstart) | [Kubernetes Bare Metal](../../user-guide/bare-metal) |
| --- | --- | --- |
| Helm is a package manager for Kubernetes. The Ambassador Edge Stack comes pre-packaged as a Helm chart. [Install via Helm.](../../user-guide/helm) | The Docker install will let you try the Ambassador Edge Stack locally in seconds, but is not supported for production. [Try with Docker.](../../about/quickstart) | Bare Metal can expose the Ambassador Edge Stack if you don't have a load balancer in place. [Install on Bare Metal.](../../user-guide/bare-metal) |

Looking for just the API Gateway? [Install the Ambassador API Gateway](../../user-guide/install-ambassador-oss).

## Upgrade Options

If you already have the Ambassador Edge Stack, here are a few different ways you can upgrade your instance:

1. [Upgrade to the Ambassador Edge Stack from the API Gateway](../../user-guide/upgrade-to-edge-stack).
2. [Upgrade your Ambassador Edge Stack instance](../../reference/upgrading) to the latest version.
3. Try out our [early access releases](../../user-guide/early-access).
