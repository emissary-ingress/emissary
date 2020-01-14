# Install and Upgrade Overview

## Installation Options

The Ambassador Edge Stack can be installed in a variety of ways:

### Standard Install

Kubernetes via YAML is the most common approach to install with our default, customizable manifest. Ambassador is designed to run in Kubernetes for production. If you're new to Kubernetes and/or Ambassador, we recommend using this method.

See the [Quick Start](../../user-guide/getting-started) installtion guide to get started now.

### Helm

Helm is a package manager for Kubernetes. The Ambassador Edge Stack comes pre-packaged as a Helm chart.

See the [Helm](../../user-guide/helm) instructions to try it now.

### Docker

Don't have Kubernetes, but want to try out Ambassador? The Ambassador Edge Stack can run locally on your laptop in a Docker container. The Docker install will let you try Ambassador locally in seconds, but is not supported for production.

See the [Docker](../../about/quickstart) instructions to try it now.

### Bare Metal

If you don't have a load balancer, you can use Kubernetes' Bare Metal, which allows you to expose the Ambassador Edge Stack via NodePort.

See the [Bare Metal](../../user-guide/bare-metal) instructions to try it now.

### The Ambassador API Gateway

The original Ambassador API Gateway. See the [instructions](../../user-guide/install-ambassador-oss) to try it out now.

## Upgrade Options

If you already have the Ambassador Edge Stack, here are a few different ways you can upgrade your instance:

1. [Upgrade to the Ambassador Edge Stack from the API Gateway](../../user-guide/upgrade-to-edge-stack).
2. [Upgrade your Ambassador Instance](../../reference/upgrading) to the latest version.
3. Try out our [early access releases](../../user-guide/early-access).
