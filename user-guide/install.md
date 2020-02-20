# Installing the Ambassador Edge Stack

The Ambassador Edge Stack can be installed in a variety of ways:

## Kubernetes

Kubernetes via YAML is the most common approach to install with our default, customizable manifest. The Ambassador Edge Stack is designed to run in Kubernetes for production. If you're new to Kubernetes and/or Ambassador, we recommend using this method.

See the [Quick Start](../../user-guide/getting-started) installation guide to get started now.

[![YAML](../../doc-images/kubernetes.png)](../../user-guide/getting-started)

## Other Methods

You can also install the Ambassador Edge Stack using Helm, Docker, Bare Metal,
the Operator, or installing manually.

| [![Helm](../../doc-images/helm.png)](../../user-guide/helm) |
[![Docker](../../doc-images/docker.png)](../../about/quickstart) | [Kubernetes
Bare Metal](../../user-guide/bare-metal) | [AES Operator](/user-guide/aes-operator) | [Manual Install](/user-guide/manual-install)
| --- | --- | --- | --- | --- |
| Helm is a package manager for Kubernetes. The Ambassador Edge Stack comes
pre-packaged as a Helm chart. [Install via Helm.](../../user-guide/helm) | The
Docker install will let you try the Ambassador Edge Stack locally in seconds,
but is not supported for production. [Try with Docker.](../../about/quickstart)
| Bare Metal can expose the Ambassador Edge Stack if you don't have a load
balancer in place. [Install on Bare Metal.](../../user-guide/bare-metal) | The
Ambassador Edge Stack Operator automates install and updates, among other
actions. [Install with the Operator](/user-guide/aes-operator) | If you want to
configure specific parameters of your installation, use the [manual installation
method](/user-guide/manual-install). |

Looking for just the API Gateway? [Install the Ambassador API Gateway](../../user-guide/install-ambassador-oss).

## Upgrade Options

If you already have the Ambassador Edge Stack, here are a few different ways you can upgrade your instance:

1. [Upgrade to the Ambassador Edge Stack from the API Gateway](../../user-guide/upgrade-to-edge-stack).
2. [Upgrade your Ambassador Edge Stack instance](../../reference/upgrading) to the latest version.
3. Try out our [early access releases](../../user-guide/early-access).
