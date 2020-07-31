# Introduction to Service Preview

One of the challenges in adopting Kubernetes and microservices is the development and testing workflow. Creating and maintaining a full development environment with many microservices and their dependencies is complex and hard.

Service Preview, based on [Telepresence](https://www.telepresence.io), enables different developers to run different virtual versions of the same microservice. These virtual versions are deployed on your CI system or local development infrastructure, enabling fast development and testing workflows.

## Getting started

To get started, follow the [installation](service-preview-install) instructions, and try the [tutorial](service-preview-tutorial).

Service Preview runs on Mac OS X, Linux, and Windows via WSL2.

### Service Preview Components

![Preview](../../../images/service-preview.png)

There are three main components to Service Preview:

1. The [Traffic Agent](service-preview-reference#traffic-agent), which controls routing to the microservice. The Traffic Agent is deployed as a sidecar on the same pod as your microservice (behind the scenes, it's a special configuration of the basic Ambassador Edge Stack image). The Traffic Agent sidecar can be manually configured or [automatically injected by the Ambassador Injector](service-preview-reference#automatic-traffic-agent-sidecar-injection-with-ambassador-injector) in any pod with a specific annotation.

2. The [Traffic Manager](service-preview-reference#traffic-manager), which manages the different instances of the Traffic Agent, and is deployed in the cluster.

3. The [Edge Control](edge-control) local client, which runs in your local environment (Linux or Mac OS X). The client is the command line interface to the Traffic Manager.

See the [Service Preview reference](service-preview-reference) for more information on how these components work.
