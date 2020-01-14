# Installing the Ambassador Edge Stack

The Ambassador Edge Stack can be installed in a variety of ways:

## Standard Install

Kubernetes via YAML is the most common approach to install with our default, customizable manifest. Ambassador is designed to run in Kubernetes for production. If you're new to Kubernetes and/or Ambassador, we recommend using this method.
See the [Quick Start](../../user-guide/getting-started) installation guide to get started now.

[![YAML](/doc-images/kubernetes.png)](/user-guide/getting-started)

## Other Methods

You can also install Ambassador using Helm, Docker, Bare Metal, or the original Ambassador API Gateway.

| [![Helm](/doc-images/helm.png)](../../user-guide/helm) | [![Docker](/doc-images/docker.png)](/about/quickstart) | [Kubernetes Bare Metal](../../user-guide/bare-metal) | [Ambassador API Gateway](../../user-guide/install-ambassador-oss) |
| --- | --- | --- | --- |
| Helm is a package manager for Kubernetes. Ambassador comes pre-packaged as a Helm chart. [Deploy to Kubernetes via Helm.](/user-guide/helm) | The Docker install will let you try Ambassador locally in seconds, but is not supported for production. [Try via Docker.](/about/quickstart) | Bare Metal can expose the Ambassador Edge Stack if you don't have a load balancer in place. [Set up with Bare Metal.](../../user-guide/bare-metal) | Want to use the original API Gateway? [Set up the Ambassador API Gateway](../../user-guide/install-ambassador-oss).|

## Upgrade Options

If you already have the Ambassador Edge Stack, here are a few different ways you can upgrade your instance:

1. [Upgrade to the Ambassador Edge Stack from the API Gateway](../../user-guide/upgrade-to-edge-stack).
2. [Upgrade your Ambassador Instance](../../reference/upgrading) to the latest version.
3. Try out our [early access releases](../../user-guide/early-access).
