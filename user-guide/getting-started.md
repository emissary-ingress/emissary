---
    description: In this tutorial, we'll walk through the process of deploying Ambassador in Kubernetes for ingress routing.
---
# Getting Started with the Ambassador Edge Stack

The Ambassador Edge Stack is a free comprehensive, self-service edge stack that is Kubernetes-native and built on [Envoy Proxy](https://www.envoyproxy.io/). With the Ambassador Edge Stack, application developers can *independently* manage their edge (e.g., authentication, routing, rate limiting) without centralized operational intervention, reducing toil. The Ambassador Edge Stack provides a comprehensive set of capabilities for the edge ranging from traffic management (e.g., rate limiting, load balancing), security (e.g., TLS, single sign-on, rate limiting), and developer onboarding (e.g., developer portal, Swagger/OpenAPI support).

## Install the Ambassador Edge Stack

The Ambassador Edge Stack installs in minutes. There are three general methods for installing the Ambassador Edge Stack:

* [Standard install](install). If you're new to Kubernetes and/or Ambassador, use this method.
* [Docker](../../about/quickstart). Don't have Kubernetes, but want to try out Ambassador? The Ambassador Edge Stack can run locally on your laptop in a Docker container.
* [Helm](helm). Helm is a popular package manager for Kubernetes. If you're using Helm, Ambassador Edge Stack comes pre-packaged as a Helm chart.

## More about the Ambassador Edge Stack

Want to learn more about the Ambassador Edge Stack? Read [Why Ambassador](../../about/why-ambassador) and review a summary of the features of the Ambassador Edge Stack below.

![Features](../../doc-images/features-table.jpg)

## Documentation and Help

Once you’ve installed the Ambassador Edge Stack, you can read about the best practices for configuration and usage in your production environment, along with tutorials and guides on just how to get you there.

We have several major sections to our documentation:

* [Core Concepts](../../concepts/overview) cover Ambassador's architecture and how it should be used.
* [Guides and tutorials](../../docs/guides) walk you through how to configure Ambassador for specific use cases, from rate limiting to Istio integration to gRPC.
* The [Reference](../../reference/configuration) section contains detailed documentation on configuring and managing all aspects of Ambassador.

If the documentation doesn't have the answers you need, join our [Slack community](https://d6e.co/slack)! Bug reports, feature requests, and pull requests are also gratefully accepted on our [GitHub](https://github.com/datawire/ambassador/).
