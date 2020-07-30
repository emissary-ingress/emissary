# Service Preview

One of the challenges in adopting Kubernetes and microservices is the development and testing workflow. Creating and maintaining a full development environment with many microservices and their dependencies is complex and hard.

Service Preview addresses this challenge by connecting your CI system or local development infrastructure to the Kubernetes cluster, and dynamically routing specific requests to your local environment.

## Installation

Service Preview is installed as an additional deployment to Ambassador Edge Stack.

See [installing Service Preview](service-preview-install) and [Service Preview in Action](service-preview-quickstart) to learn how to install and use Service Preview.

## Service Preview Components

![Preview](../../../images/service-preview.png)

There are three main components to Service Preview:

1. The [Traffic Agent](service-preview-reference#traffic-agent), which controls routing to the microservice. The Traffic Agent is deployed as a sidecar on the same pod as your microservice (behind the scenes, it's a special configuration of the basic Ambassador Edge Stack image). The Traffic Agent sidecar can be manually configured or [automatically injected by the Ambassador Injector](service-preview-reference#automatic-traffic-agent-sidecar-injection) in any pod with a specific annotation.

2. The [Traffic Manager](service-preview-reference#traffic-manager), which manages the different instances of the Traffic Agent, and is deployed in the cluster.

3. The [Edge Control](edge-control) local client, which runs in your local environment (Linux or Mac OS X). The client is the command line interface to the Traffic Manager.

See the [Service Preview reference](service-preview-reference) for more information on how these components work.
